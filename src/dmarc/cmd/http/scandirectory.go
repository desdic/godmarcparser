package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"dmarc/pkg/dmarc"
	"dmarc/pkg/input"

	log "github.com/sirupsen/logrus"
)

// ScanDirectory scans for dmarc reports in various formats
func ScanDirectory(ctx context.Context, queue chan<- dmarc.Content, errors chan<- error, path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		errors <- fmt.Errorf("Unable to list files: %v", err)
		return
	}

	var i input.Handler

DONE:
	for _, f := range files {

		switch {
		case strings.HasSuffix(f.Name(), ".xml.gz"):
			i = input.GzipInput{}
		case strings.HasSuffix(f.Name(), ".zip"):
			i = input.ZipInput{}
		case strings.HasSuffix(f.Name(), ".xml"):
			i = input.XmlInput{}
		default:
			errors <- fmt.Errorf("Unknown filetype %s, skipping", f.Name())
			continue
		}

		select {
		case <-ctx.Done():
			break DONE
		default:
		}

		fname := path + "/" + f.Name()
		log.Debugf("Found %s", fname)
		err := i.Read(ctx, fname, queue)
		if err != nil {
			errors <- err
		}
	}
}
