// Package pii detects and redacts personal data in submitted text
// (FR-MOD-3): email addresses, phone numbers and government-id-like patterns.
// It is pure: no I/O, no configuration, deterministic for a given input.
package pii

import (
	"regexp"
	"sort"
	"strings"
)

// Kind classifies a finding.
type Kind string

const (
	// KindEmail is an email address.
	KindEmail Kind = "email"
	// KindPhone is a phone number.
	KindPhone Kind = "phone"
	// KindID is a government identifier such as a national id or card number.
	KindID Kind = "id"
)

// Placeholder is the text each finding is replaced with. Keeping the kind in
// the placeholder tells the reader what was removed without keeping the value.
func (k Kind) Placeholder() string { return "[" + string(k) + "]" }

// Finding is one span of personal data in the source text.
type Finding struct {
	Kind  Kind
	Value string
	Start int
	End   int
}

// Patterns are deliberately conservative: a false positive only adds a
// confirmation step, while a false negative risks publishing personal data.
var (
	emailRe = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	phoneRe = regexp.MustCompile(`(?:\+?\d[\d\s().\-]{7,}\d)`)
	idRe    = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b|\b\d{4}[ -]?\d{4}[ -]?\d{4}[ -]?\d{4}\b`)
)

// Detect returns every personal-data span in text, ordered by position.
func Detect(text string) []Finding {
	var found []Finding
	for _, p := range []struct {
		kind Kind
		re   *regexp.Regexp
	}{
		{KindEmail, emailRe},
		{KindID, idRe},
		{KindPhone, phoneRe},
	} {
		for _, loc := range p.re.FindAllStringIndex(text, -1) {
			found = append(found, Finding{
				Kind:  p.kind,
				Value: text[loc[0]:loc[1]],
				Start: loc[0],
				End:   loc[1],
			})
		}
	}
	return dropOverlaps(found)
}

// Has reports whether text contains any personal data.
func Has(text string) bool { return len(Detect(text)) > 0 }

// Redact replaces each finding with its kind's placeholder. Spans detected more
// than once (email inside a longer phone-like run) are resolved by keeping the
// earliest, highest-priority match.
func Redact(text string) string {
	found := Detect(text)
	if len(found) == 0 {
		return text
	}
	var b strings.Builder
	last := 0
	for _, f := range found {
		if f.Start < last {
			continue
		}
		b.WriteString(text[last:f.Start])
		b.WriteString(f.Kind.Placeholder())
		last = f.End
	}
	b.WriteString(text[last:])
	return b.String()
}

// dropOverlaps sorts findings and removes any that overlap an earlier one.
func dropOverlaps(found []Finding) []Finding {
	sort.Slice(found, func(i, j int) bool {
		if found[i].Start != found[j].Start {
			return found[i].Start < found[j].Start
		}
		return found[i].End > found[j].End
	})
	out := found[:0]
	end := -1
	for _, f := range found {
		if f.Start < end {
			continue
		}
		out = append(out, f)
		end = f.End
	}
	return out
}
