package storage

// laal+build pgsql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/desdic/godmarcparser/dmarc"

	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

// Postgresql storage
type Postgresql struct {
	db  *sql.DB
	URL string
}

// Initialize creates the tables
func (h *Postgresql) Initialize(ctx context.Context) (err error) {

	if h.URL == "" {
		return fmt.Errorf("DMARCURL environment variable is empty")
	}

	h.db, err = sql.Open("postgres", h.URL)
	if err != nil {
		return fmt.Errorf("Unable to connect to postgresql database: %v", err)
	}

	dbinit := []string{`
		CREATE TABLE IF NOT EXISTS report(
			id SERIAL PRIMARY KEY,
	        report_begin VARCHAR,
	        report_end VARCHAR,
	        policy_domain VARCHAR,
	        report_org VARCHAR,
	        report_id VARCHAR,
	        report_email VARCHAR,
	        report_extra_contact_info VARCHAR,
	        policy_adkim VARCHAR,
	        policy_aspf VARCHAR,
	        policy_p VARCHAR,
	        policy_sp VARCHAR,
	        policy_pct VARCHAR,
			UNIQUE(report_begin, report_end, report_org, report_id)
		);`, `
		CREATE TABLE IF NOT EXISTS reportrow(
			id SERIAL PRIMARY KEY,
			rid INTEGER REFERENCES report(id),
			row_ip VARCHAR,
			row_count INTEGER,
			eval_disposition VARCHAR,
			eval_spf_align VARCHAR,
			eval_dkim_align VARCHAR,
			reason VARCHAR,
			dkimdomain VARCHAR,
			dkimresult VARCHAR,
			spfdomain VARCHAR,
			spfresult VARCHAR,
			identifier_hfrom VARCHAR
		);`}

	log.Debug("Initializing postgresql")
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Unable to begin transaction: %v", err)
	}
	for _, d := range dbinit {
		_, err = tx.ExecContext(ctx, d)
		if err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				return fmt.Errorf("Error doing rollback after table creation failed: %v %v", err, rerr)
			}

			return fmt.Errorf("Unable to create table: %v", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("Unable to commit while create tables: %v", err)
	}
	return nil
}

// ReadReport fetches a report
func (h *Postgresql) ReadReport(ctx context.Context, id int64) (rs dmarc.Rows, err error) {

	queryStmt, err := h.db.PrepareContext(ctx,
		`SELECT 
				r.id,
        		r.report_begin,
        		r.report_end,
        		r.policy_domain,
        		r.report_org,
        		r.report_id,
        		r.report_email,
        		r.report_extra_contact_info,
        		r.policy_adkim,
        		r.policy_aspf,
        		r.policy_p,
        		r.policy_sp,
        		r.policy_pct,
        		SUM(rr.row_count) AS rowcount,
        		MIN(rr.dkimresult) AS dkimresult,
        		MIN(rr.spfresult) AS spfresult
		 FROM   report AS r
		 	LEFT JOIN reportrow AS rr ON r.id = rr.rid
		 		WHERE r.id = $1
		 GROUP BY r.id
		 ORDER BY r.report_begin DESC`)

	if err != nil {
		return rs, fmt.Errorf("failed to prepare report: %v", err)
	}

	defer func() {
		if qerr := queryStmt.Close(); qerr != nil {
			log.Errorf("Unable to close query: %v", qerr)
		}
	}()

	var (
		r          dmarc.Report
		begin, end int64
	)
	err = queryStmt.QueryRowContext(ctx, id).Scan(&r.ID,
		&begin,
		&end,
		&rs.Report.PolicyDomain,
		&rs.Report.ReportOrg,
		&rs.Report.ReportID,
		&rs.Report.ReportEmail,
		&rs.Report.ReportExtraContactInfo,
		&rs.Report.PolicyAdkim,
		&rs.Report.PolicyAspf,
		&rs.Report.PolicyP,
		&rs.Report.PolicySP,
		&rs.Report.PolicyPCT,
		&rs.Report.Count,
		&rs.Report.DKIMResult,
		&rs.Report.SPFResult,
	)

	if err != nil {
		return rs, fmt.Errorf("Failed to query reportrow: %v", err)
	}

	rs.Report.ReportBegin = time.Unix(begin, 0)
	rs.Report.ReportEnd = time.Unix(end, 0)

	rowStmt, err := h.db.PrepareContext(ctx,
		`SELECT
			rr.row_ip,
			rr.row_count,
			rr.eval_disposition,
			rr.eval_spf_align,
			rr.eval_dkim_align,
			rr.reason,
			rr.dkimdomain,
			rr.dkimresult,
			rr.spfdomain,
			rr.spfresult,
			rr.identifier_hfrom
		FROM reportrow rr
		WHERE rr.rid = $1`)

	if err != nil {
		return rs, fmt.Errorf("Unable to prepare reportrow: %v", err)
	}

	rows, err := rowStmt.QueryContext(ctx, id)
	switch {
	case err == sql.ErrNoRows:
		return rs, nil
	case err != nil:
		return rs, fmt.Errorf("Failed to fetch recordrows: %v", err)
	}

	for rows.Next() {

		var d dmarc.Row

		err = rows.Scan(
			&d.SourceIP,
			&d.Count,
			&d.EvalDisposition,
			&d.EvalSPFAlign,
			&d.EvalDKIMAalign,
			&d.Reason,
			&d.DKIMDomain,
			&d.DKIMResult,
			&d.SPFDomain,
			&d.SPFResult,
			&d.IdentifierHFrom,
		)
		if err != nil {
			return rs, fmt.Errorf("Unable to scan: %v", err)
		}

		if d.SPFResult == "" {
			d.SPFResult = "neutral"
		}
		if d.DKIMResult == "" {
			d.DKIMResult = "neutral"
		}

		rs.Rows = append(rs.Rows, d)
	}

	return rs, nil
}

