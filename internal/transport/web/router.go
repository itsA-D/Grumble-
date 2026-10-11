package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the HTTP handler: shared middleware, static assets, health
// endpoints and the public pages. The anonymous-id middleware runs for every
// page so votes have a stable identity (NFR-PRIV-1).
func NewRouter(deps Deps) http.Handler {
	s := &Server{deps: deps, trusted: parseCIDRs(deps.TrustedProxies)}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(s.realIP)
	r.Use(s.anonID)
	r.Use(s.csrfProtection)
	r.Use(s.securityHeaders)
	r.Use(s.logRequests)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.handleHealth)
	r.Get("/readyz", s.handleReady)

	r.Handle("/static/*", staticHandler())

	r.Get("/", s.handleFeed)
	r.Get("/submit", s.handleSubmitForm)
	r.Post("/submit", s.handleSubmit)
	r.Get("/p/{slug}", s.handleDetail)
	r.Post("/p/{id}/vote", s.handleVote)
	r.Get("/c/{slug}", s.handleCategoryHub)
	r.Get("/country/{code}", s.handleCountryHub)
	r.Get("/sitemap.xml", s.handleSitemap)
	r.Get("/robots.txt", s.handleRobots)
	r.NotFound(s.handleNotFound)

	return r
}
