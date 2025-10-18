# Development Guide

This document provides guidelines for setting up Mango-Go for development and contributing to the project.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.24.4 or later**: [Install Go](https://golang.org/dl/)
- **Git**: [Install Git](https://git-scm.com/downloads)
- **Node.js 18 or later**: [Install Node.js](https://nodejs.org/) (Required for esbuild)
- **SQLite3**: Usually comes with Go, but you may need to install it separately on some systems
- **Make**: For using the provided Makefile commands
- **Docker & Docker Compose:** (Optional, for containerized development)
- **esbuild:** (Required for asset bundling) A very fast JavaScript and CSS bundler:
    ```sh
    # Using npm (recommended)
    npm install -g esbuild

    # Or, using Go
    go install github.com/evanw/esbuild/cmd/esbuild@latest
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

### 3. Development Commands

The project provides several Make commands for different development scenarios:

**For local development (recommended):**
```bash
make run
```
This command creates un-minified bundles for easier debugging and runs the application.

**For production builds:**
```bash
make build
```
This command creates minified bundles and builds the production binary.

**Other useful commands:**
```bash
make assets          # Build assets only
make clean           # Clean build artifacts
make download-go-deps # Download and tidy Go dependencies
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

**For development:**
```bash
make run
```

**For production:**
```bash
make build
./build/mango-go
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

### Development Best Practices

**Code Style:**
- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Write self-documenting code with clear comments
- Keep functions small and focused
- Use interfaces for abstraction

**Error Handling:**
- Always handle errors explicitly
- Use `errors.Wrap()` for context when wrapping errors
- Log errors at appropriate levels
- Return meaningful error messages

**Security:**
- Validate all user inputs
- Use parameterized queries for database operations
- Sanitize file paths and user-provided data
- Follow the principle of least privilege

**Testing:**
- Test edge cases and error conditions
- Use table-driven tests for multiple scenarios
- Mock external dependencies appropriately

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

#### Code Formatting

The project uses Prettier to maintain consistent code formatting for CSS and JavaScript files.

**Format all CSS and JS files:**
```bash
make format
```

**Check if files are properly formatted:**
```bash
make format-check
```

**Format only CSS files:**
```bash
npm run format:css
```

**Format only JavaScript files:**
```bash
npm run format:js
```

Prettier configuration is defined in `.prettierrc` and ignores files listed in `.prettierignore`.

### Docker Development

The project includes Docker support for containerized development and deployment:

**Development with Docker Compose:**
```bash
# Start the application with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the application
docker-compose down
```

**Building Docker Images:**
```bash
# Build the Docker image
docker build -t mango-go .

# Run the container
docker run -p 8080:8080 -v $(pwd)/manga:/app/manga mango-go
```

**Docker Configuration:**
- `Dockerfile`: Multi-stage build for production images
- `docker-compose.yml`: Development environment setup
- `.dockerignore`: Excludes unnecessary files from Docker context

**Container Development Guidelines:**
- Use volume mounts for development (manga library, config)
- Ensure proper file permissions for mounted volumes
- Use environment variables for configuration in containers
- Test both local and containerized builds before submitting PRs

## Project Structure

```
mango-go/
├── .github/           # GitHub Actions workflows and issue templates
│   ├── workflows/     # CI/CD pipelines
│   └── ISSUE_TEMPLATE/ # Issue and PR templates
├── build/             # Build artifacts (generated)
├── data/              # Runtime data directory
├── internal/          # Internal application code
│   ├── api/           # HTTP handlers and routing
│   ├── assets/        # Web assets and migrations
│   │   ├── migrations/ # Database migration files
│   │   └── web/       # Frontend assets (HTML, CSS, JS)
│   ├── auth/          # Authentication logic
│   ├── config/        # Configuration management
│   ├── core/          # Core application setup
│   ├── db/            # Database initialization
│   ├── downloader/    # Manga download functionality
│   │   └── providers/ # Download provider implementations
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
├── Makefile           # Build and development commands
├── Dockerfile         # Container configuration
├── docker-compose.yml # Docker Compose setup
└── README.md          # User documentation
```

## Contributing

### Before You Start

1. Check existing issues and pull requests to avoid duplicates
2. Discuss major changes in an issue before implementing
3. Ensure your changes align with the project's goals

### Issue Templates

The project provides issue templates to help structure bug reports and feature requests:

**Bug Reports** (`.github/ISSUE_TEMPLATE/bug_report.md`):
- Use the "[Bug Report]" prefix in the title
- Include environment details (OS, browser, Mango version)
- Provide clear reproduction steps
- Include Docker configuration if applicable

**Feature Requests** (`.github/ISSUE_TEMPLATE/feature_request.md`):
- Use the "[Feature Request]" prefix in the title
- Describe the problem and proposed solution
- Provide use cases and benefits
- Include mockups or examples when helpful

**General Questions** (`.github/ISSUE_TEMPLATE/general-question.md`):
- For questions that don't fit bug reports or feature requests
- Use for discussions and clarifications

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Plan your development in phases**:
   - Break large features into smaller, manageable phases
   - Fully implement and test each phase before proceeding
   - This approach ensures stable, incremental progress

3. **Make your changes** following the coding standards:
   - Write tests for new functionality
   - Update documentation as needed
   - Follow Go naming conventions
   - Keep changes focused and atomic

4. **Test your changes**:
   ```bash
   go test ./...
   go build .
   ```

5. **Commit your changes**:
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

### Continuous Integration

The project uses GitHub Actions for automated testing and building:

**Test Pipeline** (`.github/workflows/test.yml`):
- Runs on every push and pull request to master
- Tests on Ubuntu with Go 1.24.4 and Node.js 18
- Installs dependencies and builds assets
- Runs the full test suite with `go test ./...`

**Release Pipeline** (`.github/workflows/release.yml`):
- Triggers on version tags (v*.*.*)
- Builds binaries for multiple platforms:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
- Creates release archives with checksums
- Publishes to GitHub Releases

**Docker Pipeline** (`.github/workflows/docker-*.yml`):
- Builds and publishes Docker images
- Supports both edge and release versions

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
- Keep tests small and independent [[memory:6281978]]
- Merge duplicate tests when possible [[memory:6281978]]

### Test Utilities

Use the test utilities in `internal/testutil/` for common testing tasks:

- `testutil.SetupTestDB(t)`: Get test database connection with cleanup
- `testutil.SetupTestServer(t)`: Create test HTTP server with full app setup
- `testutil.SetupTestApp(t)`: Create test app instance with all dependencies
- `testutil.Auth()`: Authentication helpers

### Testing Patterns

**For API handlers:**
```go
func TestHandler(t *testing.T) {
    server, db, jobManager := testutil.SetupTestServer(t)
    // Your test code here
}
```

**For database operations:**
```go
func TestStore(t *testing.T) {
    db := testutil.SetupTestDB(t)
    // Your test code here
}
```

**For integration tests:**
```go
func TestIntegration(t *testing.T) {
    app := testutil.SetupTestApp(t)
    // Your test code here
}
```

### Test Data

- Use `t.TempDir()` for temporary file operations
- Test data is automatically cleaned up after each test
- Mock external services using the provider system

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
2. Add new git tag locally and push it
3. This triggers the build and release
5. Modify release notes

---

Thank you for contributing to Mango-Go! Your contributions help make this project better for everyone.