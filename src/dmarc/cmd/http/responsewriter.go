package main

import (
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	length int
}

// WriteHeader is just a wrapper
func (w *statusWriter) WriteHeader(status int) {
	if status == 0 {
		status = 200
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write is just a wrapper
func (w *statusWriter) Write(b []byte) (int, error) {
	// We can get a return code 0 in some cases
	if w.status == 0 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(b)
	w.length += n
	return n, err
}

// LogHTTP is a wrapper to log with status and length
func LogHTTP(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := statusWriter{ResponseWriter: w}
		handler.ServeHTTP(&sw, r)
		duration := time.Now().Sub(start)

		log.Infof("%s [%s] %s %d %d %s %s %s",
			r.RemoteAddr, start, strconv.Quote(r.Method+" "+r.RequestURI+" "+r.Proto),
			sw.status, sw.length, strconv.Quote(r.Referer()), strconv.Quote(r.UserAgent()), duration)
	}
}