// ReadReports fetches the list of reports paginated
func (h *Postgresql) ReadReports(ctx context.Context, offset int, pagesize int) (rs []dmarc.Report, err error) {

	queryStmt, err := h.db.PrepareContext(ctx,
		`SELECT 
				r.id,
        		r.report_begin,
        		r.report_end,
        		r.policy_domain,
        		r.report_org,
        		r.report_id,
        		r.report_email,
        		r.report_extra_contact_info,
        		r.policy_adkim,
        		r.policy_aspf,
        		r.policy_p,
        		r.policy_sp,
        		r.policy_pct,
        		SUM(rr.row_count) AS rowcount,
        		MIN(rr.dkimresult) AS dkimresult,
        		MIN(rr.spfresult) AS spfresult,
				(SELECT COUNT(*) FROM report) as items
		 FROM   report AS r
		 LEFT JOIN reportrow AS rr ON r.id = rr.rid
		 GROUP BY r.id
		 ORDER BY r.report_begin DESC OFFSET $1 LIMIT $2`)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare report: %v", err)
	}

	defer func() {
		if qerr := queryStmt.Close(); qerr != nil {
			log.Errorf("Unable to close query: %v", qerr)
		}
	}()

	rows, err := queryStmt.QueryContext(ctx, offset, pagesize)
	switch {
	case err == sql.ErrNoRows:
		return []dmarc.Report{}, nil
	case err != nil:
		return nil, fmt.Errorf("Failed to fetch rows: %v", err)
	}

	for rows.Next() {

		var (
			r          dmarc.Report
			begin, end int64
		)

		err = rows.Scan(&r.ID,
			&begin,
			&end,
			&r.PolicyDomain,
			&r.ReportOrg,
			&r.ReportID,
			&r.ReportEmail,
			&r.ReportExtraContactInfo,
			&r.PolicyAdkim,
			&r.PolicyAspf,
			&r.PolicyP,
			&r.PolicySP,
			&r.PolicyPCT,
			&r.Count,
			&r.DKIMResult,
			&r.SPFResult,
			&r.Items,
		)
		if err != nil {
			return nil, fmt.Errorf("Unable to scan: %v", err)
		}

		r.ReportBegin = time.Unix(begin, 0)
		r.ReportEnd = time.Unix(end, 0)

		if r.SPFResult == "" {
			r.SPFResult = "neutral"
		}
		if r.DKIMResult == "" {
			r.DKIMResult = "neutral"
		}

		rs = append(rs, r)
	}

	return rs, nil
}

