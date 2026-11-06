package web

import (
	"net/http"

	"grumble/internal/scoring"
	"grumble/internal/service/trending"
	"grumble/internal/transport/web/seo"
	"grumble/internal/transport/web/views"
	"grumble/internal/transport/web/views/ui"
)

var trendingWindows = []scoring.Window{
	scoring.WindowToday,
	scoring.WindowWeek,
	scoring.WindowMonth,
	scoring.WindowAll,
}

// handleTrending renders the windowed board (FR-FEED-2). Any ?window= request
// is a variant: canonical points at /trending and the response is noindex.
func (s *Server) handleTrending(w http.ResponseWriter, r *http.Request) {
	rawWindow := r.URL.Query().Get("window")
	window := scoring.WindowFrom(rawWindow)
	w.Header().Add("Vary", "HX-Request")

	page, err := s.deps.Trending.Get(r.Context(), window)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	catRows, countryRows, err := s.deps.Trending.Breakdown(r.Context(), scoring.WindowAll)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	vm := s.page(r, seo.Trending(s.deps.ProductName, r.URL.Query().Has("window")))
	saved := s.savedSet(r.Context(), r)
	cards := s.cardsFromPains(page.Pains, saved, vm.CSRFToken)
	for i := range cards {
		cards[i].Rank = i + 1
	}

	data := views.TrendingPage{
		Page:       vm,
		Window:     string(page.Window),
		WindowKeys: windowChips(page.Window),
		Cards:      cards,
		Categories: sideItems(page.Categories),
		Countries:  sideItemsCountries(page.Countries),
		Browse:     browseGroups(catRows, countryRows),
	}

	if isHTMX(r) {
		s.render(w, r, http.StatusOK, views.TrendingMain(data))
		return
	}
	s.render(w, r, http.StatusOK, views.Trending(data))
}

func windowChips(active scoring.Window) []views.WindowChip {
	chips := make([]views.WindowChip, 0, len(trendingWindows))
	for _, window := range trendingWindows {
		chips = append(chips, views.WindowChip{
			Key:    string(window),
			Label:  window.Label(),
			Href:   "/trending?window=" + string(window),
			Active: window == active,
		})
	}
	return chips
}

func sideItems(rows []trending.Category) []ui.SideListItem {
	out := make([]ui.SideListItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, ui.SideListItem{Href: "/c/" + row.Slug, Label: row.Name, Count: row.Count})
	}
	return out
}

func sideItemsCountries(rows []trending.Country) []ui.SideListItem {
	out := make([]ui.SideListItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, ui.SideListItem{Href: "/country/" + row.Code, Label: row.Name, Count: row.Count})
	}
	return out
}

func browseGroups(cats []trending.Category, countries []trending.Country) []ui.BrowseGroup {
	catLinks := make([]ui.BrowseLink, 0, len(cats))
	for _, row := range cats {
		catLinks = append(catLinks, ui.BrowseLink{Href: "/c/" + row.Slug, Label: row.Name, Count: row.Count})
	}
	countryLinks := make([]ui.BrowseLink, 0, len(countries))
	for _, row := range countries {
		countryLinks = append(countryLinks, ui.BrowseLink{Href: "/country/" + row.Code, Label: row.Name, Count: row.Count})
	}
	return []ui.BrowseGroup{
		{Title: "Categories", Links: catLinks},
		{Title: "Countries", Links: countryLinks},
	}
}
