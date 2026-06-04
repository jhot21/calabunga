package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"calabunga/internal/api"
	"calabunga/internal/auth"
	"calabunga/internal/config"
)

func TestMiddleware_MissingToken(t *testing.T) {
	cfg := config.Config{JWTSecret: "testsecret12345678901234567890ab", JWTMaxTTL: 720 * time.Hour}
	handler := api.JWTMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/calendars", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_ValidToken(t *testing.T) {
	secret := "testsecret12345678901234567890ab"
	cfg := config.Config{JWTSecret: secret, JWTMaxTTL: 720 * time.Hour, JWTDefaultTTL: 24 * time.Hour}

	creds := auth.Credentials{ServerURL: "https://dav.example.com", Username: "u", Password: "p"}
	token, _ := auth.Issue(secret, creds, 24*time.Hour, 720*time.Hour)

	reached := false
	handler := api.JWTMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/calendars", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.True(t, reached)
	assert.Equal(t, 200, w.Code)
}
