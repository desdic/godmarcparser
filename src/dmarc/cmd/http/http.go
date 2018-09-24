package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"dmarc/pkg/cfg"
	"dmarc/pkg/dmarc"
	"dmarc/pkg/spf"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

var (
	isdomain = regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z]{2,3})$`)
	isip     = regexp.MustCompile(`((^\s*((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\s*$)|(^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\s*$))`)
)

func statusHandler(ctx context.Context, fn func(context.Context, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(ctx, w, r)
	}
}

func handleReports(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	v := r.URL.Query()

	page := 1

	p := v.Get("page")
	if p != "" {
		i, err := strconv.Atoi(p)
		if err != nil {
			errors <- fmt.Errorf("Cannot convert page to int: %v", err)
			http.Error(w, "page is not a number", http.StatusBadRequest)
			return
		}
		page = i
	}

	if page < 1 {
		page = 1
	}

	pagesize := 30

	offset := (page - 1) * pagesize

	reports, err := s.ReadReports(ctx, offset, pagesize)
	if err != nil {
		errors <- fmt.Errorf("Unable to read reports: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	totalpages := 1
	if len(reports) > 0 {
		totalpages = int(math.Ceil(float64(reports[0].Items / pagesize)))
	}

	var pages []int

	before := page - 1
	if before < 0 {
		before = 0
	}

	if before > 3 {
		before = 3
	}

	after := page + 3
	if after > 3 {
		after = 3
	}

	if page+3 > totalpages {
		after = 1 + (totalpages - page)
	}

	for i := before; i > 0; i-- {
		pages = append(pages, page-i)
	}

	pages = append(pages, page)

	for i := 1; i < after+1; i++ {
		pages = append(pages, page+i)
	}

	data := dmarc.Reports{Reports: reports, CurPage: page, LastPage: page - 1, NextPage: page + 1, TotalPages: totalpages + 1, Pages: pages}

	tmpl, err := template.ParseFiles("templates/reports.html")
	if err != nil {
		errors <- fmt.Errorf("Unable to parse templates/reports.html: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err = tmpl.Execute(w, data); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		errors <- fmt.Errorf("Error running template: %v", err)
		return
	}
}

func handleReport(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		errors <- fmt.Errorf("Unable to convert %s to int64", vars["id"])
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	report, err := s.ReadReport(ctx, id)
	if err != nil {
		errors <- fmt.Errorf("Unable to read report %d: %v", id, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("templates/report.html")
	if err != nil {
		errors <- fmt.Errorf("Unable to parse templates/report.html: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if err = tmpl.Execute(w, report); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		errors <- fmt.Errorf("Error running template: %v", err)
		return
	}
}

func isIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}

func isIPv6(address string) bool {
	return strings.Count(address, ":") >= 2
}

type acidr struct {
	Cidr   string
	Within bool
}

type alist struct {
	Domain    string
	IP        string
	Reverse   string
	Spfrecord string
	Cidrs     []acidr
	Breakdown string
}

func flattenBreakDown(ip net.IP, b spf.BreakDown) (list []acidr) {
	for _, l := range b.Rules {
		iplist := strings.Split(l.Value, ",")
		for _, i := range iplist {
			_, ipnet, err := net.ParseCIDR(i)
			if err != nil {
				log.Errorf("Error parsing CIDR %s: %v:", i, err)
				return
			}

			within := false
			if ipnet.Contains(ip) {
				within = true
			}
			list = append(list, acidr{Cidr: i, Within: within})
		}
	}

	for _, i := range b.Includes {
		list = append(list, flattenBreakDown(ip, i)...)
	}

	return list
}

func handleAnalyse(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	domain := vars["domain"]
	ip := vars["ip"]

	if !isdomain.MatchString(domain) {
		_, err := fmt.Fprintf(w, "Got %#v but its not a valid domain name", domain)
		if err != nil {
			log.Errorf("Unable to send data: %v", err)
		}
		return
	}

	if !isip.MatchString(ip) {
		_, err := fmt.Fprintf(w, "Got %#v but its not a valid IP address", ip)
		if err != nil {
			log.Errorf("Unable to send data: %v", err)
		}
		return
	}

	spfresults, err := spf.Get(domain)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Unable to lookup IPs: %v", err)
		return
	}

	var analasis alist

	analasis.Domain = domain
	analasis.IP = ip
	analasis.Spfrecord = spfresults.Record

	l, err := net.LookupAddr(ip)
	if err != nil {
		log.Warningf("Unable to do reverse lookup on %s: %v", ip, err)
	}
	analasis.Reverse = strings.Join(l, ",")

	b, err := json.MarshalIndent(spfresults, "", "\t")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Unable to convert SPF results to json")
		return
	}

	analasis.Breakdown = string(b)

	ipcidr := ip + "/128"
	if isIPv4(ip) {
		ipcidr = ip + "/32"
	}

	ipB, _, err := net.ParseCIDR(ipcidr)
	if err != nil {
		log.Errorf("Error creating CIDR %s: %v:", ipcidr, err)
		return
	}

	analasis.Cidrs = flattenBreakDown(ipB, spfresults)

	tmpl, err := template.ParseFiles("templates/analyse.html")
	if err != nil {
		errors <- fmt.Errorf("Unable to parse templates/analyse.html: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err = tmpl.Execute(w, analasis); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		errors <- fmt.Errorf("Error running template: %v", err)
		return
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("Default handler invoked due to missing route or file")
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func httpHandler(ctx context.Context) http.Handler {

	r := mux.NewRouter()
	log.Debug("Adding /static")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Debug("Adding handler for /")
	r.HandleFunc("/", LogHTTP(statusHandler(ctx, handleReports)))

	log.Debug("Adding handler for /report")
	r.HandleFunc("/report/{id:[0-9]+}", LogHTTP(statusHandler(ctx, handleReport))).Name("report")

	log.Debug("Adding handler for /analyze")
	r.HandleFunc("/analyse/{domain:[a-z0-9.-]+}/{ip:[a-f0-9.:]+}", LogHTTP(statusHandler(ctx, handleAnalyse))).Name("analyse")

	// Default handler (Used for logging 404)
	r.NotFoundHandler = LogHTTP(http.HandlerFunc(defaultHandler))

	return r
}

func httpStart(ctx context.Context, cancel context.CancelFunc, cfg cfg.HTTPCfg) (*http.Server, chan os.Signal) {

	srv := &http.Server{
		Handler:      httpHandler(ctx),
		Addr:         cfg.Port,
		WriteTimeout: time.Second * time.Duration(cfg.WriteTimeout),
		ReadTimeout:  time.Second * time.Duration(cfg.ReadTimeout),
		IdleTimeout:  time.Second * time.Duration(cfg.IdleTimeout),
	}

	go func(srv *http.Server) {
		log.Infof("Starting webserver on port %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Errorf("ListenAndServe: %v", err)
			cancel()
			return
		}
	}(srv)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	return srv, c
}
