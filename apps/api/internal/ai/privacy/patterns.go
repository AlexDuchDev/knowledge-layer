package privacy

import (
	"regexp"
)

// Pattern priorities: higher wins on overlap (secrets before generic UUID).
const (
	prioSecret     = 100
	prioEmail      = 85
	prioPhone      = 80
	prioIBAN       = 75
	prioContract   = 72
	prioInvoice    = 70
	prioUUID       = 55
	prioURL        = 45
	prioStructured = 25
	prioNER        = 10
)

var (
	reEmail = regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)

	rePhone = regexp.MustCompile(`(?i)(?:\+?\d{1,3}[\s\-]?)?(?:\(?\d{2,4}\)?[\s\-]?)?\d{3}[\s\-]?\d{3}[\s\-]?\d{3,4}\b`)

	reBearer    = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9\-._~+/]+=*\b`)
	reSK        = regexp.MustCompile(`\bsk-[A-Za-z0-9]{10,}\b`)
	reAWSKey    = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	reGitHubPat = regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}\b`)

	reUUID = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)

	reURL = regexp.MustCompile(`https?://[^\s<>()]+`)

	reContract = regexp.MustCompile(`(?i)\b(?:CT|CNT|CONTRACT)[-–—/]?\d{4}[-–/]?\d{2,}\b`)
	reInvoice  = regexp.MustCompile(`(?i)\b(?:INV|INVOICE)[-–—#/]?\d{4,}\b`)

	reIBAN = regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9]{1,30}\b`)
)

type detectedSpan struct {
	Start, End int
	Type       SensitiveEntityType
	Value      string
	Priority   int
}

func findPatternSpans(text string) []detectedSpan {
	var spans []detectedSpan
	add := func(prio int, typ SensitiveEntityType, loc []int) {
		if len(loc) != 2 {
			return
		}
		spans = append(spans, detectedSpan{
			Start: loc[0], End: loc[1], Type: typ,
			Value: text[loc[0]:loc[1]], Priority: prio,
		})
	}

	for _, loc := range reEmail.FindAllStringIndex(text, -1) {
		add(prioEmail, EntityEmail, loc)
	}
	for _, loc := range rePhone.FindAllStringIndex(text, -1) {
		add(prioPhone, EntityPhone, loc)
	}
	for _, loc := range reBearer.FindAllStringIndex(text, -1) {
		add(prioSecret, EntitySecuritySecret, loc)
	}
	for _, loc := range reSK.FindAllStringIndex(text, -1) {
		add(prioSecret, EntitySecuritySecret, loc)
	}
	for _, loc := range reAWSKey.FindAllStringIndex(text, -1) {
		add(prioSecret, EntitySecuritySecret, loc)
	}
	for _, loc := range reGitHubPat.FindAllStringIndex(text, -1) {
		add(prioSecret, EntitySecuritySecret, loc)
	}
	for _, loc := range reContract.FindAllStringIndex(text, -1) {
		add(prioContract, EntityContractRef, loc)
	}
	for _, loc := range reInvoice.FindAllStringIndex(text, -1) {
		add(prioInvoice, EntityInvoiceRef, loc)
	}
	for _, loc := range reIBAN.FindAllStringIndex(text, -1) {
		if likelyIBAN(text[loc[0]:loc[1]]) {
			add(prioIBAN, EntityFinancialAccount, loc)
		}
	}
	for _, loc := range reURL.FindAllStringIndex(text, -1) {
		add(prioURL, EntityURL, loc)
	}
	for _, loc := range reUUID.FindAllStringIndex(text, -1) {
		add(prioUUID, EntityUUIDLike, loc)
	}
	return spans
}

func likelyIBAN(s string) bool {
	if len(s) < 15 || len(s) > 34 {
		return false
	}
	for _, r := range s {
		if r != ' ' && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// mergeSpansInText builds non-overlapping spans; higher priority wins; value taken from text.
func mergeSpansInText(text string, spans []detectedSpan) []detectedSpan {
	if len(spans) == 0 {
		return nil
	}
	for i := range spans {
		if spans[i].Start < 0 {
			spans[i].Start = 0
		}
		if spans[i].End > len(text) {
			spans[i].End = len(text)
		}
		spans[i].Value = text[spans[i].Start:spans[i].End]
	}
	for i := 0; i < len(spans); i++ {
		for j := i + 1; j < len(spans); j++ {
			if spans[j].Start < spans[i].Start || (spans[j].Start == spans[i].Start && spans[j].Priority > spans[i].Priority) {
				spans[i], spans[j] = spans[j], spans[i]
			}
		}
	}
	var out []detectedSpan
	cur := -1
	for _, s := range spans {
		if s.End <= cur {
			continue
		}
		if s.Start < cur {
			s.Start = cur
		}
		if s.Start >= s.End {
			continue
		}
		s.Value = text[s.Start:s.End]
		out = append(out, s)
		cur = s.End
	}
	return out
}

// addStructuredGapSpans fills gaps between pattern spans with structured type (structured-first for field semantics; patterns override locally).
func addStructuredGapSpans(text string, structured SensitiveEntityType, patterns []detectedSpan) []detectedSpan {
	patMerged := mergeSpansInText(text, patterns)
	if structured == "" {
		return patMerged
	}
	if len(patMerged) == 0 && len(text) > 0 {
		return []detectedSpan{{
			Start: 0, End: len(text), Type: structured, Priority: prioStructured, Value: text,
		}}
	}
	var spans []detectedSpan
	pos := 0
	for _, m := range patMerged {
		if pos < m.Start {
			spans = append(spans, detectedSpan{
				Start: pos, End: m.Start, Type: structured, Priority: prioStructured,
				Value: text[pos:m.Start],
			})
		}
		spans = append(spans, m)
		pos = m.End
	}
	if pos < len(text) {
		spans = append(spans, detectedSpan{
			Start: pos, End: len(text), Type: structured, Priority: prioStructured,
			Value: text[pos:],
		})
	}
	return mergeSpansInText(text, spans)
}

func appendNERSpans(text string, base []detectedSpan, ner []detectedSpan) []detectedSpan {
	for i := range ner {
		ner[i].Priority = prioNER
	}
	return mergeSpansInText(text, append(base, ner...))
}
