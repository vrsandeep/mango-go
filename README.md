<div align="center">
  <img src="internal/assets/web/static/images/logo.svg" alt="Mango-Go Logo" width="80" style="vertical-align: middle; margin-right: 20px;"/>
  <h1 style="display: inline-block; vertical-align: middle; margin: 0;">Mango-Go</h1>
</div>

<div align="center">
  <p>A self-hosted manga server and web reader written in Go</p>
  <p>
    <a href="https://github.com/vrsandeep/mango-go/actions/workflows/test.yml">
      <img src="https://github.com/vrsandeep/mango-go/actions/workflows/test.yml/badge.svg" alt="tests"/>
    </a>
  </p>
</div>

## Features

- **Manga Library Management** - Organize and browse your collection
- **Web Reader** - Read manga directly in your browser
- **Responsive Design** - Works on desktop, tablet, and mobile
- **Download Manager** - Download manga from various sources
- **Tagging System** - Organize with custom tags and folders
- **Multi-User Support** - User management with permission levels
- **Subscriptions** - Track and download new chapters automatically
- **Progress Tracking** - Keep track of your reading progress

## Quick Start

### Docker (Recommended)

1. **Create `docker-compose.yml`:**
   ```yaml
   services:
     mango:
       image: ghcr.io/vrsandeep/mango-go
       container_name: mango
       restart: unless-stopped
       ports:
         - "8080:8080"
       environment:
         - MANGO_DATABASE_PATH=/app/data/mango.db
         - MANGO_LIBRARY_PATH=/manga
         - MANGO_PLUGINS_PATH=/app/plugins
       volumes:
         - ./data:/app/data
         - ./manga:/manga
         - ./plugins:/app/plugins
   ```

2. **Start:**
   ```sh
   docker-compose up -d
   ```

3. **Get admin password:**
   ```sh
   docker-compose logs | grep "Password:"
   ```

4. **Access:** `http://localhost:8080`

### Binary

Download from [Releases](https://github.com/vrsandeep/mango-go/releases) or build with `make build`. Create `config.yml` (see [config.yml](./config.yml)) and run `./mango-go`.

## Library Organization

Organize manga with series at the root level:

```
manga/
├── One Piece/
│   ├── Volume 1/
│   │   ├── Chapter 1.cbr
│   │   └── Chapter 2.cbr
│   └── Volume 2/
└── Naruto/
    ├── Volume 1.cbz
    └── Volume 2.cbz
```

**Supported formats:** `.cbz`, `.cbr`, `.cb7`, `.zip`, `.rar`, `.7z`

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MANGO_LIBRARY_PATH` | Path to manga library | `./manga` |
| `MANGO_DATABASE_PATH` | SQLite database path | `./mango.db` |
| `MANGO_PLUGINS_PATH` | Path to plugins directory | `../mango-go-plugins` |
| `MANGO_PORT` | Web server port | `8080` |
| `MANGO_SCAN_INTERVAL` | Library scan interval (minutes) | `30` |

## Screenshots

![Home page](screenshots/home_light.png)
![Library](screenshots/library.png)
![Admin](screenshots/admin.png)
![Dark theme](screenshots/home_dark.png)

## Troubleshooting

- **Library Not Scanning**: Check file permissions and ensure the manga directory is accessible
- **Port Already in Use**: Change the port in your configuration

**View logs:**
```sh
# Docker
docker-compose logs

# Standalone
./mango-go 2>&1 | tee mango.log
```

## Documentation

- **Issues**: [GitHub Issues](https://github.com/vrsandeep/mango-go/issues)
- **Contributing**: [CONTRIBUTING.md](CONTRIBUTING.md)
- **Plugins**: [PLUGIN_SYSTEM_DESIGN.md](PLUGIN_SYSTEM_DESIGN.md)

## Acknowledgments

Original [Mango](https://github.com/hkalexling/mango/) project for inspiration
