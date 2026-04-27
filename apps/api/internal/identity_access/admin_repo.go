package identity_access

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Team is a row in teams.
type Team struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	OwnerID     *uuid.UUID `json:"owner_id,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TeamMembership is a user_team_memberships row.
type TeamMembership struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	TeamID         uuid.UUID `json:"team_id"`
	MembershipType string    `json:"membership_type"`
	CreatedAt      time.Time `json:"created_at"`
}

// DomainGrantRow is a domain_grants row with id.
type DomainGrantRow struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	DomainID       uuid.UUID  `json:"domain_id"`
	AccessLevel    string     `json:"access_level"`
	SensitivityCap int        `json:"sensitivity_cap"`
	GrantedBy      *uuid.UUID `json:"granted_by,omitempty"`
	GrantedAt      time.Time  `json:"granted_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

// RoleRow is a roles row.
type RoleRow struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserRoleBindingRow is user_role_bindings with id.
type UserRoleBindingRow struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	RoleID    uuid.UUID  `json:"role_id"`
	ScopeType string     `json:"scope_type"`
	ScopeID   *uuid.UUID `json:"scope_id,omitempty"`
	GrantedBy *uuid.UUID `json:"granted_by,omitempty"`
	GrantedAt time.Time  `json:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (r *Repo) ListTeams(ctx context.Context, limit, offset int) ([]Team, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, owner_id, status, created_at, updated_at
		FROM teams ORDER BY name ASC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Team
	for rows.Next() {
		var t Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.OwnerID, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (r *Repo) CreateTeam(ctx context.Context, name string, description *string, ownerID *uuid.UUID) (*Team, error) {
	id := uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO teams (id, name, description, owner_id, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,'active',now(),now())`, id, name, description, ownerID)
	if err != nil {
		return nil, err
	}
	return r.GetTeam(ctx, id)
}

func (r *Repo) GetTeam(ctx context.Context, id uuid.UUID) (*Team, error) {
	var t Team
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, owner_id, status, created_at, updated_at FROM teams WHERE id=$1`, id,
	).Scan(&t.ID, &t.Name, &t.Description, &t.OwnerID, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) PatchTeam(ctx context.Context, id uuid.UUID, name *string, description *string, status *string) (*Team, error) {
	t, err := r.GetTeam(ctx, id)
	if err != nil {
		return nil, err
	}
	nm := t.Name
	if name != nil {
		nm = *name
	}
	desc := t.Description
	if description != nil {
		desc = description
	}
	st := t.Status
	if status != nil {
		st = *status
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE teams SET name=$2, description=$3, status=$4, updated_at=now() WHERE id=$1`,
		id, nm, desc, st)
	if err != nil {
		return nil, err
	}
	return r.GetTeam(ctx, id)
}

