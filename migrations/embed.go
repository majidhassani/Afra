// Package migrations embeds all SQL migration files so the binary can
// apply them without filesystem access.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
