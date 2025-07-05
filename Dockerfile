FROM golang:1.24.4-alpine AS builder

# Install build tools needed for CGo and SQLite.
RUN apk add --no-cache build-base sqlite-dev

# Set the working directory inside the container.
WORKDIR /app

# Copy dependency management files first to leverage Docker layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application's source code.
COPY . .

# Build the Go application.
# -o /mango-server: Specifies the output binary name.
# -ldflags "-w -s": Strips debugging information, reducing the binary size.
# CGO_ENABLED=1: Required for the go-sqlite3 driver.
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o /mango-server ./cmd/mango-server

# Use alpine as the base image. It's lightweight but contains the necessary
# runtime libraries (like musl libc) that our binary depends on.
FROM alpine:latest

# Install runtime dependencies. ca-certificates is needed for making HTTPS requests.
# sqlite-libs provides the .so files needed by the compiled Go binary.
RUN apk add --no-cache ca-certificates sqlite-libs

# Copy the compiled binary from the builder stage.
COPY --from=builder /mango-server /mango-server

# Expose the port the application will run on.
EXPOSE 8080

# Set the entrypoint for the container.
# Note: Since we are not using a non-root user in this simple setup,
# the binary will run as root. For enhanced security, a non-root user could be added.
ENTRYPOINT ["/mango-server"]