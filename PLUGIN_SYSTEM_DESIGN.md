# Mango-Go Plugin System Design

## Quick Reference

**Plugin Template**: [mango-go-plugins/template](https://github.com/vrsandeep/mango-go-plugins/tree/master/template)

**Key Concepts:**
- Plugins are JavaScript files using goja runtime
- Lazy loading: plugins discovered on startup, loaded on first access
- Sandboxed: isolated VMs, no filesystem/network access except via mango API
- Provider interface: plugins implement `search`, `getChapters`, `getPageURLs`

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Plugin Structure](#plugin-structure)
4. [Plugin API](#plugin-api)
5. [Plugin Loader](#plugin-loader)
6. [Integration Points](#integration-points)
7. [Error Handling & Sandboxing](#error-handling--sandboxing)
8. [Testing Strategy](#testing-strategy)
9. [State Management](#state-management)
10. [HTML Parsing](#html-parsing--scraping-support)
11. [Repository System](#plugin-repository--installation)

---

## Overview

### Goals
- Enable community to extend mango-go with new comic sources
- Platform-agnostic plugins (no platform-specific binaries)
- Plugins cannot crash the main application
- Plugins should be independently testable
- Backward compatible with existing built-in providers

### Technology Choice
- **Runtime**: [goja](https://github.com/dop251/goja) - Pure Go JavaScript engine
- **Why**: Platform agnostic, crash isolation, excellent developer experience, standard JS ecosystem

---

## Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                    Mango-Go Core                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │  API Layer   │  │   Worker     │  │ Subscription │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         └──────────────────┼──────────────────┘              │
│                            ▼                                  │
│                  ┌─────────────────────┐                     │
│                  │   Provider Registry │                     │
│                  │  (Built-in + Plugin)│                     │
│                  └──────────┬──────────┘                     │
│                            ▼                                  │
│                  ┌─────────────────────┐                     │
│                  │   Plugin Manager   │                      │
│                  │   (Discovery + Lazy)│                     │
│                  └──────────┬──────────┘                     │
└─────────────────────────────┼──────────────────────────────────┘
                               │
                               ▼
                  ┌─────────────────────┐
                  │    Plugin Runtime   │
                  │   (goja VM + API)  │
                  └─────────────────────┘
```

### Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| **Plugin Manager** | Discovers plugins, manages lifecycle (load/unload/reload), lazy loading |
| **Plugin Runtime** | Creates isolated goja VM, injects Mango API, handles execution with error recovery |
| **Provider Adapter** | Adapts JS exports to Go `Provider` interface, handles type conversions |

---

## Plugin Structure

### Directory Layout

```
plugins/
├── example-site/
│   ├── plugin.json          # Plugin manifest
│   ├── index.js            # Main plugin code
│   └── state.json          # Plugin state (auto-generated)
```

### Plugin Manifest Schema (`plugin.json`)

```json
{
  "id": "example-site",
  "name": "Example Site",
  "version": "1.0.0",
  "description": "Download manga from example.com",
  "author": "Plugin Author",
  "license": "MIT",
  "api_version": "1.0",
  "plugin_type": "downloader",
  "entry_point": "index.js",
  "capabilities": {
    "search": true,
    "chapters": true,
    "download": true
  },
  "config": {
    "base_url": {
      "type": "string",
      "default": "https://api.example.com",
      "description": "Example Site API base URL"
    }
  }
}
```

**Required Fields:**
- `id`: Unique identifier (alphanumeric, hyphens, underscores)
- `name`: Human-readable name
- `version`: Semantic version
- `api_version`: Mango API version required
- `plugin_type`: `"downloader"` (default)
- `entry_point`: Main JS file (default: "index.js")

---

## Plugin API

### Core Interface

Plugins must export these functions:

```javascript
exports.getInfo = () => ProviderInfo
exports.search = async (query, mango) => SearchResult[]
exports.getChapters = async (seriesId, mango) => ChapterResult[]
exports.getPageURLs = async (chapterId, mango) => string[]
```

### Mango API Object

Injected into each plugin function call:

```javascript
mango = {
  // HTTP client
  http: {
    get(url, options?) => Response,
    post(url, body?, options?) => Response
  },

  // Configuration (from plugin.json)
  config: {
    get(key) => value,
    set(key, value) => void
  },

  // State persistence (plugins/{id}/state.json)
  state: {
    get(key) => value,
    set(key, value) => void,
    delete(key) => void
  },

  // Logging
  log: {
    info(message),
    warn(message),
    error(message)
  },

  // HTML parsing
  html: {
    parseHTML(html) => Document,
    querySelector(doc, selector) => Element,
    querySelectorAll(doc, selector) => Element[]
  }
}
```

### Type Definitions

| Type | Structure |
|------|-----------|
| **ProviderInfo** | `{ id: string, name: string, version: string }` |
| **SearchResult** | `{ title: string, identifier: string, cover_url: string }` |
| **ChapterResult** | `{ title: string, identifier: string, number: number, volume?: number }` |

### Example Plugin

See the [plugin template](https://github.com/vrsandeep/mango-go-plugins/tree/master/template) for a complete example plugin implementation.

---

## Plugin Loader

### Discovery Process

1. Scan `plugins/` directory for subdirectories
2. Load and validate `plugin.json` manifests
3. Register lazy adapters with Provider Registry
4. Load plugins on-demand (first access)

### Lazy Loading

- Plugins are **discovered** on startup (manifest loaded)
- Plugins are **loaded** on first access (runtime created)
- Plugins are **unloaded** after idle timeout (configurable, default: 30 minutes)
- Unloaded plugins automatically reload on next access

### Registry Integration

```go
// Plugins register as Provider implementations
providers.Get("example-site")  // Works for both built-in and plugin providers

// Lazy adapter wraps plugin until first access
type LazyPluginProviderAdapter struct {
    manager  *PluginManager
    pluginID string
}
```

---

## Integration Points

| Component | Integration |
|-----------|-------------|
| **Provider Registry** | Plugins register alongside built-in providers |
| **Download Worker** | Uses provider registry, works transparently with plugins |
| **Subscription Service** | References provider IDs, works with plugins |
| **API Handlers** | Lists all providers (built-in + plugins) |
| **Configuration** | Plugin directory path: `plugins.path` |

---

## Error Handling & Sandboxing

### Error Recovery

- Plugin panics caught and converted to errors
- Timeouts: 30 seconds default per call
- Errors logged, don't crash main application
- Failed plugins marked but don't prevent others

### Sandboxing

**Allowed:**
- HTTP requests via `mango.http`
- Logging via `mango.log`
- Plugin configuration access
- Standard JavaScript operations

**Restricted:**
- No filesystem access
- No network access outside `mango.http`
- No process spawning
- No native modules
- Limited memory/CPU (configurable)

### Error Types

```go
type PluginError struct {
    PluginID    string
    Function    string
    Message     string
    Cause       error
    IsTimeout   bool
    IsPanic     bool
}
```

---

## Testing Strategy

### Plugin Testing (Independent)

Mock the `mango` object for unit testing:

```javascript
const mockMango = {
  http: { get: async (url) => ({ body: '{}' }) },
  log: { info: () => {} }
};
const result = await exports.search("test", mockMango);
```

### Integration Testing

- Unit tests for loader/runtime
- Integration tests with mock plugins
- Error recovery tests
- Timeout tests

---

## State Management

### Plugin State Storage

- Location: `plugins/{plugin-id}/state.json`
- Auto-managed by mango-go
- Accessed via `mango.state` API
- Used for: auth tokens, session data, preferences

**API:**
```javascript
mango.state.set('auth_token', 'value');
const token = mango.state.get('auth_token');
mango.state.delete('auth_token');
```

---

## HTML Parsing & Scraping Support

### Usage

```javascript
const html = await mango.http.get(url).then(r => r.body);
const doc = mango.html.parseHTML(html);
const title = mango.html.querySelector(doc, "h1.title").textContent;
const links = mango.html.querySelectorAll(doc, "a.chapter-link");
```

**Features:**
- Parse HTML strings into document objects
- CSS selector support (`querySelector`, `querySelectorAll`)
- Element API: `textContent`, `innerHTML`, `getAttribute(name)`

**Limitations:**
- No JavaScript execution in HTML (static parsing only)
- Simplified API (not full browser DOM)

---

## Plugin Repository & Installation

### Repository Metadata Format

```json
{
  "name": "Mango-Go Plugins",
  "plugins": [
    {
      "id": "example-site",
      "name": "Example Site",
      "version": "1.0.0",
      "description": "Download from example.com",
      "repository": {
        "url": "https://github.com/user/repo",
        "branch": "main"
      },
      "download_url": "https://github.com/user/repo/archive/main.zip"
    }
  ]
}
```

### Installation Flow

1. User adds repository URL in admin UI
2. System fetches repository metadata
3. User selects plugins to install
4. System downloads and extracts plugin ZIP
5. System validates manifest
6. Plugin is discovered and registered

### API Endpoints

```
GET  /api/plugins                    # List installed plugins
GET  /api/plugins/repository         # Fetch available plugins
POST /api/plugins/install           # Install plugin(s)
POST /api/plugins/{id}/reload       # Reload plugin
DELETE /api/plugins/{id}            # Uninstall plugin
```

---

## Security Considerations

- Plugins run in isolated VMs
- No filesystem access except via mango API
- Network access only through `mango.http`
- Timeout limits prevent resource exhaustion
- API version checking prevents incompatible plugins

---

## Performance Considerations

- **Lazy loading**: Plugins loaded on first access
- **Idle unloading**: Unloaded after timeout (configurable, default: 30 min)
- **Isolated VMs**: Each plugin has separate runtime
- **Caching**: HTTP requests can be cached using state API

---

## File Structure

```
internal/plugins/
├── manager.go          # Plugin manager (discovery, lifecycle)
├── runtime.go          # goja runtime wrapper
├── adapter.go          # Provider interface adapter
├── lazy_adapter.go     # Lazy loading wrapper
├── manifest.go         # Manifest parsing
└── api.go              # Mango API injection
```

---

## Migration Strategy

- Built-in providers remain alongside plugins
- Existing subscriptions continue to work
- Provider IDs remain the same
- No database schema changes required
- API endpoints remain unchanged
