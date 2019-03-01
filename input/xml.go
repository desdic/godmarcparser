package input

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/desdic/godmarcparser/dmarc"
)

// xmlInput is the interface
type XmlInput struct{}

func (r XmlInput) Read(ctx context.Context, filename string, queue chan<- dmarc.Content) error {

	// Skip if not xml
	if len(filename) < 4 || !strings.HasSuffix(filename, ".xml") {
		return fmt.Errorf("%s is not a xml file", filename)
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	c := dmarc.Content{From: filename, Name: filename}
	c.Data = new(bytes.Buffer)

	c.Data = bytes.NewBuffer(data)

	select {
	case <-ctx.Done():
		return fmt.Errorf("Reading xml file cancelled")
	case queue <- c:
	}

	return nil
}
