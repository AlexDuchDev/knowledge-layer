// Package privacy implements AI privacy: policy, detection, sanitization, vault, rehydration.
package privacy

// Taxonomy groups (documentation / policy UI); each maps to SensitiveEntityType string IDs.
//
// A — Always-sensitive personal identifiers
// B — Always-sensitive security/secrets
// C — Always-sensitive financial identifiers
// D — Always-sensitive legal/contractual identifiers
// E — Business-sensitive by default
// F — Context-sensitive / configurable (use policy rules per domain/scenario)

// SensitiveEntityType is a stable identifier for policy rules and traces.
type SensitiveEntityType string

const (
	// A — Personal identifiers
	EntityPersonFirstName SensitiveEntityType = "person_first_name"
	EntityPersonLastName  SensitiveEntityType = "person_last_name"
	EntityPersonName      SensitiveEntityType = "person_name"
	EntityEmail           SensitiveEntityType = "email"
	EntityPhone           SensitiveEntityType = "phone"
	EntityAddress         SensitiveEntityType = "address"
	EntityPersonalID      SensitiveEntityType = "personal_identifier"
	EntityGovernmentID    SensitiveEntityType = "government_id"

	// B — Secrets
	EntitySecuritySecret SensitiveEntityType = "security_secret"

	// C — Financial
	EntityInvoiceRef       SensitiveEntityType = "invoice_ref"
	EntityPaymentRef       SensitiveEntityType = "payment_ref"
	EntityTransactionID    SensitiveEntityType = "transaction_id"
	EntityFinancialAccount SensitiveEntityType = "financial_account"

	// D — Legal / contractual
	EntityContractRef SensitiveEntityType = "contract_ref"
	EntityLegalRef    SensitiveEntityType = "legal_ref"

	// E — Business-sensitive
	EntityCompanyName      SensitiveEntityType = "company_name"
	EntityCustomerID       SensitiveEntityType = "customer_id"
	EntityAccountID        SensitiveEntityType = "account_id"
	EntityContactPerson    SensitiveEntityType = "contact_person_name"
	EntityInternalCodename SensitiveEntityType = "internal_codename"
	EntityDealName         SensitiveEntityType = "deal_name"
	EntityIncidentID       SensitiveEntityType = "incident_id"
	EntityEmployeeID       SensitiveEntityType = "employee_id"

	// F — Configurable / catch-all
	EntityHRPerformance   SensitiveEntityType = "hr_performance"
	EntityStrategyTerm    SensitiveEntityType = "strategy_term"
	EntitySupportInternal SensitiveEntityType = "support_case_internal"
	EntityCustomPattern   SensitiveEntityType = "custom_pattern"

	// URL and generic ID patterns (pattern layer)
	EntityURL      SensitiveEntityType = "url"
	EntityUUIDLike SensitiveEntityType = "uuid_like"
)

// TaxonomyGroup returns A–F for documentation; unknown types return "F".
func TaxonomyGroup(t SensitiveEntityType) string {
	switch t {
	case EntityPersonFirstName, EntityPersonLastName, EntityPersonName, EntityEmail, EntityPhone,
		EntityAddress, EntityPersonalID, EntityGovernmentID:
		return "A"
	case EntitySecuritySecret:
		return "B"
	case EntityInvoiceRef, EntityPaymentRef, EntityTransactionID, EntityFinancialAccount:
		return "C"
	case EntityContractRef, EntityLegalRef:
		return "D"
	case EntityCompanyName, EntityCustomerID, EntityAccountID, EntityContactPerson,
		EntityInternalCodename, EntityDealName, EntityIncidentID, EntityEmployeeID:
		return "E"
	default:
		return "F"
	}
}

// AllSensitiveEntityTypes lists types that appear in default policy seed (extend as needed).
func AllSensitiveEntityTypes() []SensitiveEntityType {
	return []SensitiveEntityType{
		EntityPersonFirstName, EntityPersonLastName, EntityPersonName,
		EntityEmail, EntityPhone, EntityAddress, EntityPersonalID, EntityGovernmentID,
		EntitySecuritySecret,
		EntityInvoiceRef, EntityPaymentRef, EntityTransactionID, EntityFinancialAccount,
		EntityContractRef, EntityLegalRef,
		EntityCompanyName, EntityCustomerID, EntityAccountID, EntityContactPerson,
		EntityInternalCodename, EntityDealName, EntityIncidentID, EntityEmployeeID,
		EntityHRPerformance, EntityStrategyTerm, EntitySupportInternal, EntityCustomPattern,
		EntityURL, EntityUUIDLike,
	}
}
