# Define the source files. This makes it easy to add new files in the future.
CSS_FILES = $(wildcard internal/assets/web/static/css/*.css)
JS_FILES = $(wildcard internal/assets/web/static/js/*.js)
IMAGE_FILES = $(wildcard internal/assets/web/static/images/*)

# Define the output directories for the minified bundles.
CSS_OUT = internal/assets/web/dist/css
JS_OUT = internal/assets/web/dist/js
IMAGE_OUT = internal/assets/web/dist/images

.PHONY: all assets clean build run

all: build

# 'run' is the new command for local development.
# It creates un-minified bundles for easier debugging and runs the app.
run: assets
	@echo "🚀 Starting development server..."
	@go run .

# 'build' is the command for production releases.
# It creates minified bundles and builds the binary with the 'prod' tag.
build: download-go-deps assets
	@echo "📦 Building production binary..."
	@CGO_ENABLED=1 go build -ldflags="-w -s" -o mango-go .
	@echo "✅ Production binary 'mango-go' created."

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
	@rm -rf ./internal/assets/web/static/bundled
	@rm -f ./mango-go
	@echo "✅ Cleanup complete."
