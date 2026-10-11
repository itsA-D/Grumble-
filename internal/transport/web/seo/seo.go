// Package seo builds page head metadata. Titles and meta descriptions must
// never advertise a count of zero (PRD FR-SEO-3); hub pages stay noindex until
// they hold at least a minimum number of pains.
package seo

import "strings"

// Meta is a page's head metadata.
type Meta struct {
	// Title is the full document title, including the product name.
	Title string
	// Description is the meta description; empty means no tag is rendered.
	Description string
	// Path is the site-relative canonical path, for example "/about".
	Path string
	// NoIndex hides the page from search engines.
	NoIndex bool
}

// New builds metadata for a page, appending the product name to the title.
// An empty title falls back to the product name alone.
func New(productName, title, description, path string) Meta {
	full := strings.TrimSpace(title)
	switch {
	case full == "":
		full = productName
	case productName != "":
		full = full + " · " + productName
	}
	return Meta{
		Title:       full,
		Description: strings.TrimSpace(description),
		Path:        path,
	}
}

// Canonical returns the absolute canonical URL for the page.
func (m Meta) Canonical(siteURL string) string {
	return strings.TrimRight(siteURL, "/") + m.Path
}

// Home returns metadata for the landing page.
func Home(productName string) Meta {
	return New(
		productName,
		"",
		"A public board of crowdsourced frustrations, ranked by urgency. See what wastes people's time, upvote the worst, and watch builders fix what matters most.",
		"/",
	)
}

// Feed returns metadata for the explore feed, varying the title by tab.
func Feed(productName, tab string) Meta {
	title := "Frustrations ranked by urgency"
	switch tab {
	case "latest":
		title = "Latest frustrations"
	case "most_voted":
		title = "Most voted frustrations"
	}
	return New(
		productName,
		title,
		"A public board of crowdsourced frustrations, ranked by urgency. Upvote the ones that waste the most time.",
		"/",
	)
}

// Pain returns metadata for a detail page. The title is a short snippet of the
// pain body; titles and descriptions never contain a count (FR-SEO-3).
func Pain(productName, body, path string) Meta {
	snippet := truncate(body, 70)
	return New(productName, snippet, truncate(body, 155), path)
}

// Hub returns metadata for a category or country landing page. Callers pass
// NoIndex when the hub is below the indexability floor (FR-SEO-3).
func Hub(productName, title, description, path string, indexable bool) Meta {
	m := New(productName, title, description, path)
	m.NoIndex = !indexable
	return m
}

// Submit returns metadata for the submission form (always noindex).
func Submit(productName string) Meta {
	m := New(
		productName,
		"Post a frustration",
		"Describe a frustration in your own words. No account needed.",
		"/submit",
	)
	m.NoIndex = true
	return m
}

// truncate shortens s to at most n runes, trimming to a word boundary and
// appending an ellipsis. It collapses internal whitespace.
func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	cut := string(r[:n])
	if i := strings.LastIndexByte(cut, ' '); i > n/2 {
		cut = cut[:i]
	}
	return strings.TrimSpace(cut) + "…"
}
