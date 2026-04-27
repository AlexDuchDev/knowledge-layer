package identity_access

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserInvitation struct {
	ID             uuid.UUID  `json:"id"`
	Email          string     `json:"email"`
	DomainID       uuid.UUID  `json:"domain_id"`
	RoleID         *uuid.UUID `json:"role_id,omitempty"`
	AccessLevel    string     `json:"access_level"`
	SensitivityCap int        `json:"sensitivity_cap"`
	InvitedBy      *uuid.UUID `json:"invited_by,omitempty"`
	ExpiresAt      time.Time  `json:"expires_at"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func hashInviteToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func (r *Repo) CreateInvitation(ctx context.Context, email string, domainID uuid.UUID, roleID *uuid.UUID, accessLevel string, sensitivityCap int, invitedBy *uuid.UUID, rawToken string, expiresAt time.Time) (*UserInvitation, error) {
	th := hashInviteToken(rawToken)
	id := uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_invitations (id, email, domain_id, role_id, access_level, sensitivity_cap, token_hash, invited_by, expires_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now())`,
		id, email, domainID, roleID, accessLevel, sensitivityCap, th, invitedBy, expiresAt)
	if err != nil {
		return nil, err
	}
	return r.getInvitationByID(ctx, id)
}

func (r *Repo) getInvitationByID(ctx context.Context, id uuid.UUID) (*UserInvitation, error) {
	var inv UserInvitation
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, domain_id, role_id, access_level, sensitivity_cap, invited_by, expires_at, accepted_at, created_at
		FROM user_invitations WHERE id=$1`, id,
	).Scan(&inv.ID, &inv.Email, &inv.DomainID, &inv.RoleID, &inv.AccessLevel, &inv.SensitivityCap, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *Repo) GetInvitationByRawToken(ctx context.Context, rawToken string) (*UserInvitation, error) {
	th := hashInviteToken(rawToken)
	var inv UserInvitation
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, domain_id, role_id, access_level, sensitivity_cap, invited_by, expires_at, accepted_at, created_at
		FROM user_invitations WHERE token_hash=$1 AND accepted_at IS NULL`, th,
	).Scan(&inv.ID, &inv.Email, &inv.DomainID, &inv.RoleID, &inv.AccessLevel, &inv.SensitivityCap, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, pgx.ErrNoRows
	}
	return &inv, nil
}

func (r *Repo) MarkInvitationAccepted(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE user_invitations SET accepted_at=now() WHERE id=$1 AND accepted_at IS NULL`, id)
	return err
}

func (r *Repo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, name, status, primary_team_id, created_at, updated_at FROM users WHERE lower(email)=lower($1)`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Status, &u.TeamID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repo) SetUserPasswordHash(ctx context.Context, userID uuid.UUID, hash *string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET password_hash=$2, updated_at=now() WHERE id=$1`, userID, hash)
	return err
}

func (r *Repo) GetPasswordHash(ctx context.Context, userID uuid.UUID) (*string, error) {
	var h *string
	err := r.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1`, userID).Scan(&h)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (r *Repo) CountDomains(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM domains`).Scan(&n)
	return n, err
}

func (r *Repo) CreateDomainWithPolicy(ctx context.Context, name string, description *string, ownerID *uuid.UUID, defaultSensitivity int) (*Domain, *uuid.UUID, error) {
	policyID := uuid.New()
	domainID := uuid.New()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		INSERT INTO access_policies (id, name, description, domain_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'active', now(), now())`,
		policyID, "default", "Bootstrap policy", domainID)
	if err != nil {
		return nil, nil, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO domains (id, name, description, owner_id, default_access_policy_id, default_sensitivity_level, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,'active',now(),now())`,
		domainID, name, description, ownerID, policyID, defaultSensitivity)
	if err != nil {
		return nil, nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE access_policies SET domain_id=$2 WHERE id=$1`, policyID, domainID)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	d, err := r.GetDomain(ctx, domainID)
	return d, &policyID, err
}

func (r *Repo) GetDomain(ctx context.Context, id uuid.UUID) (*Domain, error) {
	var d Domain
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, owner_id, default_access_policy_id,
		       default_sensitivity_level, status, created_at, updated_at
		FROM domains WHERE id=$1`, id,
	).Scan(&d.ID, &d.Name, &d.Description, &d.OwnerID, &d.DefaultAccessPolicyID,
		&d.DefaultSensitivityLevel, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repo) PatchDomain(ctx context.Context, id uuid.UUID, name *string, description *string, status *string, ownerID *uuid.UUID) (*Domain, error) {
	d, err := r.GetDomain(ctx, id)
	if err != nil {
		return nil, err
	}
	nm := d.Name
	if name != nil {
		nm = *name
	}
	desc := d.Description
	if description != nil {
		desc = description
	}
	st := d.Status
	if status != nil {
		st = *status
	}
	ow := d.OwnerID
	if ownerID != nil {
		ow = ownerID
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE domains SET name=$2, description=$3, status=$4, owner_id=$5, updated_at=now() WHERE id=$1`,
		id, nm, desc, st, ow)
	if err != nil {
		return nil, err
	}
	return r.GetDomain(ctx, id)
}
