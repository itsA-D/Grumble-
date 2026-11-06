package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"grumble/internal/domain"
	"grumble/internal/domain/pain"
	"grumble/internal/scoring"
	"grumble/internal/service/bookmarks"
	"grumble/internal/transport/web/seo"
	"grumble/internal/transport/web/views"
)

const savedLimitMessage = "Saved list is full. Remove one first."

// handleSaved renders the private list. It is deliberately noindex both in the
// document head and via X-Robots-Tag.
func (s *Server) handleSaved(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Robots-Tag", "noindex")
	if s.deps.Bookmarks == nil {
		s.render(w, r, http.StatusOK, views.Saved(views.SavedPage{
			Page: s.page(r, seo.Saved(s.deps.ProductName)),
		}))
		return
	}

	data, err := s.savedPageData(r, VoterKey(r.Context()))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, views.Saved(data))
}

// savedPageData loads the saved cards and the all-time browse grid. count comes
// from the service result so an OOB update after delete is authoritative.
func (s *Server) savedPageData(r *http.Request, ownerKey string) (views.SavedPage, error) {
	pains, err := s.deps.Bookmarks.List(r.Context(), ownerKey, 0)
	if err != nil {
		return views.SavedPage{}, err
	}
	return s.savedPageFrom(r, pains, len(pains), s.deps.Bookmarks.Limit())
}

func (s *Server) savedPageFrom(r *http.Request, pains []pain.Pain, count, limit int) (views.SavedPage, error) {
	catRows, countryRows, err := s.deps.Trending.Breakdown(r.Context(), scoring.WindowAll)
	if err != nil {
		return views.SavedPage{}, err
	}
	vm := s.page(r, seo.Saved(s.deps.ProductName))
	cards := s.cardsFromPains(pains, s.savedSet(r.Context(), r), vm.CSRFToken)
	for i := range cards {
		cards[i].FromSaved = true
	}
	return views.SavedPage{
		Page:   vm,
		Cards:  cards,
		Count:  count,
		Limit:  limit,
		Browse: browseGroups(catRows, countryRows),
	}, nil
}

// handleSave adds a pain to the saved list. HTMX receives just the toggle; a
// plain POST redirects back to the page named by return_to.
func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	s.mutateSaved(w, r, true)
}

// handleUnsave removes a pain. From the saved page the row itself is the target,
// so the response only carries OOB count/list updates.
func (s *Server) handleUnsave(w http.ResponseWriter, r *http.Request) {
	s.mutateSaved(w, r, false)
}

func (s *Server) mutateSaved(w http.ResponseWriter, r *http.Request, save bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.savedActionError(w, r, domain.ErrNotFound)
		return
	}
	ownerKey := VoterKey(r.Context())
	if ownerKey == "" || s.deps.Bookmarks == nil {
		s.savedActionError(w, r, domain.ErrNotFound)
		return
	}

	if limiter := s.deps.SaveLimiter; limiter != nil {
		ok, err := limiter.Allow(r.Context(), "save:"+ownerKey)
		if err != nil {
			s.savedActionError(w, r, err)
			return
		}
		if !ok {
			if isHTMX(r) {
				savedState := !save
				if formValue(r, "from") == "saved" {
					w.Header().Set("HX-Retarget", "#save-"+id.String())
					w.Header().Set("HX-Reswap", "outerHTML")
					savedState = true
				}
				s.renderSavedToggle(w, r, id, savedState, formValue(r, "return_to"), "Too many changes. Wait a moment.", http.StatusOK)
				return
			}
			s.renderError(w, r, domain.ErrRateLimited)
			return
		}
	}

	var result bookmarks.Result
	if save {
		result, err = s.deps.Bookmarks.Save(r.Context(), ownerKey, id)
	} else {
		result, err = s.deps.Bookmarks.Remove(r.Context(), ownerKey, id)
	}
	if err != nil {
		if errors.Is(err, bookmarks.ErrLimit) {
			if isHTMX(r) {
				s.renderSavedToggle(w, r, id, false, formValue(r, "return_to"), savedLimitMessage, http.StatusOK)
				return
			}
			s.renderSavedLimit(w, r, result.Limit)
			return
		}
		s.savedActionError(w, r, err)
		return
	}

	fromSaved := formValue(r, "from") == "saved"
	if isHTMX(r) {
		if !save && fromSaved {
			s.renderSavedRemoved(w, r, result)
			return
		}
		s.renderSavedToggle(w, r, id, result.Saved, formValue(r, "return_to"), "", http.StatusOK)
		return
	}

	fallback := "/"
	if !save {
		fallback = "/saved"
	}
	http.Redirect(w, r, returnPath(formValue(r, "return_to"), fallback), http.StatusSeeOther)
}

func (s *Server) renderSavedToggle(w http.ResponseWriter, r *http.Request, id uuid.UUID, saved bool, returnTo, message string, status int) {
	token := s.deps.Identity.CSRFToken(AnonID(r.Context()))
	s.render(w, r, status, views.SaveButtonView(views.SaveButton{
		PainID:    id.String(),
		Saved:     saved,
		CSRFToken: token,
		ReturnTo:  returnPath(returnTo, "/"),
		Error:     message,
	}))
}

func (s *Server) renderSavedRemoved(w http.ResponseWriter, r *http.Request, result bookmarks.Result) {
	data, err := s.savedPageDataForCount(r, result.Count, result.Limit)
	if err != nil {
		s.deps.Logger.Warn("saved page refresh failed", "error", err)
		data = views.SavedPage{Page: s.page(r, seo.Saved(s.deps.ProductName)), Count: result.Count, Limit: result.Limit}
	}
	s.render(w, r, http.StatusOK, views.SavedRemoved(data))
}

// savedPageDataForCount reloads the list after a delete so the zero case can
// render the full empty state with its browse grid.
func (s *Server) savedPageDataForCount(r *http.Request, count, limit int) (views.SavedPage, error) {
	pains, err := s.deps.Bookmarks.List(r.Context(), VoterKey(r.Context()), 0)
	if err != nil {
		return views.SavedPage{}, err
	}
	return s.savedPageFrom(r, pains, count, limit)
}

func (s *Server) savedActionError(w http.ResponseWriter, r *http.Request, err error) {
	status := errorStatus(err)
	if isHTMX(r) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, "Saved list changed. Reload the page.\n")
		return
	}
	s.renderError(w, r, err)
}

func (s *Server) renderSavedLimit(w http.ResponseWriter, r *http.Request, limit int) {
	meta := seo.Saved(s.deps.ProductName)
	s.render(w, r, http.StatusUnprocessableEntity, views.ErrorPage(views.ErrorPageData{
		Page:    s.page(r, meta),
		Status:  http.StatusUnprocessableEntity,
		Heading: "Saved list full",
		Message: fmt.Sprintf("Your saved list holds %d frustrations. Remove one before saving another.", limit),
	}))
}

// returnPath accepts only a local path, so return_to can never become an open
// redirect.
func returnPath(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.Contains(raw, "\\") {
		return fallback
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return fallback
	}
	return u.RequestURI()
}
