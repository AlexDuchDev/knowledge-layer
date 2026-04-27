package privacy

import "strings"

// StructuredTypeForField returns a sensitive type implied by a known field path (structured-first).
// False means "no blanket structured type" — pattern / NER layers still run.
func StructuredTypeForField(field string) (SensitiveEntityType, bool) {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "title":
		// Business-facing default: titles often name companies, products, or deals.
		return EntityCompanyName, true
	case "external_ref":
		return EntityContractRef, true
	default:
		return "", false
	}
}
