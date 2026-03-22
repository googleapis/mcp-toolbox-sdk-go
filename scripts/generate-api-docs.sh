#!/bin/bash
set -e

export PATH=$PATH:$(go env GOPATH)/bin
VERSION=$1
BASE_URL=$2

if [ -z "$VERSION" ] || [ -z "$BASE_URL" ]; then
  echo "Usage: ./scripts/generate-api-docs.sh <version> <base_url>"
  exit 1
fi

if ! command -v gomarkdoc &> /dev/null; then
    go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
fi

rm -rf docs-site/content/en/*
mkdir -p docs-site/content/en

cat <<EOF > docs-site/content/en/_index.md
---
title: "Go API Reference"
type: docs
---
# MCP Toolbox Go API Reference
Welcome to the technical reference for the MCP Toolbox Go SDK. 
Use the sidebar to explore package-level code signatures and comments.
EOF

echo "Generating API Reference Markdown..."
gomarkdoc -o docs-site/content/en/core.md ./core/...
gomarkdoc -o docs-site/content/en/tbadk.md ./tbadk/...
gomarkdoc -o docs-site/content/en/tbgenkit.md ./tbgenkit/...

cd docs-site
hugo --minify --baseURL "${BASE_URL}${VERSION}/" --destination "public/${VERSION}"

cat <<EOF > public/index.html
<!DOCTYPE html>
<html>
<head>
    <script>window.location.replace('${VERSION}/');</script>
</head>
</html>
EOF

echo "['${VERSION}']" > public/versions.json
