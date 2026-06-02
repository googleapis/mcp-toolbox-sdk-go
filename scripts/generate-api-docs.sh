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
# The package's API reference is the home page, so /<pkg>/<version>/ lands
# directly on the docs (the repo README lives only at the site root).
CONTENT_DIR="$(mktemp -d)"
trap 'rm -rf "$CONTENT_DIR"' EXIT

cat > "$CONTENT_DIR/_index.md" <<EOF
---
title: "MCP Toolbox Go SDK — ${TITLE} (${VERSION})"
type: docs
---

Viewing \`${VERSION}\`.

EOF
gomarkdoc "./${PACKAGE}/..." | sed '/^# /d' >> "$CONTENT_DIR/_index.md"

cd docs-site
HUGO_PARAMS_VERSION="${VERSION}" hugo \
  --minify \
  --contentDir "${CONTENT_DIR}" \
  --baseURL "${BASE_URL}${PACKAGE}/${VERSION}/" \
  --destination "public/${PACKAGE}/${VERSION}"
