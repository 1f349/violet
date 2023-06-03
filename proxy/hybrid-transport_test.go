package proxy

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestNewHybridTransport(t *testing.T) {
	h := NewHybridTransport()
	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	assert.NoError(t, err)
	trip, err := h.SecureRoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, trip.StatusCode)
}
