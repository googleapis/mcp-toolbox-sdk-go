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

## Further Information

*   If you encounter issues, please open an [issue](https://github.com/googleapis/mcp-toolbox-sdk-go/issues).