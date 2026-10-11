package web

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/OWNER/painboard/internal/domain/pain"
	"github.com/OWNER/painboard/internal/domain/taxonomy"
	"github.com/OWNER/painboard/internal/service/pains"
	"github.com/OWNER/painboard/internal/transport/web/seo"
	"github.com/OWNER/painboard/internal/transport/web/views"
)

func (s *Server) handleSubmitForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, views.Submit(s.submitView(r.Context(), views.SubmitValues{}, "", nil, nil, false)))
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, r, err)
		return
	}

	values := valuesFromForm(r)
	amount, _ := strconv.Atoi(strings.TrimSpace(r.PostFormValue("monthly_amount")))

	res, err := s.deps.Pains.Submit(r.Context(), pains.SubmitRequest{
		Body:               values.Body,
		CategorySlug:       values.CategorySlug,
		CountryCode:        values.CountryCode,
		Frequency:          values.Frequency,
		Disruption:         values.Disruption,
		WouldPay:           values.WouldPay,
		MonthlyAmount:      amount,
		Tried:              values.Tried,
		ClientIP:           ClientAddr(r.Context()),
		AnonID:             AnonID(r.Context()),
		TurnstileToken:     r.PostFormValue("cf-turnstile-response"),
		ConfirmedRedaction: values.ConfirmedRedaction,
	})
	if err != nil {
		s.renderSubmitError(w, r, values, err)
		return
	}

	if res.Status == pain.StatusPublished {
		http.Redirect(w, r, "/p/"+res.Pain.Slug, http.StatusSeeOther)
		return
	}

	heading, message := submittedCopy(res.Status)
	s.render(w, r, http.StatusOK, views.Submitted(views.SubmittedPage{
		Page:    s.page(r.Context(), seo.Submit(s.deps.ProductName)),
		Heading: heading,
		Message: message,
	}))
}

// renderSubmitError re-renders the form with the right status and copy. PII
// findings get the redaction preview; other validation errors keep the input.
func (s *Server) renderSubmitError(w http.ResponseWriter, r *http.Request, values views.SubmitValues, err error) {
	var pde *pains.PersonalDataError
	if errors.As(err, &pde) {
		kinds := make([]string, 0, len(pde.Kinds))
		for _, k := range pde.Kinds {
			kinds = append(kinds, k.Placeholder())
		}
		s.render(w, r, http.StatusUnprocessableEntity, views.Submit(
			s.submitView(r.Context(), values, "", kinds, []string{pde.Preview}, true)))
		return
	}

	status := errorStatus(err)
	if status == http.StatusUnprocessableEntity ||
		status == http.StatusTooManyRequests ||
		status == http.StatusForbidden {
		s.render(w, r, status, views.Submit(
			s.submitView(r.Context(), values, errorMessage(err), nil, nil, false)))
		return
	}
	s.renderError(w, r, err)
}

func (s *Server) submitView(ctx context.Context, values views.SubmitValues, errMsg string, kinds, previews []string, showRedact bool) views.SubmitPage {
	notice := ""
	if showRedact {
		notice = "We noticed something that looks like personal information. Remove it, or post the redacted version below."
	}
	preview := ""
	if len(previews) > 0 {
		preview = previews[0]
	}
	return views.SubmitPage{
		Page:             s.page(ctx, seo.Submit(s.deps.ProductName)),
		Categories:       taxonomy.Categories(),
		Countries:        taxonomy.Countries(),
		Values:           values,
		Error:            errMsg,
		PIINotice:        notice,
		PIIKinds:         kinds,
		PIIPreview:       preview,
		ShowRedact:       showRedact,
		TurnstileSiteKey: s.deps.TurnstileSiteKey,
	}
}

// valuesFromForm reads the submitted form back into the view model, preserving
// it on a re-render. The redaction preview text is only shown, never stored.
func valuesFromForm(r *http.Request) views.SubmitValues {
	return views.SubmitValues{
		Body:               r.PostFormValue("body"),
		CategorySlug:       r.PostFormValue("category"),
		CountryCode:        r.PostFormValue("country"),
		Frequency:          r.PostFormValue("frequency"),
		Disruption:         r.PostFormValue("disruption"),
		WouldPay:           r.PostFormValue("would_pay") != "",
		MonthlyAmount:      r.PostFormValue("monthly_amount"),
		Tried:              r.PostFormValue("tried"),
		ConfirmedRedaction: r.PostFormValue("confirm_redaction") != "",
	}
}

// submittedCopy explains why a non-published submission has no public link.
func submittedCopy(status pain.Status) (heading, message string) {
	switch status {
	case pain.StatusHeld:
		return "Thanks — your post is being reviewed",
			"We have received your post and a person will look at it before it goes live. It is not public yet."
	default:
		return "Thanks — your post was not published",
			"We could not publish this post. You are welcome to try again with different wording."
	}
}
