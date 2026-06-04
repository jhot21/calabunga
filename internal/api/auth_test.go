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
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(207)
		switch r.URL.Path {
		case "/.well-known/caldav", "/":
			w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"><response><href>/</href><propstat><prop><current-user-principal><href>/principals/alice/</href></current-user-principal></prop><status>HTTP/1.1 200 OK</status></propstat></response></multistatus>`))
		case "/principals/alice/":
			w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"><response><href>/principals/alice/</href><propstat><prop><calendar-home-set xmlns="urn:ietf:params:xml:ns:caldav"><href>/calendars/alice/</href></calendar-home-set></prop><status>HTTP/1.1 200 OK</status></propstat></response></multistatus>`))
		default:
			w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
		}
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
