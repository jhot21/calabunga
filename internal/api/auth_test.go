package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"calabunga/internal/api"
	"calabunga/internal/config"
	"calabunga/internal/types"
)

func TestConnect_Success(t *testing.T) {
	caldavSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "alice" || pass != "secret" {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Server", "SomeServer/1.0")
		w.WriteHeader(200)
	}))
	defer caldavSrv.Close()

	cfg := config.Config{
		JWTSecret:     "testsecret12345678901234567890ab",
		JWTMaxTTL:     720 * time.Hour,
		JWTDefaultTTL: 24 * time.Hour,
		Port:          "8080",
	}

	handler := api.NewConnectHandler(cfg)

	body, _ := json.Marshal(types.ConnectRequest{
		URL:      caldavSrv.URL,
		Username: "alice",
		Password: "secret",
		TTL:      "24h",
	})
	req := httptest.NewRequest("POST", "/api/connect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp types.ConnectResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.ExpiresAt)
}

func TestConnect_BadCredentials(t *testing.T) {
	caldavSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer caldavSrv.Close()

	cfg := config.Config{
		JWTSecret:     "testsecret12345678901234567890ab",
		JWTMaxTTL:     720 * time.Hour,
		JWTDefaultTTL: 24 * time.Hour,
	}
	handler := api.NewConnectHandler(cfg)

	body, _ := json.Marshal(types.ConnectRequest{
		URL: caldavSrv.URL, Username: "alice", Password: "wrong", TTL: "24h",
	})
	req := httptest.NewRequest("POST", "/api/connect", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
