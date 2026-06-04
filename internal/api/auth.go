package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"calabunga/internal/auth"
	"calabunga/internal/caldav"
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

		if err := validateCredentials(req.URL, req.Username, req.Password, q); err != nil {
			switch {
			case errors.Is(err, caldav.ErrUnauthorized):
				writeError(w, http.StatusUnauthorized, "auth_failed", "Incorrect username or password")
			case errors.Is(err, caldav.ErrNotFound):
				writeError(w, http.StatusUnprocessableEntity, "not_found", "Could not find a CalDAV server at this URL")
			default:
				writeError(w, http.StatusBadGateway, "connection_failed", "Could not connect to server — check the URL and try again")
			}
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

func validateCredentials(serverURL, username, password string, q quirks.Quirks) error {
	client := caldav.NewClient(serverURL, username, password, q)
	_, err := caldav.Discover(client, serverURL)
	if err != nil {
		if errors.Is(err, caldav.ErrUnauthorized) {
			return caldav.ErrUnauthorized
		}
		if errors.Is(err, caldav.ErrNotFound) {
			return caldav.ErrNotFound
		}
		return err
	}
	return nil
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
