package input

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/desdic/godmarcparser/dmarc"

	log "github.com/sirupsen/logrus"
)

type GzipInput struct{}

func (r GzipInput) Read(ctx context.Context, filename string, queue chan<- dmarc.Content) error {

	// Simple check for extension
	if len(filename) < 7 || !strings.HasSuffix(filename, ".xml.gz") {
		return fmt.Errorf("%s does not end with .xml.gz", filename)
	}

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("Unable to open file %s: %v", filename, err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Errorf("Unable to close %s: %v", filename, cerr)
		}
	}()

	buf := bufio.NewReader(f)
	zr, err := gzip.NewReader(buf)
	if err != nil {
		return fmt.Errorf("Unable to read file %s: %v", filename, err)
	}

	defer func() {
		if zerr := zr.Close(); zerr != nil {
			log.Errorf("Unable to close stream on %s: %v", filename, zerr)
		}
	}()

	for {
		zr.Multistream(false)

		c := dmarc.Content{From: filename, Name: zr.Name}
		c.Data = new(bytes.Buffer)

		if _, err := io.Copy(c.Data, zr); err != nil {
			return fmt.Errorf("Unable to extract data from %s within %s: %v", zr.Name, filename, err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("Reading gzip file cancelled")
		case queue <- c:
		}

		err = zr.Reset(f)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Reset failed on %s within %s: %v", zr.Name, filename, err)
		}
	}
	return nil
}
