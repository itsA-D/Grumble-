package pain

import (
	"errors"
	"testing"

	"grumble/internal/domain/taxonomy"
	"grumble/internal/scoring"
)

func validPain() Pain {
	return Pain{
		Body:     "My landlord never fixes the heating in winter and ignores every message.",
		Category: taxonomy.Category{Slug: "housing", Name: "Housing"},
		Country:  taxonomy.Country{Code: "GB", Name: "United Kingdom"},
	}
}

func TestValidateOK(t *testing.T) {
	if err := Validate(validPain()); err != nil {
		t.Fatalf("valid pain rejected: %v", err)
	}
}

func TestValidateFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Pain)
		field  string
	}{
		{"too short", func(p *Pain) { p.Body = "too short" }, "body"},
		{"no category", func(p *Pain) { p.Category = taxonomy.Category{} }, "category"},
		{"bad category", func(p *Pain) { p.Category = taxonomy.Category{Slug: "nope"} }, "category"},
		{"no country", func(p *Pain) { p.Country = taxonomy.Country{} }, "country"},
		{"bad frequency", func(p *Pain) { p.Details.Frequency = "hourly" }, "frequency"},
		{"negative amount", func(p *Pain) { p.Details.MonthlyAmount = -1 }, "monthly_amount"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validPain()
			tc.mutate(&p)
			err := Validate(p)
			if err == nil {
				t.Fatal("expected error")
			}
			var ve ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected ValidationError, got %T", err)
			}
			if ve.Field != tc.field {
				t.Errorf("field = %q, want %q", ve.Field, tc.field)
			}
		})
	}
}

func TestStatusPublic(t *testing.T) {
	if !StatusPublished.Public() {
		t.Error("published should be public")
	}
	for _, s := range []Status{StatusPending, StatusHeld, StatusRejected, StatusRemoved} {
		if s.Public() {
			t.Errorf("%s must not be public", s)
		}
	}
}

// TestEnumAlignment guards against the domain enums and the scoring constants
// drifting apart (scoring receives the enum values as strings).
func TestEnumAlignment(t *testing.T) {
	if string(FrequencyDaily) != scoring.FrequencyDaily ||
		string(FrequencyWeekly) != scoring.FrequencyWeekly ||
		string(FrequencySometimes) != scoring.FrequencySometimes {
		t.Error("frequency enums do not match scoring constants")
	}
	if string(DisruptionMild) != scoring.DisruptionMild ||
		string(DisruptionAnnoying) != scoring.DisruptionAnnoying ||
		string(DisruptionRuinsWeek) != scoring.DisruptionRuinsWeek {
		t.Error("disruption enums do not match scoring constants")
	}
}
