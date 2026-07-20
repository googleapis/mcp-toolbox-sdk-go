# Changelog

## [1.0.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/tbgenkit/v0.8.0...tbgenkit/v1.0.0) (2026-07-20)


### Miscellaneous Chores

* **ci:** Update GCS Bucket name after MCP Toolbox v1 ([#232](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/232)) ([1de836d](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/1de836d496750f602e16ad5af5dd3d4f788e199c))
* **ci:** Update toolbox version to 1.4.0 in integration tests ([#267](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/267)) ([ed7f993](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/ed7f993037a587bf233dcc400ee6c42a1c040654))
* **deps:** bump github.com/go-jose/go-jose/v4 in /tbgenkit ([#224](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/224)) ([b92b2f3](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/b92b2f3b96b60971bb15fd263045988a50be10d3))
* **deps:** bump go.opentelemetry.io/otel/sdk in /tbgenkit ([#227](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/227)) ([77dcab9](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/77dcab90d2e97edb52e13fdfdc4e1cd64184cb31))
* **deps:** bump golang.org/x/crypto from 0.49.0 to 0.52.0 in /tbgenkit ([#294](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/294)) ([b4b890e](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/b4b890ede21552e9936270e01306fc502c93944d))
* **deps:** bump golang.org/x/net from 0.52.0 to 0.55.0 in /tbgenkit ([#292](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/292)) ([fcf266c](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/fcf266cc065aafa6b4cf604bd430ad224c3eb6cb))
* **deps:** update mcp toolbox server for integration tests to v1.6.0 ([#284](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/284)) ([e3e02e6](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/e3e02e61427749dee21a00fd4cebbe1dae87a5f9))
* **deps:** update mcp toolbox server for integration tests to v1.7.0 ([#302](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/302)) ([c8b6f3e](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/c8b6f3e601a7956edf881144be178db3421e2286))
* **deps:** update mcp-toolbox server to v1.5.0 ([#274](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/274)) ([b3e4313](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/b3e4313e2a1f1f3743e3b912f394e4c9d6ad3387))
* **tbadk:** release 1.0.0 ([#297](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/297)) ([4605f14](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/4605f14faf3deb01128a3c5471218c1be5a98131))
* update genkit dependency to v1.10.0 ([#290](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/290)) ([c6d6937](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/c6d6937331e7b9884d5851e38ef475d32e9abb8b))


### Documentation

* Update Links to repo and new docsite ([#229](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/229)) ([81718da](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/81718daa4b96fb719c68c7da4c3154ec4dc56b18))

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