func (r *Repo) ListTeamMemberships(ctx context.Context, teamID *uuid.UUID, userID *uuid.UUID, limit, offset int) ([]TeamMembership, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	q := `SELECT id, user_id, team_id, membership_type, created_at FROM user_team_memberships WHERE 1=1`
	args := []any{}
	n := 1
	if teamID != nil {
		q += ` AND team_id = $` + strconv.Itoa(n)
		args = append(args, *teamID)
		n++
	}
	if userID != nil {
		q += ` AND user_id = $` + strconv.Itoa(n)
		args = append(args, *userID)
		n++
	}
	q += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(n) + ` OFFSET $` + strconv.Itoa(n+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []TeamMembership
	for rows.Next() {
		var m TeamMembership
		if err := rows.Scan(&m.ID, &m.UserID, &m.TeamID, &m.MembershipType, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// AddTeamMembership inserts membership; on conflict returns existing.
func (r *Repo) AddTeamMembership(ctx context.Context, userID, teamID uuid.UUID, membershipType string) (*TeamMembership, error) {
	if membershipType == "" {
		membershipType = "member"
	}
	var m TeamMembership
	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_team_memberships (id, user_id, team_id, membership_type, created_at)
		VALUES ($1,$2,$3,$4,now())
		ON CONFLICT (user_id, team_id) DO UPDATE SET membership_type = EXCLUDED.membership_type
		RETURNING id, user_id, team_id, membership_type, created_at`,
		uuid.New(), userID, teamID, membershipType,
	).Scan(&m.ID, &m.UserID, &m.TeamID, &m.MembershipType, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repo) DeleteTeamMembership(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM user_team_memberships WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repo) ListRoles(ctx context.Context) ([]RoleRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, created_at, updated_at FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []RoleRow
	for rows.Next() {
		var ro RoleRow
		if err := rows.Scan(&ro.ID, &ro.Name, &ro.Description, &ro.CreatedAt, &ro.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, ro)
	}
	return list, rows.Err()
}

func (r *Repo) ListDomainGrants(ctx context.Context, userID *uuid.UUID, domainID *uuid.UUID, limit, offset int) ([]DomainGrantRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	q := `SELECT id, user_id, domain_id, access_level, sensitivity_cap, granted_by, granted_at, expires_at
		FROM domain_grants WHERE 1=1`
	args := []any{}
	n := 1
	if userID != nil {
		q += ` AND user_id = $` + strconv.Itoa(n)
		args = append(args, *userID)
		n++
	}
	if domainID != nil {
		q += ` AND domain_id = $` + strconv.Itoa(n)
		args = append(args, *domainID)
		n++
	}
	q += ` ORDER BY granted_at DESC LIMIT $` + strconv.Itoa(n) + ` OFFSET $` + strconv.Itoa(n+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []DomainGrantRow
	for rows.Next() {
		var g DomainGrantRow
		if err := rows.Scan(&g.ID, &g.UserID, &g.DomainID, &g.AccessLevel, &g.SensitivityCap, &g.GrantedBy, &g.GrantedAt, &g.ExpiresAt); err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

func (r *Repo) GetDomainGrant(ctx context.Context, id uuid.UUID) (*DomainGrantRow, error) {
	var g DomainGrantRow
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, domain_id, access_level, sensitivity_cap, granted_by, granted_at, expires_at
		FROM domain_grants WHERE id=$1`, id,
	).Scan(&g.ID, &g.UserID, &g.DomainID, &g.AccessLevel, &g.SensitivityCap, &g.GrantedBy, &g.GrantedAt, &g.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repo) UpsertDomainGrant(ctx context.Context, userID, domainID uuid.UUID, accessLevel string, sensitivityCap int, grantedBy *uuid.UUID) (*DomainGrantRow, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO domain_grants (id, user_id, domain_id, access_level, sensitivity_cap, granted_by, granted_at)
		VALUES ($1,$2,$3,$4,$5,$6,now())
		ON CONFLICT (user_id, domain_id) DO UPDATE SET
			access_level = EXCLUDED.access_level,
			sensitivity_cap = EXCLUDED.sensitivity_cap,
			granted_by = EXCLUDED.granted_by,
			granted_at = now()
		RETURNING id`,
		uuid.New(), userID, domainID, accessLevel, sensitivityCap, grantedBy,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetDomainGrant(ctx, id)
}

func (r *Repo) PatchDomainGrant(ctx context.Context, id uuid.UUID, accessLevel *string, sensitivityCap *int, expiresAt *time.Time) (*DomainGrantRow, error) {
	g, err := r.GetDomainGrant(ctx, id)
	if err != nil {
		return nil, err
	}
	al := g.AccessLevel
	if accessLevel != nil {
		al = *accessLevel
	}
	sc := g.SensitivityCap
	if sensitivityCap != nil {
		sc = *sensitivityCap
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE domain_grants SET access_level=$2, sensitivity_cap=$3, expires_at=$4 WHERE id=$1`,
		id, al, sc, expiresAt)
	if err != nil {
		return nil, err
	}
	return r.GetDomainGrant(ctx, id)
}

func (r *Repo) DeleteDomainGrant(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM domain_grants WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repo) ListUserRoleBindings(ctx context.Context, userID *uuid.UUID, limit, offset int) ([]UserRoleBindingRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	q := `SELECT id, user_id, role_id, scope_type, scope_id, granted_by, granted_at, expires_at FROM user_role_bindings WHERE 1=1`
	args := []any{}
	n := 1
	if userID != nil {
		q += ` AND user_id = $` + strconv.Itoa(n)
		args = append(args, *userID)
		n++
	}
	q += ` ORDER BY granted_at DESC LIMIT $` + strconv.Itoa(n) + ` OFFSET $` + strconv.Itoa(n+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []UserRoleBindingRow
	for rows.Next() {
		var b UserRoleBindingRow
		if err := rows.Scan(&b.ID, &b.UserID, &b.RoleID, &b.ScopeType, &b.ScopeID, &b.GrantedBy, &b.GrantedAt, &b.ExpiresAt); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (r *Repo) GetUserRoleBinding(ctx context.Context, id uuid.UUID) (*UserRoleBindingRow, error) {
	var b UserRoleBindingRow
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, role_id, scope_type, scope_id, granted_by, granted_at, expires_at
		FROM user_role_bindings WHERE id=$1`, id,
	).Scan(&b.ID, &b.UserID, &b.RoleID, &b.ScopeType, &b.ScopeID, &b.GrantedBy, &b.GrantedAt, &b.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repo) CreateUserRoleBinding(ctx context.Context, userID, roleID uuid.UUID, scopeType string, scopeID *uuid.UUID, grantedBy *uuid.UUID) (*UserRoleBindingRow, error) {
	id := uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_role_bindings (id, user_id, role_id, scope_type, scope_id, granted_by, granted_at)
		VALUES ($1,$2,$3,$4,$5,$6,now())`, id, userID, roleID, scopeType, scopeID, grantedBy)
	if err != nil {
		return nil, err
	}
	return r.GetUserRoleBinding(ctx, id)
}

func (r *Repo) PatchUserRoleBinding(ctx context.Context, id uuid.UUID, scopeType *string, scopeID *uuid.UUID, expiresAt *time.Time) (*UserRoleBindingRow, error) {
	b, err := r.GetUserRoleBinding(ctx, id)
	if err != nil {
		return nil, err
	}
	st := b.ScopeType
	if scopeType != nil {
		st = *scopeType
	}
	sid := b.ScopeID
	if scopeID != nil {
		sid = scopeID
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE user_role_bindings SET scope_type=$2, scope_id=$3, expires_at=$4 WHERE id=$1`,
		id, st, sid, expiresAt)
	if err != nil {
		return nil, err
	}
	return r.GetUserRoleBinding(ctx, id)
}

func (r *Repo) DeleteUserRoleBinding(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM user_role_bindings WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repo) CreateUser(ctx context.Context, email, name string) (*User, error) {
	id := uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, email, name, status, created_at, updated_at)
		VALUES ($1,$2,$3,'active',now(),now())`, id, email, name)
	if err != nil {
		return nil, err
	}
	return r.GetUser(ctx, id)
}

func (r *Repo) PatchUser(ctx context.Context, id uuid.UUID, email *string, name *string, status *string, primaryTeamID *uuid.UUID) (*User, error) {
	u, err := r.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	em := u.Email
	if email != nil {
		em = *email
	}
	nm := u.Name
	if name != nil {
		nm = *name
	}
	st := u.Status
	if status != nil {
		st = *status
	}
	tid := u.TeamID
	if primaryTeamID != nil {
		tid = primaryTeamID
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE users SET email=$2, name=$3, status=$4, primary_team_id=$5, updated_at=now() WHERE id=$1`,
		id, em, nm, st, tid)
	if err != nil {
		return nil, err
	}
	return r.GetUser(ctx, id)
}
