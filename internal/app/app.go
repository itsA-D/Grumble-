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
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"grumble/internal/jobs"
	"grumble/internal/platform/clock"
	"grumble/internal/platform/config"
	"grumble/internal/platform/db"
	"grumble/internal/platform/idgen"
	"grumble/internal/platform/ratelimit"
	"grumble/internal/platform/turnstile"
	"grumble/internal/service/bookmarks"
	"grumble/internal/service/feed"
	"grumble/internal/service/hubs"
	"grumble/internal/service/identity"
	"grumble/internal/service/mapsvc"
	"grumble/internal/service/pains"
	"grumble/internal/service/seed"
	"grumble/internal/service/trending"
	"grumble/internal/service/voting"
	"grumble/internal/store/postgres"
	"grumble/internal/transport/web"
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

	tx := postgres.NewTxManager(pool)
	bookmarksSvc := bookmarks.New(postgres.NewBookmarks(pool), tx, bookmarks.Config{
		SavedLimit: cfg.Bookmarks.SavedLimit,
	})
	saveLimiter := ratelimit.NewMemory(ratelimit.Config{
		Window: cfg.RateLimit.SaveWindow,
		Max:    cfg.RateLimit.SaveMax,
	}, time.Now)

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
		Voting:             voting.New(postgres.NewPains(pool), postgres.NewVotes(pool), tx),
		Hubs:               hubs.New(postgres.NewPains(pool)),
		Map:                mapsvc.New(postgres.NewCountryStats(pool)),
		Trending:           trending.New(postgres.NewPains(pool), clock.System{}),
		Bookmarks:          bookmarksSvc,
		SaveLimiter:        saveLimiter,
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

// Seed loads the labelled demo content (PRD section 6). Every row is stored
// published and flagged seeded = true; the run is idempotent.
func Seed(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	pool, err := db.Open(ctx, dbConfig(cfg))
	if err != nil {
		return err
	}
	defer pool.Close()

	newID := func() uuid.UUID {
		id, err := idgen.New()
		if err != nil {
			return uuid.Nil
		}
		return id
	}
	svc := seed.New(postgres.NewPains(pool), postgres.NewTxManager(pool), newID, seed.DefaultItems())
	rep, err := svc.Run(ctx)
	if err != nil {
		return err
	}
	if err := jobs.RebuildCountryStats(ctx, pool); err != nil {
		return fmt.Errorf("rebuild map stats: %w", err)
	}
	logger.Info("seed complete", "created", rep.Created, "skipped", rep.Skipped, "seeded", true)
	return nil
}

// RunWorker runs background jobs. The only job today is the periodic rebuild of
// the map's country_stats read model (FR-MAP-1); River owns scheduling so a
// restart never loses the cadence.
func RunWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	pool, err := db.Open(ctx, dbConfig(cfg))
	if err != nil {
		return err
	}
	defer pool.Close()

	driver := riverpgxv5.New(pool)
	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		return fmt.Errorf("river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("river migrate: %w", err)
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, jobs.NewCountryStatsJob(pool))

	client, err := river.NewClient(driver, &river.Config{
		Logger: logger,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 2},
		},
		Workers:      workers,
		PeriodicJobs: []*river.PeriodicJob{jobs.CountryStatsPeriodicJob()},
	})
	if err != nil {
		return fmt.Errorf("river client: %w", err)
	}
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("river start: %w", err)
	}
	logger.Info("worker started", "env", cfg.Env, "jobs", "country-stats")

	<-ctx.Done()

	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Stop(stopCtx); err != nil {
		logger.Warn("river stop", "error", err)
	}
	logger.Info("worker stopped")
	return nil
}

func dbConfig(cfg config.Config) db.Config {
	return db.Config{
		URL:              cfg.Database.URL,
		MaxConns:         cfg.Database.MaxConns,
		MinConns:         cfg.Database.MinConns,
		StatementTimeout: cfg.Database.StatementTimeout,
		ApplicationName:  "grumble",
	}
}
