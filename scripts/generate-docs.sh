#!/bin/bash
set -e

OUTPUT_DIR=$1
VERSION=$2

if [ -z "$OUTPUT_DIR" ] || [ -z "$VERSION" ]; then
  echo "Usage: ./scripts/generate-docs.sh <output_directory> <version_tag>"
  exit 1
fi

echo "Generating documentation for version $VERSION..."

go install golang.org/x/pkgsite/cmd/pkgsite@latest

pkgsite -http=:8080 &
PKGSITE_PID=$!

sleep 15

wget -nv --recursive --page-requisites --convert-links \
     --restrict-file-names=windows --no-parent \
     -nH --adjust-extension \
     --reject-regex '(\?|&)(tab=versions|tab=importedby)' \
     -P "$OUTPUT_DIR/$VERSION" \
     http://localhost:8080/github.com/googleapis/mcp-toolbox-sdk-go || true

VERSION_ROOT="$OUTPUT_DIR/$VERSION"
TEMP_PATH="$VERSION_ROOT/github.com/googleapis/mcp-toolbox-sdk-go"

if [ -d "$TEMP_PATH" ]; then
    cp -r "$TEMP_PATH/"* "$VERSION_ROOT/"
    mv "$VERSION_ROOT/github.com/googleapis/mcp-toolbox-sdk-go.html" "$VERSION_ROOT/index.html"
    rm -rf "$VERSION_ROOT/github.com"

kill $PKGSITE_PID

echo "Official Go documentation captured in $OUTPUT_DIR/$VERSION"
