package config

import (
	"os"
	"testing"
)

func TestIsNonLocal(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"local", false},
		{"development", false},
		{"staging", true},
		{"production", true},
		{"  STAGING  ", true},
	}
	for _, tc := range cases {
		cfg := Config{AppEnv: tc.env}
		if got := cfg.IsNonLocal(); got != tc.want {
			t.Fatalf("AppEnv=%q IsNonLocal=%v want %v", tc.env, got, tc.want)
		}
	}
}

func TestIsProduction(t *testing.T) {
	if !(Config{AppEnv: "production"}).IsProduction() {
		t.Fatal("expected production")
	}
	if (Config{AppEnv: "staging"}).IsProduction() {
		t.Fatal("staging is not production")
	}
}

func TestValidateAPI_ProductionRequiresSessionAndSecrets(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@db.example:5432/app?sslmode=require")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://app.example")
	t.Setenv("REDIS_URL", "redis://redis:6379")
	t.Setenv("NEO4J_URL", "bolt://neo4j:7687")
	t.Cleanup(func() {
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("CORS_ALLOW_ORIGINS")
		_ = os.Unsetenv("REDIS_URL")
		_ = os.Unsetenv("NEO4J_URL")
		_ = os.Unsetenv("OPS_AUTH_TOKEN")
		_ = os.Unsetenv("SESSION_COOKIE_SECURE")
		_ = os.Unsetenv("AI_PRIVACY_VAULT_KEY")
		_ = os.Unsetenv("AI_PRIVACY_DEV_PLAINTEXT_STORE")
	})

	cfg := Config{
		AppEnv:        "production",
		AuthMode:      "development_header",
		SessionSecret: "0123456789abcdef0123456789abcdef",
		OpenSearchURL: "",
		AppPublicURL:  "https://api.example",
	}
	if err := ValidateAPI(cfg); err == nil {
		t.Fatal("expected error for development_header in production")
	}

	cfg.AuthMode = "session"
	cfg.SessionSecret = "short"
	if err := ValidateAPI(cfg); err == nil {
		t.Fatal("expected error for short SESSION_SECRET")
	}

	cfg.SessionSecret = "0123456789abcdef0123456789abcdef"
	cfg.OpenSearchURL = "http://opensearch:9200"
	if err := ValidateAPI(cfg); err == nil {
		t.Fatal("expected error for http OpenSearch without allow flag")
	}

	cfg.OpenSearchAllowInsecureHTTP = true
	if err := ValidateAPI(cfg); err == nil {
		t.Fatal("expected error: OPS_AUTH_TOKEN missing for production")
	}

	t.Setenv("OPS_AUTH_TOKEN", "0123456789abcdef0123456789abcdef")
	cfg.AppPublicURL = "http://api.example"
	if err := ValidateAPI(cfg); err == nil {
		t.Fatal("expected error: APP_PUBLIC_URL must be https in production")
	}

	cfg.AppPublicURL = "https://api.example"
	t.Setenv("SESSION_COOKIE_SECURE", "false")
	if err := ValidateAPI(cfg); err == nil {
		t.Fatal("expected error: SESSION_COOKIE_SECURE false in production")
	}

	t.Setenv("SESSION_COOKIE_SECURE", "")
	if err := ValidateAPI(cfg); err == nil {
		t.Fatal("expected error: AI_PRIVACY_VAULT_KEY missing in production")
	}

	t.Setenv("AI_PRIVACY_DEV_PLAINTEXT_STORE", "1")
	if err := ValidateAPI(cfg); err == nil {
		t.Fatal("expected error: dev-plaintext store forbidden in production")
	}

	_ = os.Unsetenv("AI_PRIVACY_DEV_PLAINTEXT_STORE")
	t.Setenv("AI_PRIVACY_VAULT_KEY", "short")
	if err := ValidateAPI(cfg); err == nil {
		t.Fatal("expected error: AI_PRIVACY_VAULT_KEY too short for production")
	}

	t.Setenv("AI_PRIVACY_VAULT_KEY", "0123456789abcdef0123456789abcdef")
	if err := ValidateAPI(cfg); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateAPI_StagingRequiresRedisOptionalOpsToken(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@db:5432/x?sslmode=require")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://staging.example")
	t.Setenv("REDIS_URL", "redis://redis:6379")
	t.Setenv("NEO4J_URL", "bolt://neo4j:7687")
	_ = os.Unsetenv("OPS_AUTH_TOKEN")
	t.Cleanup(func() {
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("CORS_ALLOW_ORIGINS")
		_ = os.Unsetenv("REDIS_URL")
		_ = os.Unsetenv("NEO4J_URL")
	})

	cfg := Config{
		AppEnv:        "staging",
		AuthMode:      "session",
		SessionSecret: "0123456789abcdef0123456789abcdef",
		AppPublicURL:  "https://staging.example",
		OpenSearchURL: "",
	}
	if err := ValidateAPI(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAPI_LocalAllowsDevHeader(t *testing.T) {
	cfg := Config{
		AppEnv:      "local",
		AuthMode:    "development_header",
		DatabaseURL: "postgres://localhost/x",
	}
	if err := ValidateAPI(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAPI_StagingAutoBootstrapRequiresCredentials(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@db:5432/x?sslmode=require")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://staging.example")
	t.Setenv("REDIS_URL", "redis://redis:6379")
	t.Setenv("NEO4J_URL", "bolt://neo4j:7687")
	t.Setenv("AUTO_BOOTSTRAP_INSTANCE", "1")
	_ = os.Unsetenv("BOOTSTRAP_ADMIN_EMAIL")
	_ = os.Unsetenv("BOOTSTRAP_ADMIN_PASSWORD")
	t.Cleanup(func() {
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("CORS_ALLOW_ORIGINS")
		_ = os.Unsetenv("REDIS_URL")
		_ = os.Unsetenv("NEO4J_URL")
		_ = os.Unsetenv("AUTO_BOOTSTRAP_INSTANCE")
	})
	cfg := Config{
		AppEnv:        "staging",
		AuthMode:      "session",
		SessionSecret: "0123456789abcdef0123456789abcdef",
		AppPublicURL:  "https://staging.example",
		OpenSearchURL: "",
	}
	if err := ValidateAPI(cfg); err == nil {
		t.Fatal("expected error when AUTO_BOOTSTRAP_INSTANCE without BOOTSTRAP_ADMIN_EMAIL")
	}
}

func TestValidateWorker_NonLocalRequiresRedis(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x:x@db:5432/x")
	t.Cleanup(func() { _ = os.Unsetenv("DATABASE_URL") })
	cfg := Config{AppEnv: "staging", OpenSearchURL: ""}
	_ = os.Unsetenv("REDIS_URL")
	if err := ValidateWorker(cfg); err == nil {
		t.Fatal("expected error without REDIS_URL")
	}
	t.Setenv("REDIS_URL", "redis://r:6379")
	t.Cleanup(func() { _ = os.Unsetenv("REDIS_URL") })
	t.Setenv("NEO4J_URL", "bolt://neo4j:7687")
	t.Cleanup(func() { _ = os.Unsetenv("NEO4J_URL") })
	if err := ValidateWorker(cfg); err != nil {
		t.Fatal(err)
	}
}
