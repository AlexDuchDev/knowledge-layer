package oauth_proxy

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Server bundles the OAuth proxy's HTTP-handler dependencies. Operators wire
// it into the Fiber app through Mount(); when MCP_ENABLED=false the bridge
// can stay nil and the routes simply 404.
//
// Audit emission for oauth.* events is currently stderr-only. v0.5.1 wires
// these through the audit_events table when the MCP endpoint lands and
// operators have a reason to query them via /audit-events.
type Server struct {
	IDP         *IDPClient
	Clients     *ClientRepo
	Bridge      *TokenBridge
	SecretKey   []byte
	Issuer      string // proxy's own issuer URL (for .well-known)
	RedirectURL string // proxy's /oauth/callback URL

	// authCodes is an in-memory map of one-time codes issued at /oauth/callback
	// and consumed at /oauth/token. PKCE makes single-instance memory safe;
	// for multi-pod deployments operators must front the proxy with a sticky
	// load balancer or move this to Redis (out of scope per ADR-0014's
	// single-instance stance).
	authCodes sync.Map // code (string) -> *authCodeEntry
}

type authCodeEntry struct {
	clientID            string
	subject             uuid.UUID
	codeChallenge       string
	codeChallengeMethod string
	redirectURI         string
	scope               string
	expiresAt           time.Time
}

// Mount registers the v0.5.0 OAuth proxy endpoints on the supplied Fiber
// router. Routes that MUST be unauthenticated per RFC 8414 / 7591 are
// declared here so the principal middleware can allow-list them upstream.
func (s *Server) Mount(r fiber.Router) {
	r.Get("/.well-known/oauth-authorization-server", s.metadata)
	r.Get("/oauth/authorize", s.authorize)
	r.Post("/oauth/register", s.register)
	r.Post("/oauth/token", s.token)
	r.Get("/oauth/callback", s.callback)
}

// PublicPaths returns the paths that must skip the principal middleware.
// Returned as exact-match strings; routes_register uses these to extend the
// existing dev-header allow-list.
func PublicPaths() []string {
	return []string{
		"/.well-known/oauth-authorization-server",
		"/oauth/authorize",
		"/oauth/register",
		"/oauth/token",
		"/oauth/callback",
	}
}

// metadata implements RFC 8414 — discovery doc that points clients at our
// /oauth/* endpoints. Stable shape; clients cache on first read.
func (s *Server) metadata(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/json")
	return c.JSON(fiber.Map{
		"issuer":                                s.Issuer,
		"authorization_endpoint":                s.Issuer + "/oauth/authorize",
		"token_endpoint":                        s.Issuer + "/oauth/token",
		"registration_endpoint":                 s.Issuer + "/oauth/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_basic", "client_secret_post"},
		"code_challenge_methods_supported":      []string{"S256"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
	})
}

// register implements RFC 7591 dynamic-client-registration. Returns the
// client_id + (for confidential clients) a one-shot client_secret.
func (s *Server) register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_client_metadata: "+err.Error())
	}
	resp, err := s.Clients.Register(c.Context(), req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	log.Printf("oauth.client.registered client_id=%s name=%q method=%s", resp.ClientID, resp.ClientName, resp.TokenEndpointAuthMethod)
	c.Status(fiber.StatusCreated)
	return c.JSON(resp)
}

// authorize is the entry point of the auth-code flow. We:
//
//	1. Validate the client + redirect_uri.
//	2. Sign a state Payload that captures everything we need at /oauth/callback.
//	3. Redirect the user-agent to the IDP's authorize URL.
//
// The MCP client never sees this redirect — it follows the 302 transparently.
func (s *Server) authorize(c *fiber.Ctx) error {
	clientID := strings.TrimSpace(c.Query("client_id"))
	redirectURI := strings.TrimSpace(c.Query("redirect_uri"))
	codeChallenge := strings.TrimSpace(c.Query("code_challenge"))
	codeChallengeMethod := strings.TrimSpace(c.Query("code_challenge_method"))
	scope := strings.TrimSpace(c.Query("scope"))

	if clientID == "" || redirectURI == "" {
		return fiber.NewError(fiber.StatusBadRequest, "client_id and redirect_uri required")
	}
	if codeChallenge == "" || codeChallengeMethod == "" {
		return fiber.NewError(fiber.StatusBadRequest, "PKCE code_challenge + code_challenge_method required (S256 only)")
	}
	if codeChallengeMethod != "S256" {
		return fiber.NewError(fiber.StatusBadRequest, "code_challenge_method must be S256")
	}

	cli, _, err := s.Clients.Get(c.Context(), clientID)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	if !cli.AllowsRedirect(redirectURI) {
		return fiber.NewError(fiber.StatusBadRequest, "redirect_uri not registered for this client")
	}

	nonce, err := randomURLSafe(16)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	state, err := Sign(Payload{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		Scope:               scope,
		Nonce:               nonce,
	}, s.SecretKey)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	idpURL := s.IDP.AuthorizeURL(state, codeChallenge, codeChallengeMethod)
	return c.Redirect(idpURL, fiber.StatusFound)
}

