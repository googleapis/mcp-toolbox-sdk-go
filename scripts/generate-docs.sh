#!/bin/bash
set -e

OUTPUT_DIR=$1
VERSION=$2

if [ -z "$OUTPUT_DIR" ] || [ -z "$VERSION" ]; then
  echo "Usage: ./scripts/generate-docs.sh <output_directory> <version_tag>"
  exit 1
fi

go install golang.org/x/pkgsite/cmd/pkgsite@latest

pkgsite -http=:8080 &
PKGSITE_PID=$!

sleep 10

mkdir -p "$OUTPUT_DIR/$VERSION"
wget --recursive --page-requisites --html-extension --convert-links \
     --restrict-file-names=windows --no-parent \
     -P "$OUTPUT_DIR/$VERSION" \
     http://localhost:8080/github.com/googleapis/mcp-toolbox-sdk-go

# 4. Gracefully shut down the official server
kill $PKGSITE_PID

echo "Official Go documentation captured in $OUTPUT_DIR/$VERSION"
