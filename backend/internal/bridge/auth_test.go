package bridge

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveHostedAuth_BridgeToken(t *testing.T) {
	auth, err := ResolveHostedAuth("http://example.com", "secret", "", nil)
	require.NoError(t, err)
	assert.Equal(t, "secret", auth.BridgeToken)
	assert.Empty(t, auth.BearerToken)
}

func TestResolveHostedAuth_OrganizerPIN(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/auth/pin", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"jwt-token","role":"admin"}`))
	}))
	defer srv.Close()

	auth, err := ResolveHostedAuth(srv.URL, "", "1738", srv.Client())
	require.NoError(t, err)
	assert.Equal(t, "jwt-token", auth.BearerToken)
}

func TestBridgeWebSocketURL(t *testing.T) {
	url, err := BridgeWebSocketURL("https://keweenawendurance.com", "laptop-finish-1")
	require.NoError(t, err)
	assert.Equal(t, "wss://keweenawendurance.com/api/rfid/bridge?device_id=laptop-finish-1", url)
}
