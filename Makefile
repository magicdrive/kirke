# Makefile for kirke

# Variables
BUILD_DIR = $(CURDIR)/build
CMD_DIRS = $(wildcard cmd/*)
BINARY_NAME = kirke
VERSION = $(shell git describe --tags --always)
LDFLAGS += -X "main.version=$(VERSION)"

PLATFORMS := linux/amd64 darwin/amd64
GO := GO111MODULE=on CGO_ENABLED=0 go

# Default target
.PHONY: all
all: help

# Build all artifacts
.PHONY: build
build: clean
	@mkdir -p $(BUILD_DIR)
	@$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags "$(LDFLAGS)"
	@chmod 755 $(BUILD_DIR)/$(BINARY_NAME)

# Build artifacts for all platforms and release
.PHONY: release-build
release-build: clean $(PLATFORMS)
	@echo "Release files are created in the $(BUILD_DIR) directory."

# Build each platform
$(PLATFORMS):
	@mkdir -p $(BUILD_DIR)
	GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@)) \
		 $(GO) build -o $(BUILD_DIR)/$(word 1,$(subst /, ,$@))-$(word 2,$(subst /, ,$@))/$(BINARY_NAME)\
		 -ldflags "$(LDFAGS)" .
	chmod 755 $(BUILD_DIR)/$(word 1,$(subst /, ,$@))-$(word 2,$(subst /, ,$@))/$(BINARY_NAME)

# Run go test for each directory
.PHONY: test
test:
	@$(GO) test $(CURDIR)/...

# Run go test with verbose output and clear test cache
.PHONY: test-verbose
test-verbose:
	@$(GO) clean -testcache
	@$(GO) test -v $(CURDIR)/...

# Install application. Use `go install`
.PHONY: install
install:
	@echo "Installing kirke..."
	@$(GO) install -ldflags "$(LDFLAGS)"

# Clean build artifacts
.PHONY: clean
clean:
	@rm -rf $(BUILD_DIR)

# Publish to github.com
.PHONY: publish
publish:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is not set. Please set it and try again."; \
		exit 1; \
	fi
	git tag $(version)
	git push origin $(version)


# Show help
.PHONY: help
help:
	@echo "Makefile commands:"
	@echo "  make build                - Build all artifacts"
	@echo "  make release-build        - Build artifacts for multiple platforms with version info"
	@echo "  make install              - Install application. Use `go install`"
	@echo "  make test                 - Run go test"
	@echo "  make test-verbose         - Run go test -v with go clean -testcache"
	@echo "  make clean                - Remove build artifacts"
	@echo "  make publish vesion=<tag> - Publish to github.com"
	@echo "  make help                 - Show this message"

