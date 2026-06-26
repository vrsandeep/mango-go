# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Mango-Go is a self-hosted manga server and web reader written in Go. It serves a Chi-based HTTP API with an embedded single-page frontend, uses SQLite for persistence, and supports plugins written in JavaScript (executed via the goja VM).

## Commands

```bash
# Development
make run            # Bundle assets (unminified) + run with go run
make assets         # Bundle/minify CSS and JS only (requires esbuild)

# Production
make build          # Download deps + bundle + compile binary to ./build/mango-go

# Testing
go test ./...                          # Run all tests
go test ./internal/api -v              # Run API tests verbosely
go test ./internal/library -run TestX  # Run a single test

# Code quality
make format-check   # Verify Go (gofmt) + CSS/JS (Prettier) formatting
make prettify       # Auto-format Go + CSS/JS

# Cleanup
make clean          # Remove ./build and ./internal/assets/web/dist
```

**Prerequisites**: `esbuild` and `npx`/Prettier must be on PATH for asset bundling. `CGO_ENABLED=1` is required (go-fitz, go-sqlite3 use CGO). On Alpine/musl Docker builds, set `GO_BUILD_TAGS=musl`.

**Config**: Copy `config.yml` template to project root and set `library_path` to your manga directory. Env overrides use `MANGO_` prefix (e.g. `MANGO_PORT=9090`).

## Architecture

```
main.go
  └─ internal/core/app.go       — App struct: single dependency hub passed everywhere
       ├─ internal/config/       — Viper-based config from config.yml + MANGO_* env vars
       ├─ internal/db/           — SQLite connection + golang-migrate schema migrations
       ├─ internal/store/        — Data access layer (one file per entity/domain)
       ├─ internal/api/          — Chi HTTP router, all handlers and middleware
       ├─ internal/library/      — Manga directory scanner, fs watcher, archive parsing
       ├─ internal/plugins/      — JS plugin loader (goja VMs, lazy load/unload)
       ├─ internal/downloader/   — Worker pool for downloading chapters as .cbz
       ├─ internal/subscription/ — Periodic new-chapter checks (6-hour interval)
       ├─ internal/jobs/         — Background job registry (scan, thumbnail gen, etc.)
       ├─ internal/websocket/    — Hub for broadcasting scan progress to clients
       └─ internal/anilist/      — AniList GraphQL client for manga metadata
```

**Key patterns**:
- `core.App` is the single dependency container; all handlers and services receive it via constructor.
- Store layer (`internal/store/`) wraps SQL queries; interfaces are defined for testability (e.g. `HomeStore`, `AnilistSearcher`).
- API handlers live in `internal/api/*_handlers.go`; each file maps to a domain (browse, downloader, plugins, etc.).
- Static frontend assets are embedded in the binary via `go:embed` from `internal/assets/web/dist/`.

## Library Scanning

The scanner in `internal/library/` walks the manga directory, hashes archive contents (SHA1), and upserts records into SQLite. It runs at startup, on a configurable interval, and in response to filesystem events (fsnotify). Progress is broadcast over WebSocket (`/ws/admin/progress`).

Supported formats: `cbz`, `cbr`, `cb7`, `zip`, `rar`, `7z`, `pdf` — each PDF file is treated as a single chapter.

## Plugin System

Plugins live in the `plugins/` directory. Each plugin is a subdirectory containing `plugin.json` (manifest) and `index.js` (logic). Plugins run in isolated goja VMs and are lazy-loaded on first access, then unloaded after idle time. They implement `search`, `getChapters`, and `getPageURLs`. See `PLUGIN_SYSTEM_DESIGN.md` for the full contract.

## Testing Approach

- Unit tests mock dependencies via interfaces.
- `internal/testutil/` provides shared test helpers.
- API handler tests (`*_handlers_test.go`) use `httptest` and mock stores.
- There are no integration tests against a real DB; store tests use in-memory SQLite.
