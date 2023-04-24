package utils

import (
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
func GetDomainWithoutPort(domain string) (string, bool) {
	// if a valid index isn't found then return false
	n := strings.LastIndexByte(domain, ':')
	if n == -1 {
		return "", false
	}
	return domain[:n], true
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
	var countDot int
	n := strings.LastIndexFunc(domain, func(r rune) bool {
		// return true if this is the second '.'
		// otherwise counts one and continues
		if r == '.' {
			if countDot == 1 {
				return true
			}
			countDot++
		}
		return false
	})
	// if a valid index isn't found then return false
	if n == -1 {
		return "", false
	}
	return domain[n+1:], true
}
