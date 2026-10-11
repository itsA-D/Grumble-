package pii

import (
	"strings"
	"testing"
)

func TestDetectKinds(t *testing.T) {
	cases := []struct {
		name string
		text string
		kind Kind
	}{
		{"email", "reach me at jane.doe+news@example.co.uk please", KindEmail},
		{"phone", "call +1 (415) 555-2671 tomorrow", KindPhone},
		{"ssn", "my number is 123-45-6789 sadly", KindID},
		{"card", "card 4242 4242 4242 4242 was declined", KindID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := Detect(tc.text)
			if len(found) == 0 {
				t.Fatalf("no finding in %q", tc.text)
			}
			if found[0].Kind != tc.kind {
				t.Fatalf("kind = %s, want %s (findings %+v)", found[0].Kind, tc.kind, found)
			}
		})
	}
}

func TestDetectCleanText(t *testing.T) {
	if Has("The trains are always late and nobody explains why.") {
		t.Fatal("ordinary text flagged as personal data")
	}
}

func TestRedact(t *testing.T) {
	got := Redact("Email jane@example.com or call 415-555-2671 about order 123-45-6789.")
	if strings.Contains(got, "jane@example.com") || strings.Contains(got, "415-555-2671") || strings.Contains(got, "123-45-6789") {
		t.Fatalf("redaction left personal data: %q", got)
	}
	if !strings.Contains(got, "[email]") || !strings.Contains(got, "[phone]") || !strings.Contains(got, "[id]") {
		t.Fatalf("expected placeholders in %q", got)
	}
}

func TestRedactCleanTextUnchanged(t *testing.T) {
	text := "I hate when the bus is late."
	if got := Redact(text); got != text {
		t.Fatalf("clean text changed: %q", got)
	}
}

func TestDetectOverlapKeepsLongest(t *testing.T) {
	// A 16-digit card also matches the looser phone pattern; the id wins.
	found := Detect("4111111111111111")
	if len(found) != 1 || found[0].Kind != KindID {
		t.Fatalf("findings = %+v, want a single id", found)
	}
}
