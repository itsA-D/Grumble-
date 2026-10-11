// Package migrations embeds the goose SQL migration files so the binary can
// apply them without external files on disk.
package migrations

import "embed"

// FS holds the embedded .sql migration files, applied in filename order by
// GitHub.com/pressly/goose's migration runner.
//
//go:embed *.sql
var FS embed.FS
