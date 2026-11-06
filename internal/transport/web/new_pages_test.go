package web

import (
	"net/http"
	"strings"
	"testing"

	"grumble/internal/domain/pain"
	"grumble/internal/service/bookmarks"
	"grumble/internal/service/trending"
)

func trendingEnv(t *testing.T, page trending.Page, bm BookmarksService) *testEnv {
	t.Helper()
	return newTestEnvCore(t, Deps{
		Trending:  fakeTrending{page: page},
		Bookmarks: bm,
	})
}

func TestTrendingFullPageAndHTMXFragment(t *testing.T) {
	env := trendingEnv(t, trending.Page{Pains: []pain.Pain{samplePain("a trend that is long enough to render")}}, fakeBookmarks{})

	full := env.get(t, "/trending")
	if full.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", full.Code)
	}
	body := full.Body.String()
	assertLayoutShell(t, "/trending", body)
	if !strings.Contains(body, "Trending frustrations") || !strings.Contains(body, `id="trending-main"`) {
		t.Fatal("trending page content missing")
	}
	if !strings.Contains(body, "Rank 1") {
		t.Fatal("trending rank missing")
	}
	if got := full.Header().Get("Vary"); !strings.Contains(got, "HX-Request") {
		t.Fatalf("Vary = %q, want HX-Request", got)
	}

	partial := env.do(t, http.MethodGet, "/trending", "", map[string]string{"HX-Request": "true"})
	if partial.Code != http.StatusOK {
		t.Fatalf("htmx status = %d", partial.Code)
	}
	fragment := partial.Body.String()
	if strings.Contains(strings.ToLower(fragment), "<!doctype") || strings.Contains(fragment, `<nav aria-label="Main"`) {
		t.Fatal("HTMX trending response must not include the shell")
	}
	if !strings.Contains(fragment, `id="trending-main"`) {
		t.Fatal("HTMX trending response must be the swap target")
	}
}

func TestTrendingWindowVariantIsNoindex(t *testing.T) {
	env := trendingEnv(t, trending.Page{}, fakeBookmarks{})
	w := env.get(t, "/trending?window=all")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "noindex") {
		t.Fatal("window variant must be noindex")
	}
	if !strings.Contains(body, `href="https://example.test/trending"`) {
		t.Fatal("window variant canonical must point at /trending")
	}
	if !strings.Contains(body, "Nothing saved yet") && !strings.Contains(body, "No signal in this window") {
		t.Fatal("trending empty state missing")
	}
}

func TestSavedPageIsNoindex(t *testing.T) {
	env := trendingEnv(t, trending.Page{}, fakeBookmarks{pains: []pain.Pain{samplePain("a saved pain that is long enough to render")}})
	w := env.get(t, "/saved")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Fatalf("X-Robots-Tag = %q, want noindex", got)
	}
	body := w.Body.String()
	assertLayoutShell(t, "/saved", body)
	if !strings.Contains(body, "1 SAVED / 5") {
		t.Fatal("saved count missing")
	}
	if !strings.Contains(body, "REMOVE") {
		t.Fatal("saved rows need a remove control")
	}
	if !strings.Contains(body, "noindex") {
		t.Fatal("saved page meta must be noindex")
	}
}

func TestSaveCapRendersInlineMessage(t *testing.T) {
	env := trendingEnv(t, trending.Page{}, fakeBookmarks{err: bookmarks.ErrLimit})
	w := env.do(t, http.MethodPost, "/saved/6f1d1f2e-6f1d-4f1d-8f1d-6f1d1f2e6f1d",
		"csrf_token="+testCSRF, map[string]string{"HX-Request": "true"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 so HTMX swaps the message", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Saved list is full") {
		t.Fatal("cap message missing")
	}
}

func TestUnsaveFromSavedUpdatesOOBListWhenEmpty(t *testing.T) {
	env := trendingEnv(t, trending.Page{}, fakeBookmarks{result: bookmarks.Result{Saved: false, Count: 0, Limit: 5}})
	w := env.do(t, http.MethodDelete, "/saved/6f1d1f2e-6f1d-4f1d-8f1d-6f1d1f2e6f1d",
		"csrf_token="+testCSRF+"&from=saved", map[string]string{"HX-Request": "true"})
	if w.Code == http.StatusForbidden {
		t.Fatal("DELETE form token must pass CSRF")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `hx-swap-oob="true"`) || !strings.Contains(body, "0 SAVED / 5") {
		t.Fatalf("OOB count missing: %s", body)
	}
	if !strings.Contains(body, "Nothing saved yet") {
		t.Fatal("empty saved state must replace the list after the last removal")
	}
}

func TestUnsaveRedirectsToLocalReturnPath(t *testing.T) {
	env := trendingEnv(t, trending.Page{}, fakeBookmarks{result: bookmarks.Result{Saved: false, Count: 1, Limit: 5}})
	w := env.do(t, http.MethodDelete, "/saved/6f1d1f2e-6f1d-4f1d-8f1d-6f1d1f2e6f1d",
		"csrf_token="+testCSRF+"&return_to=%2Fp%2Fsome-pain", nil)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/p/some-pain" {
		t.Fatalf("Location = %q", got)
	}
}

func TestReturnPathRejectsOpenRedirects(t *testing.T) {
	cases := map[string]string{
		"":                        "/saved",
		"https://evil.test":       "/saved",
		"//evil.test":             "/saved",
		`\evil`:                   "/saved",
		"/p/ok?x=1":               "/p/ok?x=1",
		"javascript:alert(1)":     "/saved",
		"https://example.com/foo": "/saved",
	}
	for in, want := range cases {
		if got := returnPath(in, "/saved"); got != want {
			t.Fatalf("returnPath(%q) = %q, want %q", in, got, want)
		}
	}
}
