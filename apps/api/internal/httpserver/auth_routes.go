package httpserver

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/auth"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/identity_access"
	"golang.org/x/crypto/bcrypt"
)

func mailerFor(cfg config.Config) auth.Mailer {
	if cfg.SMTPHost != "" && cfg.SMTPFrom != "" {
		return auth.SMTPMailer{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser,
			Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
		}
	}
	return auth.LogMailer{Prefix: "knowledge-layer"}
}

func mountAuthSettingsRoutes(f *fiber.App, d *app.Deps, cfg config.Config) {
	mailer := mailerFor(cfg)
	sec := cfg.SessionSecretBytes()
	cookieOpts := auth.SessionCookieOpts{Secure: cfg.SessionCookieSecure()}

	f.Get("/instance/status", func(c *fiber.Ctx) error {
		n, err := d.Identity.CountDomains(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		sourceFeedTotal := 0
		sourceFeedActive := 0
		hasSuccessfulSourceSync := false
		if n > 0 {
			_ = d.Pool.QueryRow(c.Context(), `SELECT count(*)::int FROM source_feeds`).Scan(&sourceFeedTotal)
			_ = d.Pool.QueryRow(c.Context(), `SELECT count(*)::int FROM source_feeds WHERE status = 'active'`).Scan(&sourceFeedActive)
			_ = d.Pool.QueryRow(c.Context(), `SELECT EXISTS (SELECT 1 FROM source_feeds WHERE last_successful_sync_at IS NOT NULL)`).Scan(&hasSuccessfulSourceSync)
		}
		return c.JSON(fiber.Map{
			"needs_bootstrap":            n == 0,
			"domain_count":               n,
			"auth_mode":                  cfg.AuthMode,
			"build_version":              cfg.BuildVersion,
			"source_feed_total_count":    sourceFeedTotal,
			"source_feed_active_count":   sourceFeedActive,
			"has_successful_source_sync": hasSuccessfulSourceSync,
		})
	})

	f.Post("/instance/bootstrap", func(c *fiber.Ctx) error {
		n, err := d.Identity.CountDomains(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if n > 0 {
			return fiber.NewError(fiber.StatusForbidden, "bootstrap only when no domains exist")
		}
		var body struct {
			AdminEmail        string  `json:"admin_email"`
			AdminName         string  `json:"admin_name"`
			AdminPassword     string  `json:"admin_password"`
			DomainName        string  `json:"domain_name"`
			DomainDescription *string `json:"domain_description"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if body.AdminEmail == "" || body.AdminName == "" || len(body.AdminPassword) < 8 || body.DomainName == "" {
			return fiber.NewError(fiber.StatusBadRequest, "admin_email, admin_name, admin_password (min 8), domain_name required")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(body.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		hs := string(hash)
		u, err := d.Identity.CreateUser(c.Context(), body.AdminEmail, body.AdminName)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if err := d.Identity.SetUserPasswordHash(c.Context(), u.ID, &hs); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		dom, _, err := d.Identity.CreateDomainWithPolicy(c.Context(), body.DomainName, body.DomainDescription, &u.ID, 0)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if _, err := d.Identity.UpsertDomainGrant(c.Context(), u.ID, dom.ID, "admin", 3, &u.ID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if _, err := d.Identity.CreateUserRoleBinding(c.Context(), u.ID, identity_access.RoleAdminSeedID, "global", nil, &u.ID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		_ = d.AuditOps.Write(c.Context(), auditInput("instance.bootstrapped", "system", nil, "domain", &dom.ID, nil))
		if cfg.AuthMode == "session" && len(sec) > 0 {
			_ = auth.IssueSessionCookie(c, u.ID, sec, 7*24*time.Hour, cookieOpts)
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": u, "domain": dom})
	})

	f.Post("/auth/login", func(c *fiber.Ctx) error {
		if cfg.AuthMode != "session" {
			return fiber.NewError(fiber.StatusBadRequest, "session auth not enabled (set AUTH_MODE=session and SESSION_SECRET)")
		}
		if len(sec) == 0 {
			return fiber.NewError(fiber.StatusServiceUnavailable, "SESSION_SECRET not configured")
		}
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&body); err != nil || body.Email == "" || body.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "email and password required")
		}
		u, err := d.Identity.GetUserByEmail(c.Context(), body.Email)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
		}
		ph, err := d.Identity.GetPasswordHash(c.Context(), u.ID)
		if err != nil || ph == nil || *ph == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
		}
		if bcrypt.CompareHashAndPassword([]byte(*ph), []byte(body.Password)) != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
		}
		_ = auth.IssueSessionCookie(c, u.ID, sec, 7*24*time.Hour, cookieOpts)
		return c.JSON(u)
	})

	f.Post("/auth/register", func(c *fiber.Ctx) error {
		if !cfg.AllowSelfRegistration {
			return fiber.NewError(fiber.StatusForbidden, "self-registration is disabled; use an invitation")
		}
		return fiber.NewError(fiber.StatusNotImplemented, "not enabled")
	})

	f.Post("/auth/logout", func(c *fiber.Ctx) error {
		auth.ClearSessionCookie(c, cookieOpts)
		return c.SendStatus(fiber.StatusNoContent)
	})

	f.Get("/auth/me", func(c *fiber.Ctx) error {
		pid, ok := httpcontext.PrincipalID(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
		}
		u, err := d.Identity.GetUser(c.Context(), pid)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "not found")
		}
		nav, err := computeNavigationVisibility(c.Context(), d, pid)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{
			"id":              u.ID,
			"email":           u.Email,
			"name":            u.Name,
			"status":          u.Status,
			"primary_team_id": u.TeamID,
			"created_at":      u.CreatedAt,
			"updated_at":      u.UpdatedAt,
			"navigation":      nav,
		})
	})

	f.Get("/invitations/preview", func(c *fiber.Ctx) error {
		tok := c.Query("token")
		if tok == "" {
			return fiber.NewError(fiber.StatusBadRequest, "token required")
		}
		inv, err := d.Identity.GetInvitationByRawToken(c.Context(), tok)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "invalid or expired invitation")
		}
		return c.JSON(fiber.Map{"email": inv.Email, "expires_at": inv.ExpiresAt})
	})

	f.Post("/invitations/accept", func(c *fiber.Ctx) error {
		var body struct {
			Token    string `json:"token"`
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&body); err != nil || body.Token == "" || body.Name == "" || len(body.Password) < 8 {
			return fiber.NewError(fiber.StatusBadRequest, "token, name, password (min 8) required")
		}
		inv, err := d.Identity.GetInvitationByRawToken(c.Context(), body.Token)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "invalid or expired invitation")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		hs := string(hash)
		var u *identity_access.User
		existing, err := d.Identity.GetUserByEmail(c.Context(), inv.Email)
		if err == nil {
			u = existing
			if err := d.Identity.SetUserPasswordHash(c.Context(), u.ID, &hs); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			if _, err := d.Identity.PatchUser(c.Context(), u.ID, nil, &body.Name, nil, nil); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
		} else if errors.Is(err, pgx.ErrNoRows) {
			u, err = d.Identity.CreateUser(c.Context(), inv.Email, body.Name)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			if err := d.Identity.SetUserPasswordHash(c.Context(), u.ID, &hs); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
		} else {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if _, err := d.Identity.UpsertDomainGrant(c.Context(), u.ID, inv.DomainID, inv.AccessLevel, inv.SensitivityCap, inv.InvitedBy); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if inv.RoleID != nil {
			if _, err := d.Identity.CreateUserRoleBinding(c.Context(), u.ID, *inv.RoleID, "domain", &inv.DomainID, inv.InvitedBy); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
		}
		if err := d.Identity.MarkInvitationAccepted(c.Context(), inv.ID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if cfg.AuthMode == "session" && len(sec) > 0 {
			_ = auth.IssueSessionCookie(c, u.ID, sec, 7*24*time.Hour, cookieOpts)
		}
		return c.JSON(u)
	})

	f.Post("/invitations", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		var body struct {
			Email          string     `json:"email"`
			DomainID       uuid.UUID  `json:"domain_id"`
			RoleID         *uuid.UUID `json:"role_id"`
			AccessLevel    string     `json:"access_level"`
			SensitivityCap int        `json:"sensitivity_cap"`
			ExpiresInDays  int        `json:"expires_in_days"`
		}
		if err := c.BodyParser(&body); err != nil || body.Email == "" || body.DomainID == uuid.Nil {
			return fiber.NewError(fiber.StatusBadRequest, "email and domain_id required")
		}
		if err := requirePublishOnDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		al := body.AccessLevel
		if al == "" {
			al = "read"
		}
		sc := body.SensitivityCap
		if sc <= 0 {
			sc = 1
		}
		days := body.ExpiresInDays
		if days <= 0 {
			days = 14
		}
		rawTok := make([]byte, 32)
		if _, err := rand.Read(rawTok); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		tokStr := hex.EncodeToString(rawTok)
		exp := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		inv, err := d.Identity.CreateInvitation(c.Context(), body.Email, body.DomainID, body.RoleID, al, sc, &principal, tokStr, exp)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		link := fmt.Sprintf("%s/invite?token=%s", cfg.AppPublicURL, tokStr)
		subj := "Knowledge Layer invitation"
		msg := fmt.Sprintf("You were invited to Knowledge Layer.\nOpen: %s\n", link)
		_ = mailer.Send([]string{body.Email}, subj, msg)
		_ = d.AuditOps.Write(c.Context(), auditInput("invitation.created", "user", &principal, "user_invitation", &inv.ID, nil))
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": inv.ID, "email": inv.Email, "expires_at": inv.ExpiresAt})
	})

	f.Post("/users/import", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		if err := requireCanManageIdentity(c, d, principal); err != nil {
			return err
		}
		sendInvites := c.FormValue("send_invites") == "true" || c.FormValue("send_invites") == "1"
		mode := c.FormValue("mode")
		if mode == "" {
			mode = "invite"
		}
		fh, err := c.FormFile("file")
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "file required (multipart field file)")
		}
		src, err := fh.Open()
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		defer src.Close()
		r := csv.NewReader(src)
		r.TrimLeadingSpace = true
		header, err := r.Read()
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "empty csv")
		}
		col := map[string]int{}
		for i, h := range header {
			col[strings.ToLower(strings.TrimSpace(h))] = i
		}
		req := []string{"email", "domain_id"}
		for _, k := range req {
			if _, ok := col[k]; !ok {
				return fiber.NewError(fiber.StatusBadRequest, "csv must include columns: email, domain_id")
			}
		}
		var report []fiber.Map
		line := 1
		for {
			rec, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				report = append(report, fiber.Map{"line": line, "error": err.Error()})
				break
			}
			line++
			email := strings.TrimSpace(rec[col["email"]])
			did, err := uuid.Parse(strings.TrimSpace(rec[col["domain_id"]]))
			if err != nil || email == "" {
				report = append(report, fiber.Map{"line": line, "error": "bad email or domain_id"})
				continue
			}
			if err := requirePublishOnDomain(c, d, principal, did); err != nil {
				report = append(report, fiber.Map{"line": line, "error": "forbidden for domain"})
				continue
			}
			name := email
			if i, ok := col["name"]; ok && i < len(rec) && strings.TrimSpace(rec[i]) != "" {
				name = strings.TrimSpace(rec[i])
			}
			al := "read"
			if i, ok := col["access_level"]; ok && i < len(rec) && strings.TrimSpace(rec[i]) != "" {
				al = strings.TrimSpace(rec[i])
			}
			sc := 1
			if i, ok := col["sensitivity_cap"]; ok && i < len(rec) {
				if n, e := strconv.Atoi(strings.TrimSpace(rec[i])); e == nil {
					sc = n
				}
			}
			var roleID *uuid.UUID
			if i, ok := col["role_id"]; ok && i < len(rec) && strings.TrimSpace(rec[i]) != "" {
				if rid, e := uuid.Parse(strings.TrimSpace(rec[i])); e == nil {
					roleID = &rid
				}
			}
			switch mode {
			case "invite":
				rawTok := make([]byte, 32)
				if _, err := rand.Read(rawTok); err != nil {
					report = append(report, fiber.Map{"line": line, "error": err.Error()})
					continue
				}
				tokStr := hex.EncodeToString(rawTok)
				inv, err := d.Identity.CreateInvitation(c.Context(), email, did, roleID, al, sc, &principal, tokStr, time.Now().Add(14*24*time.Hour))
				if err != nil {
					report = append(report, fiber.Map{"line": line, "error": err.Error()})
					continue
				}
				if sendInvites {
					link := fmt.Sprintf("%s/invite?token=%s", cfg.AppPublicURL, tokStr)
					_ = mailer.Send([]string{email}, "Knowledge Layer invitation", fmt.Sprintf("Open: %s\n", link))
				}
				report = append(report, fiber.Map{"line": line, "ok": true, "invitation_id": inv.ID.String()})
			case "active":
				u, err := d.Identity.CreateUser(c.Context(), email, name)
				if err != nil {
					report = append(report, fiber.Map{"line": line, "error": err.Error()})
					continue
				}
				if _, err := d.Identity.UpsertDomainGrant(c.Context(), u.ID, did, al, sc, &principal); err != nil {
					report = append(report, fiber.Map{"line": line, "error": err.Error()})
					continue
				}
				if roleID != nil {
					if _, err := d.Identity.CreateUserRoleBinding(c.Context(), u.ID, *roleID, "domain", &did, &principal); err != nil {
						report = append(report, fiber.Map{"line": line, "error": err.Error()})
						continue
					}
				}
				report = append(report, fiber.Map{"line": line, "ok": true, "user_id": u.ID.String()})
			default:
				return fiber.NewError(fiber.StatusBadRequest, "mode must be invite or active")
			}
		}
		return c.JSON(fiber.Map{"results": report})
	})

	f.Get("/settings/instance", func(c *fiber.Ctx) error {
		pid, ok := httpcontext.PrincipalID(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
		}
		if err := requireCanManageIdentity(c, d, pid); err != nil {
			return err
		}
		n, _ := d.Identity.CountDomains(c.Context())
		return c.JSON(fiber.Map{
			"auth_mode":               cfg.AuthMode,
			"build_version":           cfg.BuildVersion,
			"app_public_url":          cfg.AppPublicURL,
			"domain_count":            n,
			"smtp_configured":         cfg.SMTPHost != "" && cfg.SMTPFrom != "",
			"allow_self_registration": cfg.AllowSelfRegistration,
			"config_reference":        "docs/CONFIG_ENV.md and docs/SELF_HOSTED.md",
		})
	})

	f.Post("/settings/test-mail", func(c *fiber.Ctx) error {
		pid, ok := httpcontext.PrincipalID(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
		}
		if err := requireCanManageIdentity(c, d, pid); err != nil {
			return err
		}
		var body struct {
			To string `json:"to"`
		}
		if err := c.BodyParser(&body); err != nil || body.To == "" {
			return fiber.NewError(fiber.StatusBadRequest, "to required")
		}
		if err := mailer.Send([]string{body.To}, "Knowledge Layer mail test", "This is a test message from Knowledge Layer."); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(fiber.Map{"ok": true})
	})
}
