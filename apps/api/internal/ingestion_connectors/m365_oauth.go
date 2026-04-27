package ingestion_connectors

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// ErrMicrosoftOAuthNotConfigured means graph_refresh_token is set but server has no Microsoft OAuth client.
var ErrMicrosoftOAuthNotConfigured = errors.New("microsoft_365: OAuth client not configured on server (set MICROSOFT_OAUTH_CLIENT_ID and MICROSOFT_OAUTH_CLIENT_SECRET)")

func (s *Service) resolveM365AccessToken(ctx context.Context, cfg *m365FeedConfig) error {
	if cfg == nil {
		return nil
	}
	at := strings.TrimSpace(cfg.AccessToken)
	if at != "" && !m365TokenNeedsRefresh(cfg) {
		return nil
	}
	rt := strings.TrimSpace(cfg.RefreshToken)
	if rt == "" {
		if at != "" {
			return nil
		}
		return errors.New("microsoft_365: no access token")
	}
	if s.m365OAuth == nil {
		return ErrMicrosoftOAuthNotConfigured
	}
	tok := &oauth2.Token{
		AccessToken:  at,
		RefreshToken: rt,
		Expiry:       m365ParseExpiry(cfg.ExpiryRFC3339),
	}
	src := s.m365OAuth.TokenSource(ctx, tok)
	newTok, err := src.Token()
	if err != nil {
		return err
	}
	if newTok.AccessToken != "" {
		cfg.AccessToken = newTok.AccessToken
	}
	if !newTok.Expiry.IsZero() {
		cfg.ExpiryRFC3339 = newTok.Expiry.UTC().Format(time.RFC3339)
	}
	return nil
}

func m365TokenNeedsRefresh(cfg *m365FeedConfig) bool {
	if strings.TrimSpace(cfg.RefreshToken) == "" {
		return false
	}
	t := m365ParseExpiry(cfg.ExpiryRFC3339)
	if t.IsZero() {
		return true
	}
	return time.Until(t) < 2*time.Minute
}

func m365ParseExpiry(iso string) time.Time {
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
