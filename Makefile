CSS_FILES = $(wildcard internal/assets/web/static/css/*.css)
JS_FILES = $(wildcard internal/assets/web/static/js/*.js)
IMAGE_FILES = $(wildcard internal/assets/web/static/images/*)

# Output directories for the minified bundles.
CSS_OUT = internal/assets/web/dist/css
JS_OUT = internal/assets/web/dist/js
IMAGE_OUT = internal/assets/web/dist/images

BINARY_NAME=mango-go
BUILD_DIR=./build

.PHONY: all assets clean build run download-go-deps format format-check install-prettier

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
	@CGO_ENABLED=1 GIN_MODE=release go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) .
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

# Install Prettier dependencies
install-prettier:
	@echo "📦 Installing Prettier..."
	@npm install
	@echo "✅ Prettier installed successfully."

# Format CSS and JS files with Prettier
format: install-prettier
	@echo "🎨 Formatting CSS and JS files..."
	@npm run format
	@echo "✅ Files formatted successfully."

# Check if files are formatted correctly
format-check: install-prettier
	@echo "🔍 Checking file formatting..."
	@npm run format:check
