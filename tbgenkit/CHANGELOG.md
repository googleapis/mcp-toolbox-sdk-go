# Changelog

## [0.10.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbgenkit/v0.9.0...tbgenkit/v0.10.0) (2026-09-01)


### Bug Fixes

* **mcp:** include tool output in error message on execution failure ([#324](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/324)) ([910b97e](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/910b97ee7fddc7facca50bbd3cdbc4b774d00c7a))


### Miscellaneous Chores

* **core:** release 1.2.0 ([#340](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/340)) ([64f32b3](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/64f32b3bdc791bb35295f757ec065e0b2194c9c9))
* **tbgenkit:** release 0.10.0 ([#342](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/342)) ([50bb46b](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/50bb46b465a3bd6b00bd7fbc23961bc0b6e3e998))
* Update core dependency in TBADK & TBGenkit ([#345](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/345)) ([4cccc60](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/4cccc60f39572ad80f334bef8446c8ed5fcceb3a))

## [0.9.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbgenkit/v0.8.0...tbgenkit/v0.9.0) (2026-08-04)


### Features

* **core:** add MCP 2026 (July spec) stateless protocol support and auto-negotiation ([#317](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/317)) ([be1e47a](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/be1e47a551bfe300733480488fd6ff37d2e04451))

## [0.8.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbgenkit/v0.7.0...tbgenkit/v0.8.0) (2026-04-01)


### Bug Fixes

* **core:** resolve dropped default parameter values in MCP transport parsing ([#215](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/215)) ([76e39ec](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/76e39ec88686a9684b5c8a1b1e2d9ed7d98dda51))


### Documentation

* Documentation migrated to the MCP Toolbox official docsite ([#201](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/201)) ([7dac748](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/7dac74880ef0ed2055e34dc6deae09509a01fc5f))

## [0.7.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbgenkit/v0.6.0...tbgenkit/v0.7.0) (2026-03-06)


### ⚠ BREAKING CHANGES

* Remove support for Native Toolbox transport ([#189](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/189))

### Features

* Add support for default parameters ([#185](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/185)) ([6c2bf7a](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/6c2bf7ac95ba4983794d40e70064217bb71fe015))
* Remove support for Native Toolbox transport ([#189](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/189)) ([d596ef8](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/d596ef87f0dfbb361b11b85a71fb597414c5d904))

## [0.6.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbgenkit/v0.5.1...tbgenkit/v0.6.0) (2026-02-16)

> [!IMPORTANT]
> **Breaking Change Notice**: As of version `0.6.0`, this repository has transitioned to a multi-module structure.
> *   **For new versions (`v0.6.0`+)**: You must import specific modules (e.g., `go get github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit`).
> *   **For older versions (`v0.5.1` and below)**: The repository remains a single-module library (`go get github.com/googleapis/mcp-toolbox-sdk-go`).
> *   Please update your imports and `go.mod` accordingly when upgrading.

### Refactor

* Convert mcp-toolbox-go-sdk into multi-module repository ([#159](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/159)) ([da52e20](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/da52e2084095ec62df2b36824ebebccd8b82ceaf))
## Changelog
