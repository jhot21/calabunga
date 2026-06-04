package api

import (
	"encoding/json"
	"net/http"
	"time"

	"calabunga/internal/auth"
	"calabunga/internal/config"
	"calabunga/internal/quirks"
	"calabunga/internal/types"
)

func NewConnectHandler(cfg config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req types.ConnectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}

		q := quirks.Detect(req.URL, req.Username, req.Password)

		if !validateCredentials(req.URL, req.Username, req.Password) {
			writeError(w, http.StatusUnauthorized, "auth_failed", "could not authenticate with CalDAV server")
			return
		}

		ttl := parseTTL(req.TTL, cfg.JWTDefaultTTL)
		creds := auth.Credentials{
			ServerURL: req.URL,
			Username:  req.Username,
			Password:  req.Password,
			Quirks:    q.Name(),
		}
		token, err := auth.Issue(cfg.JWTSecret, creds, ttl, cfg.JWTMaxTTL)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "failed to issue token")
			return
		}

		expiresAt := time.Now().Add(minDuration(ttl, cfg.JWTMaxTTL))
		writeJSON(w, http.StatusOK, types.ConnectResponse{
			Token:     token,
			ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
		})
	})
}

func validateCredentials(serverURL, username, password string) bool {
	req, err := http.NewRequest("OPTIONS", serverURL, nil)
	if err != nil {
		return false
	}
	req.SetBasicAuth(username, password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden
}

func parseTTL(s string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