func (h *Postgresql) Write(ctx context.Context, f dmarc.Feedback) (err error) {

	log.Debug("Preparing context for report")
	queryStmt, err := h.db.PrepareContext(ctx,
		`INSERT INTO report(
			report_begin,
			report_end,
			policy_domain,
			report_org,
			report_id,
			report_email,
			report_extra_contact_info,
			policy_adkim,
			policy_aspf,
			policy_p,
			policy_sp,
			policy_pct)
	     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) 
		 RETURNING id`)

	if err != nil {
		return fmt.Errorf("failed to prepare report: %v", err)
	}

	defer func() {
		if qerr := queryStmt.Close(); qerr != nil {
			log.Errorf("Unable to close query: %v", qerr)
		}
	}()

	log.Debug("Preparing context for reportrow")
	rowStmt, err := h.db.PrepareContext(ctx,
		`INSERT INTO reportrow(rid,
                               row_ip,
                               row_count,
                               eval_disposition,
                               eval_spf_align,
                               eval_dkim_align,
                               reason,
                               dkimdomain,
                               dkimresult,
                               spfdomain,
                               spfresult,
                               identifier_hfrom)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`)
	if err != nil {
		return fmt.Errorf("failed to prepare reportrow: %v", err)
	}

	defer func() {
		if rerr := rowStmt.Close(); rerr != nil {
			log.Errorf("Unable to close statement: %v", rerr)
		}
	}()

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Unable to start transactions: %v", err)
	}

	recordStmt := tx.StmtContext(ctx, queryStmt)

	var id int64

	err = recordStmt.QueryRowContext(ctx,
		f.ReportMetadata.DateRange.Begin,
		f.ReportMetadata.DateRange.End,
		f.PolicyPublished.Domain,
		f.ReportMetadata.OrgName,
		f.ReportMetadata.ReportID,
		f.ReportMetadata.Email,
		f.ReportMetadata.ExtraContactInfo,
		f.PolicyPublished.ADKIM,
		f.PolicyPublished.ASPF,
		f.PolicyPublished.P,
		f.PolicyPublished.SP,
		f.PolicyPublished.PCT,
	).Scan(&id)

	switch {
	case err != nil:
		if pgerr, ok := err.(*pq.Error); ok {
			if pgerr.Code == "23505" {
				log.Debug("Record already exists, skipping.")
				if err = tx.Rollback(); err != nil {
					return fmt.Errorf("Rollback failed: %v", err)
				}
				return nil
			}
		}

		if rerr := tx.Rollback(); rerr != nil {
			return fmt.Errorf("Rollback failed after failed to execute query: %v %v", err, rerr)
		}
		return fmt.Errorf("Failed to execute query: %v", err)
	}

	if id == 0 {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("Rollback failed after invalid id returned: %d %v", id, err)
		}
		return fmt.Errorf("Invalid id returned: %d", id)
	}

	for _, r := range f.Records {
		for _, rw := range r.Rows {

			var (
				dkimdomain, dkimresult string
				spfdomain, spfresult   string
			)

			if len(r.AuthResults.SPF) == 1 {
				spfdomain = f.PolicyPublished.Domain
				spfresult = r.AuthResults.SPF[0].Result
			}

			if len(r.AuthResults.DKIM) == 1 {
				dkimdomain = f.PolicyPublished.Domain
				dkimresult = r.AuthResults.DKIM[0].Result
			}

			reason := ""
			if len(rw.PolicyEvaluated.Reasons) > 0 {
				var reasons []string
				for _, rs := range rw.PolicyEvaluated.Reasons {
					reasons = append(reasons, rs.Type)
				}
				reason = strings.Join(reasons, ",")
			}

			rowtxStmt := tx.StmtContext(ctx, rowStmt)
			_, err = rowtxStmt.ExecContext(ctx,
				id,
				rw.SourceIP,
				rw.Count,
				rw.PolicyEvaluated.Disposition,
				rw.PolicyEvaluated.SPF,
				rw.PolicyEvaluated.DKIM,
				reason,
				dkimdomain,
				dkimresult,
				spfdomain,
				spfresult,
				r.Identifiers.HeaderFrom,
			)
			if err != nil {
				if rerr := tx.Rollback(); rerr != nil {
					return fmt.Errorf("Rollback failed after unable to insert into recordrow: %d %v", err, rerr)
				}
				return fmt.Errorf("Unable to insert into recordrow: %v", err)
			}
		}
	}

	log.Debug("Comitting transaction")
	return tx.Commit()
}
