# Changelog

## [0.7.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/core/v0.6.2...core/v0.7.0) (2026-03-05)


### ⚠ BREAKING CHANGES

* Remove support for Native Toolbox transport ([#189](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/189))

### Features

* Add map binding options and normalize generic parameters ([#197](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/197)) ([23ee483](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/23ee483fdb696f45cca80a510c962ae7e3da9756))
* Add support for default parameters ([#185](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/185)) ([6c2bf7a](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/6c2bf7ac95ba4983794d40e70064217bb71fe015))
* Enable package-specific client version identification for MCP Transport ([#194](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/194)) ([f8ba007](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/f8ba007f85efb0cd3e22852a1be1456ec397e1c1))

## [0.6.2](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/github.com/googleapis/mcp-toolbox-sdk-go/core-v0.5.1...github.com/googleapis/mcp-toolbox-sdk-go/core-v0.6.2) (2026-02-12)

> [!IMPORTANT]
> **Breaking Change Notice**: As of version `0.6.2`, this repository has transitioned to a multi-module structure.
> *   **For new versions (`v0.6.2`+)**: You must import specific modules (e.g., `go get github.com/googleapis/mcp-toolbox-sdk-go/core`).
> *   **For older versions (`v0.5.1` and below)**: The repository remains a single-module library (`go get github.com/googleapis/mcp-toolbox-sdk-go`).
> *   Please update your imports and `go.mod` accordingly when upgrading.

### Refactor

* Convert mcp-toolbox-go-sdk into multi-module repository ([#159](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/159)) ([da52e20](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/da52e2084095ec62df2b36824ebebccd8b82ceaf))
