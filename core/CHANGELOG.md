# Changelog

## [1.0.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/core/v1.0.0...core/v1.0.0) (2026-07-20)


### Miscellaneous Chores

* Add MCPLatest protocol ([#271](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/271)) ([7154330](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/71543303a19cc5c57bec76eefc2143f453271e74))
* **ci:** Update GCS Bucket name after MCP Toolbox v1 ([#232](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/232)) ([1de836d](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/1de836d496750f602e16ad5af5dd3d4f788e199c))
* **ci:** Update toolbox version to 1.4.0 in integration tests ([#267](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/267)) ([ed7f993](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/ed7f993037a587bf233dcc400ee6c42a1c040654))
* **deps:** bump github.com/go-jose/go-jose/v4 in /core ([#223](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/223)) ([3ff7232](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/3ff7232cbd524905b52e55f32bf01737bb15bba7))
* **deps:** bump go.opentelemetry.io/otel/sdk in /core ([#228](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/228)) ([3e5a1bb](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/3e5a1bbc620a24fa5831e5b5b7a0c265c49e5302))
* **deps:** bump golang.org/x/crypto from 0.51.0 to 0.52.0 in /core ([#296](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/296)) ([eb0e202](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/eb0e20226b3f1cf3498deaf15af2e6393afaf02a))
* **deps:** bump golang.org/x/net from 0.52.0 to 0.55.0 in /core ([#291](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/291)) ([368cc40](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/368cc402e3036d0a4aede8851bc4e4f8a3cfe9df))
* **deps:** bump google.golang.org/grpc from 1.79.2 to 1.79.3 in /core ([#237](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/237)) ([0eefd84](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/0eefd84012e894e19ddf8ae5f828632b252e12e9))
* **deps:** update mcp toolbox server for integration tests to v1.6.0 ([#284](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/284)) ([e3e02e6](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/e3e02e61427749dee21a00fd4cebbe1dae87a5f9))
* **deps:** update mcp toolbox server for integration tests to v1.7.0 ([#302](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/302)) ([c8b6f3e](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/c8b6f3e601a7956edf881144be178db3421e2286))
* **deps:** update mcp-toolbox server to v1.5.0 ([#274](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/274)) ([b3e4313](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/b3e4313e2a1f1f3743e3b912f394e4c9d6ad3387))
* **tbadk:** release 1.0.0 ([#297](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/297)) ([4605f14](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/4605f14faf3deb01128a3c5471218c1be5a98131))


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
