CSS_FILES = $(wildcard internal/assets/web/static/css/*.css)
CSS_EXT_FILES = $(wildcard internal/assets/web/static/css/ext/*.css)
JS_FILES = $(wildcard internal/assets/web/static/js/*.js)
IMAGE_FILES = $(wildcard internal/assets/web/static/images/*)
PRETTIER_DIRS = internal/assets/web/static/css internal/assets/web/static/js

# Output directories for the minified bundles.
CSS_OUT = internal/assets/web/dist/css
CSS_EXT_OUT = internal/assets/web/dist/css/ext
JS_OUT = internal/assets/web/dist/js
IMAGE_OUT = internal/assets/web/dist/images

BINARY_NAME=mango-go
BUILD_DIR=./build

# go-fitz (PDF): on Alpine/musl (e.g. Docker), set GO_BUILD_TAGS=musl so bundled MuPDF
# archives match libc. Leave empty on glibc (Ubuntu, macOS).
GO_BUILD_TAGS ?=
GO_TAGS_FLAG := $(if $(strip $(GO_BUILD_TAGS)),-tags=$(GO_BUILD_TAGS),)

.PHONY: all assets clean build run download-go-deps prettify format format-check

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
	@CGO_ENABLED=1 GIN_MODE=release go build $(GO_TAGS_FLAG) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Production binary created at '$(pwd)/$(BUILD_DIR)/$(BINARY_NAME)'."

download-go-deps:
	@go mod download
	@go mod tidy

# Creates minified bundles for production.
assets:
	@echo "📦 Bundling and minifying assets for production..."
	@mkdir -p $(CSS_OUT)
	@mkdir -p $(CSS_EXT_OUT)
	@esbuild $(CSS_FILES) --bundle --minify --outdir=$(CSS_OUT)
	@cp $(CSS_EXT_FILES) $(CSS_EXT_OUT)
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

# Format Go, then prettify static CSS/JS (Prettier: .prettierrc / .prettierignore)
prettify:
	@echo "🎨 Formatting Go..."
	@go fmt ./...
	@echo "🎨 Prettifying CSS and JS..."
	@npx --yes prettier --write $(PRETTIER_DIRS)
	@echo "✅ Prettify complete."

format: prettify

format-check:
	@echo "🔍 Checking Go formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Go files need formatting; run make prettify" >&2 && gofmt -l . >&2 && exit 1)
	@echo "🔍 Checking CSS/JS formatting..."
	@npx --yes prettier --check $(PRETTIER_DIRS)
	@echo "✅ Format check passed."
