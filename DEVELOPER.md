# Development

This guide provides instructions for setting up your development environment to
contribute to the `core` package, which is part of the
`mcp-toolbox-sdk-go` monorepo.

## Prerequisites

Before you begin, ensure you have the following installed:

* [Go](https://go.dev/doc/install)

## Setup

These steps will guide you through setting up the monorepo and this specific package for development.

1. Clone the repository:

    ```bash
    git clone https://github.com/googleapis/mcp-toolbox-sdk-go.git
    ```

2. Navigate to the **package directory**:

    ```bash
    cd mcp-toolbox-sdk-go/core
    ```

3. Install dependencies for your package:

    ```bash
    go get
    go mod tidy
    ```

4. Local Testing
    If you need to test changes in `mcp-toolbox-sdk-go` against another package that consumes `mcp-toolbox-sdk-go`, you can use:

    * Replace Directives
        In the go.mod of the consuming project, add the line
        ```go
        replace github.com/googleapis/mcp-toolbox-sdk-go => ../path/to/your/local/mcp-toolbox-sdk-go
        ```
        And reinstall the dependencies
        ```bash
        go mod tidy
        ```

      Remember to remove the replace directive before committing your changes!

    * Use a Go Workspace
      Clone the `mcp-toolbox-sdk-go` and your package in the same directory
      (ex. /development).

        ```bash
        cd /development
        go work init
        go work use ./my-consuming-project ./mcp-toolbox-sdk-go
        ```

      Remember, the generated go.work file should not be committed with your changes!

    Using either of these approaches will make sure any change in the `mcp-toolbox-sdk-go` will be reflected in the consuming project.

## Testing

Ensure all tests pass before submitting your changes. Tests are typically run from within the root directory.

> [!IMPORTANT]
> Dependencies (including testing tools) should have been installed during the initial `go get` at the monorepo root.

1. **Run Unit & Integration Tests:**

    ```bash
    go test ./... -v -race
    ```

## Linting and Formatting

This project uses golangci to maintain code quality and consistency.

1. **Run Linter & Fix Issues:**
    Check your code for linting errors and fix fixable linting and formatting issues:

    ```bash
    golangci-lint run
    ```

## Committing Changes

* **Branching:** Create a new branch for your feature or bug fix (e.g., `feature/my-new-feature` or `fix/issue-123`).
* **Commit Messages:** Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit message conventions.
* **Pre-submit checks:** On any PRs, presubmit checks like linters, unit tests
  and integration tests etc. are run. Make sure all checks are green before
  proceeding.
* **Submitting a PR:** On approval by a repo maintainer, *Squash and Merge* your PR.

## Further Information

* If you encounter issues or have questions, please open an [issue](https://github.com/googleapis/mcp-toolbox-sdk-go/issues) on the GitHub repository.