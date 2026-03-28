.PHONY: help build all install clean test lint run release vet fmt

# Variables
VERSION=0.1.0
BINARY_NAME=lazy-cli
LINUX_AMD64=$(BINARY_NAME)-linux-amd64
LINUX_ARM64=$(BINARY_NAME)-linux-arm64
MACOS_AMD64=$(BINARY_NAME)-macos-amd64
MACOS_ARM64=$(BINARY_NAME)-macos-arm64

# Default target
help:
	@echo "lazy-cli Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build      Build for current platform"
	@echo "  make all        Build for all platforms (Linux/macOS, AMD64/ARM64)"
	@echo "  make install    Build and install to ~/go/bin/"
	@echo "  make clean      Remove built binaries"
	@echo "  make test       Run tests"
	@echo "  make vet        Run go vet"
	@echo "  make fmt        Format code"
	@echo "  make lint       Run vet + fmt"
	@echo "  make run        Build and run (current platform)"
	@echo "  make release    Tag, build all, and upload to GitHub Releases"
	@echo ""

# Build for current platform
build:
	go build -o $(BINARY_NAME) .
	@echo "✓ Built $(BINARY_NAME)"

# Build for all platforms
all: clean
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build -o $(LINUX_AMD64) .
	GOOS=linux GOARCH=arm64 go build -o $(LINUX_ARM64) .
	GOOS=darwin GOARCH=amd64 go build -o $(MACOS_AMD64) .
	GOOS=darwin GOARCH=arm64 go build -o $(MACOS_ARM64) .
	@echo "✓ Built all platforms:"
	@ls -lh $(LINUX_AMD64) $(LINUX_ARM64) $(MACOS_AMD64) $(MACOS_ARM64)

# Install to ~/go/bin
install: build
	go install .
	@echo "✓ Installed to ~/go/bin/$(BINARY_NAME)"
	@echo "  Make sure ~/go/bin is in your PATH"

# Clean up binaries
clean:
	rm -f $(BINARY_NAME) $(LINUX_AMD64) $(LINUX_ARM64) $(MACOS_AMD64) $(MACOS_ARM64)
	@echo "✓ Cleaned"

# Run go test
test:
	go test ./...
	@echo "✓ Tests passed"

# Run go vet
vet:
	go vet ./...
	@echo "✓ Vet passed"

# Format code
fmt:
	go fmt ./...
	@echo "✓ Code formatted"

# Lint (vet + fmt)
lint: vet fmt
	@echo "✓ Lint complete"

# Build and run
run: build
	./$(BINARY_NAME)

# Build, tag, and release to GitHub
release: all
	@echo "Creating release v$(VERSION)..."
	git tag v$(VERSION)
	git push origin v$(VERSION)
	gh release create v$(VERSION) \
		$(LINUX_AMD64) \
		$(LINUX_ARM64) \
		$(MACOS_AMD64) \
		$(MACOS_ARM64) \
		--title "lazy-cli v$(VERSION)" \
		--notes "Releases for Linux (AMD64/ARM64) and macOS (Intel/Apple Silicon)"
	@echo "✓ Released v$(VERSION)"
