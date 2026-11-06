package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"grumble/internal/service/hubs"
	"grumble/internal/transport/web/seo"
	"grumble/internal/transport/web/views"
)

// hubPageSize is how many pains a hub page lists.
const hubPageSize = 20

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	detail, err := s.deps.Pains.Get(r.Context(), slug)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	voted := false
	if key := VoterKey(r.Context()); key != "" {
		if v, err := s.deps.Voting.Voted(r.Context(), detail.Pain.ID, key); err == nil {
			voted = v
		} else {
			s.deps.Logger.Warn("voted lookup failed", "error", err, "pain_id", detail.Pain.ID)
		}
	}

	vm := s.page(r, seo.Pain(s.deps.ProductName, detail.Pain.Body, "/p/"+detail.Pain.Slug))
	saved := s.savedSet(r.Context(), r)
	s.render(w, r, http.StatusOK, views.Detail(views.DetailPage{
		Page:      vm,
		Pain:      s.cardFromPain(detail.Pain, saved, vm.CSRFToken),
		Related:   s.cardsFromPains(detail.Related, saved, vm.CSRFToken),
		Voted:     voted,
		PainID:    detail.Pain.ID.String(),
		VoteCount: detail.Pain.VoteCount,
	}))
}

func (s *Server) handleCategoryHub(w http.ResponseWriter, r *http.Request) {
	s.renderHub(w, r, hubs.Category, chi.URLParam(r, "slug"), "/c/")
}

func (s *Server) handleCountryHub(w http.ResponseWriter, r *http.Request) {
	s.renderHub(w, r, hubs.Country, chi.URLParam(r, "code"), "/country/")
}

func (s *Server) renderHub(w http.ResponseWriter, r *http.Request, kind hubs.Kind, slug, prefix string) {
	hub, err := s.deps.Hubs.Get(r.Context(), hubs.Query{
		Kind:  kind,
		Slug:  slug,
		Limit: hubPageSize,
	})
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	faq := make([]views.FAQ, 0, len(hub.FAQ))
	for _, q := range hub.FAQ {
		faq = append(faq, views.FAQ{Question: q.Question, Answer: q.Answer})
	}

	vm := s.page(r, seo.Hub(s.deps.ProductName, hub.Title, hub.Description, prefix+slug, hub.Indexable))
	s.render(w, r, http.StatusOK, views.Hub(views.HubPage{
		Page:      vm,
		Heading:   hub.Title,
		Intro:     hub.Intro,
		FAQ:       faq,
		Cards:     s.cardsFromPains(hub.Pains, s.savedSet(r.Context(), r), vm.CSRFToken),
		Count:     hub.Count,
		Indexable: hub.Indexable,
	}))
}
