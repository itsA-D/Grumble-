package web

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"grumble/internal/domain"
	"grumble/internal/domain/pain"
	"grumble/internal/domain/pii"
	"grumble/internal/domain/taxonomy"
	"grumble/internal/service/feed"
	"grumble/internal/service/hubs"
	"grumble/internal/service/pains"
	"grumble/internal/service/voting"
)

// --- fakes ------------------------------------------------------------------

type fakeFeed struct {
	pains []pain.Pain
	err   error
}

func (f fakeFeed) List(context.Context, feed.Query) ([]pain.Pain, error) {
	return f.pains, f.err
}

// feedRecorder captures the query the feed handler builds.
type feedRecorder struct {
	record func(feed.Query)
}

func (f feedRecorder) List(_ context.Context, q feed.Query) ([]pain.Pain, error) {
	if f.record != nil {
		f.record(q)
	}
	return nil, nil
}

type fakePains struct {
	submit  func(pains.SubmitRequest) (pains.SubmitResult, error)
	get     func(string) (pains.Detail, error)
	sitemap []pain.Pain
}

func (f fakePains) Submit(_ context.Context, req pains.SubmitRequest) (pains.SubmitResult, error) {
	if f.submit != nil {
		return f.submit(req)
	}
	return pains.SubmitResult{}, nil
}

func (f fakePains) Get(_ context.Context, slug string) (pains.Detail, error) {
	if f.get != nil {
		return f.get(slug)
	}
	return pains.Detail{}, domain.ErrNotFound
}

func (f fakePains) RecentForSitemap(context.Context, int) ([]pain.Pain, error) {
	return f.sitemap, nil
}

type fakeVoting struct {
	set    func(voting.Request) (voting.Result, error)
	voteEr error
}

func (f fakeVoting) Set(_ context.Context, req voting.Request) (voting.Result, error) {
	if f.set != nil {
		return f.set(req)
	}
	return voting.Result{Voted: req.Voted, VoteCount: 1}, nil
}

func (f fakeVoting) Voted(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, f.voteEr
}

type fakeHubs struct {
	hub hubs.Hub
	err error
}

func (f fakeHubs) Get(context.Context, hubs.Query) (hubs.Hub, error) {
	return f.hub, f.err
}

type fakeIdentity struct{}

func (fakeIdentity) SignAnonID(id string) string { return "t." + id }
func (fakeIdentity) VerifyAnonID(token string) (string, bool) {
	if strings.HasPrefix(token, "t.") {
		return strings.TrimPrefix(token, "t."), true
	}
	return "", false
}
func (fakeIdentity) CSRFToken(anonID string) string { return testCSRF }
func (fakeIdentity) VerifyCSRF(anonID, token string) bool {
	return anonID != "" && token == testCSRF
}

// testCSRF is the token fakeIdentity issues for any non-empty anon id, matching
// the deterministic NewAnonID used by the test router.
const testCSRF = "csrf.test-anon"

// --- helpers ---------------------------------------------------------------

type testEnv struct {
	handler http.Handler
	logs    *bytes.Buffer
}

func newTestEnv(t *testing.T, painsSvc PainsService, feedSvc FeedService, voteSvc VotingService, hubSvc HubsService) *testEnv {
	t.Helper()
	return newTestEnvEnv(t, "dev", painsSvc, feedSvc, voteSvc, hubSvc)
}

func newTestEnvEnv(t *testing.T, env string, painsSvc PainsService, feedSvc FeedService, voteSvc VotingService, hubSvc HubsService) *testEnv {
	t.Helper()
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	h := NewRouter(Deps{
		Logger:             logger,
		ProductName:        "Grumble",
		SiteURL:            "https://example.test",
		Env:                env,
		TrustedProxyHeader: "X-Forwarded-For",
		Feed:               feedSvc,
		Pains:              painsSvc,
		Voting:             voteSvc,
		Hubs:               hubSvc,
		Identity:           fakeIdentity{},
		NewAnonID:          func() string { return "test-anon" },
	})
	return &testEnv{handler: h, logs: buf}
}

func (e *testEnv) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, http.MethodGet, path, "", nil)
}

