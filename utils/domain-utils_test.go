package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSplitDomainPort(t *testing.T) {
	domain, port, ok := SplitDomainPort("www.example.com:5612", 443)
	assert.True(t, ok, "Output should be true")
	assert.Equal(t, "www.example.com", domain)
	assert.Equal(t, int(5612), port)

	domain, port, ok = SplitDomainPort("example.com", 443)
	assert.True(t, ok, "Output should be true")
	assert.Equal(t, "example.com", domain)
	assert.Equal(t, int(443), port)
}

func TestDomainWithoutPort(t *testing.T) {
	domain := GetDomainWithoutPort("www.example.com:5612")
	assert.Equal(t, "www.example.com", domain)

	domain = GetDomainWithoutPort("example.com:443")
	assert.Equal(t, "example.com", domain)

	domain = GetDomainWithoutPort("www.example.com")
	assert.Equal(t, "www.example.com", domain)

	domain = GetDomainWithoutPort("example.com")
	assert.Equal(t, "example.com", domain)
}

func TestReplaceSubdomainWithWildcard(t *testing.T) {
	domain, ok := ReplaceSubdomainWithWildcard("www.example.com")
	assert.True(t, ok, "Output should be true")
	assert.Equal(t, "*.example.com", domain)

	domain, ok = ReplaceSubdomainWithWildcard("www.example.com:5612")
	assert.True(t, ok, "Output should be true")
	assert.Equal(t, "*.example.com:5612", domain)
}

func TestGetBaseDomain(t *testing.T) {
	domain, ok := GetParentDomain("www.example.com")
	assert.True(t, ok, "Output should be true")
	assert.Equal(t, "example.com", domain)

	domain, ok = GetParentDomain("www.example.com:5612")
	assert.True(t, ok, "Output should be true")
	assert.Equal(t, "example.com:5612", domain)
}

func TestGetTopFqdn(t *testing.T) {
	domain, ok := GetTopFqdn("www.example.com")
	assert.True(t, ok, "Output should be true")
	assert.Equal(t, "example.com", domain)

	domain, ok = GetTopFqdn("www.www.example.com")
	assert.True(t, ok, "Output should be true")
	assert.Equal(t, "example.com", domain)
}
