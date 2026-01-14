package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_WithToken(t *testing.T) {
	c := New("test-token", "my-org")

	assert.NotNil(t, c)
	assert.Implements(t, (*Client)(nil), c)
}

func TestNew_WithoutToken(t *testing.T) {
	c := New("", "my-org")

	assert.NotNil(t, c)
	assert.Implements(t, (*Client)(nil), c)
}

func TestAuthTransport_RoundTrip(t *testing.T) {
	transport := &authTransport{token: "my-secret-token"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer my-secret-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	assert.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
