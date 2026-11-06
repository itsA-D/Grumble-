package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"grumble/internal/domain"
	"grumble/internal/domain/taxonomy"
	"grumble/internal/service/mapsvc"
	"grumble/internal/transport/web/seo"
	"grumble/internal/transport/web/views"
)

// handleMap renders the world pain map (FR-MAP-1). The category filter is a
// query parameter; invalid slugs fall back to all categories so the page always
// renders.
func (s *Server) handleMap(w http.ResponseWriter, r *http.Request) {
	category := resolveFeedCategory(r.URL.Query().Get("category"))
	w.Header().Add("Vary", "HX-Request")

	page, err := s.deps.Map.Get(r.Context(), mapsvc.Query{Category: category})
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	subtitle := "No signal yet. Be the first to post."
	if page.PainTotal > 0 {
		subtitle = fmt.Sprintf("%d frustrations across %d countries. Brighter means more pain.",
			page.PainTotal, page.CountryTotal)
	}

	data := views.MapPageData{
		Page:           s.page(r, seo.Map(s.deps.ProductName, category != "")),
		Subtitle:       subtitle,
		Countries:      page.Countries,
		CountryPaths:   page.CountryPaths,
		MaxPain:        page.MaxPain,
		ActiveCategory: category,
		Categories:     taxonomy.Categories(),
		Region:         taxonomy.RegionWorld,
		Regions:        taxonomy.AllRegions(),
		SidePanel:      page.SidePanel,
		PainTotal:      page.PainTotal,
		CountryTotal:   page.CountryTotal,
		VoteTotal:      page.VoteTotal,
	}

	// A category chip swaps only the map section; the shell (and its nav) is
	// never re-sent for an HTMX request.
	if isHTMX(r) {
		s.render(w, r, http.StatusOK, views.MapSection(data))
		return
	}
	s.render(w, r, http.StatusOK, views.MapPage(data))
}

// handleMapCountry serves the side panel for a selected country. HTMX requests
// get only the panel so the map keeps its pan/zoom state; a plain request (for
// example a shared link) renders the full map focused on that country.
func (s *Server) handleMapCountry(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	category := resolveFeedCategory(r.URL.Query().Get("category"))
	w.Header().Add("Vary", "HX-Request")

	if _, ok := taxonomy.CountryByCode(code); !ok {
		s.renderError(w, r, domain.ErrNotFound)
		return
	}

	page, err := s.deps.Map.Get(r.Context(), mapsvc.Query{Category: category, Country: code})
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	if isHTMX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if err := views.MapSidePanel(page.SidePanel).Render(r.Context(), w); err != nil {
			s.deps.Logger.Error("render map side panel failed", "error", err, "country", code)
		}
		return
	}

	subtitle := "No signal yet. Be the first to post."
	if page.PainTotal > 0 {
		subtitle = fmt.Sprintf("%d frustrations across %d countries. Brighter means more pain.",
			page.PainTotal, page.CountryTotal)
	}
	s.render(w, r, http.StatusOK, views.MapPage(views.MapPageData{
		Page:           s.page(r, seo.Map(s.deps.ProductName, category != "")),
		Subtitle:       subtitle,
		Countries:      page.Countries,
		CountryPaths:   page.CountryPaths,
		MaxPain:        page.MaxPain,
		ActiveCategory: category,
		Categories:     taxonomy.Categories(),
		Region:         taxonomy.RegionWorld,
		Regions:        taxonomy.AllRegions(),
		SidePanel:      page.SidePanel,
		PainTotal:      page.PainTotal,
		CountryTotal:   page.CountryTotal,
		VoteTotal:      page.VoteTotal,
	}))
}

// handleMapSideClear returns the default side panel (no country selected).
func (s *Server) handleMapSideClear(w http.ResponseWriter, r *http.Request) {
	category := resolveFeedCategory(r.URL.Query().Get("category"))
	w.Header().Add("Vary", "HX-Request")

	page, err := s.deps.Map.Get(r.Context(), mapsvc.Query{Category: category})
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	if !isHTMX(r) {
		http.Redirect(w, r, "/map", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := views.MapSidePanel(page.SidePanel).Render(r.Context(), w); err != nil {
		s.deps.Logger.Error("render map side panel failed", "error", err)
	}
}
