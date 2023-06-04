package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAcmeChallenges(t *testing.T) {
	a := NewAcmeChallenge()
	assert.Equal(t, "", a.Get("example.com", "123"))

	// The challenge should be created
	a.Put("example.com", "123", "123abc")
	assert.Equal(t, "123abc", a.Get("example.com", "123"))

	// The challenge should be deleted
	a.Delete("example.com", "123")
	assert.Equal(t, "", a.Get("example.com", "123"))

	// This should not crash or stop execution
	a.Delete("example.com", "123")
	assert.Equal(t, "", a.Get("example.com", "123"))

	// This should not crash or stop execution
	a.Delete("www.example.com", "123")
	assert.Equal(t, "", a.Get("example.com", "123"))
}
