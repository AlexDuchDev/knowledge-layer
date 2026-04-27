package instancebootstrap

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/audit"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/identity_access"
	"golang.org/x/crypto/bcrypt"
)

const pgUniqueViolation = "23505"

// Ensure creates the first admin user, default domain, grants, and admin role binding
// when the instance has no domains and config allows auto-bootstrap.
// It is idempotent for concurrent API startups (unique email / existing workspace).
func Ensure(ctx context.Context, d *app.Deps, cfg config.Config) error {
	if !cfg.AutoBootstrapEmptyInstance() {
		return nil
	}
	n, err := d.Identity.CountDomains(ctx)
	if err != nil {
		return fmt.Errorf("instancebootstrap: count domains: %w", err)
	}
	if n > 0 {
		return nil
	}

	email := cfg.BootstrapAdminEmail()
	password := cfg.BootstrapAdminPassword()
	name := cfg.BootstrapAdminName()
	domainName := cfg.BootstrapDomainName()

	if email == "" || len(password) < 8 || name == "" || domainName == "" {
		return fmt.Errorf("instancebootstrap: invalid bootstrap config (email, password min 8, name, domain required); set BOOTSTRAP_* or disable auto-bootstrap")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("instancebootstrap: hash password: %w", err)
	}
	hs := string(hash)

	u, err := d.Identity.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("instancebootstrap: lookup user: %w", err)
		}
		u, err = d.Identity.CreateUser(ctx, email, name)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
				u, err = d.Identity.GetUserByEmail(ctx, email)
			}
			if err != nil {
				n2, cerr := d.Identity.CountDomains(ctx)
				if cerr == nil && n2 > 0 {
					return nil
				}
				return fmt.Errorf("instancebootstrap: create user: %w", err)
			}
		}
	}

	if err := d.Identity.SetUserPasswordHash(ctx, u.ID, &hs); err != nil {
		return fmt.Errorf("instancebootstrap: set password: %w", err)
	}

	n, err = d.Identity.CountDomains(ctx)
	if err != nil {
		return fmt.Errorf("instancebootstrap: recount domains: %w", err)
	}
	if n > 0 {
		return nil
	}

	dom, _, err := d.Identity.CreateDomainWithPolicy(ctx, domainName, nil, &u.ID, 0)
	if err != nil {
		n2, cerr := d.Identity.CountDomains(ctx)
		if cerr == nil && n2 > 0 {
			return nil
		}
		return fmt.Errorf("instancebootstrap: create domain: %w", err)
	}

	if _, err := d.Identity.UpsertDomainGrant(ctx, u.ID, dom.ID, "admin", 3, &u.ID); err != nil {
		return fmt.Errorf("instancebootstrap: domain grant: %w", err)
	}
	if _, err := d.Identity.CreateUserRoleBinding(ctx, u.ID, identity_access.RoleAdminSeedID, "global", nil, &u.ID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			n2, cerr := d.Identity.CountDomains(ctx)
			if cerr == nil && n2 > 0 {
				return nil
			}
			return fmt.Errorf("instancebootstrap: role binding duplicate but workspace incomplete: %w", err)
		}
		return fmt.Errorf("instancebootstrap: role binding: %w", err)
	}

	_ = d.AuditOps.Write(ctx, audit.WriteInput{
		EventType:  "instance.bootstrapped",
		ActorType:  "system",
		TargetType: "domain",
		TargetID:   &dom.ID,
	})
	log.Printf("instance-bootstrap: first workspace ready (domain=%q admin_email=%q user_id=%s)", domainName, email, u.ID.String())
	return nil
}
