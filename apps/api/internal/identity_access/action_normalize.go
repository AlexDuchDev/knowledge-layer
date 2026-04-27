package identity_access

import "strings"

// NormalizeAction maps legacy or alias action codes to the canonical permission catalog.
func NormalizeAction(action string) string {
	a := strings.TrimSpace(strings.ToLower(action))
	switch a {
	case "manage_source_feed":
		// Legacy name; catalog uses manage_sources (see migration 000017).
		return "manage_sources"
	default:
		return a
	}
}
