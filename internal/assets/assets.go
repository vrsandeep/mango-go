package assets

import "embed"

//go:embed all:web/*.html
//go:embed all:web/dist
var WebFS embed.FS

//go:embed all:migrations
var MigrationsFS embed.FS
