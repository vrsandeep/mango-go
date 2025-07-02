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

FROM alpine:latest

# Install runtime dependencies. ca-certificates is needed for making HTTPS requests (e.g., to MangaDex).
# sqlite is needed for the database driver.
RUN apk add --no-cache ca-certificates sqlite

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /mango-server /usr/local/bin/mango-server
COPY web ./web
COPY migrations ./migrations
COPY config.yml .

RUN mkdir /app/data && chown -R appuser:appgroup /app/data
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["mango-server"]