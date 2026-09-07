// Package migrations embeds the SQL migration files so they ship inside the
// compiled binary instead of being read from disk at runtime.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
