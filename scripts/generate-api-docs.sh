#!/bin/bash
set -e

export PATH=$PATH:$(go env GOPATH)/bin
BASE_URL=${1:-"/"}

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

echo "Generating API Reference Markdown with per-package dropdowns..."

printf -- "---\ntitle: \"Core\"\ntype: docs\nweight: 10\n---\n\n" > docs-site/content/docs/core.md
cat <<EOF >> docs-site/content/docs/core.md
<div style="margin-bottom: 2rem; padding: 1rem; background-color: #f8f9fa; border-radius: 8px; border: 1px solid #e9ecef; display: inline-block;">
  <label for="core-version" style="font-weight: bold; margin-right: 10px; color: #4a4a4a;">Package Version:</label>
  <select id="core-version" style="padding: 5px 10px; border-radius: 4px; border: 1px solid #ccc; background-color: white;">
    <option value="">main (latest)</option>
    </select>
</div>
EOF
gomarkdoc ./core/... | sed '/^# /d' >> docs-site/content/docs/core.md

printf -- "---\ntitle: \"Tbadk\"\ntype: docs\nweight: 20\n---\n\n" > docs-site/content/docs/tbadk.md
cat <<EOF >> docs-site/content/docs/tbadk.md
<div style="margin-bottom: 2rem; padding: 1rem; background-color: #f8f9fa; border-radius: 8px; border: 1px solid #e9ecef; display: inline-block;">
  <label for="tbadk-version" style="font-weight: bold; margin-right: 10px; color: #4a4a4a;">Package Version:</label>
  <select id="tbadk-version" style="padding: 5px 10px; border-radius: 4px; border: 1px solid #ccc; background-color: white;">
    <option value="">main (latest)</option>
    </select>
</div>
EOF
gomarkdoc ./tbadk/... | sed '/^# /d' >> docs-site/content/docs/tbadk.md

printf -- "---\ntitle: \"Tbgenkit\"\ntype: docs\nweight: 30\n---\n\n" > docs-site/content/docs/tbgenkit.md
cat <<EOF >> docs-site/content/docs/tbgenkit.md
<div style="margin-bottom: 2rem; padding: 1rem; background-color: #f8f9fa; border-radius: 8px; border: 1px solid #e9ecef; display: inline-block;">
  <label for="tbgenkit-version" style="font-weight: bold; margin-right: 10px; color: #4a4a4a;">Package Version:</label>
  <select id="tbgenkit-version" style="padding: 5px 10px; border-radius: 4px; border: 1px solid #ccc; background-color: white;">
    <option value="">main (latest)</option>
    </select>
</div>
EOF
gomarkdoc ./tbgenkit/... | sed '/^# /d' >> docs-site/content/docs/tbgenkit.md

cd docs-site
sed -i "s|PLACEHOLDER_BASE_URL|${BASE_URL}|g" hugo.toml

hugo --minify --baseURL "${BASE_URL}" --destination "public"