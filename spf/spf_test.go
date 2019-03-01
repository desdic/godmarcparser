package spf

import (
	"testing"
)

func TestGetIP(t *testing.T) {

	tt := []struct {
		name       string
		shouldwork bool
	}{
		{"example.com", true},
		{"a.b.c.d.notatoplevel", false},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			l, err := getIP(tc.name)
			if err != nil && tc.shouldwork {
				t.Fatal(err)
			}

			if err == nil && !tc.shouldwork {
				t.Fatal("Test successed but it should not have")
			}

			if !tc.shouldwork {
				return
			}

			if len(l) < 1 {
				t.Fatal("The IP list is empty")
			}

		})
	}
}

func TestGetMX(t *testing.T) {

	tt := []struct {
		name       string
		shouldwork bool
	}{
		{"nasa.gov", true},
		{"a.b.c.d.notatoplevel", false},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			l, err := getMXs(tc.name)
			if err != nil && tc.shouldwork {
				t.Fatal(err)
			}

			if err == nil && !tc.shouldwork {
				t.Fatal("Test successed but it should not have")
			}

			if !tc.shouldwork {
				return
			}

			if len(l) < 1 {
				t.Fatal("The IP list is empty")
			}

		})
	}
}

func TestToCIDR(t *testing.T) {

	tt := []struct {
		IP       string
		expected string
	}{
		{"10.0.0.1", "10.0.0.1/32"},
		{"10.0.0.1/32", "10.0.0.1/32"},
		{"127.0.0.1/8", "127.0.0.1/8"},
		{"::1", "::1/128"},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.IP, func(t *testing.T) {

			c := toCIDR(tc.IP)
			if c != tc.expected {
				t.Fatalf("Got %v expected %v", c, tc.expected)
			}

		})
	}
}

func TestGet(t *testing.T) {

	tt := []struct {
		domain   string
		expected string
	}{
		{"greyhat.dk", "v=spf1 a:send.one.com include:_custspf.one.com mx -all"},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.domain, func(t *testing.T) {

			c, err := Get(tc.domain)
			if err != nil {
				t.Fatal(err)
			}

			if c.Record != tc.expected {
				t.Fatalf("Got '%#v' expected '%#v'", c.Record, tc.expected)
			}

		})
	}
}
