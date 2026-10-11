package views

import (
	"grumble/internal/domain/taxonomy"
	"grumble/internal/transport/web/seo"
)

// Page holds the data every page template needs.
type Page struct {
	// Meta is the head metadata.
	Meta seo.Meta
	// SiteURL is the absolute base URL, used for canonicals and links.
	SiteURL string
	// ProductName is the display name shown in the chrome.
	ProductName string
	// CSRFToken is the per-identity synchronizer token embedded in forms
	// (NFR-SEC-2).
	CSRFToken string
}

// PainCard is a compact rendering of a pain in a list.
type PainCard struct {
	Slug         string
	Body         string
	CategoryName string
	CategorySlug string
	CountryName  string
	CountryCode  string
	VoteCount    int
	Urgency      int
	// Seeded marks labelled demo content so it is never presented as organic.
	Seeded bool
}

// FeedPage is the explore feed (FR-FEED-1).
type FeedPage struct {
	Page
	Tab       string
	TabLabel  string
	Cards     []PainCard
	PageNum   int
	HasPrev   bool
	HasNext   bool
	PrevPage  int
	NextPage  int
	EmptyFeed bool
}

// DetailPage is one pain with its related list and vote control (FR-DET-1).
type DetailPage struct {
	Page
	Pain      PainCard
	Related   []PainCard
	Voted     bool
	PainID    string
	VoteCount int
}

// FAQ is one question and answer rendered on a hub.
type FAQ struct {
	Question string
	Answer   string
}

// HubPage is a category or country landing page (FR-HUB-1).
type HubPage struct {
	Page
	Heading   string
	Intro     string
	FAQ       []FAQ
	Cards     []PainCard
	Count     int
	Indexable bool
}

// SubmitValues preserves the form on a re-render.
type SubmitValues struct {
	Body               string
	CategorySlug       string
	CountryCode        string
	Frequency          string
	Disruption         string
	WouldPay           bool
	MonthlyAmount      string
	Tried              string
	ConfirmedRedaction bool
}

// SubmitPage is the submission form (FR-SUB-1 to FR-SUB-5).
type SubmitPage struct {
	Page
	Categories []taxonomy.Category
	Countries  []taxonomy.Country
	Values     SubmitValues
	Error      string
	PIINotice  string
	PIIKinds   []string
	PIIPreview string
	ShowRedact bool
	// TurnstileSiteKey, when set, renders the widget (FR-MOD-7).
	TurnstileSiteKey string
}

// SubmittedPage confirms a submission that is not publicly visible (held or
// rejected), so the author gets a response without a public link.
type SubmittedPage struct {
	Page
	Heading string
	Message string
}

// VoteButton is the HTMX-swappable vote control (FR-VOTE-1/2).
type VoteButton struct {
	PainID    string
	Voted     bool
	VoteCount int
	// CSRFToken is the per-identity token embedded in the vote form.
	CSRFToken string
}

// ErrorPageData is a non-404 error page (rate limit, forbidden, server error).
type ErrorPageData struct {
	Page
	Status  int
	Heading string
	Message string
}
