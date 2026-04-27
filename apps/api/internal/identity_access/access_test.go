package identity_access

import (
	"testing"
)

func TestLevelAllowsAction(t *testing.T) {
	tests := []struct {
		level  string
		action string
		want   bool
	}{
		{"read", "view", true},
		{"read", "search", true},
		{"read", "view_raw", true},
		{"read", "edit", false},
		{"write", "view_raw", true},
		{"write", "edit", true},
		{"write", "run_job", true},
		{"write", "approve", true},
		{"write", "create", true},
		{"write", "export", true},
		{"write", "manage_jobs", true},
		{"write", "manage_permissions", false},
		{"write", "manage_policies", false},
		{"read", "export", false},
		{"admin", "approve", true},
		{"admin", "manage_source_feed", true},
		{"admin", "manage_permissions", true},
		{"none", "view", false},
	}
	for _, tt := range tests {
		if got := levelAllowsAction(tt.level, tt.action); got != tt.want {
			t.Fatalf("levelAllowsAction(%q,%q)=%v want %v", tt.level, tt.action, got, tt.want)
		}
	}
}

func TestNormalizeAction(t *testing.T) {
	if got := NormalizeAction("manage_source_feed"); got != "manage_sources" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeAction("view"); got != "view" {
		t.Fatalf("got %q", got)
	}
}