func (e *testEnv) do(t *testing.T, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(context.Background(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	e.handler.ServeHTTP(w, r)
	return w
}

func samplePain(body string) pain.Pain {
	cat, _ := taxonomy.CategoryBySlug("tech")
	country, _ := taxonomy.CountryByCode("GB")
	return pain.Pain{
		ID:        uuid.New(),
		Slug:      "sample-pain",
		Body:      body,
		Category:  cat,
		Country:   country,
		Status:    pain.StatusPublished,
		VoteCount: 3,
		Urgency:   64,
	}
}

// --- feed & escaping --------------------------------------------------------

func TestFeedRendersAndEscapesUserText(t *testing.T) {
	body := `<script>alert(1)</script> the buses are always late here`
	env := newTestEnv(t, fakePains{}, fakeFeed{pains: []pain.Pain{samplePain(body)}}, fakeVoting{}, fakeHubs{})

	w := env.get(t, "/")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	got := w.Body.String()
	if strings.Contains(got, "<script>alert(1)</script>") {
		t.Fatalf("user text rendered unescaped: %s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatalf("escaped user text not found in output")
	}
}

func TestFeedEmptyState(t *testing.T) {
	env := newTestEnv(t, fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{})
	w := env.get(t, "/")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Nothing has been posted") {
		t.Fatalf("empty state copy missing")
	}
}

// --- detail -----------------------------------------------------------------

func TestDetailNotFound(t *testing.T) {
	env := newTestEnv(t, fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{})
	w := env.get(t, "/p/does-not-exist")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDetailRendersVoteButton(t *testing.T) {
	p := samplePain("a perfectly ordinary frustration that is long enough")
	env := newTestEnv(t, fakePains{get: func(string) (pains.Detail, error) {
		return pains.Detail{Pain: p}, nil
	}}, fakeFeed{}, fakeVoting{}, fakeHubs{})

	w := env.get(t, "/p/"+p.Slug)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `id="vote-`+p.ID.String()+`"`) {
		t.Fatalf("vote control missing from detail page")
	}
}

// --- submit -----------------------------------------------------------------

func postSubmit(env *testEnv, t *testing.T) *httptest.ResponseRecorder {
	form := "csrf_token=" + testCSRF +
		"&body=the+buses+are+always+late+and+nobody+explains+why&category=tech&country=GB"
	return env.do(t, http.MethodPost, "/submit", form, nil)
}

func TestSubmitPublishedRedirectsToDetail(t *testing.T) {
	cat, _ := taxonomy.CategoryBySlug("tech")
	country, _ := taxonomy.CountryByCode("GB")
	env := newTestEnv(t, fakePains{submit: func(pains.SubmitRequest) (pains.SubmitResult, error) {
		return pains.SubmitResult{
			Pain:   pain.Pain{Slug: "late-buses", Category: cat, Country: country},
			Status: pain.StatusPublished,
		}, nil
	}}, fakeFeed{}, fakeVoting{}, fakeHubs{})

	w := postSubmit(env, t)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/p/late-buses" {
		t.Fatalf("Location = %q, want /p/late-buses", loc)
	}
}

func TestSubmitHeldShowsConfirmationWithoutPublicLink(t *testing.T) {
	env := newTestEnv(t, fakePains{submit: func(pains.SubmitRequest) (pains.SubmitResult, error) {
		return pains.SubmitResult{Pain: pain.Pain{Slug: "held-slug"}, Status: pain.StatusHeld}, nil
	}}, fakeFeed{}, fakeVoting{}, fakeHubs{})

	w := postSubmit(env, t)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	got := w.Body.String()
	if !strings.Contains(got, "being reviewed") {
		t.Fatalf("held confirmation copy missing")
	}
	if strings.Contains(got, "/p/held-slug") {
		t.Fatalf("held pain leaked a public detail link")
	}
}

func TestSubmitRateLimited(t *testing.T) {
	env := newTestEnv(t, fakePains{submit: func(pains.SubmitRequest) (pains.SubmitResult, error) {
		return pains.SubmitResult{}, domain.ErrRateLimited
	}}, fakeFeed{}, fakeVoting{}, fakeHubs{})

	w := postSubmit(env, t)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", w.Code)
	}
	if !strings.Contains(w.Body.String(), "too often") {
		t.Fatalf("rate-limit message missing")
	}
}

func TestSubmitTurnstileForbidden(t *testing.T) {
	env := newTestEnv(t, fakePains{submit: func(pains.SubmitRequest) (pains.SubmitResult, error) {
		return pains.SubmitResult{}, domain.ErrForbidden
	}}, fakeFeed{}, fakeVoting{}, fakeHubs{})

	w := postSubmit(env, t)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestSubmitPIIReRendersWithRedactionPreview(t *testing.T) {
	env := newTestEnv(t, fakePains{submit: func(pains.SubmitRequest) (pains.SubmitResult, error) {
		return pains.SubmitResult{}, &pains.PersonalDataError{
			Kinds:   []pii.Kind{pii.KindEmail},
			Preview: "email me at [email] about this",
		}
	}}, fakeFeed{}, fakeVoting{}, fakeHubs{})

	w := postSubmit(env, t)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", w.Code)
	}
	got := w.Body.String()
	if !strings.Contains(got, "email me at [email] about this") {
		t.Fatalf("redaction preview missing")
	}
	if !strings.Contains(got, `name="confirm_redaction"`) {
		t.Fatalf("confirm-redaction control missing")
	}
}

