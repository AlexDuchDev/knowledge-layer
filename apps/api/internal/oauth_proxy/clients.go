package oauth_proxy

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Client is one row of the oauth_clients table — a single MCP client (Claude
// Desktop, Cursor, etc.) that registered itself via /oauth/register.
type Client struct {
	ClientID                string
	ClientName              string
	RedirectURIs            []string
	TokenEndpointAuthMethod string
	RevokedAt               *time.Time
	CreatedAt               time.Time
}

// ClientRepo wraps the oauth_clients table. Operations are intentionally
// narrow: register a client, look one up by id, mark used. No list/delete
// public API in v0.5.0; operators inspect via psql or audit_events.
type ClientRepo struct {
	pool *pgxpool.Pool
}

func NewClientRepo(pool *pgxpool.Pool) *ClientRepo { return &ClientRepo{pool: pool} }

// RegisterRequest is what /oauth/register accepts (RFC 7591). Only the
// fields the v0.5.0 proxy actually consumes are surfaced.
type RegisterRequest struct {
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
}

// RegisterResponse is what /oauth/register returns (RFC 7591 §3.2.1). The
// secret is shown once at registration time and never again — clients must
// persist it locally.
type RegisterResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
}

// Register inserts a new client row. PKCE-only ("none") clients receive no
// secret; confidential clients get a random 32-byte url-safe secret.
func (r *ClientRepo) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	if len(req.RedirectURIs) == 0 {
		return nil, errors.New("oauth_proxy: redirect_uris required")
	}
	for _, u := range req.RedirectURIs {
		if !strings.HasPrefix(u, "http://localhost") && !strings.HasPrefix(u, "https://") {
			// Allow localhost for native MCP clients (Claude Desktop's
			// internal callback) and HTTPS otherwise. Plain http:// to
			// non-localhost is rejected.
			return nil, fmt.Errorf("oauth_proxy: redirect_uri %q must be https:// or http://localhost...", u)
		}
	}
	method := strings.ToLower(strings.TrimSpace(req.TokenEndpointAuthMethod))
	switch method {
	case "", "none":
		method = "none"
	case "client_secret_basic", "client_secret_post":
	default:
		return nil, fmt.Errorf("oauth_proxy: unsupported token_endpoint_auth_method %q", req.TokenEndpointAuthMethod)
	}

	clientID, err := randomURLSafe(16)
	if err != nil {
		return nil, err
	}
	var secret string
	var secretHash string
	if method != "none" {
		secret, err = randomURLSafe(32)
		if err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		secretHash = string(hash)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO oauth_clients (client_id, client_secret_hash, redirect_uris, client_name, token_endpoint_auth_method)
		VALUES ($1, $2, $3, $4, $5)`,
		clientID, secretHash, req.RedirectURIs, strings.TrimSpace(req.ClientName), method)
	if err != nil {
		return nil, err
	}
	return &RegisterResponse{
		ClientID:                clientID,
		ClientSecret:            secret,
		ClientName:              req.ClientName,
		RedirectURIs:            req.RedirectURIs,
		TokenEndpointAuthMethod: method,
		ClientIDIssuedAt:        time.Now().Unix(),
	}, nil
}

// Get loads one row. Caller is expected to verify revoked_at IS NULL.
func (r *ClientRepo) Get(ctx context.Context, clientID string) (*Client, string, error) {
	var c Client
	var secretHash string
	err := r.pool.QueryRow(ctx, `
		SELECT client_id, client_name, redirect_uris, token_endpoint_auth_method,
		       revoked_at, created_at, client_secret_hash
		FROM oauth_clients WHERE client_id = $1`, clientID,
	).Scan(&c.ClientID, &c.ClientName, &c.RedirectURIs, &c.TokenEndpointAuthMethod,
		&c.RevokedAt, &c.CreatedAt, &secretHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", errors.New("oauth_proxy: unknown client_id")
		}
		return nil, "", err
	}
	if c.RevokedAt != nil {
		return nil, "", errors.New("oauth_proxy: client revoked")
	}
	return &c, secretHash, nil
}

// VerifySecret constant-time-compares the supplied secret against the stored
// bcrypt hash. PKCE-only clients (auth_method=none) accept any secret call
// but the caller is expected to skip secret verification entirely for those.
func (r *ClientRepo) VerifySecret(secretHash, supplied string) error {
	if secretHash == "" {
		return errors.New("oauth_proxy: client has no secret (use PKCE)")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(secretHash), []byte(supplied)); err != nil {
		return errors.New("oauth_proxy: client secret mismatch")
	}
	return nil
}

// MarkUsed bumps last_used_at. Best-effort; failures are not fatal.
func (r *ClientRepo) MarkUsed(ctx context.Context, clientID string) {
	_, _ = r.pool.Exec(ctx, `UPDATE oauth_clients SET last_used_at = now() WHERE client_id = $1`, clientID)
}

// AllowsRedirect compares supplied redirect_uri against the client's
// allow-list (verbatim — no path-level partial match, no query-param
// stripping). Spec-compliant operators get the documented behavior;
// loosening this is a security regression.
func (c *Client) AllowsRedirect(uri string) bool {
	for _, u := range c.RedirectURIs {
		if u == uri {
			return true
		}
	}
	return false
}

func randomURLSafe(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
