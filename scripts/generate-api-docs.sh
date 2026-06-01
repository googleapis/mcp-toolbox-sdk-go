#!/bin/bash
set -euo pipefail

export PATH="$PATH:$(go env GOPATH)/bin"

PACKAGE="${1:?package required (core|tbadk|tbgenkit)}"
VERSION="${2:?version required (e.g. v1.0.0 or dev)}"
BASE_URL="${3:-/}"

case "$PACKAGE" in
  core)     TITLE="Core" ;;
  tbadk)    TITLE="Tbadk" ;;
  tbgenkit) TITLE="Tbgenkit" ;;
  *)        echo "Unknown package: $PACKAGE" >&2; exit 1 ;;
esac

go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

# Per-build content tree in a temp dir, kept out of the checked-in
# docs-site/content so concurrent package builds never trample each other.
CONTENT_DIR="$(mktemp -d)"
trap 'rm -rf "$CONTENT_DIR"' EXIT
mkdir -p "$CONTENT_DIR/docs"

cat > "$CONTENT_DIR/_index.md" <<EOF
---
title: "MCP Toolbox Go SDK — ${TITLE} (${VERSION})"
type: docs
---
EOF
cat README.md >> "$CONTENT_DIR/_index.md"

cat > "$CONTENT_DIR/docs/_index.md" <<EOF
---
title: "Packages"
type: docs
weight: 1
alwaysopen: true
---
Public variables, functions, and structs for the ${TITLE} package.
EOF

MD_FILE="$CONTENT_DIR/docs/${PACKAGE}.md"
cat > "$MD_FILE" <<EOF
---
title: "${TITLE}"
type: docs
weight: 10
---

Viewing \`${VERSION}\`.

EOF
gomarkdoc "./${PACKAGE}/..." | sed '/^# /d' >> "$MD_FILE"

cd docs-site
HUGO_PARAMS_VERSION="${VERSION}" hugo \
  --minify \
  --contentDir "${CONTENT_DIR}" \
  --baseURL "${BASE_URL}${PACKAGE}/${VERSION}/" \
  --destination "public/${PACKAGE}/${VERSION}"
