BINARY_NAME = SockTail
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME  = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS     = -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"
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

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build           - Build for current platform"
	@echo "  build-all       - Build for all platforms"
	@echo "  release         - Build and archive binaries"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Download Go dependencies"
	@echo "  fmt             - Format code"
	@echo "  lint            - Lint code with golangci-lint"
	@echo "  test            - Run tests"
	@echo "  run             - Build and run locally"
	@echo "  help            - Show this help"
