#!/bin/bash
set -e

export PATH=$PATH:$(go env GOPATH)/bin
VERSION=$1
BASE_URL=$2

if [ -z "$VERSION" ] || [ -z "$BASE_URL" ]; then
  echo "Usage: ./scripts/generate-api-docs.sh <version> <base_url>"
  exit 1
fi

go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

rm -rf docs-site/content/en/*
mkdir -p docs-site/content/en/docs

cat <<EOF > docs-site/content/en/_index.md
---
title: "Go API Reference"
type: docs
---
# MCP Toolbox Go API Reference

Welcome to the automated technical reference for the MCP Toolbox Go SDK. 
Use the sidebar to explore the technical definitions for each package.
EOF

# 2. FIX: Added 'type: docs' so the sidebar appears
cat <<EOF > docs-site/content/en/docs/_index.md
---
title: "Packages"
type: docs
weight: 1
---
EOF

echo "Generating API Reference Markdown..."
gomarkdoc -o docs-site/content/en/docs/core.md ./core/...
gomarkdoc -o docs-site/content/en/docs/tbadk.md ./tbadk/...
gomarkdoc -o docs-site/content/en/docs/tbgenkit.md ./tbgenkit/...

cd docs-site
hugo --minify --baseURL "${BASE_URL}${VERSION}/" --destination "public/${VERSION}"

cat <<EOF > public/index.html
<!DOCTYPE html>
<html>
<head>
  <meta http-equiv="refresh" content="0; url=${BASE_URL}${VERSION}/" />
  <script>window.location.replace('${BASE_URL}${VERSION}/');</script>
</head>
<body>Redirecting to latest version...</body>
</html>
EOF

echo "[\"${VERSION}\"]" > public/versions.json