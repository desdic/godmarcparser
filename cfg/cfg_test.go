package cfg

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadConfig(t *testing.T) {

	var tt = []struct {
		name       string
		config     string
		expected   Config
		shouldwork bool
	}{
		{"valid",
			"testdata/config.json",
			Config{
				HTTP: HTTPCfg{
					Port:         ":1234",
					WriteTimeout: 17,
					ReadTimeout:  18, IdleTimeout: 61},
				Storage: StorageCfg{
					Type: "postgresql",
					URL:  "postgres://dmarc:dmarc@127.0.0.1:5432/dmarc?sslmode=disable",
				},
				Log:       LogCfg{Level: "info"},
				Directory: ScanDirectory{Path: "/files", Interval: 45},
			}, true,
		},
		{"sanitize",
			"testdata/sanitize.json",
			Config{
				HTTP: HTTPCfg{
					Port:         ":8080",
					WriteTimeout: 15,
					ReadTimeout:  15,
					IdleTimeout:  30},
				Storage: StorageCfg{
					Type: "postgresql",
					URL:  "postgres://dmarc:dmarc@127.0.0.1:5432/dmarc?sslmode=disable",
				},
				Log:       LogCfg{Level: "info"},
				Directory: ScanDirectory{Path: "/files", Interval: 30},
			}, true,
		},
		{"missing",
			"testdata/missing.json",
			Config{},
			false,
		},
		{"notvalid",
			"testdata/notvalid.json",
			Config{},
			false,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			c, err := ReadConfig(tc.config)
			if err != nil && tc.shouldwork {
				t.Fatalf("Unable to open config file: %v", err)
			}

			if err != nil && !tc.shouldwork {
				return
			}

			if diff := cmp.Diff(c, tc.expected); diff != "" {
				t.Errorf("%v: config differs: (-want +got)\n%s", tc.expected, diff)
			}
		})
	}
}
