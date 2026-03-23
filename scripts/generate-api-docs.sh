#!/bin/bash
set -e

export PATH=$PATH:$(go env GOPATH)/bin
VERSION=$1
BASE_URL=$2

if [ -z "$VERSION" ] || [ -z "$BASE_URL" ]; then
  echo "Usage: ./scripts/generate-api-docs.sh <version> <base_url>"
  exit 1
fi

go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

rm -rf docs-site/content/en/*
mkdir -p docs-site/content/en/docs

cat <<EOF > docs-site/content/en/_index.md
---
title: "MCP Toolbox Go SDK"
---
{{< blocks/cover title="MCP Toolbox Go API Reference" height="full" >}}
<p class="lead mt-4">Automated technical reference for the MCP Toolbox Go SDK.</p>
<div class="mx-auto mt-4">
  <a class="btn btn-lg btn-primary" href="${BASE_URL}${VERSION}/docs/">
    Explore Packages
  </a>
</div>
{{< /blocks/cover >}}
EOF

cat <<EOF > docs-site/content/en/docs/_index.md
---
title: "Packages"
type: docs
weight: 1
---
EOF

echo "Generating API Reference Markdown..."
gomarkdoc -o docs-site/content/en/docs/core.md ./core/...
gomarkdoc -o docs-site/content/en/docs/tbadk.md ./tbadk/...
gomarkdoc -o docs-site/content/en/docs/tbgenkit.md ./tbgenkit/...

cd docs-site
hugo --minify --baseURL "${BASE_URL}${VERSION}/" --destination "public/${VERSION}"

cat <<EOF > public/index.html
<!DOCTYPE html>
<html>
<head>
  <meta http-equiv="refresh" content="0; url=${BASE_URL}${VERSION}/" />
</head>
<body style="background-color: #f8f9fa; text-align: center; padding-top: 50px; font-family: sans-serif;">
  <p>Redirecting to the latest API version...</p>
  <script>window.location.replace('${BASE_URL}${VERSION}/');</script>
</body>
</html>
EOF

echo "[\"${VERSION}\"]" > public/versions.json