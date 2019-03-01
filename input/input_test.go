package input

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/desdic/godmarcparser/dmarc"
)

func TestInput(t *testing.T) {

	tt := []struct {
		name       string
		filename   string
		handler    Handler
		timeout    time.Duration
		expected   string
		shouldwork bool
	}{
		{"not_xml", "testdata/text.text", XmlInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", false},
		{"not_zip", "testdata/text.text", ZipInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", false},
		{"not_gz", "testdata/text.text", GzipInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", false},
		{"valid_xml", "testdata/valid.xml", XmlInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", true},
		{"missing_xml", "testdata/missing.xml", XmlInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", false},
		{"valid_zip", "testdata/valid.zip", ZipInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", true},
		{"missing_zip", "testdata/missing.zip", ZipInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", false},
		{"corrupt_zip", "testdata/corrupt.zip", ZipInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", false},
		{"valid_gz", "testdata/valid.xml.gz", GzipInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", true},
		{"missing_gz", "testdata/missing.xml.gz", GzipInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", false},
		{"corrupt_gz", "testdata/corrupt.xml.gz", GzipInput{}, 300 * time.Second, "b5d12fa6e477a00d62bfa1c09896a1de", false},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			queue := make(chan dmarc.Content)

			go func(expected string) {
				for q := range queue {
					h := md5.New()
					io.WriteString(h, q.Data.String())
					result := fmt.Sprintf("%x", h.Sum(nil))
					if result != expected {
						fmt.Printf("Content is not correct: %v vs %v", result, tc.expected)
					}
				}
			}(tc.expected)

			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			err := tc.handler.Read(ctx, tc.filename, queue)
			if err != nil && tc.shouldwork {
				t.Fatalf("Unable to read file: %v", err)
			}

			if err == nil && !tc.shouldwork {
				t.Fatal("Test worked but should have failed")
			}

			if !tc.shouldwork {
				return
			}

		})
	}
}
