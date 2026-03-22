#!/bin/bash
set -e

export PATH=$PATH:$(go env GOPATH)/bin

if ! command -v gomarkdoc &> /dev/null; then
    go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
fi

mkdir -p docs-site/content/en
if [ ! -f docs-site/content/en/_index.md ]; then
    cat <<EOF > docs-site/content/en/_index.md
---
title: "Go API Reference"
type: docs
---
Welcome to the MCP Toolbox Go SDK Technical Reference. Use the sidebar to explore package definitions.
EOF
fi

echo "Generating API Reference Markdown..."
gomarkdoc -o docs-site/content/en/core.md ./core/...
gomarkdoc -o docs-site/content/en/tbadk.md ./tbadk/...
gomarkdoc -o docs-site/content/en/tbgenkit.md ./tbgenkit/...

cd docs-site
hugo --minify
