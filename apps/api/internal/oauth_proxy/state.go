// Package oauth_proxy implements a stateless OAuth 2.1 authorization-server
// proxy that fronts an operator-configured OIDC issuer. Inspired by Hugr's
// /oauth/* endpoints — the goal is to let MCP clients (Claude Desktop, Cursor)
// authenticate against the operator's existing IDP (Keycloak / Auth0 / Okta /
// Dex) without any server-side session state.
//
// State round-trip is the security-critical piece: the `state` query param
// shipped to the IDP must come back unmolested in the callback. We HMAC-sign
// it (with a per-instance OAUTH_SECRET_KEY) and reject anything that fails
// verification. The protected payload is small — original PKCE code_challenge,
// the MCP client's redirect_uri, and a per-flow nonce — so we don't need to
// page in any DB row to complete the callback.
package oauth_proxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

// stateMaxAge bounds how long an in-flight authorization can take. Most IDPs
// enforce a similar ceiling; ours mirrors the typical OAuth flow envelope.
const stateMaxAge = 10 * time.Minute

// Payload is what we hand the IDP via the `state` query param and expect
// back in the callback. Encoded as base64url(JSON || HMAC).
type Payload struct {
	// ClientID is the MCP client's id (from /oauth/register).
	ClientID string `json:"c"`
	// RedirectURI is the MCP client's redirect_uri — verified verbatim
	// against the registered set on callback before we forward the code.
	RedirectURI string `json:"r"`
	// CodeChallenge is the PKCE code_challenge from the original request,
	// verified again at /oauth/token when the client supplies code_verifier.
	CodeChallenge string `json:"cc"`
	// CodeChallengeMethod is "S256" (we reject "plain" upstream).
	CodeChallengeMethod string `json:"cm"`
	// Scope is the requested scope string (passed through to the IDP and
	// also pinned into the issued token).
	Scope string `json:"s"`
	// Nonce defends against state replay across flows.
	Nonce string `json:"n"`
	// IssuedAt limits replay window via stateMaxAge.
	IssuedAt int64 `json:"iat"`
}

// Sign produces an opaque, tamper-evident encoding of the payload bound to
// the operator's OAUTH_SECRET_KEY. Format: base64url(JSON).base64url(HMAC).
func Sign(p Payload, key []byte) (string, error) {
	if len(key) < 32 {
		return "", errors.New("oauth_proxy: OAUTH_SECRET_KEY must be at least 32 bytes")
	}
	if p.IssuedAt == 0 {
		p.IssuedAt = time.Now().Unix()
	}
	body, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	bodyB64 := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(bodyB64))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return bodyB64 + "." + sig, nil
}

// Verify decodes a signed state and rejects anything that fails HMAC, parses
// to a malformed payload, or has aged past stateMaxAge. Returns the payload
// on success; never returns a partial value on error.
func Verify(token string, key []byte) (*Payload, error) {
	if len(key) < 32 {
		return nil, errors.New("oauth_proxy: OAUTH_SECRET_KEY must be at least 32 bytes")
	}
	dot := -1
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			dot = i
			break
		}
	}
	if dot <= 0 || dot >= len(token)-1 {
		return nil, errors.New("oauth_proxy: malformed state")
	}
	bodyB64 := token[:dot]
	sigB64 := token[dot+1:]
	expected := hmac.New(sha256.New, key)
	expected.Write([]byte(bodyB64))
	expectedSig := base64.RawURLEncoding.EncodeToString(expected.Sum(nil))
	// Constant-time compare so a timing oracle can't enumerate valid sigs.
	if !hmac.Equal([]byte(sigB64), []byte(expectedSig)) {
		return nil, errors.New("oauth_proxy: state signature mismatch")
	}
	body, err := base64.RawURLEncoding.DecodeString(bodyB64)
	if err != nil {
		return nil, errors.New("oauth_proxy: state body decode")
	}
	var p Payload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, errors.New("oauth_proxy: state body parse")
	}
	if p.IssuedAt == 0 {
		return nil, errors.New("oauth_proxy: state missing issued_at")
	}
	age := time.Since(time.Unix(p.IssuedAt, 0))
	if age < 0 || age > stateMaxAge {
		return nil, errors.New("oauth_proxy: state expired")
	}
	return &p, nil
}
