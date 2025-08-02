# Development Guide

This document provides guidelines for setting up Mango-Go for development and contributing to the project.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.24.4 or later**: [Install Go](https://golang.org/dl/)
- **Git**: [Install Git](https://git-scm.com/downloads)
- **SQLite3**: Usually comes with Go, but you may need to install it separately on some systems
- **Make** (optional): For using the provided Makefile commands
- **Docker & Docker Compose:** (Recommended for production)
- **esbuild:** (Required for production builds) A very fast JavaScript and CSS bundler. You can install it with npm or Go:
    ```sh
    # Using npm (requires Node.js)
    npm install -g esbuild

    # Or, using Go
    go install [github.com/evanw/esbuild/cmd/esbuild@latest](https://github.com/evanw/esbuild/cmd/esbuild@latest)
    ```

## Local Development Setup

### 1. Clone the Repository

```bash
git clone https://github.com/vrsandeep/mango-go.git
cd mango-go
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Application

```bash
make build
```

### 4. Create Configuration

Create a `config.yml` file in the project root:

```yaml
library:
  path: "./manga"  # Path to your manga library
database:
  path: "./mango.db"  # SQLite database path
port: 8080
scan_interval: 30  # Library scan interval in minutes
```

### 5. Set Up Test Data

Create a test manga library structure in the `./manga` directory:

```
manga/
├── Series A/
│   ├── Volume 1/
│   │   ├── Chapter 1.cbr
│   │   └── Chapter 2.cbr
│   └── Volume 2/
│       └── Chapter 3.cbr
└── Series B/
    └── Chapter 1.cbz
```

**Note**: While any folder structure works, it's recommended to organize series at the root level of your library folder for optimal scanning performance.

### 6. Run the Application

```bash
./mango-go
```

The application will start on `http://localhost:8080`. On first run, it will create a default admin user with credentials printed to the console.

## Development Workflow

### Running Tests

Run all tests:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

Run tests for a specific package:
```bash
go test ./internal/api
```

Run tests with coverage:
```bash
go test -cover ./...
```

### Code Quality

The project uses several tools to maintain code quality:

1. **Format code**:
   ```bash
   go fmt ./...
   ```

2. **Vet code**:
   ```bash
   go vet ./...
   ```

3. **Run linter** (if you have golangci-lint installed):
   ```bash
   golangci-lint run
   ```

### Database Migrations

The application uses SQLite with migrations stored in `internal/assets/migrations/`. When making database schema changes:

1. Create new migration files in the migrations directory
2. Follow the naming convention: `000XXX_description.up.sql` and `000XXX_description.down.sql`
3. Test migrations both up and down

### Frontend Development

The frontend is built with vanilla HTML, CSS, and JavaScript. Files are located in `internal/assets/web/`:

- HTML templates: `internal/assets/web/*.html`
- CSS styles: `internal/assets/web/static/css/`
- JavaScript: `internal/assets/web/static/js/`
- Images: `internal/assets/web/static/images/`

## Project Structure

```
mango-go/
├── internal/
│   ├── api/           # HTTP handlers and routing
│   ├── assets/        # Web assets and migrations
│   ├── auth/          # Authentication logic
│   ├── config/        # Configuration management
│   ├── core/          # Core application setup
│   ├── db/            # Database initialization
│   ├── downloader/    # Manga download functionality
│   ├── jobs/          # Background job management
│   ├── library/       # Library scanning and parsing
│   ├── models/        # Data structures
│   ├── store/         # Database queries and data access
│   ├── subscription/  # Subscription management
│   ├── testutil/      # Testing utilities
│   ├── util/          # Utility functions
│   └── websocket/     # WebSocket functionality
├── main.go            # Application entry point
├── go.mod             # Go module definition
├── go.sum             # Dependency checksums
├── config.yml         # Configuration file
└── README.md          # User documentation
```

## Contributing

### Before You Start

1. Check existing issues and pull requests to avoid duplicates
2. Discuss major changes in an issue before implementing
3. Ensure your changes align with the project's goals

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the coding standards:
   - Write tests for new functionality
   - Update documentation as needed
   - Follow Go naming conventions

3. **Test your changes**:
   ```bash
   go test ./...
   go build .
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add new feature description"
   ```

### Commit Message Format

Use conventional commit format:

- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `style:` for formatting changes
- `refactor:` for code refactoring
- `test:` for adding tests
- `chore:` for maintenance tasks

### Submitting a Pull Request

1. **Push your branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a Pull Request** on GitHub with:
   - Clear description of changes
   - Reference to related issues
   - Screenshots for UI changes
   - Test results

3. **Ensure CI passes** before requesting review

### Code Review Process

1. All PRs require at least one review
2. Address review comments promptly
3. Maintainers may request changes before merging
4. Squash commits when requested

## Testing Guidelines

### Unit Tests

- Write tests for all new functionality
- Aim for good test coverage
- Use descriptive test names
- Mock external dependencies


### Test Utilities

Use the test utilities in `internal/testutil/` for common testing tasks:

- `testutil.DB()`: Get test database connection
- `testutil.Server()`: Create test HTTP server
- `testutil.Auth()`: Authentication helpers

## Debugging

### Enable Debug Logging

Set the log level in your config or use environment variables:

```bash
export MANGO_LOG_LEVEL=debug
./mango-go
```

### Database Inspection

Use SQLite CLI to inspect the database:

```bash
sqlite3 mango.db
.tables
.schema users
SELECT * FROM users;
```

### Profiling

Enable HTTP profiling by adding the import and route:

```go
import _ "net/http/pprof"
```

Then access profiling data at `http://localhost:8080/debug/pprof/`

## Common Issues

### Build Issues

- Ensure you're using Go 1.24.4+
- Run `go mod tidy` to clean dependencies
- Check that all dependencies are properly installed

### Database Issues

- Ensure SQLite is properly installed
- Check file permissions for database directory
- Verify migration files are up to date

### Library Scanning Issues

- Check file permissions on manga directory
- Ensure supported archive formats (.cbz, .cbr, .zip, .rar)
- Verify manga files are not corrupted
- Do not have folders with just images.

## Getting Help

- Check existing issues and discussions
- Create a new issue with detailed information
- Join the project discussions
- Review the codebase for similar implementations

## Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create release tag
4. Build and test release artifacts
5. Publish release notes

---

Thank you for contributing to Mango-Go! Your contributions help make this project better for everyone.