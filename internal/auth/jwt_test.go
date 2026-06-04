package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"calabunga/internal/auth"
)

func TestIssueAndValidate(t *testing.T) {
	secret := "testsecret1234567890123456789012"

	creds := auth.Credentials{
		ServerURL: "https://dav.example.com",
		Username:  "alice",
		Password:  "hunter2",
		Quirks:    "baikal",
	}

	token, err := auth.Issue(secret, creds, 24*time.Hour, 720*time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	got, err := auth.Validate(secret, token)
	require.NoError(t, err)
	assert.Equal(t, creds.ServerURL, got.ServerURL)
	assert.Equal(t, creds.Quirks, got.Quirks)

	decrypted, err := auth.Decrypt(secret, got)
	require.NoError(t, err)
	assert.Equal(t, creds.Username, decrypted.Username)
	assert.Equal(t, creds.Password, decrypted.Password)
}

func TestValidate_Expired(t *testing.T) {
	secret := "testsecret1234567890123456789012"
	creds := auth.Credentials{ServerURL: "https://dav.example.com", Username: "u", Password: "p"}

	token, err := auth.Issue(secret, creds, -1*time.Second, 720*time.Hour)
	require.NoError(t, err)

	_, err = auth.Validate(secret, token)
	assert.ErrorIs(t, err, auth.ErrTokenExpired)
}

func TestShouldRefresh(t *testing.T) {
	secret := "testsecret1234567890123456789012"
	creds := auth.Credentials{ServerURL: "https://dav.example.com", Username: "u", Password: "p"}

	token, err := auth.Issue(secret, creds, 1*time.Hour, 720*time.Hour)
	require.NoError(t, err)

	claims, err := auth.Validate(secret, token)
	require.NoError(t, err)

	assert.False(t, auth.ShouldRefresh(claims))
}

func TestTTLCapEnforced(t *testing.T) {
	secret := "testsecret1234567890123456789012"
	creds := auth.Credentials{ServerURL: "https://dav.example.com", Username: "u", Password: "p"}

	token, err := auth.Issue(secret, creds, 60*24*time.Hour, 30*24*time.Hour)
	require.NoError(t, err)

	claims, err := auth.Validate(secret, token)
	require.NoError(t, err)

	actualTTL := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	assert.LessOrEqual(t, actualTTL, 30*24*time.Hour)
}
