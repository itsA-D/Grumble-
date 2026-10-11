package web

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/OWNER/painboard/internal/domain"
)

type clientAddrKey struct{}

// ClientAddr returns the resolved client address stored by the realIP
// middleware. The raw address is never persisted; the identity service hashes
// it into a salted source hash (PRD NFR-PRIV-1). It is used only in-request.
func ClientAddr(ctx context.Context) string {
	if v, ok := ctx.Value(clientAddrKey{}).(string); ok {
		return v
	}
	return ""
}

type anonIDKey struct{}

// AnonID returns the raw anonymous id for the current request.
func AnonID(ctx context.Context) string {
	if v, ok := ctx.Value(anonIDKey{}).(string); ok {
		return v
	}
	return ""
}

// VoterKey is the storage key for this visitor's votes: a signed-in user would
// use "u:<id>" (M2); anonymous visitors use "a:<anon_id>" (ARCHITECTURE.md).
func VoterKey(ctx context.Context) string {
	if id := AnonID(ctx); id != "" {
		return "a:" + id
	}
	return ""
}

// anonCookieName is the httpOnly cookie carrying the signed anonymous id.
const anonCookieName = "pb_aid"

// anonCookieMaxAge matches identity.AnonIDMaxAge (90 days).
const anonCookieMaxAge = 90 * 24 * time.Hour

// realIP resolves the client address, honouring the proxy header only when the
// direct peer is in the trusted CIDRs.
func (s *Server) realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addr := resolveClientAddr(r, s.deps.TrustedProxyHeader, s.trusted)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientAddrKey{}, addr)))
	})
}

// anonID reads, verifies or mints the anonymous id and stores it on the
// request. A tampered or missing token is replaced silently.
func (s *Server) anonID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := ""
		if c, err := r.Cookie(anonCookieName); err == nil {
			if raw, ok := s.deps.Identity.VerifyAnonID(c.Value); ok {
				id = raw
			}
		}
		if id == "" {
			id = s.deps.NewAnonID()
			http.SetCookie(w, &http.Cookie{
				Name:     anonCookieName,
				Value:    s.deps.Identity.SignAnonID(id),
				Path:     "/",
				MaxAge:   int(anonCookieMaxAge / time.Second),
				HttpOnly: true,
				Secure:   s.deps.Env == "prod",
				SameSite: http.SameSiteLaxMode,
			})
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), anonIDKey{}, id)))
	})
}

// csrfHeader is the header HTMX and fetch clients may send the token in; the
// form field name is the fallback for plain HTML forms.
const (
	csrfHeader   = "X-CSRF-Token"
	csrfFormName = "csrf_token"
)

// csrfProtection rejects unsafe requests (POST/PUT/PATCH/DELETE) that lack a
// valid per-identity token, and cross-site requests whose Origin is not this
// host (NFR-SEC-2). Safe methods pass through untouched. The token is bound to
// the anonymous id, which only a same-site page can present, so a foreign page
// cannot forge or replay it.
func (s *Server) csrfProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !sameHost(origin, r.Host) {
			s.renderError(w, r, domain.ErrForbidden)
			return
		}
		token := r.Header.Get(csrfHeader)
		if token == "" {
			token = r.PostFormValue(csrfFormName)
		}
		if !s.deps.Identity.VerifyCSRF(AnonID(r.Context()), token) {
			s.renderError(w, r, domain.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// sameHost reports whether an Origin/Referer URL shares this request's host.
func sameHost(raw, host string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Host == host
}

// securityHeaders sets conservative response headers. All scripts are served
// from this origin, so no inline script or nonce is required.
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	const csp = "default-src 'self'; base-uri 'self'; form-action 'self'; " +
		"img-src 'self' data:; style-src 'self'; script-src 'self'; " +
		"connect-src 'self'; frame-ancestors 'none'"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// logRequests logs one structured line per request. It deliberately logs no
// client address: raw IPs are never stored or logged (NFR-PRIV-1).
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		s.deps.Logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

func resolveClientAddr(r *http.Request, header string, trusted []*net.IPNet) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer != nil && header != "" && isTrusted(peer, trusted) {
		if v := r.Header.Get(header); v != "" {
			if cand := net.ParseIP(strings.TrimSpace(strings.Split(v, ",")[0])); cand != nil {
				return cand.String()
			}
		}
	}
	if peer != nil {
		return peer.String()
	}
	return host
}

func isTrusted(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func parseCIDRs(list []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(list))
	for _, c := range list {
		if _, n, err := net.ParseCIDR(strings.TrimSpace(c)); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}
