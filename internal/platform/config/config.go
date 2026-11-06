// Package config loads and validates all runtime settings from the
// environment. Missing or invalid values fail startup (twelve-factor). See the
// environment section of CLAUDE.md and PRD NFR-OBS-1.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ErrMissing marks a configuration problem so callers can distinguish it.
type lookupFunc func(key string) (string, bool)

// Config is the fully validated runtime configuration.
type Config struct {
	// Env is the deployment environment: "dev" or "prod".
	Env string
	// ProductName is the single source of the product's display name.
	ProductName string
	// SiteURL is the absolute public base URL, used for canonicals and links.
	SiteURL string

	Log          Log
	HTTP         HTTP
	Database     Database
	Identity     Identity
	TrustedProxy TrustedProxy
	Turnstile    Turnstile
	RateLimit    RateLimit
	Bookmarks    Bookmarks
	Model        Model
	Email        Email
	Billing      Billing
	Realtime     Realtime
	Metrics      Metrics
}

// Log configures structured logging.
type Log struct {
	// Level is the minimum slog level: debug, info, warn or error.
	Level string
}

// HTTP configures the web server.
type HTTP struct {
	// Addr is the listen address, for example ":3000".
	Addr string
	// ReadHeaderTimeout bounds time spent reading request headers.
	ReadHeaderTimeout time.Duration
	// ReadTimeout bounds time spent reading the whole request.
	ReadTimeout time.Duration
	// WriteTimeout bounds time spent writing a response.
	WriteTimeout time.Duration
	// IdleTimeout bounds keep-alive idle time.
	IdleTimeout time.Duration
	// ShutdownTimeout bounds graceful shutdown.
	ShutdownTimeout time.Duration
}

// Database configures the Postgres connection pool.
type Database struct {
	// URL is the pooled connection string used by the app.
	URL string
	// DirectURL is a direct (non-pooled) connection used for LISTEN and
	// migrations; it defaults to URL when unset.
	DirectURL string
	// MaxConns is the maximum pool size.
	MaxConns int32
	// MinConns is the minimum pool size kept warm.
	MinConns int32
	// StatementTimeout caps every statement.
	StatementTimeout time.Duration
}

// Identity configures anonymous identity and privacy-preserving hashing.
type Identity struct {
	// AnonIDSecret signs the anonymous-id cookie.
	AnonIDSecret string
	// IPHashSalt salts the source hash; rotate monthly.
	IPHashSalt string
	// IPHashSaltPrevious keeps the prior salt during rotation.
	IPHashSaltPrevious string
}

// TrustedProxy configures how the client address is derived.
type TrustedProxy struct {
	// Header is the header the platform sets, for example "X-Forwarded-For".
	Header string
	// Proxies lists trusted proxy CIDRs; the header is trusted only from these.
	Proxies []string
}

// Turnstile configures Cloudflare Turnstile bot defence.
type Turnstile struct {
	// SiteKey is the public widget key.
	SiteKey string
	// SecretKey is the server-side verification key.
	SecretKey string
}

// RateLimit selects the rate limiter backing store.
type RateLimit struct {
	// Mode is "memory" or "redis".
	Mode string
	// RedisURL is required when Mode is "redis".
	RedisURL string
	// SubmitWindow is the sliding window for submissions (FR-MOD-2).
	SubmitWindow time.Duration
	// SubmitMax is how many submissions one source may make per window.
	SubmitMax int
	// SaveWindow is the sliding window for save/unsave mutations.
	SaveWindow time.Duration
	// SaveMax is how many save mutations one identity may make per window.
	SaveMax int
}

// Bookmarks configures the saved-list cap. PRD FR-BKM-1 allows five saved
// frustrations without a plan and unlimited for Pro; anonymous visitors have no
// plan, so this is the anonymous cap.
type Bookmarks struct {
	// SavedLimit is the maximum saved frustrations per identity.
	SavedLimit int
}

// Model configures the LLM provider and model names.
type Model struct {
	// APIKey authenticates with the provider.
	APIKey string
	// Fast is the model used for moderation and classification.
	Fast string
	// Brief is the larger model used for founder briefs.
	Brief string
}

// Email configures the transactional email provider.
type Email struct {
	// APIKey authenticates with the provider.
	APIKey string
	// From is the envelope sender address.
	From string
}

