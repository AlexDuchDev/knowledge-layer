package identity_access

import "testing"

func TestSensitivityCode(t *testing.T) {
	if got := SensitivityCode(SensitivityPublicInternal); got != "public_internal" {
		t.Fatalf("got %q", got)
	}
	if got := SensitivityCode(SensitivityStrictlyConfidential); got != "strictly_confidential" {
		t.Fatalf("got %q", got)
	}
}
