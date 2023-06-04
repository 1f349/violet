package utils

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestGetBearer(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.com", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer abc")
	assert.Equal(t, "abc", GetBearer(req))
}

func TestGetBearer_Empty(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.com", nil)
	assert.NoError(t, err)
	assert.Equal(t, "", GetBearer(req))
}
