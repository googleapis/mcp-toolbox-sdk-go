# Changelog

## [0.3.0](https://github.com/googleapis/mcp-toolbox-sdk-go/compare/github.com/googleapis/mcp-toolbox-sdk-go/core-v0.5.1...github.com/googleapis/mcp-toolbox-sdk-go/core-v0.3.0) (2026-02-11)


### ⚠ BREAKING CHANGES

* Convert mcp-toolbox-go-sdk into multi-module repository ([#159](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/159))

### refactor

* Convert mcp-toolbox-go-sdk into multi-module repository ([#159](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/159)) ([da52e20](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/da52e2084095ec62df2b36824ebebccd8b82ceaf))


### Features

* Add MCP Transport version 2024-11-05 ([#128](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/128)) ([f4784ea](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/f4784ea4fc43e7e9d3dcd36fb78c22e217145495))
* Add MCP Transport version 2025-03-26 ([#131](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/131)) ([dba6000](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/dba6000ae3031881f88e5fd33c4de01b93bce938))
* Add MCP Transport version 2025-06-18 ([#132](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/132)) ([8e70cd3](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/8e70cd3595efdf459078c0ca64c9eeb9a28d1789))
* Add support for Map parameter type ([#51](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/51)) ([c80d8d6](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/c80d8d6e2b6a48910c6f596dc8ef3f130a94b544))
* Add support for MCP protocol in core SDK ([#133](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/133)) ([240c976](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/240c9767d10ab08505ef3d6ad95412f8b5d9cdb8))
* Add support for MCP Version 2025-11-25 ([#146](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/146)) ([92defe2](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/92defe2a3d3645709e95d118a210616d058d8da3))
* Add support for nested maps in generic map parameter ([#61](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/61)) ([3b33c52](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/3b33c52150dc07e0427e5fb9e0f3ff07f0a62390))
* Adding Toolbox Tool and other utils ([#9](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/9)) ([3384fc1](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/3384fc18b577f5f7b52a7d94292b8ff34ddaf72b))
* Adding ToolboxClient and it's functionalities ([#8](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/8)) ([2ed22b9](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/2ed22b963caeb22a3481bcdcfa2d6519f5b5f614))
* **core:** initial release ([#36](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/36)) ([7bbb620](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/7bbb6202cfae15370506d3770867d3b630bfec9a))
* Enable package-specific client identification for MCP ([#155](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/155)) ([75cb30e](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/75cb30ecf5e47634aafafed4bab7617b7dc6243c))
* Manually trigerring a new release ([#66](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/66)) ([98003ce](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/98003ce383945e95150455efb461a032be914c28))
* Manually trigger minor release ([#64](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/64)) ([1d43be4](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/1d43be4f756238972c4adceed7fa3b249a8acaa5))
* Remove support for nested Maps ([#60](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/60)) ([402c987](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/402c987eeee0285f3f89c47570b7bca7889329a6))
* **tbadk:** Add E2E integration tests for tbadk package ([#96](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/96)) ([de7bbfe](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/de7bbfe360ffb29a3eb028a152d372d885498f9c))


### Bug Fixes

* **mcp:** Fix header propogation in mcp protocols ([#140](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/140)) ([25dff40](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/25dff406059142170baeeecf0a3e0e47c840d9f3))
* **mcp:** merge multiple JSON objects in MCP tool output ([#142](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/142)) ([a51eee4](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/a51eee4b7df060db0042cedf7abbac7bd61dc814))
* trigger patch release to fix corrupt filename ([#151](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/151)) ([8c725b8](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/8c725b862cfcc3da952c18c76d2b5bb19695ab1a))


### Miscellaneous Chores

* Add a getter for the tool's input schema ([#30](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/30)) ([006a9e4](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/006a9e4054bb50759a191441007ce72b2ebc3975))
* Add base structs for Toolbox Go Core SDK ([#3](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/3)) ([fb9a90a](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/fb9a90ac20f6a4115c3658a3765a644905435bbd))
* Add better assert condition for a flaky test ([#27](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/27)) ([d5a6714](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/d5a67148ff8364e52102a885beaf44adb14090bb))
* Add build tag to auth tests ([#40](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/40)) ([e74dcc9](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/e74dcc9063ff8e5dc5f76b7de212052e0748ff95))
* add coverage tests flag ([#81](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/81)) ([212a294](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/212a294fc33eab5f2e984a46edc194a568789527))
* Add DefaultToolOptions usage in the readme ([#29](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/29)) ([73236b2](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/73236b20f741a999004f777a273ff80447f98e92))
* Add deprecation warning for Toolbox Native protocol ([#158](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/158)) ([72c0d01](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/72c0d01cb9bda289f78457551e43b8007622f6f3))
* Add documentation for Go Core SDK ([#18](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/18)) ([922ccd2](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/922ccd2d939b7fd67759a24850190b02a9c89ffb))
* Add E2E code samples for various frameworks. ([#31](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/31)) ([74bbd30](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/74bbd302d19ad008db0ca9be0e87bdeea1a31bba))
* Add E2E tests for Core package ([#15](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/15)) ([1d77236](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/1d772364ae590de0102c0331d7238229a43e712b))
* Add Functional Options for Toolbox Go Core SDK ([#7](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/7)) ([a01d489](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/a01d48928d2a4cc4d43dfbf36603e7f96a8d84a6))
* Add GetGoogleIDToken method ([#24](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/24)) ([c02d3af](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/c02d3aff8eb6e7db7c4069457296eac20bb8fe36))
* Add HTTP unsecure connection warning on tool invocation ([#157](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/157)) ([bcef80b](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/bcef80b143b21872ef11c60df900bc484c4d7ad7))
* Add invoke function to the ToolboxTool ([#10](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/10)) ([350a79d](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/350a79d73a8d9995bf8104fc3e0ed104a84c39c5))
* Add support for optional parameters ([#25](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/25)) ([f43d767](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/f43d767a5702a0d1bdee71a8906632a91c16a754))
* Add Transport interface and Toolbox Transport ([#126](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/126)) ([5b83315](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/5b83315d11225dc52896b56a1a55e877013bc974))
* Add util functions ([#6](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/6)) ([fbf8ad5](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/fbf8ad58804378b439d68979d4a075625b57e3c1))
* Add warning for insecure connection ([#58](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/58)) ([d95dcf9](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/d95dcf9fd37e4da532bd5128319ebb7e62f57b08))
* Change the module name to match the repo path ([#17](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/17)) ([7aa70c7](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/7aa70c77d55b26cdc6f071db7a4e8b4948df50e6))
* keep default mcp version as v20250618 ([#148](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/148)) ([2298b4e](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/2298b4ea92cd04eeecabbdd20e225730c4cd9ef0))
* **main:** release 0.2.0 ([#46](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/46)) ([5003602](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/500360251893f12c6d55ce4cc257ba713124cd79))
* **main:** release 0.5.0 ([#122](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/122)) ([aa8929f](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/aa8929f7ff0e59e0594ef54c45f51f60b97580b7))
* **main:** release 0.5.1 ([#152](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/152)) ([6b9518d](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/6b9518da9d0d46dede3924ee24fad106fb033411))
* pick tools config version from cloudbuild.yaml ([#68](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/68)) ([aa001ba](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/aa001ba0f6aff09a2a7ed0edd53c4cb90291a098))
* release 0.1.0 ([#34](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/34)) ([36fe0e1](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/36fe0e136b8dd6ef359c2d56f86c5a4c77648842))
* release 0.3.0 ([#59](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/59)) ([a89adcf](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/a89adcf440b30ba4b3a74a4ff4a2a12afa05a586))
* Seperate E2E tests from unit tests ([#28](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/28)) ([c4ef3d8](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/c4ef3d82d95f1b03c00049b5cbe809ffc2c17758))
* **tbgenkit:** Add extra tests for tbgenkit ([#88](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/88)) ([48633a7](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/48633a7c8ff142d64aca505816e85592dbbf91ef))
* Update README.md ([#72](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/72)) ([b3271fc](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/b3271fcaaba5e8d40d6a97f439604366fdebab2d))


### Documentation

* Add documentation for MCP protocol support ([#139](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/139)) ([2687553](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/268755395c6045bd0e0b702f9f0e1c2a4e8ca2cc))
* fix auth_methods link ([#54](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/54)) ([cc23e3c](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/cc23e3cda23f9d45bb7a13d69c51eac050d4c5ba))
* **tbgenkit:** Add documentation for the tbgenkit package ([#43](https://github.com/googleapis/mcp-toolbox-sdk-go/issues/43)) ([d75afbf](https://github.com/googleapis/mcp-toolbox-sdk-go/commit/d75afbf48d6d148debcc72ebcc68f9b6791e1539))

## Changelog
