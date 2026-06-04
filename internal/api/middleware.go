package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"calabunga/internal/auth"
	"calabunga/internal/config"
	"calabunga/internal/types"
)

type contextKey string

const claimsKey contextKey = "claims"

func JWTMiddleware(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "token_expired", "missing or invalid authorization header")
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.Validate(cfg.JWTSecret, tokenStr)
			if err != nil {
				if errors.Is(err, auth.ErrTokenExpired) {
					writeError(w, http.StatusUnauthorized, "token_expired", "session expired — please log in again")
					return
				}
				writeError(w, http.StatusUnauthorized, "token_expired", "invalid token")
				return
			}

			if auth.ShouldRefresh(claims) {
				creds, err := auth.Decrypt(cfg.JWTSecret, claims)
				if err == nil {
					newToken, err := auth.Issue(cfg.JWTSecret, creds,
						time.Duration(claims.OriginalTTL)*time.Second,
						cfg.JWTMaxTTL)
					if err == nil {
						w.Header().Set("X-Refresh-Token", newToken)
					}
				}
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func claimsFromCtx(r *http.Request) *auth.Claims {
	c, _ := r.Context().Value(claimsKey).(*auth.Claims)
	return c
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(types.APIError{Code: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
