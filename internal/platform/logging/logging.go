// Package logging builds the structured logger used across the application.
// Logs are JSON on stdout with a request id on every line (PRD NFR-OBS-1).
package logging

import (
	"context"
	"log/slog"
	"os"
)

type contextKey struct{}

// New returns a JSON slog.Logger at the given level.
func New(level string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	}))
}

// WithLogger stores a logger in the context.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext returns the logger stored in the context, or the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
