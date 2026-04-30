package config

import (
	"encoding/hex"
	"os"
	"strings"
)

type Config struct {
	AppEnv                      string
	APIPort                     string
	DatabaseURL                 string
	OpenSearchURL               string
	OpenSearchAllowInsecureHTTP bool
	RedisURL                    string
	Neo4jURL                    string
	Neo4jUser                   string
	Neo4jPassword               string
	AuthMode                    string
	SessionSecret               string
	OpsAuthToken                string
	AllowSelfRegistration       bool
	AppPublicURL                string
	SMTPHost                    string
	SMTPPort                    string
	SMTPUser                    string
	SMTPPassword                string
	SMTPFrom                    string
	BuildVersion                string
	BlobStoreBackend            string
	BlobStoreS3Endpoint         string
	BlobStoreS3Region           string
	BlobStoreS3Bucket           string
	BlobStoreS3AccessKey        string
	BlobStoreS3SecretKey        string
	BlobStoreS3UseSSL           bool
	BlobStoreS3PathStyle        bool
	// Second Brain overlay (optional webhooks / bots).
	SecondBrainWebhookSecret       string
	TelegramBotToken               string
	SecondBrainBotScenarioCode     string
	MattermostOutgoingWebhookToken string
	// OAuth for connector onboarding (Gmail / Microsoft 365). Optional; empty disables authorize-url.
	APIPublicURL               string // Browser-reachable API base for OAuth redirect_uri (default: APP_PUBLIC_URL).
	OAuthWebRedirectURL        string // Web URL after OAuth (query oauth_sid appended). Default http://localhost:3000/source-feeds
	GmailOAuthClientID         string
	GmailOAuthClientSecret     string
	MicrosoftOAuthClientID     string
	MicrosoftOAuthClientSecret string
	MicrosoftOAuthTenant       string // default "common"
	// L1 in-process cache for hot read paths. Off by default to keep
	// memory ceiling predictable; flip on for the API container only
	// (workers don't need it).
	CacheL1Enabled    bool
	CacheL1TTLSeconds int
	CacheL1MaxMB      int
	// OAuth 2.1 proxy fronting an operator-supplied OIDC issuer (v0.5.0).
	// Off by default. Required for the v0.5.1 MCP endpoint when shipped.
	OAuthProxyEnabled  bool
	OAuthSecretKey     string // ≥32 bytes; signs OAuth state + JWTs
	OIDCIssuerURL      string // operator's IDP discovery URL
	OIDCClientID       string // proxy's client_id at the IDP
	OIDCClientSecret   string // proxy's client_secret at the IDP
	OIDCSubColumn      string // users-table column for OIDC sub matching: "email" (default) or "external_idp_subject"
	OAuthProxyIssuer   string // proxy's own issuer URL for .well-known (defaults to APP_PUBLIC_URL)
	OAuthProxyCallback string // proxy's /oauth/callback URL (defaults to ${OAuthProxyIssuer}/oauth/callback)
	// MCP endpoint (v0.5.1). Off by default. When MCP_ENABLED=true the
	// /mcp route is mounted and bearer-protected via OAuth proxy. Hardening
	// rejects MCP_ENABLED=true without OAUTH_PROXY_ENABLED=true.
	MCPEnabled bool
}

