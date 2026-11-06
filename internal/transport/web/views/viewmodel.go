package views

import (
	"grumble/internal/domain/taxonomy"
	"grumble/internal/service/mapsvc"
	"grumble/internal/transport/web/seo"
	"grumble/internal/transport/web/views/ui"
)

// Page holds the data every page template needs.
type Page struct {
	// Meta is the head metadata.
	Meta seo.Meta
	// SiteURL is the absolute base URL, used for canonicals and links.
	SiteURL string
	// ProductName is the display name shown in the chrome.
	ProductName string
	// Path is the request URI (path plus query) used to mark the active nav
	// item, for example "/?tab=latest".
	Path string
	// CSRFToken is the per-identity synchronizer token embedded in forms
	// (NFR-SEC-2).
	CSRFToken string
}

// PainCard is a compact rendering of a pain in a list.
type PainCard struct {
	ID           string
	Slug         string
	Body         string
	CategoryName string
	CategorySlug string
	CountryName  string
	CountryCode  string
	VoteCount    int
	Urgency      int
	// Rank is the 1-based trending position; zero hides the rank column.
	Rank int
	// Saved is the server-rendered state of the save toggle.
	Saved bool
	// CSRFToken is embedded in the save form.
	CSRFToken string
	// FromSaved makes a removal replace the row on the saved page.
	FromSaved bool
	// Seeded marks labelled demo content so it is never presented as organic.
	Seeded bool
}

// FeedPage is the explore feed (FR-FEED-1).
type FeedPage struct {
	Page
	Tab      string
	TabLabel string
	// Categories are the filter chips; ActiveCategory is the selected slug ("" is
	// "All").
	Categories     []taxonomy.Category
	ActiveCategory string
	// Search is the current body search term ("" when unset).
	Search     string
	Cards      []PainCard
	NextCursor string
	HasNext    bool
	HasCursor  bool
	EmptyFeed  bool
	// Totals behind the stat strip. They are published-only counts from the
	// feed service, not a sum of the visible page.
	PainTotal    int
	CountryTotal int
	VoteTotal    int
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

// WindowChip is one trending window link.
type WindowChip struct {
	Key    string
	Label  string
	Href   string
	Active bool
}

// TrendingPage is the windowed trending board.
type TrendingPage struct {
	Page
	Window     string
	WindowKeys []WindowChip
	Cards      []PainCard
	Categories []ui.SideListItem
	Countries  []ui.SideListItem
	Browse     []ui.BrowseGroup
}

// SavedPage is the anonymous saved list. It is always noindex.
type SavedPage struct {
	Page
	Cards      []PainCard
	Count      int
	Limit      int
	Browse     []ui.BrowseGroup
	LimitError string
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

// MapPageData is the view model for the world pain map (FR-MAP-1).
type MapPageData struct {
	Page
	Subtitle       string
	Countries      []mapsvc.CountryStat
	CountryPaths   []mapsvc.CountryPathData
	MaxPain        int
	ActiveCategory string
	Categories     []taxonomy.Category
	Region         taxonomy.Region
	Regions        []taxonomy.Region
	SidePanel      mapsvc.SidePanelData
	PainTotal      int
	CountryTotal   int
	VoteTotal      int
}
