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
# RUN CGO_ENABLED=0 go build -tags "sqlite_omit_load_extension" -ldflags="-w -s" -o /mango-server ./cmd/mango-server

FROM scratch
# FROM alpine:latest

COPY --from=builder /mango-server /mango-server

# Copy the CA certificates, which are necessary for making HTTPS requests
# (e.g., to the MangaDex API).
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

CMD ["/mango-server"]
