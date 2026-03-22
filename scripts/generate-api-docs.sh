#!/bin/bash
set -e

export PATH=$PATH:$(go env GOPATH)/bin

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

Welcome to the automated technical reference for the MCP Toolbox Go SDK. 
Use the sidebar to explore package-level code signatures and comments.
EOF

echo "Generating API Reference Markdown..."
gomarkdoc -o docs-site/content/en/core.md ./core/...
gomarkdoc -o docs-site/content/en/tbadk.md ./tbadk/...
gomarkdoc -o docs-site/content/en/tbgenkit.md ./tbgenkit/...

cd docs-site
hugo --minify
