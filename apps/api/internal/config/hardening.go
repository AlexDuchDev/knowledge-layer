package config

import (
	"fmt"
	"os"
	"strings"
)

// IsNonLocal is true for staging and production profiles (stricter security rules apply).
func (c Config) IsNonLocal() bool {
	switch strings.ToLower(strings.TrimSpace(c.AppEnv)) {
	case "staging", "production":
		return true
	default:
		return false
	}
}

// IsLocalDev is true when IsNonLocal is false (local, development, test, etc.).
func (c Config) IsLocalDev() bool {
	return !c.IsNonLocal()
}

// IsProduction is true only for APP_ENV=production (stricter than staging in a few checks).
func (c Config) IsProduction() bool {
	return strings.ToLower(strings.TrimSpace(c.AppEnv)) == "production"
}

// SessionCookieSecure controls the session cookie Secure flag.
// SESSION_COOKIE_SECURE=1|true forces secure; 0|false forces off; unset defaults to secure when IsNonLocal.
func (c Config) SessionCookieSecure() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("SESSION_COOKIE_SECURE")))
	switch v {
	case "false", "0", "no":
		return false
	case "true", "1", "yes":
		return true
	default:
		return c.IsNonLocal()
	}
}

func validateOpenSearchURL(cfg Config) error {
	u := strings.TrimSpace(cfg.OpenSearchURL)
	if u == "" {
		return nil
	}
	lower := strings.ToLower(u)
	if strings.HasPrefix(lower, "http://") && !cfg.OpenSearchAllowInsecureHTTP {
		return fmt.Errorf("OPENSEARCH_URL must use https when APP_ENV=%q (or set OPENSEARCH_ALLOW_INSECURE_HTTP=1 only on private networks)",
			cfg.AppEnv)
	}
	return nil
}

const minOpsAuthTokenLen = 16

// validatePrivacyVaultProduction enforces that the AI privacy vault is wired
// with a real AES-256 key in production (no dev-plaintext fallback). The vault
// stores sensitive placeholder cleartext for sanitize/rehydrate flows; running
// production with plaintext-at-rest defeats the privacy boundary.
func validatePrivacyVaultProduction() error {
	if strings.TrimSpace(os.Getenv("AI_PRIVACY_DEV_PLAINTEXT_STORE")) == "1" {
		return fmt.Errorf("AI_PRIVACY_DEV_PLAINTEXT_STORE=1 is not allowed when APP_ENV=production (vault must encrypt placeholder mappings)")
	}
	key := strings.TrimSpace(os.Getenv("AI_PRIVACY_VAULT_KEY"))
	if key == "" {
		return fmt.Errorf("AI_PRIVACY_VAULT_KEY must be set when APP_ENV=production (32 raw bytes or base64-of-32 for AES-256)")
	}
	if len(key) < 32 {
		return fmt.Errorf("AI_PRIVACY_VAULT_KEY must be 32 raw bytes or base64-of-32 bytes when APP_ENV=production (got length %d)", len(key))
	}
	return nil
}

// ValidateAPI enforces fail-closed rules for the HTTP API outside local dev.
func ValidateAPI(cfg Config) error {
	if cfg.IsNonLocal() {
		if cfg.AuthMode == "development_header" {
			return fmt.Errorf("AUTH_MODE=development_header is not allowed when APP_ENV=%q; use session", cfg.AppEnv)
		}
		if cfg.AuthMode != "session" {
			return fmt.Errorf("AUTH_MODE must be session when APP_ENV=%q (got %q)", cfg.AppEnv, cfg.AuthMode)
		}
		if len(cfg.SessionSecretBytes()) == 0 {
			return fmt.Errorf("SESSION_SECRET is required (min 16 bytes raw or hex) when APP_ENV=%q", cfg.AppEnv)
		}
		if os.Getenv("DATABASE_URL") == "" {
			return fmt.Errorf("DATABASE_URL must be set explicitly when APP_ENV=%q", cfg.AppEnv)
		}
		if os.Getenv("REDIS_URL") == "" {
			return fmt.Errorf("REDIS_URL must be set explicitly when APP_ENV=%q (queue-backed jobs require Redis)", cfg.AppEnv)
		}
		if os.Getenv("NEO4J_URL") == "" {
			return fmt.Errorf("NEO4J_URL must be set explicitly when APP_ENV=%q (GraphRAG requires Neo4j)", cfg.AppEnv)
		}
		if os.Getenv("CORS_ALLOW_ORIGINS") == "" {
			return fmt.Errorf("CORS_ALLOW_ORIGINS must be set when APP_ENV=%q", cfg.AppEnv)
		}
		if err := validateOpenSearchURL(cfg); err != nil {
			return err
		}
		if cfg.IsProduction() {
			tok := strings.TrimSpace(os.Getenv("OPS_AUTH_TOKEN"))
			if len(tok) < minOpsAuthTokenLen {
				return fmt.Errorf("OPS_AUTH_TOKEN must be set in the environment (min %d chars) when APP_ENV=production so /ops/health and /metrics are not anonymously reachable with full or stub detail", minOpsAuthTokenLen)
			}
			pub := strings.TrimSpace(cfg.AppPublicURL)
			if !strings.HasPrefix(strings.ToLower(pub), "https://") {
				return fmt.Errorf("APP_PUBLIC_URL must use https:// when APP_ENV=production (got %q)", pub)
			}
			if !cfg.SessionCookieSecure() {
				return fmt.Errorf("SESSION_COOKIE_SECURE must not be disabled when APP_ENV=production (cookies require HTTPS)")
			}
			if err := validatePrivacyVaultProduction(); err != nil {
				return err
			}
		}
		if cfg.AutoBootstrapEmptyInstance() && cfg.IsNonLocal() {
			if cfg.BootstrapAdminEmail() == "" {
				return fmt.Errorf("BOOTSTRAP_ADMIN_EMAIL is required when AUTO_BOOTSTRAP_INSTANCE is enabled and APP_ENV=%q", cfg.AppEnv)
			}
			if len(cfg.BootstrapAdminPassword()) < 8 {
				return fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD must be at least 8 characters when AUTO_BOOTSTRAP_INSTANCE is enabled for APP_ENV=%q", cfg.AppEnv)
			}
		}
	}
	return nil
}

// ValidateWorker enforces minimal production rules for background workers.
func ValidateWorker(cfg Config) error {
	if cfg.IsNonLocal() {
		if os.Getenv("DATABASE_URL") == "" {
			return fmt.Errorf("DATABASE_URL must be set explicitly when APP_ENV=%q", cfg.AppEnv)
		}
		if os.Getenv("REDIS_URL") == "" {
			return fmt.Errorf("REDIS_URL must be set explicitly when APP_ENV=%q", cfg.AppEnv)
		}
		if os.Getenv("NEO4J_URL") == "" {
			return fmt.Errorf("NEO4J_URL must be set explicitly when APP_ENV=%q (GraphRAG requires Neo4j)", cfg.AppEnv)
		}
		if err := validateOpenSearchURL(cfg); err != nil {
			return err
		}
		if cfg.IsProduction() {
			if err := validatePrivacyVaultProduction(); err != nil {
				return err
			}
		}
	}
	return nil
}
