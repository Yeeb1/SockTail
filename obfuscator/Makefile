# Build variables
BINARY_NAME=socks5-proxy
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

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
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Build for all platforms
.PHONY: build-all
build-all: clean deps build-linux build-windows build-darwin

# Build for Linux (amd64)
.PHONY: build-linux
build-linux:
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .

# Build for Linux (arm64)
.PHONY: build-linux-arm64
build-linux-arm64:
	mkdir -p dist
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .

# Build for Windows (amd64)
.PHONY: build-windows
build-windows:
	mkdir -p dist
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .

# Build for macOS (amd64)
.PHONY: build-darwin
build-darwin:
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .

# Build for macOS (arm64 - Apple Silicon)
.PHONY: build-darwin-arm64
build-darwin-arm64:
	mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .

# Create release archives
.PHONY: release
release: build-all
	cd dist && \
	tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
	tar -czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
	tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
	tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
	zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe

# Run the proxy (for development)
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

# Test
.PHONY: test
test:
	go test -v ./...

# Obfuscate auth key
.PHONY: obfuscate-key
obfuscate-key:
	@echo "Usage: make obfuscate-key KEY=your-tailscale-auth-key"
	@if [ -z "$(KEY)" ]; then \
		echo "Error: Please provide KEY parameter"; \
		echo "Example: make obfuscate-key KEY=tskey-auth-xxxxx-your-key"; \
		exit 1; \
	fi
	go run obfuscate_key.go $(KEY)

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build for current platform"
	@echo "  build-all      - Build for all platforms"
	@echo "  build-linux    - Build for Linux amd64"
	@echo "  build-windows  - Build for Windows amd64"
	@echo "  build-darwin   - Build for macOS amd64"
	@echo "  release        - Build all and create release archives"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  obfuscate-key  - Obfuscate auth key for embedding"
	@echo "  fmt            - Format code"
	@echo "  test           - Run tests"
	@echo "  run            - Build and run"
	@echo "  help           - Show this help"