func TestSubmitFormHasLabelledInputs(t *testing.T) {
	env := newTestEnv(t, fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{})
	w := env.get(t, "/submit")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	got := w.Body.String()
	for _, field := range []string{"body", "category", "country", "frequency", "disruption", "would_pay", "monthly_amount", "tried"} {
		if !strings.Contains(got, `for="`+field+`"`) || !strings.Contains(got, `id="`+field+`"`) {
			t.Fatalf("field %q is missing its label or input id", field)
		}
	}
}

// --- vote -------------------------------------------------------------------

func TestVoteHTMXReturnsPartialAndIsIdempotent(t *testing.T) {
	id := uuid.New()
	env := newTestEnv(t, fakePains{}, fakeFeed{}, fakeVoting{
		set: func(req voting.Request) (voting.Result, error) {
			return voting.Result{Voted: true, VoteCount: 1}, nil
		},
	}, fakeHubs{})

	headers := map[string]string{"HX-Request": "true"}
	form := "csrf_token=" + testCSRF + "&voted=1"
	first := env.do(t, http.MethodPost, "/p/"+id.String()+"/vote", form, headers)
	second := env.do(t, http.MethodPost, "/p/"+id.String()+"/vote", form, headers)

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("status = %d/%d, want 200/200", first.Code, second.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("vote response changed on repeat (not idempotent):\n%s\n%s", first.Body.String(), second.Body.String())
	}
	if !strings.Contains(first.Body.String(), `aria-pressed="true"`) {
		t.Fatalf("vote partial did not render voted state")
	}
}

func TestVoteNonHTMXRedirects(t *testing.T) {
	id := uuid.New()
	env := newTestEnv(t, fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{})
	w := env.do(t, http.MethodPost, "/p/"+id.String()+"/vote", "csrf_token="+testCSRF+"&voted=1", nil)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
}

// --- CSRF -------------------------------------------------------------------

