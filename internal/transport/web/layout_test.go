package web

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"grumble/internal/transport/web/views"
)

// TestEveryFullPageRendersOneShell is the regression guard for the shared
// layout: every registered GET route that serves HTML must include exactly one
// main navigation and one main landmark. Adding a route without a sample path
// fails the test, so a new page cannot ship without the shell.
func TestEveryFullPageRendersOneShell(t *testing.T) {
	env := newTestEnvFull(t, "dev", fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{}, fakeMap{page: sampleMapPage()})
	mux, ok := env.handler.(*chi.Mux)
	if !ok {
		t.Fatalf("handler is %T, want *chi.Mux", env.handler)
	}

	var routes []string
	if err := chi.Walk(mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method == http.MethodGet {
			routes = append(routes, route)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	// Representative requests for dynamic routes. Non-HTML endpoints and
	// fragment-only endpoints are skipped explicitly.
	samples := map[string]string{
		"/":                   "/",
		"/trending":           "/trending",
		"/saved":              "/saved",
		"/map":                "/map",
		"/map/country/{code}": "/map/country/GB",
		"/submit":             "/submit",
		"/p/{slug}":           "/p/password-resets-take-twenty-minutes-and-land-in",
		"/c/{slug}":           "/c/tech",
		"/country/{code}":     "/country/US",
	}
	skipped := map[string]bool{
		"/healthz":        true,
		"/readyz":         true,
		"/static/*":       true,
		"/sitemap.xml":    true,
		"/robots.txt":     true,
		"/map/side/clear": true,
	}

	checked := 0
	for _, route := range routes {
		if skipped[route] {
			continue
		}
		path, ok := samples[route]
		if !ok {
			t.Fatalf("route %s has no representative path; add one to samples", route)
		}
		checked++
		w := env.get(t, path)
		assertLayoutShell(t, route, w.Body.String())
	}
	if checked != len(samples) {
		t.Fatalf("checked %d full pages, want %d", checked, len(samples))
	}

	// The 404 page and the shared 500 page render through the same shell.
	w := env.get(t, "/definitely-not-a-page")
	if w.Code != http.StatusNotFound {
		t.Fatalf("404 status = %d, want 404", w.Code)
	}
	assertLayoutShell(t, "404", w.Body.String())

	var b strings.Builder
	err := views.ErrorPage(views.ErrorPageData{
		Status:  http.StatusInternalServerError,
		Heading: "Server error",
		Message: "Something went wrong.",
	}).Render(context.Background(), &b)
	if err != nil {
		t.Fatalf("render error page: %v", err)
	}
	assertLayoutShell(t, "500", b.String())
}

func assertLayoutShell(t *testing.T, name, body string) {
	t.Helper()
	if n := strings.Count(body, `<nav aria-label="Main"`); n != 1 {
		t.Fatalf("%s: %d main navigations, want 1", name, n)
	}
	if n := strings.Count(body, "<main"); n != 1 {
		t.Fatalf("%s: %d main landmarks, want 1", name, n)
	}
	if !strings.Contains(body, `<main id="content"`) {
		t.Fatalf("%s: main landmark has no id=content", name)
	}
	if n := strings.Count(body, `href="#content"`); n != 1 {
		t.Fatalf("%s: %d skip links, want 1", name, n)
	}
}

// TestMapCategoryHTMXReturnsSectionFragment keeps the chip swap from replacing
// the document shell, and marks the response as variant-dependent.
func TestMapCategoryHTMXReturnsSectionFragment(t *testing.T) {
	env := newTestEnvFull(t, "dev", fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{}, fakeMap{page: sampleMapPage()})
	w := env.do(t, http.MethodGet, "/map?category=tech", "", map[string]string{"HX-Request": "true"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Vary"); !strings.Contains(got, "HX-Request") {
		t.Fatalf("Vary = %q, want HX-Request", got)
	}
	body := w.Body.String()
	if strings.Contains(strings.ToLower(body), "<!doctype") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatal("HTMX response must not include the document shell")
	}
	if !strings.Contains(body, `id="map-section"`) {
		t.Fatal("HTMX response must contain the map section")
	}
	if strings.Contains(body, `<nav`) {
		t.Fatal("HTMX fragment must not contain navigation")
	}
}

// TestMapFullPageVariesOnHXRequest ensures caches keep the shell and the
// fragment apart for the same URL.
func TestMapFullPageVariesOnHXRequest(t *testing.T) {
	env := newTestEnvFull(t, "dev", fakePains{}, fakeFeed{}, fakeVoting{}, fakeHubs{}, fakeMap{page: sampleMapPage()})
	w := env.get(t, "/map")
	if sh := w.Header().Get("Vary"); !strings.Contains(sh, "HX-Request") {
		t.Fatalf("Vary = %q, want HX-Request", sh)
	}
	assertLayoutShell(t, "/map", w.Body.String())
}
