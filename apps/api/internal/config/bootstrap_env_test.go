package config

import (
	"os"
	"testing"
)

func TestAutoBootstrapEmptyInstance(t *testing.T) {
	t.Run("local default on", func(t *testing.T) {
		t.Setenv("APP_ENV", "local")
		t.Setenv("AUTO_BOOTSTRAP_INSTANCE", "")
		if !Load().AutoBootstrapEmptyInstance() {
			t.Fatal("expected auto-bootstrap on for local when unset")
		}
	})
	t.Run("local explicit off", func(t *testing.T) {
		t.Setenv("APP_ENV", "local")
		t.Setenv("AUTO_BOOTSTRAP_INSTANCE", "0")
		if Load().AutoBootstrapEmptyInstance() {
			t.Fatal("expected auto-bootstrap off")
		}
	})
	t.Run("staging default off", func(t *testing.T) {
		t.Setenv("APP_ENV", "staging")
		t.Setenv("AUTO_BOOTSTRAP_INSTANCE", "")
		if Load().AutoBootstrapEmptyInstance() {
			t.Fatal("expected auto-bootstrap off for staging unless explicit")
		}
	})
	t.Run("staging explicit on", func(t *testing.T) {
		t.Setenv("APP_ENV", "staging")
		t.Setenv("AUTO_BOOTSTRAP_INSTANCE", "1")
		if !Load().AutoBootstrapEmptyInstance() {
			t.Fatal("expected auto-bootstrap on")
		}
	})
}

func TestBootstrapAdminDefaults_Local(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("BOOTSTRAP_ADMIN_EMAIL", "")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "")
	_ = os.Unsetenv("BOOTSTRAP_ADMIN_EMAIL")
	_ = os.Unsetenv("BOOTSTRAP_ADMIN_PASSWORD")
	cfg := Load()
	if cfg.BootstrapAdminEmail() != "admin@local.test" {
		t.Fatalf("email: %q", cfg.BootstrapAdminEmail())
	}
	if cfg.BootstrapAdminPassword() != "changeme12345" {
		t.Fatalf("password default")
	}
	if cfg.BootstrapDomainName() != "Default" {
		t.Fatalf("domain default")
	}
}
