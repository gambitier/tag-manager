# Tag Manager Makefile

# Binary name
BINARY_NAME=tag-manager

# Build directory
BUILD_DIR=build

# Go build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build clean run list test help

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application (builds first if needed)
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME) update $(ARGS)

# List discovered packages
list: build
	@echo "Discovering Go packages..."
	@$(BUILD_DIR)/$(BINARY_NAME) list

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Dependencies installed"

# Test the application
test:
	@echo "Running tests..."
	go test ./...
	@echo "Tests complete"

# Show help
help:
	@echo "Available targets:"
	@echo "  build   - Build the application"
	@echo "  run     - Build and run the application (update command)"
	@echo "  list    - Build and list discovered packages"
	@echo "  clean   - Clean build artifacts"
	@echo "  deps    - Install dependencies"
	@echo "  test    - Run tests"
	@echo "  help    - Show this help message"
	@echo ""
	@echo "Usage examples:"
	@echo "  make run                    # Interactive tag update"
	@echo "  make list                   # List discovered packages"
	@echo "  make run ARGS=\"\"            # Same as make run"
