# Changelog

## [1.2.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/core/v1.1.0...core/v1.2.0) (2026-09-01)


### Features

* **core:** client options and tool/set loading with secure parameters ([5541ff2](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/5541ff2becf0dc5cede7f34489700c873266ec38))
* **core:** protocol and wire transport support for secure parameters ([4e5d650](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/4e5d65060472b0024fe698609877262390443b49))
* **core:** tool-level secure parameter binding, fast-fail and validation ([ef2f9cb](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/ef2f9cbbdcc28b18dfed4bb41b7f94f46c40d040))


### Bug Fixes

* **mcp:** include tool output in error message on execution failure ([#324](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/324)) ([910b97e](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/910b97ee7fddc7facca50bbd3cdbc4b774d00c7a))


## [1.1.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/core/v1.0.0...core/v1.1.0) (2026-08-04)


### Features

*  Add MCP 2026 (July spec) stateless protocol support and auto-negotiation ([#317](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/317)) ([be1e47a](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/be1e47a551bfe300733480488fd6ff37d2e04451))

* Add automatic protocol version fallback and negotiation (https://github.com/googleapis/mcp-toolbox-sdk-go/pull/305)

* Add support response _meta serverInfo and resultType in July Spec (https://github.com/googleapis/mcp-toolbox-sdk-go/pull/308)

* Add MCPLatest protocol ([#271](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/271)) ([7154330](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/71543303a19cc5c57bec76eefc2143f453271e74))

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