// callback is what the IDP redirects to. We:
//
//	1. Verify the state HMAC.
//	2. Exchange the IDP code for an id_token.
//	3. Map the verified subject through TokenBridge.
//	4. Mint our own one-time auth code and redirect the MCP client back
//	   to its registered redirect_uri with code + the original state.
func (s *Server) callback(c *fiber.Ctx) error {
	rawState := strings.TrimSpace(c.Query("state"))
	idpCode := strings.TrimSpace(c.Query("code"))
	if rawState == "" || idpCode == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing state or code from IDP")
	}
	p, err := Verify(rawState, s.SecretKey)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid state: "+err.Error())
	}
	claims, err := s.IDP.Exchange(c.Context(), idpCode)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	subjectUUID, err := s.Bridge.Resolve(c.Context(), claims)
	if err != nil {
		// IDP authenticated *someone* but we don't have a KL row for them.
		// 401 (not 403) — the user has no Knowledge Layer identity.
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	code, err := randomURLSafe(24)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	s.authCodes.Store(code, &authCodeEntry{
		clientID:            p.ClientID,
		subject:             subjectUUID,
		codeChallenge:       p.CodeChallenge,
		codeChallengeMethod: p.CodeChallengeMethod,
		redirectURI:         p.RedirectURI,
		scope:               p.Scope,
		expiresAt:           time.Now().Add(2 * time.Minute),
	})

	dest, err := url.Parse(p.RedirectURI)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "redirect_uri invalid")
	}
	q := dest.Query()
	q.Set("code", code)
	// MCP clients pass their own state through /oauth/authorize; we pass it
	// back here. RFC 6749 §4.1.2.
	if cliState := strings.TrimSpace(c.Query("client_state")); cliState != "" {
		q.Set("state", cliState)
	}
	dest.RawQuery = q.Encode()
	return c.Redirect(dest.String(), fiber.StatusFound)
}

// token implements the auth-code-grant token endpoint. Validates PKCE
// code_verifier (S256), then mints a JWT signed with our SecretKey. The
// JWT's `sub` claim is the resolved KL principal UUID.
func (s *Server) token(c *fiber.Ctx) error {
	grantType := c.FormValue("grant_type")
	if grantType != "authorization_code" {
		return fiber.NewError(fiber.StatusBadRequest, "only authorization_code grant supported")
	}
	code := strings.TrimSpace(c.FormValue("code"))
	verifier := strings.TrimSpace(c.FormValue("code_verifier"))
	clientID := strings.TrimSpace(c.FormValue("client_id"))
	clientSecret := strings.TrimSpace(c.FormValue("client_secret"))
	redirectURI := strings.TrimSpace(c.FormValue("redirect_uri"))

	if code == "" || verifier == "" || clientID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code, code_verifier, client_id required")
	}

	raw, ok := s.authCodes.LoadAndDelete(code)
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "auth code invalid or already used")
	}
	entry := raw.(*authCodeEntry)
	if time.Now().After(entry.expiresAt) {
		return fiber.NewError(fiber.StatusBadRequest, "auth code expired")
	}
	if entry.clientID != clientID {
		return fiber.NewError(fiber.StatusBadRequest, "client_id mismatch")
	}
	if entry.redirectURI != redirectURI {
		return fiber.NewError(fiber.StatusBadRequest, "redirect_uri mismatch")
	}
	if !verifyPKCE(entry.codeChallenge, entry.codeChallengeMethod, verifier) {
		return fiber.NewError(fiber.StatusBadRequest, "code_verifier mismatch")
	}

	cli, secretHash, err := s.Clients.Get(c.Context(), clientID)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}
	if cli.TokenEndpointAuthMethod != "none" {
		if err := s.Clients.VerifySecret(secretHash, clientSecret); err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}
	}
	s.Clients.MarkUsed(c.Context(), clientID)

	token, err := s.signJWT(entry.subject, clientID, entry.scope)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	log.Printf("oauth.token.issued client_id=%s principal=%s scope=%q", clientID, entry.subject, entry.scope)
	c.Set("Cache-Control", "no-store")
	c.Set("Pragma", "no-cache")
	return c.JSON(fiber.Map{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        entry.scope,
	})
}

// signJWT mints the bearer the MCP client uses on subsequent /mcp calls.
// HMAC-SHA256 with the proxy's secret key — symmetric so we can verify
// without holding a KMS key. The token's `sub` claim is the KL principal
// UUID; the validating middleware (added in v0.5.1 alongside MCP) uses
// this directly with AccessEvaluator.
func (s *Server) signJWT(principal uuid.UUID, clientID, scope string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":       s.Issuer,
		"sub":       principal.String(),
		"aud":       clientID,
		"scope":     scope,
		"iat":       now.Unix(),
		"exp":       now.Add(time.Hour).Unix(),
		"client_id": clientID,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.SecretKey)
}

// VerifyBearer is the proxy's bearer-validation surface, used by the future
// MCP middleware (v0.5.1) to map a token back to a principal UUID.
func (s *Server) VerifyBearer(rawToken string) (uuid.UUID, error) {
	parsed, err := jwt.Parse(rawToken, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("oauth_proxy: unexpected jwt alg")
		}
		return s.SecretKey, nil
	})
	if err != nil || !parsed.Valid {
		return uuid.Nil, errors.New("oauth_proxy: invalid bearer")
	}
	claims, _ := parsed.Claims.(jwt.MapClaims)
	sub, _ := claims["sub"].(string)
	id, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, errors.New("oauth_proxy: bearer sub not a uuid")
	}
	return id, nil
}

// verifyPKCE recomputes the S256 code_challenge from a code_verifier and
// constant-time compares against the stored challenge.
func verifyPKCE(challenge, method, verifier string) bool {
	if method != "S256" {
		return false
	}
	h := sha256.Sum256([]byte(verifier))
	derived := base64.RawURLEncoding.EncodeToString(h[:])
	return subtleEqual(challenge, derived)
}

// subtleEqual is constant-time string equality. Avoids importing
// crypto/subtle just for this — small enough to inline.
func subtleEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
