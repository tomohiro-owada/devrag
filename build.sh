#!/bin/bash

set -e

echo "Building devrag..."

# Output directory
mkdir -p bin

# Build flags
LDFLAGS="-s -w"
TAGS=""

# Version info
VERSION=${VERSION:-"1.0.0"}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS="$LDFLAGS -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT"

echo "Version: $VERSION"
echo "Build Time: $BUILD_TIME"
echo "Git Commit: $GIT_COMMIT"
echo ""

# Note: CGO is required for sqlite-vec, so cross-compilation is limited
# Build for current platform first
echo "Building for current platform..."
CGO_ENABLED=1 go build -ldflags="$LDFLAGS" -o bin/devrag cmd/main.go

# macOS (Apple Silicon) - only on macOS arm64
if [[ "$OSTYPE" == "darwin"* ]] && [[ "$(uname -m)" == "arm64" ]]; then
  echo "Building for macOS (arm64)..."
  CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" \
    -o bin/devrag-darwin-arm64 cmd/main.go

  echo "Building for macOS (amd64)..."
  CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" \
    -o bin/devrag-darwin-amd64 cmd/main.go
fi

# Note: For Windows and Linux builds from macOS, you would need:
# - Windows: Install mingw-w64 cross-compiler
# - Linux: Install appropriate cross-compilation tools
# For now, we'll skip cross-platform CGO builds

echo ""
echo "Build complete!"
echo "Binaries:"
ls -lh bin/

echo ""
echo "Total size:"
du -sh bin/
