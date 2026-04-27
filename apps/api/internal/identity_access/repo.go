package identity_access

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	TeamID    *uuid.UUID `json:"primary_team_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Domain struct {
	ID                      uuid.UUID  `json:"id"`
	Name                    string     `json:"name"`
	Description             *string    `json:"description,omitempty"`
	OwnerID                 *uuid.UUID `json:"owner_id,omitempty"`
	DefaultAccessPolicyID   *uuid.UUID `json:"default_access_policy_id,omitempty"`
	DefaultSensitivityLevel int        `json:"default_sensitivity_level"`
	Status                  string     `json:"status"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, email, name, status, primary_team_id, created_at, updated_at
		FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Status, &u.TeamID, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func (r *Repo) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, name, status, primary_team_id, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Status, &u.TeamID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repo) ListDomains(ctx context.Context) ([]Domain, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, owner_id, default_access_policy_id,
		       default_sensitivity_level, status, created_at, updated_at
		FROM domains ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.OwnerID, &d.DefaultAccessPolicyID,
			&d.DefaultSensitivityLevel, &d.Status, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// ListDomainsForUser returns domains the user has a non-expired domain_grants row for.
func (r *Repo) ListDomainsForUser(ctx context.Context, userID uuid.UUID) ([]Domain, error) {
	if userID == uuid.Nil {
		return []Domain{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT d.id, d.name, d.description, d.owner_id, d.default_access_policy_id,
		       d.default_sensitivity_level, d.status, d.created_at, d.updated_at
		FROM domains d
		INNER JOIN domain_grants dg ON dg.domain_id = d.id AND dg.user_id = $1
			AND (dg.expires_at IS NULL OR dg.expires_at > now())
		ORDER BY d.name ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.OwnerID, &d.DefaultAccessPolicyID,
			&d.DefaultSensitivityLevel, &d.Status, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}
