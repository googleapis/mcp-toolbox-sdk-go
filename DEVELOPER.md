# Development

This guide provides instructions for setting up your development environment to
contribute to the `mcp-toolbox-sdk-go` repository, which is a multi-module workspace.

## Prerequisites

Before you begin, ensure you have the following installed:

*   [Go](https://go.dev/doc/install) (v1.24.4 or higher)

## Setup

This repository contains multiple Go modules:
*   `core`: The core SDK.
*   `tbadk`: ADK Go integration.
*   `tbgenkit`: Genkit Go integration.

### Working with the Workspace

We use a `go.work` file to manage local development across these modules.

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/googleapis/mcp-toolbox-sdk-go.git
    cd mcp-toolbox-sdk-go
    ```

2.  **Initialize Workspace (Optional but Recommended)**:
    Create a `go.work` file in the root to easily work with all modules simultaneously.
    ```bash
    go work init ./core ./tbadk ./tbgenkit
    ```
    *Note: `go.work` is git-ignored to prevent conflicts between developers.*

3.  **Install Dependencies**:
    Navigate to each module and install dependencies if needed:
    ```bash
    cd core && go mod tidy
    cd ../tbadk && go mod tidy
    cd ../tbgenkit && go mod tidy
    ```

## Testing

Tests are separated into **Unit Tests** and **End-to-End (E2E) Tests**.

### Unit Tests
Unit tests are fast and do not require external dependencies.
*   **Run all unit tests**:
    ```bash
    go test -tags=unit ./core/... ./tbadk/... ./tbgenkit/...
    ```
    *Note: If using `go.work`, this runs tests for all modules.*

### E2E Tests
E2E tests require a running Toolbox server and specific environment variables. They are guarded by the `e2e` build tag.
*   **Run E2E tests**:
    ```bash
    go test -tags=e2e -p 1 ./core/... ./tbadk/... ./tbgenkit/...
    ```

## Linting and Formatting

This project uses `golangci-lint`.

1.  **Run Linter**:
    You generally need to run this within each module directory:
    ```bash
    cd core && golangci-lint run
    cd ../tbadk && golangci-lint run
    cd ../tbgenkit && golangci-lint run
    ```

## Committing Changes

*   **Conventional Commits**: Please follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).
    *   Prefix keys: `core:`, `tbadk:`, `tbgenkit:`, `chore:`, `docs:`.
    *   Example: `feat(core): add new transport protocol`
*   **Pre-submit checks**: Ensure all tests (unit) pass before sending a PR.

## Release Process

Releases are managed by **Release Please**.
*   Each module (`core`, `tbadk`, `tbgenkit`) is released independently.
*   Tags will be in the format `module/vX.Y.Z` (e.g., `core/v0.6.0`).
*   Before releasing, add a `[[params.versions.<pkg>]]` block for the new version to
    `docs-site/hugo.toml` so it appears in the API reference version picker. See
    [Adding a version to the picker](#adding-a-version-to-the-picker).

## API Reference Documentation

The API reference is published to [go.mcp-toolbox.dev](https://go.mcp-toolbox.dev).
It is generated with [`gomarkdoc`](https://github.com/princjef/gomarkdoc) and
rendered by [Hugo](https://gohugo.io/) + [Docsy](https://www.docsy.dev/) from the
`docs-site/` directory. Docs are built **per package, per version** and served at
`/<package>/<version>/` (e.g. `/core/v1.0.0/`), with a `/<package>/latest/`
redirect to the newest release.

### Workflows

The `api-docs.yml` workflow deploys to the `gh-pages` branch. It runs only on
the upstream repository and uses the `api-docs-deploy` concurrency group, so it
never races another deploy.

The automatic flow is as follows:
*   Push to `main` (or manual dispatch) → builds all three packages as `dev`.
*   Push of a per-package tag `<pkg>/vX.Y.Z` → builds that one version **and**
    rebuilds the root README landing page.
*   Other tags are skipped.

### Adding a version to the picker

Before each **new release**, add a `[[params.versions.<pkg>]]` block for the version
to `docs-site/hugo.toml` (newest first). On a successful release, the tag is created
automatically and triggers the `api-docs.yml` workflow, which builds and deploys
that version.

### Backfilling old docs

Use the **`api-docs-backfill.yml`** (API Reference Backfill) workflow to publish docs
for a version whose pages are missing — typically releases that predate the docs
tooling, or a deployment that failed. It builds **one historical version per run**.

Unlike `api-docs.yml`, this workflow does **not** deploy to production directly. Each
run opens a **pull request into the `gh-pages` branch**, so the docs are reviewed
before they go live. The page is published only when you merge that PR.

How a run works:

1.  It checks out `main` for the current docs tooling (layouts, scripts, version
    picker), then overlays the requested version's package source from its release
    tag, so `gomarkdoc` documents that version's API.
2.  It builds `/<package>/<version>/` (plus the package's `releases`/`latest` files).
3.  It overlays the build onto a clone of the live `gh-pages` tree — existing
    versions, `CNAME`, and `.nojekyll` are preserved — and opens a PR from branch
    `backfill/<pkg>-<ver>` with `gh-pages` as the base.

Steps to backfill:

1.  Make sure the version is listed in `docs-site/hugo.toml` (see
    [Adding a version to the picker](#adding-a-version-to-the-picker)), so the
    dropdown links to it.
2.  Trigger the workflow from the Actions tab, or with:

    ```bash
    gh workflow run api-docs-backfill.yml -f package=core -f version=v1.0.0
    ```

    To catch up several versions, dispatch it once per `package`/`version`. The
    concurrency group is scoped per version, so the runs are independent and none
    are cancelled — each opens its own PR.
3.  Review the resulting `backfill/<pkg>-<ver>` PR (the diff should be just that
    version's directory) and **merge it into `gh-pages`** to publish. Re-running the
    workflow for the same version updates the existing PR's branch.

### Building locally

```bash
# Build a single package/version (base URL must end in a slash).
./scripts/generate-api-docs.sh core dev http://localhost:8080/

# Serve the output.
(cd docs-site/public && python3 -m http.server 8080)
# → http://localhost:8080/core/dev/
```

## Further Information

*   If you encounter issues, please open an [issue](https://github.com/googleapis/mcp-toolbox-sdk-go/issues).