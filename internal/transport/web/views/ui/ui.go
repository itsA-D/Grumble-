// Package ui holds the shared visual primitives for the public site. Components
// carry structure and behaviour; design tokens live in web/src/app.css so a
// palette change stays in one place.
package ui

import "strconv"

// Form control classes. Templates use these constants instead of repeating the
// token-backed classes at every call site.
const (
	LabelClass    = "field-label"
	HintClass     = "field-hint"
	InputClass    = "field-input"
	SelectClass   = "field-select"
	TextareaClass = "field-textarea"
	CheckClass    = "field-check"
)

// Variants accepted by ButtonLink and SubmitButton.
const (
	Primary = "primary"
	Ghost   = "ghost"
)

// SideListItem is an item in a SideList.
type SideListItem struct {
	Href  string
	Label string
	Count int
}

// SidePain is a pain rendered in the side panel.
type SidePain struct {
	Slug         string
	Body         string
	Category     string
	CategorySlug string
	Urgency      int
	Seeded       bool
}

// BrowseLink is one link in a BrowseGrid group.
type BrowseLink struct {
	Href  string
	Label string
	Count int
}

// BrowseGroup is one titled column in a BrowseGrid. Empty groups and links with
// a zero count are not rendered.
type BrowseGroup struct {
	Title string
	Links []BrowseLink
}

// FormatCount adds thousands separators to a non-negative count.
func FormatCount(n int) string {
	if n < 0 {
		return "0"
	}
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
