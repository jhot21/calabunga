package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"calabunga/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := config.Load()
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, 720*time.Hour, cfg.JWTMaxTTL)
	assert.Equal(t, 24*time.Hour, cfg.JWTDefaultTTL)
	assert.NotEmpty(t, cfg.JWTSecret)
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("JWT_SECRET", "mysecret")
	t.Setenv("JWT_MAX_TTL", "48h")
	t.Setenv("JWT_DEFAULT_TTL", "8h")

	cfg := config.Load()
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "mysecret", cfg.JWTSecret)
	assert.Equal(t, 48*time.Hour, cfg.JWTMaxTTL)
	assert.Equal(t, 8*time.Hour, cfg.JWTDefaultTTL)
}
