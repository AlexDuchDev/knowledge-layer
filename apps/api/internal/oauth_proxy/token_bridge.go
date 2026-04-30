package oauth_proxy

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TokenBridge maps an OIDC `sub` claim to a known KL principal UUID. This is
// the canonical chokepoint where the OAuth proxy "earns its right to issue a
// bearer". A `sub` that doesn't already correspond to a row in `users` is
// rejected (no auto-provision today; that's a deliberate v0.5.x scope cut).
//
// Why no admin bypass: per the v0.5.0 plan + ADR-0015, every issued token
// must resolve to a real user so AccessEvaluator can score it. An
// "elevated" or "service-account" token that bypasses the 9-step evaluator
// is explicitly rejected.
type TokenBridge struct {
	pool *pgxpool.Pool
	// idpAttribute is the users-table column that holds the OIDC sub
	// for matching. Operators with an existing IDP integration (e.g.
	// stuffing sub into users.external_idp_subject) can override; default
	// matches against email when sub is an email-format identifier.
	subColumn string
}

// NewTokenBridge wires the bridge to a pool and chooses the column used to
// match OIDC sub claims. `subColumn` is one of: "email", "external_idp_subject".
// "email" is the safe default — almost every IDP issues either email-as-sub
// or a sub matching the user's email; operators who use opaque subs can
// migrate to "external_idp_subject" by adding the column and re-pointing.
func NewTokenBridge(pool *pgxpool.Pool, subColumn string) *TokenBridge {
	if subColumn == "" {
		subColumn = "email"
	}
	return &TokenBridge{pool: pool, subColumn: subColumn}
}

// Resolve maps an IDP-verified sub claim to a KL user. Returns
// uuid.Nil + error if no matching user exists; never returns an "anonymous"
// or "system" UUID. Caller must handle the error path as a 401, not a 403:
// the IDP authenticated *someone*, but that someone has no KL identity.
func (b *TokenBridge) Resolve(ctx context.Context, claims *IDTokenClaims) (uuid.UUID, error) {
	if claims == nil {
		return uuid.Nil, errors.New("oauth_proxy: nil claims")
	}
	// We use the subColumn name as a static route — no operator-supplied
	// strings reach this query, so SQL injection isn't a vector. Limited
	// to the two supported values via a switch.
	var query string
	var arg string
	switch b.subColumn {
	case "external_idp_subject":
		query = `SELECT id FROM users WHERE external_idp_subject = $1 AND status = 'active' LIMIT 1`
		arg = claims.Subject
	case "email":
		fallthrough
	default:
		query = `SELECT id FROM users WHERE email = $1 AND status = 'active' LIMIT 1`
		// Many IDPs set sub to email; if claims.Email is set, prefer that
		// (more reliable when sub is opaque).
		if claims.Email != "" {
			arg = claims.Email
		} else {
			arg = claims.Subject
		}
	}

	var id uuid.UUID
	if err := b.pool.QueryRow(ctx, query, arg).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("oauth_proxy: no KL user matches sub=%q (column=%s)", arg, b.subColumn)
		}
		return uuid.Nil, err
	}
	return id, nil
}
