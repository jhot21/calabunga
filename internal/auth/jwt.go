package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrTokenExpired = errors.New("token expired")

type Credentials struct {
	ServerURL string
	Username  string
	Password  string
	Quirks    string
}

type Claims struct {
	jwt.RegisteredClaims
	ServerURL   string `json:"url"`
	EncCreds    string `json:"creds"`
	Quirks      string `json:"quirks"`
	OriginalTTL int64  `json:"orig_ttl"`
}

func Issue(secret string, creds Credentials, requestedTTL, maxTTL time.Duration) (string, error) {
	ttl := requestedTTL
	if ttl > maxTTL {
		ttl = maxTTL
	}

	encCreds, err := encryptCreds(secret, creds.Username, creds.Password)
	if err != nil {
		return "", fmt.Errorf("encrypt creds: %w", err)
	}

	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		ServerURL:   creds.ServerURL,
		EncCreds:    encCreds,
		Quirks:      creds.Quirks,
		OriginalTTL: int64(ttl.Seconds()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func Validate(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func Decrypt(secret string, claims *Claims) (Credentials, error) {
	user, pass, err := decryptCreds(secret, claims.EncCreds)
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{
		ServerURL: claims.ServerURL,
		Username:  user,
		Password:  pass,
		Quirks:    claims.Quirks,
	}, nil
}

func ShouldRefresh(claims *Claims) bool {
	if claims.OriginalTTL == 0 {
		return false
	}
	remaining := time.Until(claims.ExpiresAt.Time)
	half := time.Duration(claims.OriginalTTL/2) * time.Second
	return remaining < half
}

func encryptCreds(secret, username, password string) (string, error) {
	key := deriveKey(secret)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	plaintext := []byte(username + ":" + password)
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptCreds(secret, encoded string) (username, password string, err error) {
	key := deriveKey(secret)
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return "", "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(string(plaintext), ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New("malformed credentials")
	}
	return parts[0], parts[1], nil
}

func deriveKey(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}
