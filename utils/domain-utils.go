package utils

import (
	"golang.org/x/net/publicsuffix"
	"strconv"
	"strings"
)

// SplitDomainPort takes an input host and default port then outputs the domain,
// port and true or empty values and false if the split failed
func SplitDomainPort(host string, defaultPort int) (domain string, port int, ok bool) {
	a := strings.SplitN(host, ":", 2)
	switch len(a) {
	case 2:
		domain = a[0]
		p, err := strconv.Atoi(a[1])
		port = p
		ok = err == nil
	case 1:
		domain = a[0]
		port = defaultPort
		ok = true
	}
	return
}

// GetDomainWithoutPort takes an input domain + port and outputs the domain
// without the port.
//
// example.com:443 => example.com
func GetDomainWithoutPort(domain string) string {
	// if a valid index isn't found then return false
	n := strings.LastIndexByte(domain, ':')
	if n == -1 {
		return domain
	}
	return domain[:n]
}

// ReplaceSubdomainWithWildcard returns the domain with the subdomain replaced
// with a wildcard '*' character.
//
// www.example.com => *.example.com
func ReplaceSubdomainWithWildcard(domain string) (string, bool) {
	// if a valid index isn't found then return false
	n := strings.IndexByte(domain, '.')
	if n == -1 {
		return "", false
	}
	return "*" + domain[n:], true
}

// GetParentDomain returns the parent domain stripping off the subdomain.
//
// www.example.com => example.com
func GetParentDomain(domain string) (string, bool) {
	// if a valid index isn't found then return false
	n := strings.IndexByte(domain, '.')
	if n == -1 {
		return "", false
	}
	return domain[n+1:], true
}

// GetTopFqdn returns the top domain stripping off multiple layers of subdomains.
//
// hello.world.example.com => example.com
func GetTopFqdn(domain string) (string, bool) {
	out, err := publicsuffix.EffectiveTLDPlusOne(domain)
	return out, err == nil
}

// SplitHostPath extracts the host/path from the input
func SplitHostPath(a string) (host, path string) {
	// check if source has path
	n := strings.IndexByte(a, '/')
	if n == -1 {
		// set host then path to /
		host = a
		path = "/"
	} else {
		// set host then custom path
		host = a[:n]
		path = a[n:] // this required to keep / at the start of the path
	}
	return
}

// SplitHostPathQuery extracts the host/path?query from the input
func SplitHostPathQuery(a string) (host, path, query string) {
	host, path = SplitHostPath(a)
	if path == "/" {
		n := strings.IndexByte(host, '?')
		if n != -1 {
			query = host[n+1:]
			host = host[:n]
		}
		return
	}
	n := strings.IndexByte(path, '?')
	if n != -1 {
		query = path[n+1:]
		path = path[:n] // reassign happens after
	}
	return
}
