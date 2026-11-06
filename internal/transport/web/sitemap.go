package web

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	"grumble/internal/domain/taxonomy"
	"grumble/internal/service/hubs"
)

// sitemapPains bounds how many recent pains the sitemap lists. Phase 2 can page
// this through a sitemap index when the board grows.
const sitemapPains = 500

type urlset struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []urlEntry
}

type urlEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func (s *Server) handleSitemap(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(s.deps.SiteURL, "/")
	entries := []urlEntry{
		{Loc: base + "/"},
		{Loc: base + "/trending"},
		{Loc: base + "/map"},
	}

	// Category hubs, but only once they pass the indexability floor
	// (FR-SEO-3).
	for _, cat := range taxonomy.Categories() {
		hub, err := s.deps.Hubs.Get(r.Context(), hubs.Query{Kind: hubs.Category, Slug: cat.Slug, Limit: 1})
		if err != nil {
			s.deps.Logger.Warn("sitemap hub failed", "error", err, "category", cat.Slug)
			continue
		}
		if hub.Indexable {
			entries = append(entries, urlEntry{Loc: base + "/c/" + cat.Slug})
		}
	}

	pains, err := s.deps.Pains.RecentForSitemap(r.Context(), sitemapPains)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	for _, p := range pains {
		e := urlEntry{Loc: base + "/p/" + p.Slug}
		if p.PublishedAt != nil {
			e.LastMod = p.PublishedAt.UTC().Format(time.RFC3339)
		}
		entries = append(entries, e)
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=600")
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(urlset{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: entries}); err != nil {
		s.deps.Logger.Error("sitemap encode failed", "error", err)
	}
}

func (s *Server) handleRobots(w http.ResponseWriter, _ *http.Request) {
	base := strings.TrimRight(s.deps.SiteURL, "/")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("User-agent: *\nAllow: /\n\nSitemap: " + base + "/sitemap.xml\n"))
}
