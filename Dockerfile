FROM golang:1.24.4-alpine AS builder

# Install build tools needed for CGo and SQLite.
RUN apk add --no-cache build-base sqlite-dev nodejs npm git make

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
RUN CGO_ENABLED=1 GIN_MODE=release make build

# Use alpine as the base image. It's lightweight but contains the necessary
# runtime libraries (like musl libc) that our binary depends on.
FROM alpine:latest

# Create new user and group
RUN addgroup -S mango && adduser -S mango -G mango

WORKDIR /app

# Install runtime dependencies. ca-certificates is needed for making HTTPS requests.
# sqlite-libs provides the .so files needed by the compiled Go binary.
RUN apk add --no-cache ca-certificates sqlite-libs

RUN mkdir /app/data && chown -R mango:mango /app/data
RUN chown -R mango:mango /app

# Change to the new user and group
USER mango

# Copy the compiled binary from the builder stage.
COPY --from=builder /app/mango-go /mango-go

# Expose the port the application will run on.
EXPOSE 8080

ENTRYPOINT ["/mango-go"]
