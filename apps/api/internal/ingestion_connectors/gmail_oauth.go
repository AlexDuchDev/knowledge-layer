package ingestion_connectors

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// ErrGmailOAuthNotConfigured is returned when a refresh token is present but API env has no Gmail OAuth client.
var ErrGmailOAuthNotConfigured = errors.New("gmail: OAuth client not configured on server (set GMAIL_OAUTH_CLIENT_ID and GMAIL_OAUTH_CLIENT_SECRET)")

func (s *Service) resolveGmailAccessToken(ctx context.Context, cfg *gmailFeedConfig) (string, error) {
	if cfg == nil {
		return "", nil
	}
	at := strings.TrimSpace(cfg.AccessToken)
	if at != "" && !gmailTokenNeedsRefresh(cfg) {
		return at, nil
	}
	rt := strings.TrimSpace(cfg.RefreshToken)
	if rt == "" {
		if at != "" {
			return at, nil
		}
		return "", nil
	}
	if s.gmailOAuth == nil {
		return "", ErrGmailOAuthNotConfigured
	}
	tok := &oauth2.Token{
		AccessToken:  at,
		RefreshToken: rt,
		Expiry:       gmailParseExpiry(cfg.ExpiryRFC3339),
	}
	src := s.gmailOAuth.TokenSource(ctx, tok)
	newTok, err := src.Token()
	if err != nil {
		return "", err
	}
	if newTok.AccessToken != "" {
		cfg.AccessToken = newTok.AccessToken
	}
	if !newTok.Expiry.IsZero() {
		cfg.ExpiryRFC3339 = newTok.Expiry.UTC().Format(time.RFC3339)
	}
	return strings.TrimSpace(cfg.AccessToken), nil
}

func gmailTokenNeedsRefresh(cfg *gmailFeedConfig) bool {
	if strings.TrimSpace(cfg.RefreshToken) == "" {
		return false
	}
	t := gmailParseExpiry(cfg.ExpiryRFC3339)
	if t.IsZero() {
		return true
	}
	return time.Until(t) < 2*time.Minute
}

func gmailParseExpiry(iso string) time.Time {
	iso = strings.TrimSpace(iso)
	if iso == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return time.Time{}
	}
	return t
}
