package views

import (
	"fmt"
	"net/url"
	"strings"
)

// boolStr renders a boolean as the "true"/"false" string ARIA attributes want.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// countLabel renders a neutral, singular/plural-safe count label. It never
// renders a bare zero in a heading; callers use empty-state copy instead.
func countLabel(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}

// voteGlyph is the arrow shown on the vote toggle.
func voteGlyph(voted bool) string {
	if voted {
		return "▼"
	}
	return "▲"
}

// voteLabel is the accessible text on the vote toggle.
func voteLabel(v VoteButton) string {
	if v.Voted {
		return countLabel(v.VoteCount, "vote", "votes") + " · voted"
	}
	return countLabel(v.VoteCount, "vote", "votes")
}

// feedLink builds a feed URL preserving the tab and the active filters. An empty
// cursor starts at the newest page of the ordering.
func feedLink(tab, category, search, cursor string) string {
	v := url.Values{}
	v.Set("tab", tab)
	if category != "" {
		v.Set("category", category)
	}
	if search != "" {
		v.Set("q", search)
	}
	if cursor != "" {
		v.Set("cursor", cursor)
	}
	return "/?" + v.Encode()
}

// joinKinds renders the PII kind labels as a readable list.
func joinKinds(kinds []string) string {
	return strings.Join(kinds, ", ")
}

// errorStatusLabel is the short tag shown on an error page.
func errorStatusLabel(status int) string {
	return fmt.Sprintf("%d", status)
}
