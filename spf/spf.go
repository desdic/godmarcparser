package spf

import (
	"fmt"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"
)

func getIP(domain string) (ips []string, err error) {
	a, err := net.LookupIP(domain)
	if err != nil {
		return nil, fmt.Errorf("Unable to lookup a: %v", err)
	}
	for _, ip := range a {
		ips = append(ips, ip.String())
		log.Debugf("Got: %v", ip.String())
	}
	return ips, nil
}

func getMXs(domain string) (ips []string, err error) {

	mx, err := net.LookupMX(domain)
	if err != nil {
		return nil, fmt.Errorf("Unable to lookup mx: %v", err)
	}

	for _, m := range mx {

		i, err := getIP(m.Host)
		if err != nil {
			return nil, err
		}

		ips = append(ips, i...)
	}
	return ips, nil
}

func toCIDR(address string) string {

	if strings.Contains(address, "/") {
		return address
	}

	if strings.Count(address, ":") < 2 {
		return address + "/32"
	}

	return address + "/128"
}

// Rule is just for key/value for a breakdown
type Rule struct {
	Key   string
	Value string
}

// BreakDown is a breakdown of the SPF record
type BreakDown struct {
	Domain   string
	Record   string
	Rules    []Rule
	Includes []BreakDown
}

// Get breakdown of SPF record
func Get(domain string) (result BreakDown, err error) {

	txt, err := net.LookupTXT(domain)
	if err != nil {
		return BreakDown{}, fmt.Errorf("Unable to get txt record for %s: %v", domain, err)
	}

	result.Domain = domain

	for t := range txt {

		if strings.HasPrefix(txt[t], "v=spf1 ") {
			result.Record = txt[t]
			elem := strings.Split(txt[t], " ")
			for _, e := range elem {
				log.Debugf("Element: %v", e)

				switch {
				case strings.HasPrefix(e, "v="):
					log.Debugf("Got %s", e)
					v := strings.Split(e, "=")[1]
					if v == "" {
						return BreakDown{}, fmt.Errorf("SPF version is missing: %v", v)
					}
					//result.Rules = append(result.Rules, Rule{"v", v})

				case e == "mx":
					log.Debugf("Got %s for current domain", e)
					ips, err := getMXs(domain)
					if err != nil {
						return BreakDown{}, fmt.Errorf("Unable to get MX record: %v", err)
					}

					var cidrs []string
					for _, i := range ips {
						cidrs = append(cidrs, toCIDR(i))
					}
					result.Rules = append(result.Rules, Rule{e, strings.Join(cidrs, ",")})

				case strings.HasPrefix(e, "mx:"):
					log.Debugf("Got %s", e)
					d := strings.Split(e, ":")[1]
					if d == "" {
						return BreakDown{}, fmt.Errorf("MX with domain is missing domain: %v", e)
					}

					ips, err := getMXs(d)
					if err != nil {
						return BreakDown{}, fmt.Errorf("Unable to get MX:%s record: %v", d, err)
					}
					var cidrs []string
					for _, i := range ips {
						cidrs = append(cidrs, toCIDR(i))
					}
					result.Rules = append(result.Rules, Rule{e, strings.Join(cidrs, ",")})

				case strings.HasPrefix(e, "a/"):
					log.Debugf("Got %s", e)

					cidr := strings.Split(e, ":")[1]
					if cidr == "" {
						return BreakDown{}, fmt.Errorf("CIDR is missing: %v", e)
					}

					ips, err := getIP(domain)
					if err != nil {
						return BreakDown{}, fmt.Errorf("Unable to get IP for %s: %v", e, err)
					}
					var cidrs []string
					for _, i := range ips {
						tmp := i
						if !strings.Contains(i, "/") {
							tmp = i + cidr
						}
						cidrs = append(cidrs, tmp)
					}
					result.Rules = append(result.Rules, Rule{e, strings.Join(cidrs, ",")})

				case strings.HasPrefix(e, "a:"):
					log.Debugf("Got %s", e)

					d := strings.Split(e, ":")[1]
					if d == "" {
						return BreakDown{}, fmt.Errorf("Domain is missing: %v", e)
					}

					ips, err := getIP(d)
					if err != nil {
						return BreakDown{}, fmt.Errorf("Unable to get IP for %s: %v", e, err)
					}
					var cidrs []string
					for _, i := range ips {
						cidrs = append(cidrs, toCIDR(i))
					}
					result.Rules = append(result.Rules, Rule{e, strings.Join(cidrs, ",")})

				case strings.HasPrefix(e, "a"):
					log.Debugf("Got %s for current domain", e)

					ips, err := getIP(domain)
					if err != nil {
						return BreakDown{}, fmt.Errorf("Unable to get IP for %s: %v", e, err)
					}
					var cidrs []string
					for _, i := range ips {
						cidrs = append(cidrs, toCIDR(i))
					}
					result.Rules = append(result.Rules, Rule{e, strings.Join(cidrs, ",")})

				case strings.HasPrefix(e, "ip4:"):
					log.Debugf("Got IPv4: %s", e)

					cidr := e[4:]
					if cidr == "" {
						return BreakDown{}, fmt.Errorf("IP/CIDR is missing: %v", e)
					}

					if !strings.Contains(cidr, "/") {
						cidr = cidr + "/32"
					}
					result.Rules = append(result.Rules, Rule{e, cidr})

				case strings.HasPrefix(e, "ip6:"):
					log.Debugf("Got IPv6: %s", e)

					cidr := e[4:]
					if cidr == "" {
						return BreakDown{}, fmt.Errorf("IP/CIDR is missing: %v", e)
					}

					if !strings.Contains(cidr, "/") {
						cidr = cidr + "/128"
					}
					result.Rules = append(result.Rules, Rule{e, cidr})

				case strings.HasPrefix(e, "include:"):
					log.Debugf("Got include:%s", e)

					d := strings.Split(e, ":")[1]
					if d == "" {
						return BreakDown{}, fmt.Errorf("Include is missing data: %v", e)
					}
					r, err := Get(d)
					if err != nil {
						return BreakDown{}, fmt.Errorf("Failed to get include:%s %v", d, err)
					}
					result.Includes = append(result.Includes, r)

					// Just ignore these cases for now
				case e == "-all",
					e == "~all",
					e == "?all":

				default:
					log.Warningf("Unhandled: %#v", e)
				}
			}
		}
	}

	return result, nil
}
