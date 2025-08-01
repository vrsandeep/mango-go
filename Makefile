# Define the source files. This makes it easy to add new files in the future.
CSS_FILES = $(wildcard internal/assets/web/static/css/*.css)
JS_FILES = $(wildcard internal/assets/web/static/js/*.js)
IMAGE_FILES = $(wildcard internal/assets/web/static/images/*)

# Define the output directories for the minified bundles.
CSS_OUT = internal/assets/web/dist/css
JS_OUT = internal/assets/web/dist/js
IMAGE_OUT = internal/assets/web/dist/images

BINARY_NAME=mango-go
BUILD_DIR=./build

.PHONY: all assets clean build run release-assets build-linux-amd64 build-macos-amd64 build-macos-arm64 build-linux-amd64-docker

all: build

# 'run' is the command for local development.
# It creates un-minified bundles for easier debugging and runs the app.
run: assets
	@echo "🚀 Starting development server..."
	@go run .

# 'build' is the command for production releases.
# It creates minified bundles and builds the binary with the 'prod' tag.
build: download-go-deps assets
	@echo "📦 Building production binary..."
	@CGO_ENABLED=1 GIN_MODE=release go build -ldflags="-w -s" -o  $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Production binary created at '$(pwd)/$(BUILD_DIR)/$(BINARY_NAME)'."

download-go-deps:
	@go mod download
	@go mod tidy

# Creates minified bundles for production.
assets:
	@echo "📦 Bundling and minifying assets for production..."
	@mkdir -p $(CSS_OUT)
	@esbuild $(CSS_FILES) --bundle --minify --outdir=$(CSS_OUT)
	@esbuild $(JS_FILES) --bundle --minify --outdir=$(JS_OUT)
	@mkdir -p $(IMAGE_OUT)
	@cp $(IMAGE_FILES) $(IMAGE_OUT)
	@echo "✅ Production assets built successfully."

# The 'clean' target removes generated files.
clean:
	@echo "🧹 Cleaning up build artifacts..."
	@rm -rf ./internal/assets/web/dist
	@rm -rf ./$(BUILD_DIR)
	@echo "✅ Cleanup complete."


# This target builds all binaries needed for a release.
release-assets: build-linux-amd64-docker build-macos-amd64 build-macos-arm64

# Use Docker for Linux cross-compilation to avoid CGO issues
build-linux-amd64-docker: assets
	@echo "📦 Building for linux/amd64 using Docker..."
	@mkdir -p $(BUILD_DIR)
	@docker run --rm --platform linux/amd64 -v $(PWD):/app -w /app golang:1.24.4 \
		bash -c "GOOS=linux GIN_MODE=release GOARCH=amd64 CGO_ENABLED=1 CC=gcc go build -ldflags='-w -s' -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ."

# Original target kept for reference but will fail on macOS
build-linux-amd64: assets
	@echo "📦 Building for linux/amd64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GIN_MODE=release GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .

build-macos-amd64: assets
	@echo "📦 Building for darwin/amd64 (Intel)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GIN_MODE=release GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-macos-amd64 .

build-macos-arm64: assets
	@echo "📦 Building for darwin/arm64 (Apple Silicon)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GIN_MODE=release GOARCH=arm64 CGO_ENABLED=1 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-macos-arm64 .
