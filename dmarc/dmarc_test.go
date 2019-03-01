package dmarc

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRead(t *testing.T) {

	tt := []struct {
		name       string
		path       string
		expected   Feedback
		shouldwork bool
	}{
		{"valid", "testdata/valid.xml",
			Feedback{XMLName: xml.Name{Space: "", Local: "feedback"}, FromFile: "", ReportMetadata: reportMetadata{XMLName: xml.Name{Space: "", Local: "report_metadata"}, OrgName: "example.com", Email: "double-bounce@example.com", ExtraContactInfo: "", ReportID: "myid123", DateRange: dateRange{XMLName: xml.Name{Space: "", Local: "date_range"}, Begin: 1534111200, End: 1534197600}}, PolicyPublished: policyPublished{XMLName: xml.Name{Space: "", Local: "policy_published"}, Domain: "greyhat.dk", ADKIM: "r", ASPF: "r", P: "quarantine", SP: "reject", PCT: "100"}, Records: []record{record{XMLName: xml.Name{Space: "", Local: "record"}, Rows: []row{row{XMLName: xml.Name{Space: "", Local: "row"}, SourceIP: "10.10.10.1", Count: 1, PolicyEvaluated: policyEvaluated{XMLName: xml.Name{Space: "", Local: "policy_evaluated"}, Disposition: "quarantine", DKIM: "fail", SPF: "fail", Reasons: []reason(nil)}}}, Identifiers: identify{XMLName: xml.Name{Space: "", Local: "identifiers"}, HeaderFrom: "greyhat.dk"}, AuthResults: authResult{XMLName: xml.Name{Space: "", Local: "auth_results"}, SPF: []spf{spf{XMLName: xml.Name{Space: "", Local: "spf"}, Result: "permerror"}}, DKIM: []dkim(nil)}}}}, true},
		{"notvalid", "testdata/notxml.xml", Feedback{}, false},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			f, err := os.Open(tc.path)
			if err != nil {
				t.Fatalf("Unable to open file: %v", err)
			}

			defer f.Close()

			b, err := ioutil.ReadAll(f)
			if err != nil {
				t.Fatalf("Unable to read file: %v", err)
			}
			result, err := Read(b)
			if err != nil && tc.shouldwork {
				t.Fatalf("Failed to read file %v and it should not fail", tc.path)
			}

			if err == nil && !tc.shouldwork {
				t.Fatal("The test should have failed but did not")
			}

			if !tc.shouldwork {
				return
			}

			if diff := cmp.Diff(result, tc.expected); diff != "" {
				t.Fatalf("%v: feedback differs: (-want +got)\n%s %#v", tc.expected, diff, result)
			}

		})
	}
}
