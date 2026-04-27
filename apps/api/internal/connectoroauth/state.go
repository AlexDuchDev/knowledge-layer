package connectoroauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Payload is embedded in the OAuth state parameter (signed).
type Payload struct {
	Principal uuid.UUID `json:"p"`
	Provider  string    `json:"o"`
	ExpUnix   int64     `json:"e"`
	Nonce     string    `json:"n"`
}

// SignState returns an opaque state string for OAuth authorize URLs.
func SignState(secret []byte, principal uuid.UUID, provider string, ttl time.Duration) (string, error) {
	if len(secret) < 16 {
		return "", errors.New("oauth state: secret must be at least 16 bytes")
	}
	pl := Payload{
		Principal: principal,
		Provider:  provider,
		ExpUnix:   time.Now().Add(ttl).Unix(),
		Nonce:     uuid.NewString(),
	}
	raw, err := json.Marshal(pl)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(raw)
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sig) + "." + base64.RawURLEncoding.EncodeToString(raw), nil
}

// VerifyState parses and authenticates a state string.
func VerifyState(secret []byte, state string) (Payload, error) {
	var zero Payload
	if len(secret) < 16 {
		return zero, errors.New("oauth state: secret must be at least 16 bytes")
	}
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return zero, errors.New("oauth state: malformed")
	}
	sigWant, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return zero, fmt.Errorf("oauth state: sig: %w", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return zero, fmt.Errorf("oauth state: body: %w", err)
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(raw)
	if !hmac.Equal(sigWant, mac.Sum(nil)) {
		return zero, errors.New("oauth state: bad signature")
	}
	var pl Payload
	if err := json.Unmarshal(raw, &pl); err != nil {
		return zero, err
	}
	if time.Now().Unix() > pl.ExpUnix {
		return zero, errors.New("oauth state: expired")
	}
	if pl.Principal == uuid.Nil || pl.Provider == "" {
		return zero, errors.New("oauth state: invalid payload")
	}
	return pl, nil
}
