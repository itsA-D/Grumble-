package web

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/OWNER/painboard/internal/domain"
	"github.com/OWNER/painboard/internal/service/voting"
	"github.com/OWNER/painboard/internal/transport/web/views"
)

func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.renderError(w, r, domain.ErrNotFound)
		return
	}
	key := VoterKey(r.Context())
	if key == "" {
		s.renderError(w, r, domain.ErrForbidden)
		return
	}

	// The hidden field carries the desired end state, so a double-click sends
	// "add" twice and the idempotent Set applies it once (FR-VOTE-1).
	want := r.PostFormValue("voted") == "1"
	res, err := s.deps.Voting.Set(r.Context(), voting.Request{
		PainID:   id,
		VoterKey: key,
		Voted:    want,
	})
	if err != nil {
		s.renderVoteError(w, r, err)
		return
	}

	if isHTMX(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.VoteButtonView(views.VoteButton{
			PainID:    id.String(),
			Voted:     res.Voted,
			VoteCount: res.VoteCount,
			CSRFToken: s.deps.Identity.CSRFToken(AnonID(r.Context())),
		}).Render(r.Context(), w); err != nil {
			s.deps.Logger.Error("render vote failed", "error", err)
		}
		return
	}
	http.Redirect(w, r, redirectTarget(r), http.StatusSeeOther)
}

// renderVoteError answers an HTMX vote failure with the right status and a tiny
// text body; non-HTMX callers get the standard error page.
func (s *Server) renderVoteError(w http.ResponseWriter, r *http.Request, err error) {
	if isHTMX(r) {
		http.Error(w, errorMessage(err), errorStatus(err))
		return
	}
	s.renderError(w, r, err)
}

// redirectTarget sends a non-HTMX vote back where it came from, or home when
// there is no usable same-origin referer. Only the path is used, so a spoofed
// absolute referer cannot turn this into an open redirect.
func redirectTarget(r *http.Request) string {
	ref := r.Referer()
	if ref == "" {
		return "/"
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host != "" && u.Host != r.Host {
		return "/"
	}
	if u.Path == "" {
		return "/"
	}
	if u.RawQuery != "" {
		return u.Path + "?" + u.RawQuery
	}
	return u.Path
}
