package web

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

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
	s.render(w, r, http.StatusNotFound, views.NotFound(s.page(r, meta)))
}

// renderError maps a domain error to a status and renders the error page.
func (s *Server) renderError(w http.ResponseWriter, r *http.Request, err error) {
	status := errorStatus(err)
	meta := seo.New(s.deps.ProductName, http.StatusText(status), "", r.URL.Path)
	meta.NoIndex = true
	s.render(w, r, status, views.ErrorPage(views.ErrorPageData{
		Page:    s.page(r, meta),
		Status:  status,
		Heading: http.StatusText(status),
		Message: errorMessage(err),
	}))
}

func (s *Server) page(r *http.Request, meta seo.Meta) views.Page {
	return views.Page{
		Meta:        meta,
		SiteURL:     s.deps.SiteURL,
		ProductName: s.deps.ProductName,
		Path:        r.URL.RequestURI(),
		CSRFToken:   s.deps.Identity.CSRFToken(AnonID(r.Context())),
	}
}

// savedSet returns the current identity's saved ids. A failed lookup is
// logged and treated as empty so a page still renders.
func (s *Server) savedSet(ctx context.Context, r *http.Request) map[uuid.UUID]bool {
	if s.deps.Bookmarks == nil {
		return nil
	}
	set, err := s.deps.Bookmarks.IDs(ctx, VoterKey(r.Context()))
	if err != nil {
		s.deps.Logger.Warn("saved ids lookup failed", "error", err)
		return nil
	}
	return set
}

// cardFromPain projects a domain pain into the list view model.
func (s *Server) cardFromPain(p pain.Pain, saved map[uuid.UUID]bool, csrf string) views.PainCard {
	return views.PainCard{
		ID:           p.ID.String(),
		Slug:         p.Slug,
		Body:         p.Body,
		CategoryName: p.Category.Name,
		CategorySlug: p.Category.Slug,
		CountryName:  p.Country.Name,
		CountryCode:  p.Country.Code,
		VoteCount:    p.VoteCount,
		Urgency:      p.Urgency,
		Saved:        saved[p.ID],
		CSRFToken:    csrf,
		Seeded:       p.Seeded,
	}
}

func (s *Server) cardsFromPains(pains []pain.Pain, saved map[uuid.UUID]bool, csrf string) []views.PainCard {
	cards := make([]views.PainCard, 0, len(pains))
	for _, p := range pains {
		cards = append(cards, s.cardFromPain(p, saved, csrf))
	}
	return cards
}

// isHTMX reports whether the request came from HTMX (a partial swap).
func isHTMX(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}