// Billing configures the payment provider.
type Billing struct {
	// APIKey authenticates with the provider.
	APIKey string
	// WebhookSecret verifies webhook signatures.
	WebhookSecret string
}

// Realtime configures SSE and the LISTEN/NOTIFY bridge.
type Realtime struct {
	// Mode is "listen" (Postgres LISTEN) or "memory" (single instance).
	Mode string
	// MaxClients caps concurrent SSE clients per instance.
	MaxClients int
	// MaxPerIP caps concurrent SSE clients per source hash.
	MaxPerIP int
	// Heartbeat is the SSE keep-alive interval.
	Heartbeat time.Duration
	// MapRefreshMinInterval throttles map read-model rebuilds.
	MapRefreshMinInterval time.Duration
}

// Metrics configures the Prometheus endpoint.
type Metrics struct {
	// Addr is the internal listen address for /metrics.
	Addr string
}

// Load reads configuration from the process environment.
func Load() (Config, error) { return load(os.LookupEnv) }

func load(lookup lookupFunc) (Config, error) {
	var problems []string

	cfg := Config{
		Env:         opt(lookup, "APP_ENV", "dev"),
		ProductName: opt(lookup, "PRODUCT_NAME", "Grumble"),
		SiteURL:     strings.TrimRight(opt(lookup, "SITE_URL", "http://localhost:3000"), "/"),
		Log: Log{
			Level: opt(lookup, "LOG_LEVEL", "info"),
		},
		HTTP: HTTP{
			Addr:              opt(lookup, "HTTP_ADDR", ":3000"),
			ReadHeaderTimeout: optDuration(lookup, "HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			ReadTimeout:       optDuration(lookup, "HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      optDuration(lookup, "HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:       optDuration(lookup, "HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout:   optDuration(lookup, "HTTP_SHUTDOWN_TIMEOUT", 20*time.Second),
		},
		Database: Database{
			URL:              require(lookup, "DATABASE_URL", &problems),
			MaxConns:         int32(optInt(lookup, "DATABASE_MAX_CONNS", 10, &problems)),
			MinConns:         int32(optInt(lookup, "DATABASE_MIN_CONNS", 1, &problems)),
			StatementTimeout: optDuration(lookup, "DATABASE_STATEMENT_TIMEOUT", 5*time.Second),
		},
		Identity: Identity{
			AnonIDSecret:       require(lookup, "ANON_ID_SECRET", &problems),
			IPHashSalt:         require(lookup, "IP_HASH_SALT", &problems),
			IPHashSaltPrevious: opt(lookup, "IP_HASH_SALT_PREVIOUS", ""),
		},
		TrustedProxy: TrustedProxy{
			Header:  opt(lookup, "TRUSTED_PROXY_HEADER", "X-Forwarded-For"),
			Proxies: splitCSV(opt(lookup, "TRUSTED_PROXIES", "")),
		},
		Turnstile: Turnstile{
			SiteKey:   opt(lookup, "TURNSTILE_SITE_KEY", ""),
			SecretKey: opt(lookup, "TURNSTILE_SECRET_KEY", ""),
		},
		RateLimit: RateLimit{
			Mode:         opt(lookup, "RATELIMIT_MODE", "memory"),
			RedisURL:     opt(lookup, "RATELIMIT_REDIS_URL", ""),
			SubmitWindow: optDuration(lookup, "RATELIMIT_SUBMIT_WINDOW", time.Hour),
			SubmitMax:    optInt(lookup, "RATELIMIT_SUBMIT_MAX", 5, &problems),
			SaveWindow:   optDuration(lookup, "RATELIMIT_SAVE_WINDOW", time.Hour),
			SaveMax:      optInt(lookup, "RATELIMIT_SAVE_MAX", 120, &problems),
		},
		Bookmarks: Bookmarks{
			SavedLimit: optInt(lookup, "BOOKMARKS_MAX", 5, &problems),
		},
		Model: Model{
			APIKey: opt(lookup, "MODEL_API_KEY", ""),
			Fast:   opt(lookup, "MODEL_FAST", ""),
			Brief:  opt(lookup, "MODEL_BRIEF", ""),
		},
		Email: Email{
			APIKey: opt(lookup, "EMAIL_API_KEY", ""),
			From:   opt(lookup, "EMAIL_FROM", ""),
		},
		Billing: Billing{
			APIKey:        opt(lookup, "BILLING_API_KEY", ""),
			WebhookSecret: opt(lookup, "BILLING_WEBHOOK_SECRET", ""),
		},
		Realtime: Realtime{
			Mode:                  opt(lookup, "REALTIME_MODE", "memory"),
			MaxClients:            optInt(lookup, "SSE_MAX_CLIENTS", 2000, &problems),
			MaxPerIP:              optInt(lookup, "SSE_MAX_PER_IP", 10, &problems),
			Heartbeat:             optDuration(lookup, "SSE_HEARTBEAT", 20*time.Second),
			MapRefreshMinInterval: optDuration(lookup, "MAP_REFRESH_MIN_INTERVAL", 5*time.Second),
		},
		Metrics: Metrics{
			Addr: opt(lookup, "METRICS_ADDR", "127.0.0.1:9090"),
		},
	}

	cfg.Database.DirectURL = opt(lookup, "DATABASE_DIRECT_URL", cfg.Database.URL)

	problems = append(problems, cfg.validate()...)
	if len(problems) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %s", strings.Join(problems, "; "))
	}
	return cfg, nil
}

func (c Config) validate() []string {
	var problems []string

	switch c.Env {
	case "dev", "prod":
	default:
		problems = append(problems, fmt.Sprintf("APP_ENV must be dev or prod, got %q", c.Env))
	}

	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		problems = append(problems, fmt.Sprintf("LOG_LEVEL must be debug, info, warn or error, got %q", c.Log.Level))
	}

	if c.Database.URL == "" {
		problems = append(problems, "DATABASE_URL is required")
	}
	if c.Database.MaxConns < 1 {
		problems = append(problems, "DATABASE_MAX_CONNS must be at least 1")
	}
	if c.Database.MinConns < 0 || c.Database.MinConns > c.Database.MaxConns {
		problems = append(problems, "DATABASE_MIN_CONNS must be between 0 and DATABASE_MAX_CONNS")
	}
	if len(c.Identity.AnonIDSecret) < 16 {
		problems = append(problems, "ANON_ID_SECRET must be at least 16 characters")
	}
	if len(c.Identity.IPHashSalt) < 16 {
		problems = append(problems, "IP_HASH_SALT must be at least 16 characters")
	}

	switch c.RateLimit.Mode {
	case "memory":
	case "redis":
		if c.RateLimit.RedisURL == "" {
			problems = append(problems, "RATELIMIT_REDIS_URL is required when RATELIMIT_MODE=redis")
		}
	default:
		problems = append(problems, fmt.Sprintf("RATELIMIT_MODE must be memory or redis, got %q", c.RateLimit.Mode))
	}

	if c.RateLimit.SubmitMax < 1 {
		problems = append(problems, "RATELIMIT_SUBMIT_MAX must be at least 1")
	}
	if c.RateLimit.SubmitWindow <= 0 {
		problems = append(problems, "RATELIMIT_SUBMIT_WINDOW must be positive")
	}
	if c.RateLimit.SaveMax < 1 {
		problems = append(problems, "RATELIMIT_SAVE_MAX must be at least 1")
	}
	if c.RateLimit.SaveWindow <= 0 {
		problems = append(problems, "RATELIMIT_SAVE_WINDOW must be positive")
	}
	if c.Bookmarks.SavedLimit < 1 {
		problems = append(problems, "BOOKMARKS_MAX must be at least 1")
	}

	switch c.Realtime.Mode {
	case "memory", "listen":
	default:
		problems = append(problems, fmt.Sprintf("REALTIME_MODE must be memory or listen, got %q", c.Realtime.Mode))
	}

	return problems
}

func require(lookup lookupFunc, key string, problems *[]string) string {
	v, ok := lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		*problems = append(*problems, key+" is required")
		return ""
	}
	return v
}

func opt(lookup lookupFunc, key, fallback string) string {
	if v, ok := lookup(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func optInt(lookup lookupFunc, key string, fallback int, problems *[]string) int {
	v, ok := lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		*problems = append(*problems, fmt.Sprintf("%s must be an integer, got %q", key, v))
		return fallback
	}
	return n
}

func optDuration(lookup lookupFunc, key string, fallback time.Duration) time.Duration {
	v, ok := lookup(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return d
}

func splitCSV(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
