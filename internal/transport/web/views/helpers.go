package views

import (
	"fmt"
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

// tabClass styles the active feed tab differently from the others.
func tabClass(active bool) string {
	if active {
		return "rounded-md bg-brand px-3 py-1.5 text-sm font-semibold text-white"
	}
	return "rounded-md border border-edge px-3 py-1.5 text-sm text-muted transition hover:text-ink"
}

// feedURL builds a feed page link preserving the tab.
func feedURL(tab string, page int) string {
	return fmt.Sprintf("/?tab=%s&page=%d", tab, page)
}

// joinKinds renders the PII kind labels as a readable list.
func joinKinds(kinds []string) string {
	return strings.Join(kinds, ", ")
}

// errorStatusLabel is the short tag shown on an error page.
func errorStatusLabel(status int) string {
	return fmt.Sprintf("%d", status)
}
