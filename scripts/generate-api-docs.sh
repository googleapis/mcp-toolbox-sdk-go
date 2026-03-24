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

This is the official Go SDK for the MCP Toolbox. Use the sidebar to navigate the technical API reference for each package.

## Installation

The SDK is distributed as individual Go modules for new versions (v0.6.0+). Depending on your use case, install the required packages using the commands below:

### Core
Provides the foundational types and interfaces for the MCP Toolbox.
\`\`\`bash
go get github.com/googleapis/mcp-toolbox-sdk-go/core
\`\`\`

### ADK
Provides tools and frameworks for building and managing MCP agents.
\`\`\`bash
go get github.com/googleapis/mcp-toolbox-sdk-go/tbadk
\`\`\`

### Genkit
Integrates the MCP Toolbox with the Genkit framework.
\`\`\`bash
go get github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit
\`\`\`
EOF

cat <<EOF > docs-site/content/docs/_index.md
---
title: "Packages"
type: docs
weight: 1
alwaysopen: true
---
Select a framework to view its exported variables, functions, and structs.
EOF

echo "Generating API Reference Markdown..."

printf -- "---\ntitle: \"Core\"\ntype: docs\nweight: 10\n---\n\n" > docs-site/content/docs/core.md
gomarkdoc ./core/... | sed '/^# /d' >> docs-site/content/docs/core.md

printf -- "---\ntitle: \"Tbadk\"\ntype: docs\nweight: 20\n---\n\n" > docs-site/content/docs/tbadk.md
gomarkdoc ./tbadk/... | sed '/^# /d' >> docs-site/content/docs/tbadk.md

printf -- "---\ntitle: \"Tbgenkit\"\ntype: docs\nweight: 30\n---\n\n" > docs-site/content/docs/tbgenkit.md
gomarkdoc ./tbgenkit/... | sed '/^# /d' >> docs-site/content/docs/tbgenkit.md

cd docs-site
VERSION=${1:-"main"}
HUGO_PARAMS_VERSION="${VERSION}" hugo --minify --baseURL "${BASE_URL}${VERSION}/" --destination "public/${VERSION}"
cat <<EOF > public/index.html
<!DOCTYPE html>
<html>
<head>
  <meta http-equiv="refresh" content="0; url=${BASE_URL}${VERSION}/" />
</head>
<body style="background-color: rgb(64, 63, 76); color: white; text-align: center; padding-top: 50px; font-family: sans-serif;">
  <p>Redirecting to the latest API version (${VERSION})...</p>
  <script>window.location.replace('${BASE_URL}${VERSION}/');</script>
</body>
</html>
EOF