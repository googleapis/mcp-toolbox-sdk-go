# Changelog

## [1.0.1](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/core/v1.0.0...core/v1.0.1) (2026-04-09)


### Miscellaneous Chores

* **deps:** bump github.com/go-jose/go-jose/v4 in /core ([#223](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/223)) ([3ff7232](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/3ff7232cbd524905b52e55f32bf01737bb15bba7))
* **deps:** bump go.opentelemetry.io/otel/sdk in /core ([#228](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/228)) ([3e5a1bb](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/3e5a1bbc620a24fa5831e5b5b7a0c265c49e5302))


### Documentation

* Update Links to repo and new docsite ([#229](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/229)) ([81718da](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/81718daa4b96fb719c68c7da4c3154ec4dc56b18))

## [1.0.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/core/v0.7.0...core/v1.0.0) (2026-03-31)


### Bug Fixes

* **core:** resolve dropped default parameter values in MCP transport parsing ([#215](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/215)) ([76e39ec](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/76e39ec88686a9684b5c8a1b1e2d9ed7d98dda51))


### Documentation

Documentation migrated to the MCP Toolbox official docsite ([#201](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/201)) ([7dac748](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/7dac74880ef0ed2055e34dc6deae09509a01fc5f))

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
