#!/bin/bash
set -e

export PATH=$PATH:$(go env GOPATH)/bin
BASE_URL="https://anmolshukla2002.github.io/mcp-toolbox-sdk-go/"

go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

rm -rf docs-site/content/*
mkdir -p docs-site/content/docs

cat <<EOF > docs-site/content/_index.md
---
title: "MCP Toolbox Go SDK"
type: docs
---
# Welcome to the MCP Toolbox Go SDK

This is the official Go SDK for the MCP Toolbox. Use the sidebar on the left to navigate the technical API reference for each package.

## Installation

To install the SDK, run the following command in your Go project:

\`\`\`bash
go get github.com/googleapis/mcp-toolbox-sdk-go
\`\`\`
EOF

cat <<EOF > docs-site/content/docs/_index.md
---
title: "Packages"
type: docs
weight: 1
---
# Package Overview
Select a framework from the left-hand sidebar to view its exported variables, functions, and structs.
EOF

echo "Generating API Reference Markdown..."

printf -- "---\ntitle: \"Core\"\nhide_title: true\ntype: docs\nweight: 10\n---\n\n" > docs-site/content/docs/core.md
gomarkdoc ./core/... >> docs-site/content/docs/core.md

printf -- "---\ntitle: \"Tbadk\"\nhide_title: true\ntype: docs\nweight: 20\n---\n\n" > docs-site/content/docs/tbadk.md
gomarkdoc ./tbadk/... >> docs-site/content/docs/tbadk.md

printf -- "---\ntitle: \"Tbgenkit\"\nhide_title: true\ntype: docs\nweight: 30\n---\n\n" > docs-site/content/docs/tbgenkit.md
gomarkdoc ./tbgenkit/... >> docs-site/content/docs/tbgenkit.md

cd docs-site
hugo --minify --baseURL "${BASE_URL}" --destination "public"