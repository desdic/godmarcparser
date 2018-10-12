package dmarc

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"time"
)

var brokenschema = regexp.MustCompile(`\<xs:schema[^>]*>`)
var matchschema = regexp.MustCompile(`\<\/xs:schema[^>]*>`)

// Content is the structure for processing data
type Content struct {
	From string
	Name string
	Data *bytes.Buffer
}

// Row is the dmarc row in a report
type Row struct {
	SourceIP        string
	Count           int64
	EvalDisposition string
	EvalSPFAlign    string
	EvalDKIMAalign  string
	Reason          string
	DKIMDomain      string
	DKIMResult      string
	SPFDomain       string
	SPFResult       string
	IdentifierHFrom string
}

// Rows is jus the report and the rows of a report
type Rows struct {
	Report Report
	Rows   []Row
}

// Report is the content of the report
type Report struct {
	ID                     int64
	ReportBegin            time.Time
	ReportEnd              time.Time
	PolicyDomain           string
	ReportOrg              string
	ReportID               string
	ReportEmail            string
	ReportExtraContactInfo string
	PolicyAdkim            string
	PolicyAspf             string
	PolicyP                string
	PolicySP               string
	PolicyPCT              string
	Count                  int64
	DKIMResult             string
	SPFResult              string
	Items                  int
}

// Reports is the collection of reports
type Reports struct {
	Reports    []Report
	LastPage   int
	CurPage    int
	NextPage   int
	TotalPages int
	Pages      []int
}

type dateRange struct {
	XMLName xml.Name `xml:"date_range"`
	Begin   int64    `xml:"begin"`
	End     int64    `xml:"end"`
}

type reportMetadata struct {
	XMLName          xml.Name  `xml:"report_metadata"`
	OrgName          string    `xml:"org_name"`
	Email            string    `xml:"email"`
	ExtraContactInfo string    `xml:"extra_contact_info,omitempty"`
	ReportID         string    `xml:"report_id"`
	DateRange        dateRange `xml:"date_range"`
}

type policyPublished struct {
	XMLName xml.Name `xml:"policy_published"`
	Domain  string   `xml:"domain"`
	ADKIM   string   `xml:"adkim"`
	ASPF    string   `xml:"aspf"`
	P       string   `xml:"p"`
	SP      string   `xml:"sp"`
	PCT     string   `xml:"pct"`
}

type reason struct {
	XMLName xml.Name `xml:"reason"`
	Type    string   `xml:"type"`
	Comment string   `xml:"comment"`
}

type policyEvaluated struct {
	XMLName     xml.Name `xml:"policy_evaluated"`
	Disposition string   `xml:"disposition"`
	DKIM        string   `xml:"dkim"`
	SPF         string   `xml:"spf"`
	Reasons     []reason `xml:"reason"`
}

type row struct {
	XMLName         xml.Name        `xml:"row"`
	SourceIP        string          `xml:"source_ip"`
	Count           int64           `xml:"count"`
	PolicyEvaluated policyEvaluated `xml:"policy_evaluated"`
}

type identify struct {
	XMLName    xml.Name `xml:"identifiers"`
	HeaderFrom string   `xml:"header_from"`
}

type spf struct {
	XMLName xml.Name `xml:"spf"`
	Result  string   `xml:"result"`
}

type dkim struct {
	XMLName xml.Name `xml:"dkim"`
	Result  string   `xml:"result"`
}

type authResult struct {
	XMLName xml.Name `xml:"auth_results"`
	SPF     []spf    `xml:"spf"`
	DKIM    []dkim   `xml:"dkim"`
}

type record struct {
	XMLName     xml.Name   `xml:"record"`
	Rows        []row      `xml:"row"`
	Identifiers identify   `xml:"identifiers"`
	AuthResults authResult `xml:"auth_results"`
}

// Feedback contains the reports and file information
type Feedback struct {
	XMLName         xml.Name `xml:"feedback"`
	FromFile        string
	ReportMetadata  reportMetadata  `xml:"report_metadata"`
	PolicyPublished policyPublished `xml:"policy_published"`
	Records         []record        `xml:"record"`
}

// Read a dmarc xml report
func Read(b []byte) (Feedback, error) {
	var f Feedback

	s := string(b[:])

	// It seems that some vendors has a broken schema tag added. Its not closed and should not be there
	// So this is a hack to remove it
	if !matchschema.MatchString(s) {
		s = brokenschema.ReplaceAllString(string(b[:]), "")
	}

	if err := xml.Unmarshal([]byte(s), &f); err != nil {
		return Feedback{}, err
	}
	return f, nil
}
