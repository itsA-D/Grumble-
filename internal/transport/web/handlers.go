package web

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"grumble/internal/domain/pain"
	"grumble/internal/transport/web/seo"
	"grumble/internal/transport/web/views"
)

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok\n")
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if s.deps.Pool == nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.deps.Pool.Ping(ctx); err != nil {
		s.deps.Logger.Warn("readiness check failed", "error", err)
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ready\n")
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	meta := seo.New(s.deps.ProductName, "Page not found", "", r.URL.Path)
	meta.NoIndex = true
	s.render(w, r, http.StatusNotFound, views.NotFound(s.page(r.Context(), meta)))
}

// renderError maps a domain error to a status and renders the error page.
func (s *Server) renderError(w http.ResponseWriter, r *http.Request, err error) {
	status := errorStatus(err)
	meta := seo.New(s.deps.ProductName, http.StatusText(status), "", r.URL.Path)
	meta.NoIndex = true
	s.render(w, r, status, views.ErrorPage(views.ErrorPageData{
		Page:    s.page(r.Context(), meta),
		Status:  status,
		Heading: http.StatusText(status),
		Message: errorMessage(err),
	}))
}

func (s *Server) page(ctx context.Context, meta seo.Meta) views.Page {
	return views.Page{
		Meta:        meta,
		SiteURL:     s.deps.SiteURL,
		ProductName: s.deps.ProductName,
		CSRFToken:   s.deps.Identity.CSRFToken(AnonID(ctx)),
	}
}

// cardFromPain projects a domain pain into the list view model.
func (s *Server) cardFromPain(p pain.Pain) views.PainCard {
	return views.PainCard{
		Slug:         p.Slug,
		Body:         p.Body,
		CategoryName: p.Category.Name,
		CategorySlug: p.Category.Slug,
		CountryName:  p.Country.Name,
		CountryCode:  p.Country.Code,
		VoteCount:    p.VoteCount,
		Urgency:      p.Urgency,
		Seeded:       p.Seeded,
	}
}

func (s *Server) cardsFromPains(pains []pain.Pain) []views.PainCard {
	cards := make([]views.PainCard, 0, len(pains))
	for _, p := range pains {
		cards = append(cards, s.cardFromPain(p))
	}
	return cards
}

// isHTMX reports whether the request came from HTMX (a partial swap).
func isHTMX(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}