func TestPOSTWithoutCSRFTokenIsForbidden(t *testing.T) {
	env := newTestEnv(t, fakePains{submit: func(pains.SubmitRequest) (pains.SubmitResult, error) {
		t.Fatal("service must not be reached without a CSRF token")
		return pains.SubmitResult{}, nil
	}}, fakeFeed{}, fakeVoting{}, fakeHubs{})

	body := "body=the+buses+are+always+late+and+nobody+explains+why&category=tech&country=GB"
	w := env.do(t, http.MethodPost, "/submit", body, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestPOSTWithWrongCSRFTokenIsForbidden(t *testing.T) {
	env := newTestEnv(t, fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{})
	body := "csrf_token=forged&body=the+buses+are+always+late+and+nobody+explains+why&category=tech&country=GB"
	w := env.do(t, http.MethodPost, "/submit", body, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestPOSTFromForeignOriginIsForbidden(t *testing.T) {
	env := newTestEnv(t, fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{})
	body := "csrf_token=" + testCSRF + "&body=the+buses+are+always+late+and+nobody+explains+why&category=tech&country=GB"
	w := env.do(t, http.MethodPost, "/submit", body, map[string]string{"Origin": "https://evil.test"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestPOSTWithSameOriginTokenIsAccepted(t *testing.T) {
	env := newTestEnv(t, fakePains{submit: func(pains.SubmitRequest) (pains.SubmitResult, error) {
		return pains.SubmitResult{Pain: pain.Pain{Slug: "ok"}, Status: pain.StatusPublished}, nil
	}}, fakeFeed{}, fakeVoting{}, fakeHubs{})
	body := "csrf_token=" + testCSRF + "&body=the+buses+are+always+late+and+nobody+explains+why&category=tech&country=GB"
	w := env.do(t, http.MethodPost, "/submit", body, map[string]string{"Origin": "https://example.com"})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
}

// --- X-Forwarded-For spoofing -----------------------------------------------

func TestSpoofedXFFFromUntrustedPeerDoesNotChangeClientAddr(t *testing.T) {
	var seen []string
	env := newTestEnv(t, fakePains{submit: func(req pains.SubmitRequest) (pains.SubmitResult, error) {
		seen = append(seen, req.ClientIP)
		return pains.SubmitResult{}, domain.ErrRateLimited
	}}, fakeFeed{}, fakeVoting{}, fakeHubs{})

	body := "csrf_token=" + testCSRF + "&body=the+buses+are+always+late+and+nobody+explains+why&category=tech&country=GB"
	for _, spoof := range []string{"1.2.3.4", "5.6.7.8", "9.10.11.12"} {
		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/submit", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("X-Forwarded-For", spoof)
		r.RemoteAddr = "203.0.113.7:40000"
		env.handler.ServeHTTP(httptest.NewRecorder(), r)
	}

	if len(seen) != 3 {
		t.Fatalf("service called %d times, want 3", len(seen))
	}
	for _, ip := range seen {
		if ip != "203.0.113.7" {
			t.Fatalf("client addr = %q, want the peer 203.0.113.7 (spoofed XFF must be ignored)", ip)
		}
	}
}

// --- hubs & SEO -------------------------------------------------------------

func TestHubNoIndexBelowFloor(t *testing.T) {
	cat, _ := taxonomy.CategoryBySlug("tech")
	env := newTestEnv(t, fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{hub: hubs.Hub{
		Kind:        hubs.Category,
		Slug:        cat.Slug,
		Title:       "Frustrations about " + cat.Name,
		Description: "Browse frustrations about " + cat.Name + ", ranked by how urgent the community finds them.",
		Intro:       "intro",
		Count:       2,
		Indexable:   false,
	}})

	w := env.get(t, "/c/"+cat.Slug)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	got := w.Body.String()
	if !strings.Contains(got, `name="robots" content="noindex"`) {
		t.Fatalf("hub below floor should be noindex")
	}
}

func TestSitemapListsPublishedPainsOnly(t *testing.T) {
	pub := samplePain("a published frustration that is long enough to pass")
	env := newTestEnv(t, fakePains{sitemap: []pain.Pain{pub}}, fakeFeed{}, fakeVoting{}, fakeHubs{hub: hubs.Hub{Indexable: true, Slug: "tech"}})

	w := env.get(t, "/sitemap.xml")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	got := w.Body.String()
	if !strings.Contains(got, "https://example.test/p/"+pub.Slug) {
		t.Fatalf("published pain missing from sitemap")
	}
	if strings.Contains(got, "held") || strings.Contains(got, "rejected") {
		t.Fatalf("non-published content leaked into sitemap")
	}
}

func TestRobotsPointsAtSitemap(t *testing.T) {
	env := newTestEnv(t, fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{})
	w := env.get(t, "/robots.txt")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "https://example.test/sitemap.xml") {
		t.Fatalf("robots.txt missing sitemap URL")
	}
}

// --- privacy ----------------------------------------------------------------

func TestRequestLogsNeverContainRawIP(t *testing.T) {
	const rawIP = "203.0.113.9"
	env := newTestEnv(t, fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{})

	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	r.RemoteAddr = rawIP + ":44321"
	w := httptest.NewRecorder()
	env.handler.ServeHTTP(w, r)

	if strings.Contains(env.logs.String(), rawIP) {
		t.Fatalf("request log leaked raw IP: %s", env.logs.String())
	}
	if strings.Contains(env.logs.String(), "client") {
		t.Fatalf("request log still contains a client field: %s", env.logs.String())
	}
}

func TestAnonCookieFlags(t *testing.T) {
	tests := []struct {
		env        string
		wantSecure bool
	}{
		{env: "dev", wantSecure: false},
		{env: "prod", wantSecure: true},
	}
	for _, tc := range tests {
		t.Run(tc.env, func(t *testing.T) {
			env := newTestEnvEnv(t, tc.env, fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{})
			w := env.get(t, "/")
			var cookie *http.Cookie
			for _, c := range w.Result().Cookies() {
				if c.Name == anonCookieName {
					cookie = c
				}
			}
			if cookie == nil {
				t.Fatalf("anonymous-id cookie was not set")
			}
			if !cookie.HttpOnly {
				t.Fatalf("cookie must be HttpOnly")
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Fatalf("cookie SameSite = %v, want Lax", cookie.SameSite)
			}
			if cookie.Secure != tc.wantSecure {
				t.Fatalf("cookie Secure = %v, want %v for env %q", cookie.Secure, tc.wantSecure, tc.env)
			}
			if cookie.Path != "/" {
				t.Fatalf("cookie Path = %q, want /", cookie.Path)
			}
		})
	}
}

func TestFeedDefaultsToTrending(t *testing.T) {
	var got feed.Tab
	rec := feedRecorder{record: func(q feed.Query) { got = q.Tab }}
	env := newTestEnv(t, fakePains{}, rec, fakeVoting{}, fakeHubs{})
	w := env.get(t, "/")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got != feed.Trending {
		t.Fatalf("default feed tab = %v, want Trending", got)
	}
}

// --- error mapping ----------------------------------------------------------

func TestServerErrorRenders500(t *testing.T) {
	env := newTestEnv(t, fakePains{get: func(string) (pains.Detail, error) {
		return pains.Detail{}, errors.New("boom")
	}}, fakeFeed{}, fakeVoting{}, fakeHubs{})
	w := env.get(t, "/p/anything")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
