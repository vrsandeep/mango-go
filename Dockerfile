FROM golang:1.24.4-alpine AS builder

# Install build tools needed for CGo and SQLite.
RUN apk add --no-cache build-base sqlite-dev nodejs npm

# Set the working directory inside the container.
WORKDIR /app

# Copy dependency management files first to leverage Docker layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application's source code.
COPY . .

# Install esbuild for building the web assets.
RUN npm install -g esbuild

# Build the Go application.
# -o /mango-go: Specifies the output binary name.
# -ldflags "-w -s": Strips debugging information, reducing the binary size.
# CGO_ENABLED=1: Required for the go-sqlite3 driver.
# GIN_MODE=release: Sets Gin to production mode for better performance.
RUN make build

# Use alpine as the base image. It's lightweight but contains the necessary
# runtime libraries (like musl libc) that our binary depends on.
FROM alpine:latest

# Install runtime dependencies. ca-certificates is needed for making HTTPS requests.
# sqlite-libs provides the .so files needed by the compiled Go binary.
# curl is needed for health checks.
RUN apk add --no-cache ca-certificates curl

# Copy the compiled binary from the builder stage.
COPY --from=builder /app/build/mango-go /mango-go

# Create data directory and set initial ownership
RUN mkdir -p /app/data

# Expose the port the application will run on.
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=15s --start-period=30s --retries=3 CMD curl -fsS http://localhost:5000/api/health || exit 1

ENTRYPOINT ["/mango-go"]
