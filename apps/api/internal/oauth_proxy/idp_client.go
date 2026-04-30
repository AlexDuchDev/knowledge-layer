package oauth_proxy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// IDPConfig holds the operator-supplied OIDC configuration. Issuer must be
// reachable at startup; ClientID/Secret are this proxy's identity at the IDP.
type IDPConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string // proxy's /oauth/callback URL
	Scopes       []string
}

// IDPClient wraps the OIDC provider + an oauth2.Config built from it. The
// type's job is twofold: (a) build the authorize URL we redirect the MCP
// client to, (b) exchange the IDP's authorization code for an ID token + map
// the token's `sub` claim through TokenBridge to a KL principal.
type IDPClient struct {
	provider    *oidc.Provider
	verifier    *oidc.IDTokenVerifier
	oauthConfig *oauth2.Config
}

// NewIDPClient performs OIDC discovery against the issuer URL. Errors here
// fail-start in production; in local dev we return an explanatory error and
// let the caller decide whether to disable the proxy or boot without it.
func NewIDPClient(ctx context.Context, cfg IDPConfig) (*IDPClient, error) {
	if strings.TrimSpace(cfg.Issuer) == "" {
		return nil, errors.New("oauth_proxy: OIDC issuer URL required")
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, errors.New("oauth_proxy: OIDC client_id required")
	}
	dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	prov, err := oidc.NewProvider(dctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oauth_proxy: oidc discovery: %w", err)
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	return &IDPClient{
		provider: prov,
		verifier: prov.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     prov.Endpoint(),
			Scopes:       scopes,
		},
	}, nil
}

// AuthorizeURL builds the URL the proxy redirects MCP clients to so they
// can complete the IDP-side login. PKCE params are forwarded so the IDP
// (or the user-agent's PKCE-aware UA) can validate code_verifier on token
// exchange. The state arg is our signed Payload (state.go).
func (c *IDPClient) AuthorizeURL(state, codeChallenge, codeChallengeMethod string) string {
	opts := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOnline,
	}
	if codeChallenge != "" {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", codeChallengeMethod),
		)
	}
	return c.oauthConfig.AuthCodeURL(state, opts...)
}

// Exchange swaps the IDP's auth code for an oauth2.Token + verifies the
// included id_token. Returns the verified subject (`sub` claim) which the
// TokenBridge maps to a KL principal.
type IDTokenClaims struct {
	Subject string
	Email   string
	Name    string
}

func (c *IDPClient) Exchange(ctx context.Context, code string) (*IDTokenClaims, error) {
	tok, err := c.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oauth_proxy: idp token exchange: %w", err)
	}
	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, errors.New("oauth_proxy: idp response missing id_token")
	}
	idTok, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("oauth_proxy: id_token verify: %w", err)
	}
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idTok.Claims(&claims); err != nil {
		return nil, fmt.Errorf("oauth_proxy: id_token claims: %w", err)
	}
	if claims.Sub == "" {
		return nil, errors.New("oauth_proxy: id_token missing sub claim")
	}
	return &IDTokenClaims{Subject: claims.Sub, Email: claims.Email, Name: claims.Name}, nil
}
