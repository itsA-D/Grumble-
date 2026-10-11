package pain

import (
	"github.com/google/uuid"

	"grumble/internal/domain/taxonomy"
)

// New is the fully resolved input for creating a pain (service to store).
// Status is set by the caller after the moderation decision.
type New struct {
	ID             uuid.UUID
	Slug           string
	Body           string
	Category       taxonomy.Category
	Country        taxonomy.Country
	Details        Details
	SourceHash     string
	AnonID         string
	Seeded         bool
	ScoringVersion int
}

// Trending is one candidate for the trending tab: the pain plus the votes it
// received inside the 7-day window used by the hot rank (FR-FEED-2).
type Trending struct {
	Pain    Pain
	Votes7d int
}

// VoteContext is the row the voting flow locks while it changes a vote. It
// carries exactly the signals the urgency score needs (PRD section 9.1).
type VoteContext struct {
	ID            uuid.UUID
	Status        Status
	VoteCount     int
	Frequency     Frequency
	Disruption    Disruption
	WouldPay      bool
	MonthlyAmount int
}
