package views

import (
	"net/url"
	"strings"
)

// NavItem is one primary navigation link. The slice built by NavItems is the
// only source for both the desktop row and the mobile menu, so adding a route
// changes one place.
type NavItem struct {
	Label  string
	Href   string
	Active bool
}

// NavItems returns the primary navigation for a request URI. The home feed's
// tabs stay at "/", so Explore is active for every home-feed variant.
func NavItems(requestURI string) []NavItem {
	u, err := url.Parse(requestURI)
	if err != nil {
		u = &url.URL{Path: "/"}
	}

	return []NavItem{
		{Label: "Explore", Href: "/", Active: u.Path == "/"},
		{Label: "Trending", Href: "/trending", Active: u.Path == "/trending"},
		{Label: "Map", Href: "/map", Active: u.Path == "/map" || strings.HasPrefix(u.Path, "/map/")},
		{Label: "Saved", Href: "/saved", Active: u.Path == "/saved"},
	}
}
