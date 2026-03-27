#!/bin/bash
set -e

export PATH=$PATH:$(go env GOPATH)/bin

VERSION=${1:-"main"}
BASE_URL=${2:-"/"}

go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

rm -rf docs-site/content/*
mkdir -p docs-site/content/docs

cat <<EOF > docs-site/content/_index.md
---
title: "MCP Toolbox Go SDK"
type: docs
---

EOF

cat README.md >> docs-site/content/_index.md

cat <<EOF > docs-site/content/docs/_index.md
---
title: "Packages"
type: docs
weight: 1
alwaysopen: true
---
Select a framework to view its public variables, functions, and structs.
EOF

generate_package() {
  local PKG_DIR=$1
  local TITLE=$2
  local WEIGHT=$3
  local MANUAL_VERSIONS=$4
  local MD_FILE="docs-site/content/docs/${PKG_DIR}.md"

  printf -- "---\ntitle: \"%s\"\ntype: docs\nweight: %s\n---\n\n" "$TITLE" "$WEIGHT" > "$MD_FILE"

  cat <<EOF >> "$MD_FILE"
<div style="margin-bottom: 2rem; padding: 1rem; background-color: #f8f9fa; border-radius: 8px; border: 1px solid #e9ecef; display: inline-block;">
  <label for="${PKG_DIR}-version" style="font-weight: bold; margin-right: 10px; color: #4a4a4a;">Package Version:</label>
  
  <select id="${PKG_DIR}-version" onchange="if (this.value) window.location.href=this.value;" style="padding: 5px 10px; border-radius: 4px; border: 1px solid #ccc; background-color: white; color: #333333; cursor: pointer;">
    <option value="${BASE_URL}main/docs/${PKG_DIR}/">main (latest)</option>
EOF

  for VER in $MANUAL_VERSIONS; do
    echo "    <option value=\"${BASE_URL}${VER}/docs/${PKG_DIR}/\">${VER}</option>" >> "$MD_FILE"
  done

  cat <<EOF >> "$MD_FILE"
  </select>
</div>
EOF

  gomarkdoc ./${PKG_DIR}/... | sed '/^# /d' >> "$MD_FILE"
}

# --- EXECUTE GENERATOR (UPDATE THESE BEFORE RELEASING!) ---
# To add a version to the dropdown, just type it inside the quotes separated by a space.
# Example: generate_package "core" "Core" "10" "v1.0.0 v0.9.0"

generate_package "core" "Core" "10" ""
generate_package "tbadk" "Tbadk" "20" ""
generate_package "tbgenkit" "Tbgenkit" "30" ""

cd docs-site
sed -i "s|PLACEHOLDER_BASE_URL|${BASE_URL}|g" hugo.toml

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