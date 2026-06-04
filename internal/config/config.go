package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"
)

type Config struct {
	Port          string
	JWTSecret     string
	JWTMaxTTL     time.Duration
	JWTDefaultTTL time.Duration
}

func Load() Config {
	return Config{
		Port:          env("PORT", "8080"),
		JWTSecret:     env("JWT_SECRET", randomHex(32)),
		JWTMaxTTL:     envDuration("JWT_MAX_TTL", 720*time.Hour),
		JWTDefaultTTL: envDuration("JWT_DEFAULT_TTL", 24*time.Hour),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
