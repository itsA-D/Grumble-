// Command app is the single Grumble binary. It runs one of four subcommands:
// serve (web and API), worker (background jobs), migrate (apply migrations)
// and seed (load labelled seed content and local fixtures).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"grumble/internal/app"
	"grumble/internal/platform/config"
	"grumble/internal/platform/logging"
	"grumble/internal/store/postgres"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(args) == 0 {
		usage()
		return errors.New("missing subcommand")
	}
	switch args[0] {
	case "serve":
		return runServe(ctx)
	case "worker":
		return runWorker(ctx)
	case "migrate":
		return runMigrate(ctx)
	case "seed":
		return runSeed(ctx)
	case "healthcheck":
		return runHealthcheck(ctx)
	case "-h", "--help", "help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Grumble

Usage:
  app serve        run the web server
  app worker       run only the background worker
  app migrate      apply database migrations
  app seed         load labelled seed content and local fixtures
  app healthcheck  probe the local readiness endpoint (for containers)
`)
}

// runHealthcheck probes the local readiness endpoint and exits non-zero on
// failure. It lets a distroless image report health without a shell or curl.
func runHealthcheck(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, readinessURL(cfg.HTTP.Addr), nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: readiness returned %s", resp.Status)
	}
	return nil
}

func readinessURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://127.0.0.1:3000/readyz"
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/readyz"
}

func runMigrate(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(cfg.Log.Level)

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := postgres.Migrate(ctx, cfg.Database.DirectURL); err != nil {
		return err
	}
	logger.Info("migrations applied")
	return nil
}

func runSeed(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(cfg.Log.Level)
	return app.Seed(ctx, cfg, logger)
}

func runServe(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(cfg.Log.Level)
	slog.SetDefault(logger)

	a, err := app.New(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer a.Close()
	return a.Run(ctx)
}

func runWorker(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(cfg.Log.Level)
	return app.RunWorker(ctx, cfg, logger)
}
