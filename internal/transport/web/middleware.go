package web

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"grumble/internal/domain"
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
const anonCookieName = "grumble_aid"

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
			token = formValue(r, csrfFormName)
		}
		if !s.deps.Identity.VerifyCSRF(AnonID(r.Context()), token) {
			s.renderError(w, r, domain.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// formValue reads a form field for any unsafe method. HTMX appends form values
// to the query string for DELETE, while Go's PostFormValue only reads POST-style
// bodies; FormValue covers the query and the body. The DELETE body fallback
// keeps a hand-written form working too. The body is buffered and restored so
// the handler can read it again.
func formValue(r *http.Request, name string) string {
	if v := r.FormValue(name); v != "" {
		return v
	}
	if r.Method != http.MethodDelete || r.Body == nil {
		return ""
	}
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		return ""
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return ""
	}
	return values.Get(name)
}

// sameHost reports whether an Origin/Referer URL shares this request's host.
func sameHost(raw, host string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Host == host
}

// contentSecurityPolicy builds the CSP for this deployment. Style and script
// sources are locked to this origin with no 'unsafe-inline': HTMX is configured
// via a meta tag to inject no <style>, so the widget needs no inline exemption
// (fix 4). Cloudflare Turnstile is allowed only when a site key is configured
// (FR-MOD-7).
func (s *Server) contentSecurityPolicy() string {
	script := "script-src 'self'"
	frame := "frame-src 'self'"
	if s.deps.TurnstileSiteKey != "" {
		script += " https://challenges.cloudflare.com"
		frame += " https://challenges.cloudflare.com"
	}
	return "default-src 'self'; base-uri 'self'; form-action 'self'; " +
		"img-src 'self' data:; style-src 'self'; " + script + "; " + frame + "; " +
		"connect-src 'self'; frame-ancestors 'none'"
}

// securityHeaders sets conservative response headers. All scripts are served
// from this origin, so no inline script or nonce is required.
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	csp := s.contentSecurityPolicy()
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