func Load() Config {
	return Config{
		AppEnv:                         strings.ToLower(strings.TrimSpace(getenv("APP_ENV", "local"))),
		APIPort:                        getenv("API_PORT", "8080"),
		DatabaseURL:                    getenv("DATABASE_URL", "postgres://knowledge:knowledge@localhost:5432/knowledge?sslmode=disable"),
		OpenSearchURL:                  getenv("OPENSEARCH_URL", ""),
		OpenSearchAllowInsecureHTTP:    getenv("OPENSEARCH_ALLOW_INSECURE_HTTP", "") == "1" || strings.ToLower(getenv("OPENSEARCH_ALLOW_INSECURE_HTTP", "")) == "true",
		RedisURL:                       getenv("REDIS_URL", ""),
		Neo4jURL:                       getenv("NEO4J_URL", ""),
		Neo4jUser:                      getenv("NEO4J_USER", ""),
		Neo4jPassword:                  getenv("NEO4J_PASSWORD", ""),
		AuthMode:                       strings.ToLower(strings.TrimSpace(getenv("AUTH_MODE", "development_header"))),
		SessionSecret:                  getenv("SESSION_SECRET", ""),
		OpsAuthToken:                   getenv("OPS_AUTH_TOKEN", ""),
		AllowSelfRegistration:          getenv("ALLOW_SELF_REGISTRATION", "false") == "true",
		AppPublicURL:                   strings.TrimRight(getenv("APP_PUBLIC_URL", "http://localhost:8080"), "/"),
		SMTPHost:                       getenv("SMTP_HOST", ""),
		SMTPPort:                       getenv("SMTP_PORT", "587"),
		SMTPUser:                       getenv("SMTP_USER", ""),
		SMTPPassword:                   getenv("SMTP_PASSWORD", ""),
		SMTPFrom:                       getenv("SMTP_FROM", ""),
		BuildVersion:                   getenv("BUILD_VERSION", "dev"),
		BlobStoreBackend:               strings.ToLower(strings.TrimSpace(getenv("BLOBSTORE_BACKEND", ""))),
		BlobStoreS3Endpoint:            getenv("BLOBSTORE_S3_ENDPOINT", ""),
		BlobStoreS3Region:              getenv("BLOBSTORE_S3_REGION", "us-east-1"),
		BlobStoreS3Bucket:              getenv("BLOBSTORE_S3_BUCKET", ""),
		BlobStoreS3AccessKey:           getenv("BLOBSTORE_S3_ACCESS_KEY", ""),
		BlobStoreS3SecretKey:           getenv("BLOBSTORE_S3_SECRET_KEY", ""),
		BlobStoreS3UseSSL:              getenv("BLOBSTORE_S3_USE_SSL", "true") == "1" || strings.EqualFold(getenv("BLOBSTORE_S3_USE_SSL", "true"), "true"),
		BlobStoreS3PathStyle:           getenv("BLOBSTORE_S3_PATH_STYLE", "") == "1" || strings.EqualFold(getenv("BLOBSTORE_S3_PATH_STYLE", ""), "true"),
		SecondBrainWebhookSecret:       getenv("SECOND_BRAIN_WEBHOOK_SECRET", ""),
		TelegramBotToken:               getenv("TELEGRAM_BOT_TOKEN", ""),
		SecondBrainBotScenarioCode:     getenv("SECOND_BRAIN_BOT_SCENARIO_CODE", ""),
		MattermostOutgoingWebhookToken: getenv("MATTERMOST_OUTGOING_WEBHOOK_TOKEN", ""),
		APIPublicURL:                   strings.TrimRight(getenv("API_PUBLIC_URL", ""), "/"),
		OAuthWebRedirectURL:            strings.TrimSpace(getenv("OAUTH_WEB_REDIRECT_URL", "http://localhost:3000/source-feeds")),
		GmailOAuthClientID:             getenv("GMAIL_OAUTH_CLIENT_ID", ""),
		GmailOAuthClientSecret:         getenv("GMAIL_OAUTH_CLIENT_SECRET", ""),
		MicrosoftOAuthClientID:         getenv("MICROSOFT_OAUTH_CLIENT_ID", ""),
		MicrosoftOAuthClientSecret:     getenv("MICROSOFT_OAUTH_CLIENT_SECRET", ""),
		MicrosoftOAuthTenant:           strings.TrimSpace(getenv("MICROSOFT_OAUTH_TENANT", "common")),
		CacheL1Enabled:                 strings.EqualFold(getenv("CACHE_L1_ENABLED", "false"), "true") || getenv("CACHE_L1_ENABLED", "") == "1",
		CacheL1TTLSeconds:              parsePositiveInt(getenv("CACHE_L1_TTL_SECONDS", "60"), 60),
		CacheL1MaxMB:                   parsePositiveInt(getenv("CACHE_L1_MAX_MB", "64"), 64),
		OAuthProxyEnabled:              strings.EqualFold(getenv("OAUTH_PROXY_ENABLED", "false"), "true") || getenv("OAUTH_PROXY_ENABLED", "") == "1",
		OAuthSecretKey:                 getenv("OAUTH_SECRET_KEY", ""),
		OIDCIssuerURL:                  strings.TrimSpace(getenv("OIDC_ISSUER_URL", "")),
		OIDCClientID:                   strings.TrimSpace(getenv("OIDC_CLIENT_ID", "")),
		OIDCClientSecret:               getenv("OIDC_CLIENT_SECRET", ""),
		OIDCSubColumn:                  strings.ToLower(strings.TrimSpace(getenv("OIDC_SUB_COLUMN", "email"))),
		OAuthProxyIssuer:               strings.TrimRight(getenv("OAUTH_PROXY_ISSUER", ""), "/"),
		OAuthProxyCallback:             strings.TrimSpace(getenv("OAUTH_PROXY_CALLBACK", "")),
		MCPEnabled:                     strings.EqualFold(getenv("MCP_ENABLED", "false"), "true") || getenv("MCP_ENABLED", "") == "1",
	}
}

func parsePositiveInt(s string, def int) int {
	v := strings.TrimSpace(s)
	n := 0
	for _, r := range v {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	if n <= 0 {
		return def
	}
	return n
}

// APIPublicOrigin returns the public base URL of this API (for OAuth redirect_uri). Falls back to AppPublicURL.
func (c Config) APIPublicOrigin() string {
	if u := strings.TrimRight(strings.TrimSpace(c.APIPublicURL), "/"); u != "" {
		return u
	}
	return strings.TrimRight(strings.TrimSpace(c.AppPublicURL), "/")
}

// SessionSecretBytes decodes SESSION_SECRET as hex or uses raw bytes (min 16 bytes).
func (c Config) SessionSecretBytes() []byte {
	if c.SessionSecret == "" {
		return nil
	}
	if b, err := hex.DecodeString(c.SessionSecret); err == nil && len(b) >= 16 {
		return b
	}
	if len(c.SessionSecret) >= 16 {
		return []byte(c.SessionSecret)
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
