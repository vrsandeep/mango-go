package assets

import "embed"

//go:embed all:web
var WebFS embed.FS

//go:embed all:migrations
var MigrationsFS embed.FS
