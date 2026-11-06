package web

import (
	"log/slog"
	"net"
	"net/http"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps is everything the HTTP layer needs, injected by the composition root.
// Passing plain fields keeps the transport layer independent of the config
// package.
type Deps struct {
	// Logger is the application logger.
	Logger *slog.Logger
	// Pool is the Postgres pool used by readiness checks. May be nil in tests.
	Pool *pgxpool.Pool
	// ProductName is the display name.
	ProductName string
	// SiteURL is the absolute base URL.
	SiteURL string
	// Env is "dev" or "prod"; it controls the Secure cookie flag.
	Env string
	// TrustedProxyHeader is the header carrying the client address.
	TrustedProxyHeader string
	// TrustedProxies are the CIDRs whose header values are trusted.
	TrustedProxies []string

	// Feed, Pains, Voting, Hubs, Map, Trending and Bookmarks are the service
	// surface (see services.go).
	Feed      FeedService
	Pains     PainsService
	Voting    VotingService
	Hubs      HubsService
	Map       MapService
	Trending  TrendingService
	Bookmarks BookmarksService
	// SaveLimiter throttles save/unsave mutations.
	SaveLimiter SaveLimiter
	// Identity signs and verifies the anonymous-id cookie.
	Identity IDSigner
	// NewAnonID mints a fresh raw anonymous id for a first-time visitor.
	NewAnonID func() string
	// TurnstileSiteKey renders the widget; empty disables it (dev).
	TurnstileSiteKey string
}

// Server holds shared handler state.
type Server struct {
	deps    Deps
	trusted []*net.IPNet
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := c.Render(r.Context(), w); err != nil {
		s.deps.Logger.Error("render failed", "error", err, "path", r.URL.Path)
	}
}
