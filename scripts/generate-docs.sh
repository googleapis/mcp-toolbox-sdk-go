#!/bin/bash
set -e

# Usage: ./scripts/generate-docs.sh <output_directory> <version_tag>
OUTPUT_DIR=$1
VERSION=$2

if [ -z "$OUTPUT_DIR" ] || [ -z "$VERSION" ]; then
  echo "Usage: ./scripts/generate-docs.sh <output_directory> <version_tag>"
  exit 1
fi

echo "Generating documentation for version $VERSION..."

go install github.com/ankit-shukla/doc2go@latest

# Generate static HTML for core, tbadk, and tbgenkit
# -out: destination directory
# -internal: include internal packages if needed
# -pkg-main: identifies the root module
doc2go -out "$OUTPUT_DIR/$VERSION" \
       -internal \
       -pkg-main "github.com/googleapis/mcp-toolbox-sdk-go" \
       ./...

echo "Documentation generated in $OUTPUT_DIR/$VERSION"
