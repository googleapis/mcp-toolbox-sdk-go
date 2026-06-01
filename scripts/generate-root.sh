#!/bin/bash
set -euo pipefail

BASE_URL="${1:-/}"

# Render the repo README (from the checked-out tag) as the root landing page.
# Built only on tag pushes so the root URL tracks the latest release and stays
# stable between main-branch dev builds.
CONTENT_DIR="$(mktemp -d)"
trap 'rm -rf "$CONTENT_DIR"' EXIT

cat > "$CONTENT_DIR/_index.md" <<EOF
---
title: "MCP Toolbox Go SDK"
type: docs
---
EOF
cat README.md >> "$CONTENT_DIR/_index.md"

cd docs-site
hugo \
  --minify \
  --contentDir "${CONTENT_DIR}" \
  --baseURL "${BASE_URL}" \
  --destination "public"
