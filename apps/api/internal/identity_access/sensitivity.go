package identity_access

// Canonical sensitivity levels (stored as INT in PostgreSQL). Higher means more restricted.
const (
	SensitivityPublicInternal       = 0
	SensitivityTeamRestricted       = 1
	SensitivityDomainRestricted     = 2
	SensitivityLeadershipRestricted = 3
	SensitivityStrictlyConfidential = 4
)

// SensitivityCode returns the stable string name for traces and documentation.
func SensitivityCode(level int) string {
	switch level {
	case SensitivityPublicInternal:
		return "public_internal"
	case SensitivityTeamRestricted:
		return "team_restricted"
	case SensitivityDomainRestricted:
		return "domain_restricted"
	case SensitivityLeadershipRestricted:
		return "leadership_restricted"
	case SensitivityStrictlyConfidential:
		return "strictly_confidential"
	default:
		return "unknown"
	}
}
