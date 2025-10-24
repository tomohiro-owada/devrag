#!/bin/bash

set -e

# Get version from tag or environment variable
VERSION=${VERSION:-$(git describe --tags --always 2>/dev/null || echo "dev")}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "================================="
echo "Building markdown-vector-mcp"
echo "================================="
echo "Version: $VERSION"
echo "Build Time: $BUILD_TIME"
echo "Git Commit: $GIT_COMMIT"
echo "================================="
echo ""

# Output directory
DIST_DIR="dist"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Build flags
LDFLAGS="-s -w -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT"

# Function to build for a platform
build_platform() {
  local goos=$1
  local goarch=$2
  local ext=$3

  local output_name="markdown-vector-mcp-${goos}-${goarch}${ext}"
  echo "Building for ${goos}/${goarch}..."

  CGO_ENABLED=1 GOOS=$goos GOARCH=$goarch go build \
    -ldflags="$LDFLAGS" \
    -o "${DIST_DIR}/${output_name}" \
    cmd/main.go

  if [ $? -eq 0 ]; then
    echo "✓ Built ${output_name}"

    # Create archive
    if [ "$goos" = "windows" ]; then
      (cd "$DIST_DIR" && zip "${output_name}.zip" "$output_name")
      shasum -a 256 "${DIST_DIR}/${output_name}.zip" > "${DIST_DIR}/${output_name}.zip.sha256"
    else
      tar -czf "${DIST_DIR}/${output_name}.tar.gz" -C "$DIST_DIR" "$output_name"
      shasum -a 256 "${DIST_DIR}/${output_name}.tar.gz" > "${DIST_DIR}/${output_name}.tar.gz.sha256"
    fi
    echo "✓ Created archive for ${output_name}"
  else
    echo "✗ Failed to build ${output_name}"
    return 1
  fi

  echo ""
}

# Detect current OS
CURRENT_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
CURRENT_ARCH=$(uname -m)

if [ "$CURRENT_ARCH" = "x86_64" ]; then
  CURRENT_ARCH="amd64"
fi

echo "Current platform: ${CURRENT_OS}/${CURRENT_ARCH}"
echo ""

# Build for current platform first
if [ "$CURRENT_OS" = "darwin" ]; then
  echo "=== macOS Builds ==="
  build_platform "darwin" "amd64" ""
  build_platform "darwin" "arm64" ""

  # For Linux builds on macOS, you need cross-compilation tools
  echo ""
  echo "=== Linux Builds (requires cross-compilation tools) ==="
  echo "Note: Cross-compiling CGO to Linux requires appropriate toolchains"
  echo "Skipping Linux builds from macOS. Use GitHub Actions for full cross-platform builds."

elif [ "$CURRENT_OS" = "linux" ]; then
  echo "=== Linux Builds ==="
  build_platform "linux" "amd64" ""

  # For ARM64, you might need: apt-get install gcc-aarch64-linux-gnu
  if command -v aarch64-linux-gnu-gcc &> /dev/null; then
    CC=aarch64-linux-gnu-gcc build_platform "linux" "arm64" ""
  else
    echo "Skipping linux/arm64 (install gcc-aarch64-linux-gnu for cross-compilation)"
  fi

  echo ""
  echo "Note: Cross-compiling CGO to macOS/Windows from Linux requires appropriate toolchains"
  echo "Use GitHub Actions for full cross-platform builds."

elif [ "$CURRENT_OS" = "mingw" ] || [ "$CURRENT_OS" = "msys" ]; then
  echo "=== Windows Builds ==="
  build_platform "windows" "amd64" ".exe"

  echo ""
  echo "Note: Cross-compiling CGO to Linux/macOS from Windows requires appropriate toolchains"
  echo "Use GitHub Actions for full cross-platform builds."
fi

echo ""
echo "================================="
echo "Build Summary"
echo "================================="
echo "Output directory: $DIST_DIR"
echo ""
echo "Files created:"
ls -lh "$DIST_DIR"
echo ""
echo "Total size:"
du -sh "$DIST_DIR"
echo ""
echo "✓ Build complete!"
echo ""
echo "To create a release:"
echo "  1. Push a tag: git tag v${VERSION} && git push origin v${VERSION}"
echo "  2. GitHub Actions will automatically build for all platforms"
echo "  3. Binaries will be attached to the release"
