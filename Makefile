# Makefile for kirke

# Variables
BUILD_DIR = $(CURDIR)/build
CMD_DIRS = $(wildcard cmd/*)

# Default target
.PHONY: all
all: help

# Build all artifacts
.PHONY: build
build: clean 
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/kirke

# Run go test for each directories
.PHONY: test
test:
	go test $(CURDIR)/...

# Run go test for each directories
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
	@echo "  make build   - Build all artifacts"
	@echo "  make test    - Run go test"
	@echo "  make test    - Run go test -v with git clean -testcache"
	@echo "  make clean   - Remove build artifacts"
	@echo "  make run     - Run a specific application"
	@echo "  make help    - Show this message"
