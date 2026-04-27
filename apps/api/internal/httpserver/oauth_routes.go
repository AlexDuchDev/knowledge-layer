package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/connectoroauth"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"golang.org/x/oauth2"
)

var connectorOAuthPending = connectoroauth.NewPendingStore()

func oauthStateHMACKey(cfg config.Config) ([]byte, error) {
	b := cfg.SessionSecretBytes()
	if len(b) >= 16 {
		return b, nil
	}
	if cfg.IsLocalDev() {
		return []byte("dev-oauth-state-key-min16b!"), nil
	}
	return nil, errors.New("SESSION_SECRET must be at least 16 bytes for OAuth state signing (non-local)")
}

func registerConnectorOAuthRoutes(f *fiber.App, d *app.Deps, cfg config.Config) {
	gmailConf, msConf := connectoroauth.OAuthConfigs(cfg)

	webBase := strings.TrimSpace(cfg.OAuthWebRedirectURL)
	if webBase == "" {
		webBase = "http://localhost:3000/source-feeds"
	}

	authorize := func(provider string, conf *oauth2.Config) func(*fiber.Ctx) error {
		return func(c *fiber.Ctx) error {
			principal, err := httpcontext.RequirePrincipal(c)
			if err != nil {
				return err
			}
			var body struct {
				DomainID uuid.UUID `json:"domain_id"`
			}
			_ = c.BodyParser(&body)
			if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
				return err
			}
			if conf == nil {
				return fiber.NewError(fiber.StatusServiceUnavailable, provider+": OAuth is not configured on this server")
			}
			key, err := oauthStateHMACKey(cfg)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			state, err := connectoroauth.SignState(key, principal, provider, 15*time.Minute)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			opts := []oauth2.AuthCodeOption{
				oauth2.AccessTypeOffline,
				oauth2.SetAuthURLParam("prompt", "consent"),
			}
			authURL := conf.AuthCodeURL(state, opts...)
			return c.JSON(fiber.Map{"authorize_url": authURL})
		}
	}

	f.Post("/integrations/oauth/gmail/authorize-url", authorize("gmail", gmailConf))
	f.Post("/integrations/oauth/microsoft/authorize-url", authorize("microsoft", msConf))

	callback := func(provider string, conf *oauth2.Config) func(*fiber.Ctx) error {
		return func(c *fiber.Ctx) error {
			code := strings.TrimSpace(c.Query("code"))
			stateStr := strings.TrimSpace(c.Query("state"))
			oerr := strings.TrimSpace(c.Query("error"))
			redirectWith := func(key, val string) error {
				u, parseErr := url.Parse(webBase)
				if parseErr != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "invalid OAUTH_WEB_REDIRECT_URL")
				}
				q := u.Query()
				q.Set(key, val)
				u.RawQuery = q.Encode()
				return c.Redirect(u.String(), fiber.StatusFound)
			}
			if oerr != "" {
				return redirectWith("oauth_error", oerr)
			}
			if code == "" || stateStr == "" {
				return redirectWith("oauth_error", "missing_code")
			}
			key, err := oauthStateHMACKey(cfg)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			pl, err := connectoroauth.VerifyState(key, stateStr)
			if err != nil || pl.Provider != provider {
				return redirectWith("oauth_error", "bad_state")
			}
			if conf == nil {
				return redirectWith("oauth_error", "not_configured")
			}
			tok, err := conf.Exchange(c.UserContext(), code)
			if err != nil {
				return redirectWith("oauth_error", "token_exchange_failed")
			}
			patch, err := oauthTokenConnectorPatch(provider, tok)
			if err != nil {
				return redirectWith("oauth_error", "patch_build_failed")
			}
			sid := uuid.NewString()
			connectorOAuthPending.Put(sid, &connectoroauth.PendingEntry{
				Principal: pl.Principal,
				Provider:  provider,
				Patch:     patch,
				Expires:   time.Now().Add(5 * time.Minute),
			})
			u, err := url.Parse(webBase)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "invalid OAUTH_WEB_REDIRECT_URL")
			}
			q := u.Query()
			q.Set("oauth_sid", sid)
			u.RawQuery = q.Encode()
			return c.Redirect(u.String(), fiber.StatusFound)
		}
	}

	f.Get("/integrations/oauth/gmail/callback", callback("gmail", gmailConf))
	f.Get("/integrations/oauth/microsoft/callback", callback("microsoft", msConf))

	f.Post("/integrations/oauth/consume", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			OAuthSID string `json:"oauth_sid"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		sid := strings.TrimSpace(body.OAuthSID)
		if sid == "" {
			return fiber.NewError(fiber.StatusBadRequest, "oauth_sid required")
		}
		entry, ok := connectorOAuthPending.Take(sid)
		if !ok {
			return fiber.NewError(fiber.StatusGone, "oauth_sid expired or invalid")
		}
		if entry.Principal != principal {
			return fiber.NewError(fiber.StatusForbidden, "oauth_sid does not belong to this session")
		}
		var patch map[string]any
		if err := json.Unmarshal(entry.Patch, &patch); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "corrupt oauth patch")
		}
		return c.JSON(fiber.Map{"connector_config_patch": patch})
	})
}

func oauthTokenConnectorPatch(provider string, tok *oauth2.Token) (json.RawMessage, error) {
	if tok == nil {
		return nil, errors.New("nil token")
	}
	switch provider {
	case "gmail":
		m := map[string]string{
			"gmail_oauth_access_token": strings.TrimSpace(tok.AccessToken),
		}
		if rt := strings.TrimSpace(tok.RefreshToken); rt != "" {
			m["gmail_oauth_refresh_token"] = rt
		}
		if !tok.Expiry.IsZero() {
			m["gmail_oauth_expiry"] = tok.Expiry.UTC().Format(time.RFC3339)
		}
		return json.Marshal(m)
	case "microsoft":
		m := map[string]string{
			"graph_access_token": strings.TrimSpace(tok.AccessToken),
		}
		if rt := strings.TrimSpace(tok.RefreshToken); rt != "" {
			m["graph_refresh_token"] = rt
		}
		if !tok.Expiry.IsZero() {
			m["graph_token_expiry"] = tok.Expiry.UTC().Format(time.RFC3339)
		}
		return json.Marshal(m)
	default:
		return nil, fmt.Errorf("unknown provider %q", provider)
	}
}
