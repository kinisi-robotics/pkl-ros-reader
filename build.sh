#!/bin/bash
# Build script for pkl-ros-reader external reader binary

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_BINARY="$SCRIPT_DIR/../pkl-ros-reader"

echo "Building pkl-ros-reader..."

# Ensure Go is available
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Build the binary
cd "$SCRIPT_DIR"
go build -o "$OUTPUT_BINARY" .

echo "Successfully built: $OUTPUT_BINARY"
ls -lh "$OUTPUT_BINARY"
