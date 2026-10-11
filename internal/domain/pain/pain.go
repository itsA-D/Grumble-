// Package pain defines the submission entity, its enums and the pure rules
// that validate it (PRD FR-SUB-1 to FR-SUB-5, FR-MOD-1). No I/O.
package pain

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"grumble/internal/domain/taxonomy"
)

// Body length limits (PRD FR-SUB-1) and the "what was tried" limit (FR-SUB-5).
const (
	MinBodyLen  = 20
	MaxBodyLen  = 2000
	MaxTriedLen = 500
)

// Status is the moderation state of a pain (ARCHITECTURE.md section 5).
type Status string

const (
	// StatusPending awaits automated checks; not publicly visible.
	StatusPending Status = "pending"
	// StatusPublished is publicly visible.
	StatusPublished Status = "published"
	// StatusHeld awaits human review; not publicly visible.
	StatusHeld Status = "held"
	// StatusRejected failed automated checks; not publicly visible.
	StatusRejected Status = "rejected"
	// StatusRemoved was taken down; not publicly visible.
	StatusRemoved Status = "removed"
)

// Valid reports whether s is a defined status.
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusPublished, StatusHeld, StatusRejected, StatusRemoved:
		return true
	default:
		return false
	}
}

// Public reports whether a pain in this status is visible to the public.
func (s Status) Public() bool { return s == StatusPublished }

// Frequency is how often the submitter hits the frustration (FR-SUB-5).
type Frequency string

// Frequency values (FR-SUB-5).
const (
	FrequencyDaily     Frequency = "daily"
	FrequencyWeekly    Frequency = "weekly"
	FrequencySometimes Frequency = "sometimes"
)

// Valid reports whether f is a defined frequency.
func (f Frequency) Valid() bool {
	switch f {
	case FrequencyDaily, FrequencyWeekly, FrequencySometimes:
		return true
	default:
		return false
	}
}

// Disruption is how much the frustration interferes with life (FR-SUB-5).
type Disruption string

// Disruption values (FR-SUB-5).
const (
	DisruptionMild      Disruption = "mild"
	DisruptionAnnoying  Disruption = "annoying"
	DisruptionRuinsWeek Disruption = "ruins_my_week"
)

// Valid reports whether d is a defined disruption level.
func (d Disruption) Valid() bool {
	switch d {
	case DisruptionMild, DisruptionAnnoying, DisruptionRuinsWeek:
		return true
	default:
		return false
	}
}

// Details is the optional urgency signal a submitter may add (FR-SUB-5).
type Details struct {
	// Frequency, when set, must be valid.
	Frequency Frequency
	// Disruption, when set, must be valid.
	Disruption Disruption
	// WouldPay is the submitter's willingness-to-pay flag.
	WouldPay bool
	// MonthlyAmount is USD per month; only meaningful when WouldPay is true.
	// Zero means "unspecified".
	MonthlyAmount int
	// Tried is free text (max MaxTriedLen) describing what the submitter tried.
	Tried string
}

// Pain is one submission.
type Pain struct {
	ID       uuid.UUID
	Slug     string
	Body     string
	Category taxonomy.Category
	Country  taxonomy.Country
	Details  Details
	Status   Status

	VoteCount int
	Urgency   int

	// Seeded marks content loaded by `app seed`; never shown as organic.
	Seeded bool

	CreatedAt   time.Time
	PublishedAt *time.Time
}

// ValidationError describes why a pain is not acceptable. It is a value type so
// callers can map it to a form error.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks the submitter-controlled fields of p. It does not check
// moderation content rules (those live in the moderation service).
func Validate(p Pain) error {
	if n := len([]rune(p.Body)); n < MinBodyLen {
		return ValidationError{Field: "body", Message: fmt.Sprintf("must be at least %d characters", MinBodyLen)}
	} else if n > MaxBodyLen {
		return ValidationError{Field: "body", Message: fmt.Sprintf("must be at most %d characters", MaxBodyLen)}
	}
	if !p.Category.Valid() {
		return ValidationError{Field: "category", Message: "choose a category"}
	}
	if !p.Country.Valid() {
		return ValidationError{Field: "country", Message: "choose a country"}
	}
	if p.Details.Frequency != "" && !p.Details.Frequency.Valid() {
		return ValidationError{Field: "frequency", Message: "is not a recognised value"}
	}
	if p.Details.Disruption != "" && !p.Details.Disruption.Valid() {
		return ValidationError{Field: "disruption", Message: "is not a recognised value"}
	}
	if n := len([]rune(p.Details.Tried)); n > MaxTriedLen {
		return ValidationError{Field: "tried", Message: fmt.Sprintf("must be at most %d characters", MaxTriedLen)}
	}
	if p.Details.MonthlyAmount < 0 {
		return ValidationError{Field: "monthly_amount", Message: "must not be negative"}
	}
	return nil
}
