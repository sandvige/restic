# Build configuration
CGO_ENABLED := 1
GO := go
PACKAGE := github.com/restic/restic/pkg/cresticlib

# Output configuration
LIB_NAME := librestic
BUILD_DIR := build

# Platform detection
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Default platform
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Automatic platform detection for shared libraries
ifeq ($(GOOS),linux)
	SHARED_EXT := .so
endif
ifeq ($(GOOS),darwin)
	SHARED_EXT := .dylib
endif

# Cross-compilation targets
TARGETS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.PHONY: all clean static shared cross-compile help

# Default target
all: static

# Build static library for current platform
static: $(BUILD_DIR)/$(LIB_NAME)-$(GOOS)-$(GOARCH).a

$(BUILD_DIR)/$(LIB_NAME)-$(GOOS)-$(GOARCH).a: cbridge.go
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -buildmode=c-archive -trimpath -ldflags="-s -w" -tags="disable_grpc_modules" -o $@ .
	@mv $(BUILD_DIR)/$(LIB_NAME)-$(GOOS)-$(GOARCH).h $(BUILD_DIR)/ 2>/dev/null || true

# Build shared library for current platform (Linux/macOS only)
shared: $(BUILD_DIR)/$(LIB_NAME)-$(GOOS)-$(GOARCH)$(SHARED_EXT)

$(BUILD_DIR)/$(LIB_NAME)-$(GOOS)-$(GOARCH).so: cbridge.go
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -buildmode=c-shared -trimpath -ldflags="-s -w" -tags="disable_grpc_modules" -o $@ .

$(BUILD_DIR)/$(LIB_NAME)-$(GOOS)-$(GOARCH).dylib: cbridge.go
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -buildmode=c-shared -trimpath -ldflags="-s -w" -tags="disable_grpc_modules" -o $@ .

# Cross-compile for all platforms
cross-compile: $(addprefix cross-,$(TARGETS))

cross-%: 
	$(eval OS_ARCH := $(subst /, ,$*))
	$(eval TARGET_OS := $(word 1,$(OS_ARCH)))
	$(eval TARGET_ARCH := $(word 2,$(OS_ARCH)))
	@echo "Building for $(TARGET_OS)/$(TARGET_ARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH) $(GO) build -buildmode=c-archive -trimpath -ldflags="-s -w" -tags="disable_grpc_modules" -o $(BUILD_DIR)/$(LIB_NAME)-$(TARGET_OS)-$(TARGET_ARCH).a .
	@mv $(BUILD_DIR)/$(LIB_NAME)-$(TARGET_OS)-$(TARGET_ARCH).h $(BUILD_DIR)/librestic-$(TARGET_OS)-$(TARGET_ARCH).h 2>/dev/null || true

# Development build (for testing)
dev: $(BUILD_DIR)/$(LIB_NAME).a

$(BUILD_DIR)/$(LIB_NAME).a: cbridge.go
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -buildmode=c-archive -trimpath -ldflags="-s -w" -tags="disable_grpc_modules" -o $@ .
	@mv $(BUILD_DIR)/$(LIB_NAME).h $(BUILD_DIR)/ 2>/dev/null || true

# Install headers and libraries to system directories (optional)
install: static
	@echo "Installing headers and libraries..."
	sudo cp resticlib.h /usr/local/include/
	sudo cp ResticLib.hpp /usr/local/include/
	sudo cp $(BUILD_DIR)/$(LIB_NAME)-$(GOOS)-$(GOARCH).a /usr/local/lib/$(LIB_NAME).a
	@echo "Installation complete"

# Test build (compile but don't link)
test-build:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -buildmode=c-archive -trimpath -ldflags="-s -w" -tags="disable_grpc_modules" -o /dev/null .

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f *.a *.h *.so *.dylib *.dll

# Show help
help:
	@echo "ResticLib C/C++ Build System"
	@echo ""
	@echo "All builds use optimized release flags: -trimpath -ldflags='-s -w' -tags='disable_grpc_modules'"
	@echo ""
	@echo "Targets:"
	@echo "  all           - Build static library for current platform (default)"
	@echo "  static        - Build static library for current platform"
	@echo "  shared        - Build shared library for current platform"
	@echo "  cross-compile - Build static libraries for all supported platforms"
	@echo "  dev           - Development build (librestic.a)"
	@echo "  install       - Install headers and library to system directories"
	@echo "  test-build    - Test compilation without creating artifacts"
	@echo "  clean         - Remove all build artifacts"
	@echo "  help          - Show this help message"
	@echo ""
	@echo "Cross-compilation targets:"
	@echo "  $(TARGETS)"
	@echo ""
	@echo "Environment variables:"
	@echo "  CGO_ENABLED   - Enable/disable CGO (default: 1)"
	@echo "  GO            - Go compiler command (default: go)"