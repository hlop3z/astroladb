#!/bin/sh

# ==============================================================================
# CONFIGURATION
# ==============================================================================
GOOS_TARGET="windows"             # Target OS (windows, linux, darwin, etc.)
GOARCH_TARGET="amd64"             # Target architecture (amd64, arm64, etc.)
USE_UPX=1                         # 1 = compress binary with UPX, 0 = skip
OUT_FILE="alab.exe"               # Output binary name

# ==============================================================================
# BUILD
# ==============================================================================
echo "Building alab (OS=$GOOS_TARGET, ARCH=$GOARCH_TARGET)..."

CGO_ENABLED=0 GOOS=$GOOS_TARGET GOARCH=$GOARCH_TARGET \
go build -trimpath -ldflags="-s -w -buildid=" -o "$OUT_FILE" ./cmd/alab

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# Optional UPX compression
if [ "$USE_UPX" -eq 1 ] && command -v upx >/dev/null 2>&1; then
    upx --best --lzma "$OUT_FILE" || echo "Warning: UPX failed, skipping"
fi

echo "Build complete: $OUT_FILE"
