package privacy

import (
	"fmt"
	"strings"
	"sync"
)

// PlaceholderMapping records one tokenization for vault persistence (no raw value in traces).
type PlaceholderMapping struct {
	Placeholder string
	EntityType  SensitiveEntityType
}

// PlaceholderTokenizer assigns stable TYPE_N placeholders per correlation run.
type PlaceholderTokenizer struct {
	mu               sync.Mutex
	byKey            map[string]string // entityType + NUL + value -> placeholder
	rawByPlaceholder map[string]struct {
		Value string
		Type  SensitiveEntityType
	}
	counts map[string]int // tag -> next seq
	order  []PlaceholderMapping
}

// NewPlaceholderTokenizer creates a tokenizer for one request/job correlation id.
func NewPlaceholderTokenizer() *PlaceholderTokenizer {
	return &PlaceholderTokenizer{
		byKey: map[string]string{},
		rawByPlaceholder: map[string]struct {
			Value string
			Type  SensitiveEntityType
		}{},
		counts: map[string]int{},
	}
}

func typeTag(t SensitiveEntityType) string {
	switch t {
	case EntityEmail:
		return "EMAIL"
	case EntityPhone:
		return "PHONE"
	case EntityPersonName, EntityPersonFirstName, EntityPersonLastName, EntityContactPerson:
		return "PERSON"
	case EntityCompanyName:
		return "COMPANY"
	case EntityCustomerID:
		return "CUSTOMER_ID"
	case EntityAccountID:
		return "ACCOUNT_ID"
	case EntityContractRef:
		return "CONTRACT"
	case EntityInvoiceRef:
		return "INVOICE"
	case EntitySecuritySecret:
		return "SECRET"
	case EntityURL:
		return "URL"
	case EntityUUIDLike:
		return "UUID"
	case EntityFinancialAccount:
		return "FINANCIAL"
	case EntityGovernmentID:
		return "GOV_ID"
	case EntityLegalRef:
		return "LEGAL"
	case EntityInternalCodename:
		return "CODENAME"
	case EntityAddress:
		return "ADDRESS"
	default:
		return strings.ToUpper(strings.ReplaceAll(string(t), " ", "_"))
	}
}

// Placeholder returns a stable placeholder for value within this tokenizer scope.
func (t *PlaceholderTokenizer) Placeholder(typ SensitiveEntityType, value string) string {
	key := string(typ) + "\x00" + value
	t.mu.Lock()
	defer t.mu.Unlock()
	if p, ok := t.byKey[key]; ok {
		return p
	}
	tag := typeTag(typ)
	t.counts[tag]++
	n := t.counts[tag]
	p := fmt.Sprintf("%s_%d", tag, n)
	t.byKey[key] = p
	t.rawByPlaceholder[p] = struct {
		Value string
		Type  SensitiveEntityType
	}{Value: value, Type: typ}
	t.order = append(t.order, PlaceholderMapping{Placeholder: p, EntityType: typ})
	return p
}

// Mappings returns tokenization records for the vault (values held by caller until encrypted).
func (t *PlaceholderTokenizer) Mappings() []PlaceholderMapping {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]PlaceholderMapping, len(t.order))
	copy(out, t.order)
	return out
}

// RawValueByPlaceholder returns the original value for a placeholder (in-memory only; used before vault persist).
func (t *PlaceholderTokenizer) RawValueByPlaceholder(placeholder string) (string, SensitiveEntityType, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if e, ok := t.rawByPlaceholder[placeholder]; ok {
		return e.Value, e.Type, true
	}
	return "", "", false
}
