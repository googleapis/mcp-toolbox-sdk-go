# Changelog

## [1.3.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbadk/v1.2.0...tbadk/v1.3.0) (2026-09-01)


### Features

* **tbadk:** support secure parameters in ADK tools and client ([f05e55d](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/f05e55da22b7b52b6d443b680175fca17a06f3b4))


### Bug Fixes

* **mcp:** include tool output in error message on execution failure ([#324](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/324)) ([910b97e](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/910b97ee7fddc7facca50bbd3cdbc4b774d00c7a))

### Miscellaneous Chores

* Update core dependency in TBADK & TBGenkit ([#345](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/345)) ([4cccc60](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/4cccc60f39572ad80f334bef8446c8ed5fcceb3a))

## [1.2.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbadk/v1.1.0...tbadk/v1.2.0) (2026-09-01)


### Features

* **tbadk:** support secure parameters in ADK tools and client ([f05e55d](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/f05e55da22b7b52b6d443b680175fca17a06f3b4))


### Bug Fixes

* **mcp:** include tool output in error message on execution failure ([#324](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/324)) ([910b97e](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/910b97ee7fddc7facca50bbd3cdbc4b774d00c7a))


## [1.1.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbadk/v1.0.0...tbadk/v1.1.0) (2026-08-04)


### Features

* **core:** add MCP 2026 (July spec) stateless protocol support and auto-negotiation ([#317](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/317)) ([be1e47a](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/be1e47a551bfe300733480488fd6ff37d2e04451))

## [1.0.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbadk/v0.8.0...tbadk/v1.0.0) (2026-07-09)


### ⚠ BREAKING CHANGES

* **deps:** Update ADK to v2 ([#289](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/289)) ([9e71e9f](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/9e71e9f19c7a4757fe7e2f283f45c7b5811c656b))
Please use the updated Sample as a guide to use ADK v2 with TBADK v1


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
