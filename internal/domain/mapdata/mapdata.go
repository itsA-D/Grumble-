// Package mapdata holds the storage-agnostic rows behind the world pain map
// (PRD FR-MAP-1). Both the map service and the Postgres adapter depend on these
// plain types, so the service never imports the store and the store never
// imports the service (ARCHITECTURE.md dependency direction).
package mapdata

// StatRow is one country's precomputed totals for a category ("" = all).
type StatRow struct {
	Country    string
	PainCount  int
	VoteCount  int
	AvgUrgency int
}

// CountryRow is one entry in a top-countries list.
type CountryRow struct {
	Country   string
	PainCount int
}

// PainRow is one published pain shown in the side panel for a country.
type PainRow struct {
	Slug      string
	Body      string
	Category  string
	Country   string
	VoteCount int
	Urgency   int
	Seeded    bool
}
