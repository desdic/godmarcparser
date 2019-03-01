package input

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/desdic/godmarcparser/dmarc"

	log "github.com/sirupsen/logrus"
)

type ZipInput struct{}

func (r ZipInput) Read(ctx context.Context, filename string, queue chan<- dmarc.Content) error {

	z, err := zip.OpenReader(filename)
	if err != nil {
		return fmt.Errorf("Unable to open file %s: %v", filename, err)
	}

	defer func() {
		if cerr := z.Close(); cerr != nil {
			log.Errorf("Unable to close %s: %v", filename, cerr)
		}
	}()

	for _, f := range z.File {
		zc, err := f.Open()
		if err != nil {
			return fmt.Errorf("Unable to read %s from %s: %v", f.Name, filename, err)
		}

		// TODO: defer in loop
		defer func() {
			if cerr := zc.Close(); cerr != nil {
				log.Errorf("Unable to close file %s within %s: %v", f.Name, filename, err)
			}
		}()

		// Skip if zip file contains anything else than xml
		if len(f.Name) < 4 || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}

		c := dmarc.Content{From: filename, Name: f.Name}
		c.Data = new(bytes.Buffer)

		if _, err = io.Copy(c.Data, zc); err != nil {
			return fmt.Errorf("Unable to extract data from %s within %s: %v", f.Name, filename, err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("Reading zip file cancelled")
		case queue <- c:
		}
	}
	return nil
}
