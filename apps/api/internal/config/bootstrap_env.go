package config

import (
	"os"
	"strings"
)

// AutoBootstrapEmptyInstance controls startup creation of the first admin user + domain
// when the database has zero domains.
//
// Local (APP_ENV not staging/production): defaults to true unless AUTO_BOOTSTRAP_INSTANCE
// is 0, false, or no.
//
// Staging/production: only when AUTO_BOOTSTRAP_INSTANCE is 1 or true (explicit opt-in).
func (c Config) AutoBootstrapEmptyInstance() bool {
	v := strings.TrimSpace(os.Getenv("AUTO_BOOTSTRAP_INSTANCE"))
	if c.IsNonLocal() {
		return v == "1" || strings.EqualFold(v, "true")
	}
	if v == "0" || strings.EqualFold(v, "false") || strings.EqualFold(v, "no") {
		return false
	}
	return true
}

// BootstrapAdminEmail returns BOOTSTRAP_ADMIN_EMAIL, or a local-only default.
func (c Config) BootstrapAdminEmail() string {
	v := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL"))
	if v != "" {
		return v
	}
	if c.IsLocalDev() {
		return "admin@local.test"
	}
	return ""
}

// BootstrapAdminPassword returns BOOTSTRAP_ADMIN_PASSWORD, or a local-only default.
func (c Config) BootstrapAdminPassword() string {
	v := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"))
	if v != "" {
		return v
	}
	if c.IsLocalDev() {
		return "changeme12345"
	}
	return ""
}

// BootstrapAdminName returns BOOTSTRAP_ADMIN_NAME, or a sensible default.
func (c Config) BootstrapAdminName() string {
	v := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_NAME"))
	if v != "" {
		return v
	}
	if c.IsLocalDev() {
		return "Administrator"
	}
	return "Administrator"
}

// BootstrapDomainName returns BOOTSTRAP_DOMAIN_NAME, or "Default".
func (c Config) BootstrapDomainName() string {
	v := strings.TrimSpace(os.Getenv("BOOTSTRAP_DOMAIN_NAME"))
	if v != "" {
		return v
	}
	return "Default"
}
