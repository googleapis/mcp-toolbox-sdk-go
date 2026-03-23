#!/bin/bash
set -e

export PATH=$PATH:$(go env GOPATH)/bin

BASE_URL="https://anmolshukla2002.github.io/mcp-toolbox-sdk-go/"

go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

rm -rf docs-site/content/en/*
mkdir -p docs-site/content/en/docs

cat <<EOF > docs-site/content/en/_index.md
---
title: "MCP Toolbox Go SDK"
type: docs
---
# Welcome to the API Reference

**If you can read this text, the pipeline is finally working.** Please click on the **Docs** link in the top navigation bar, or use the sidebar on the left to navigate your Go packages.
EOF

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
hugo --minify --baseURL "${BASE_URL}" --destination "public"