# Changelog

## [0.8.1](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbadk/v0.8.0...tbadk/v0.8.1) (2026-04-09)


### Miscellaneous Chores

* **deps:** bump github.com/go-jose/go-jose/v4 in /tbadk ([#222](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/222)) ([efc8c03](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/efc8c03dfe3ee1e1effe0113d4665bcd0603b084))
* **deps:** bump go.opentelemetry.io/otel/sdk in /tbadk ([#226](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/226)) ([4e5c604](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/4e5c6044ea220649593dc0bfbc0d8f38a5033b19))


### Documentation

* Update Links to repo and new docsite ([#229](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/229)) ([81718da](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/81718daa4b96fb719c68c7da4c3154ec4dc56b18))

## [0.8.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbadk/v0.7.0...tbadk/v0.8.0) (2026-04-01)

### Bug Fixes

* **core:** resolve dropped default parameter values in MCP transport parsing ([#215](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/215)) ([76e39ec](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/76e39ec88686a9684b5c8a1b1e2d9ed7d98dda51))


### Documentation

* Documentation migrated to the MCP Toolbox official docsite ([#201](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/201)) ([7dac748](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/7dac74880ef0ed2055e34dc6deae09509a01fc5f))

## [0.7.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbadk/v0.6.0...tbadk/v0.7.0) (2026-03-05)

### ⚠ BREAKING CHANGES

* Remove support for Native Toolbox transport ([#189](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/189))

### Features

* Add support for default parameters ([#185](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/185)) ([6c2bf7a](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/6c2bf7ac95ba4983794d40e70064217bb71fe015))
* Enable package-specific client version identification for MCP Transport ([#194](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/194)) ([f8ba007](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/f8ba007f85efb0cd3e22852a1be1456ec397e1c1))


## [0.6.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbadk/v0.5.1...tbadk/v0.6.0) (2026-02-16)

> [!IMPORTANT]
> **Breaking Change Notice**: As of version `0.6.0`, this repository has transitioned to a multi-module structure.
> *   **For new versions (`v0.6.0`+)**: You must import specific modules (e.g., `go get github.com/googleapis/mcp-toolbox-sdk-go/tbadk`).
> *   **For older versions (`v0.5.1` and below)**: The repository remains a single-module library (`go get github.com/googleapis/mcp-toolbox-sdk-go`).
> *   Please update your imports and `go.mod` accordingly when upgrading.

### Refactor

* Convert mcp-toolbox-go-sdk into multi-module repository ([#159](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/159)) ([da52e20](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/da52e2084095ec62df2b36824ebebccd8b82ceaf))


## Changelog
