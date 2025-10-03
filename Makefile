BINARY_NAME = SockTail
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME  = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
AUTH_KEY    ?=
CONTROL_URL ?=
LDFLAGS     = -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)$(if $(AUTH_KEY), -X main.buildTimeObfuscatedKey=$(shell go run obfuscator/obfuscate_key_hex.go '$(AUTH_KEY)'))$(if $(CONTROL_URL), -X main.buildTimeControlURL=$(CONTROL_URL))"
BUILD_ENV   = CGO_ENABLED=0

PLATFORMS = \
	linux/amd64 \
	linux/arm64 \
	windows/amd64 \
	darwin/amd64 \
	darwin/arm64

# Default target
.PHONY: all
all: clean build

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf dist/
	go clean

# Build for current platform
.PHONY: build
build:
	$(BUILD_ENV) go build $(LDFLAGS) -o $(BINARY_NAME) .

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Build for all platforms
.PHONY: build-all
build-all: clean deps
	@$(foreach platform, $(PLATFORMS), \
		$(MAKE) build-platform PLATFORM=$(platform);)

# Build for a specific platform
.PHONY: build-platform
build-platform:
	@mkdir -p dist
	@os=$(word 1, $(subst /, ,$(PLATFORM))); \
	arch=$(word 2, $(subst /, ,$(PLATFORM))); \
	ext=$$( [ $$os = windows ] && echo .exe || echo ); \
	out=dist/$(BINARY_NAME)-$$os-$$arch$$ext; \
	echo "Building $$os/$$arch -> $$out"; \
	GOOS=$$os GOARCH=$$arch $(BUILD_ENV) go build $(LDFLAGS) -o $$out .; \
	[ -x "$$(command -v upx)" ] && upx --best --lzma $$out || true

# Create release archives
.PHONY: release
release: build-all
	cd dist && \
	tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
	tar -czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
	tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
	tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
	zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe

# Run the proxy
.PHONY: run
run: build
	./$(BINARY_NAME)

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	golangci-lint run

# Run tests
.PHONY: test
test:
	go test -v ./...

# Build with auth key
.PHONY: build-with-key
build-with-key:
	@if [ -z "$(AUTH_KEY)" ]; then \
		echo "Error: AUTH_KEY is required. Usage: make build-with-key AUTH_KEY=tskey-auth-xxxxx [CONTROL_URL=https://headscale.example.com]"; \
		exit 1; \
	fi
	$(BUILD_ENV) go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build all platforms with auth key
.PHONY: build-all-with-key
build-all-with-key: clean deps
	@if [ -z "$(AUTH_KEY)" ]; then \
		echo "Error: AUTH_KEY is required. Usage: make build-all-with-key AUTH_KEY=tskey-auth-xxxxx [CONTROL_URL=https://headscale.example.com]"; \
		exit 1; \
	fi
	@$(foreach platform, $(PLATFORMS), \
		$(MAKE) build-platform PLATFORM=$(platform) AUTH_KEY="$(AUTH_KEY)" CONTROL_URL="$(CONTROL_URL)";)

# Build with both auth key and control URL
.PHONY: build-with-config
build-with-config:
	@if [ -z "$(AUTH_KEY)" ] || [ -z "$(CONTROL_URL)" ]; then \
		echo "Error: Both AUTH_KEY and CONTROL_URL are required."; \
		echo "Usage: make build-with-config AUTH_KEY=tskey-auth-xxxxx CONTROL_URL=https://headscale.example.com"; \
		exit 1; \
	fi
	$(BUILD_ENV) go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build all platforms with both auth key and control URL
.PHONY: build-all-with-config
build-all-with-config: clean deps
	@if [ -z "$(AUTH_KEY)" ] || [ -z "$(CONTROL_URL)" ]; then \
		echo "Error: Both AUTH_KEY and CONTROL_URL are required."; \
		echo "Usage: make build-all-with-config AUTH_KEY=tskey-auth-xxxxx CONTROL_URL=https://headscale.example.com"; \
		exit 1; \
	fi
	@$(foreach platform, $(PLATFORMS), \
		$(MAKE) build-platform PLATFORM=$(platform) AUTH_KEY="$(AUTH_KEY)" CONTROL_URL="$(CONTROL_URL)";)

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build                     - Build for current platform (embedded key)"
	@echo "  build-with-key            - Build with custom auth key (optional control URL)"
	@echo "  build-with-config         - Build with both auth key and control URL"
	@echo "  build-all                 - Build for all platforms (embedded key)"
	@echo "  build-all-with-key        - Build for all platforms with custom auth key (optional control URL)"
	@echo "  build-all-with-config     - Build for all platforms with auth key and control URL"
	@echo "  release                   - Build and archive binaries"
	@echo "  clean                     - Clean build artifacts"
	@echo "  deps                      - Download Go dependencies"
	@echo "  fmt                       - Format code"
	@echo "  lint                      - Lint code with golangci-lint"
	@echo "  test                      - Run tests"
	@echo "  run                       - Build and run locally"
	@echo "  help                      - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make build-with-key AUTH_KEY=tskey-auth-client-xxxxx-your-key"
	@echo "  make build-with-key AUTH_KEY=tskey-auth-client-xxxxx-your-key CONTROL_URL=https://headscale.example.com"
	@echo "  make build-with-config AUTH_KEY=tskey-auth-client-xxxxx-your-key CONTROL_URL=https://headscale.example.com"
	@echo "  make build-all-with-config AUTH_KEY=tskey-auth-client-xxxxx-your-key CONTROL_URL=https://headscale.example.com"
