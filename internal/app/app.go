// Package app is the composition root: it builds the logger, the database
// pool, the HTTP server and the background worker, and wires them together.
// Dependency injection is manual so the wiring is explicit and testable.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OWNER/painboard/internal/platform/clock"
	"github.com/OWNER/painboard/internal/platform/config"
	"github.com/OWNER/painboard/internal/platform/db"
	"github.com/OWNER/painboard/internal/platform/idgen"
	"github.com/OWNER/painboard/internal/platform/ratelimit"
	"github.com/OWNER/painboard/internal/platform/turnstile"
	"github.com/OWNER/painboard/internal/service/feed"
	"github.com/OWNER/painboard/internal/service/hubs"
	"github.com/OWNER/painboard/internal/service/identity"
	"github.com/OWNER/painboard/internal/service/pains"
	"github.com/OWNER/painboard/internal/service/voting"
	"github.com/OWNER/painboard/internal/store/postgres"
	"github.com/OWNER/painboard/internal/transport/web"
)

// App owns the long-lived resources of a running server.
type App struct {
	cfg    config.Config
	logger *slog.Logger
	pool   *pgxpool.Pool
	server *http.Server
}

// New builds the application: it opens the database pool and constructs the
// HTTP server. Call Run to serve and Close to release resources.
func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	pool, err := db.Open(ctx, dbConfig(cfg))
	if err != nil {
		return nil, err
	}

	handler := web.NewRouter(web.Deps{
		Logger:             logger,
		Pool:               pool,
		ProductName:        cfg.ProductName,
		SiteURL:            cfg.SiteURL,
		Env:                cfg.Env,
		TrustedProxyHeader: cfg.TrustedProxy.Header,
		TrustedProxies:     cfg.TrustedProxy.Proxies,
		Feed:               feed.New(postgres.NewPains(pool), clock.System{}),
		Pains:              newPainsService(cfg, pool),
		Voting:             voting.New(postgres.NewPains(pool), postgres.NewVotes(pool), postgres.NewTxManager(pool)),
		Hubs:               hubs.New(postgres.NewPains(pool)),
		Identity:           identity.New(cfg.Identity.AnonIDSecret, cfg.Identity.IPHashSalt, cfg.Identity.IPHashSaltPrevious),
		NewAnonID:          newAnonID,
		TurnstileSiteKey:   cfg.Turnstile.SiteKey,
	})

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	return &App{cfg: cfg, logger: logger, pool: pool, server: server}, nil
}

// newPainsService assembles the submission service and its adapters.
func newPainsService(cfg config.Config, pool *pgxpool.Pool) *pains.Service {
	limit := ratelimit.NewMemory(ratelimit.Config{
		Window: cfg.RateLimit.SubmitWindow,
		Max:    cfg.RateLimit.SubmitMax,
	}, time.Now)
	// Turnstile is enabled exactly when a secret is configured, so the local
	// path works without keys (FR-MOD-7).
	verifier := turnstile.New(cfg.Turnstile.SecretKey != "", cfg.Turnstile.SecretKey, http.DefaultClient)
	return pains.New(
		postgres.NewPains(pool),
		postgres.NewTxManager(pool),
		identity.New(cfg.Identity.AnonIDSecret, cfg.Identity.IPHashSalt, cfg.Identity.IPHashSaltPrevious),
		limit,
		verifier,
		clock.System{},
		idGenFunc(idgen.New),
		pains.Config{},
	)
}

// idGenFunc adapts a function to the pains.IDGen interface.
type idGenFunc func() (uuid.UUID, error)

func (f idGenFunc) New() (uuid.UUID, error) { return f() }

// newAnonID mints a fresh raw anonymous id for a first-time visitor.
func newAnonID() string {
	s, err := idgen.String()
	if err != nil {
		return ""
	}
	return s
}

// Run serves HTTP until the context is cancelled, then shuts down gracefully.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("http server listening", "addr", a.cfg.HTTP.Addr, "env", a.cfg.Env)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		a.logger.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	return nil
}

// Close releases the database pool.
func (a *App) Close() {
	if a.pool != nil {
		a.pool.Close()
	}
}

// Seed verifies database connectivity and loads labelled seed content. Seed
// content is added with the first tables (M1).
func Seed(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, dbConfig(cfg))
	if err != nil {
		return err
	}
	defer pool.Close()

	logger.Info("seed: database reachable; seed content is added with the first tables (M1)")
	return nil
}

// RunWorker runs background jobs. The job queue (River) is introduced with the
// first asynchronous work; until then the worker idles so the process stays up.
func RunWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	logger.Info("worker started", "env", cfg.Env, "message", "no jobs registered yet")
	<-ctx.Done()
	logger.Info("worker stopped")
	return nil
}

func dbConfig(cfg config.Config) db.Config {
	return db.Config{
		URL:              cfg.Database.URL,
		MaxConns:         cfg.Database.MaxConns,
		MinConns:         cfg.Database.MinConns,
		StatementTimeout: cfg.Database.StatementTimeout,
		ApplicationName:  "painboard",
	}
}
