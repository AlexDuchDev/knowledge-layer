package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const CookieName = "kl_session"

// SessionCookieOpts configures browser attributes for the session cookie.
type SessionCookieOpts struct {
	// Secure should be true when the API is served only over HTTPS (typical staging/production).
	Secure bool
}

// IssueSessionCookie sets a signed session cookie for userID with ttl.
func IssueSessionCookie(c *fiber.Ctx, userID uuid.UUID, secret []byte, ttl time.Duration, opts SessionCookieOpts) error {
	exp := time.Now().Add(ttl).Unix()
	payload := fmt.Sprintf("%s|%d", userID.String(), exp)
	sig := signPayload(payload, secret)
	val := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + hex.EncodeToString(sig)
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    val,
		Path:     "/",
		HTTPOnly: true,
		SameSite: "Lax",
		MaxAge:   int(ttl.Seconds()),
		Secure:   opts.Secure,
	})
	return nil
}

// ClearSessionCookie clears the session cookie; opts.Secure must match IssueSessionCookie for the browser to drop it.
func ClearSessionCookie(c *fiber.Ctx, opts SessionCookieOpts) {
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   opts.Secure,
	})
}

// SessionUserID returns user id if cookie is valid.
func SessionUserID(c *fiber.Ctx, secret []byte) (uuid.UUID, bool) {
	raw := c.Cookies(CookieName)
	if raw == "" {
		return uuid.Nil, false
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return uuid.Nil, false
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return uuid.Nil, false
	}
	payload := string(b)
	sig, err := hex.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, false
	}
	if !hmac.Equal(signPayload(payload, secret), sig) {
		return uuid.Nil, false
	}
	uidStr, expStr, ok := strings.Cut(payload, "|")
	if !ok {
		return uuid.Nil, false
	}
	expUnix, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || time.Now().Unix() > expUnix {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(uidStr)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func signPayload(payload string, secret []byte) []byte {
	m := hmac.New(sha256.New, secret)
	_, _ = m.Write([]byte(payload))
	return m.Sum(nil)
}
