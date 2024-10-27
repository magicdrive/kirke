# Makefile for kirke

# Variables
BUILD_DIR = $(CURDIR)/build
CMD_DIRS = $(wildcard cmd/*)
BINARY_NAME = kirke
VERSION = $(shell git describe --tags --always)
PLATFORMS := linux/amd64 windows/amd64 darwin/amd64

# Default target
.PHONY: all
all: help

# Build all artifacts
.PHONY: build
build: clean
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags "-X main.version=$(VERSION)"

# Build artifacts for all platforms and release
.PHONY: release
release: clean $(PLATFORMS)
	@echo "Release files are created in the $(BUILD_DIR) directory."

# Build each platform
$(PLATFORMS):
	@mkdir -p $(BUILD_DIR)
	GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@)) \
		 go build -o $(BUILD_DIR)/$(BINARY_NAME)-$(word 1,$(subst /, ,$@))-$(word 2,$(subst /, ,$@)) \
		 -ldflags "-X main.version=$(VERSION)" .

# Run go test for each directory
.PHONY: test
test:
	go test $(CURDIR)/...

# Run go test with verbose output and clear test cache
.PHONY: test-verbose
test-verbose:
	go clean -testcache
	go test -v $(CURDIR)/...

# Clean build artifacts
.PHONY: clean
clean:
	@rm -rf $(BUILD_DIR)

# Show help
.PHONY: help
help:
	@echo "Makefile commands:"
	@echo "  make build          - Build all artifacts"
	@echo "  make release        - Build artifacts for multiple platforms with version info"
	@echo "  make test           - Run go test"
	@echo "  make test-verbose   - Run go test -v with go clean -testcache"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make help           - Show this message"

