#!/bin/bash
set -e

export PATH=$PATH:$(go env GOPATH)/bin

if ! command -v gomarkdoc &> /dev/null; then
    go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
fi

mkdir -p docs-site/content/en

echo "Generating API Reference Markdown..."
gomarkdoc -o docs-site/content/en/core.md ./core/...
gomarkdoc -o docs-site/content/en/tbadk.md ./tbadk/...
gomarkdoc -o docs-site/content/en/tbgenkit.md ./tbgenkit/...

cd docs-site
hugo --minify
