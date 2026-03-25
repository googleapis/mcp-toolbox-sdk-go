![MCP Toolbox Logo](https://raw.githubusercontent.com/googleapis/genai-toolbox/main/logo.png)

# MCP Toolbox TBGenkit Package

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

> [!IMPORTANT]
> **Breaking Change Notice**: As of version `0.6.0`, this repository has transitioned to a multi-module structure.
> *   **For new versions (`v0.6.0`+)**: You must import specific modules (e.g., `go get github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit`).
> *   **For older versions (`v0.5.1` and below)**: The repository remains a single-module library (`go get github.com/googleapis/mcp-toolbox-sdk-go`).
> *   Please update your imports and `go.mod` accordingly when upgrading.

This module allows you to seamlessly integrate the functionalities of
[Toolbox](https://github.com/googleapis/genai-toolbox) allowing you to load and
use tools defined in the service as standard Genkit Tools within your Genkit Go
applications.

This simplifies integrating external functionalities (like APIs, databases, or
custom logic) managed by the Toolbox into your workflows, especially those
involving Large Language Models (LLMs).


<!-- TOC ignore:true -->
<!-- TOC -->

- [Installation](#installation)
- [Quickstart](#quickstart)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

<!-- /TOC -->

## Installation

```bash
go get github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit
```
This SDK is supported on Go version 1.24.4 and higher.

## Quickstart

For more information on how to load a ToolboxTool, see [the core package](https://github.com/googleapis/mcp-toolbox-sdk-go/tree/main/core)

## Usage

The `tbgenkit` package provides a framework-agnostic way to interact with your MCP Toolbox server. For detailed guides and advanced configuration, please visit the following sections on our [Documentation Site](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbgenkit):

- [Convert Toolbox Tool to a Genkit Tool](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbgenkit/#convert-toolbox-tool-to-a-genkit-tool)

# Contributing

Contributions are welcome! Please refer to the [DEVELOPER.md](/DEVELOPER.md)
file for guidelines on how to set up a development environment and run tests.

# License

This project is licensed under the Apache License 2.0. See the
[LICENSE](https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/LICENSE) file for details.

# Support

If you encounter issues or have questions, check the existing [GitHub Issues](https://github.com/googleapis/genai-toolbox/issues) for the main Toolbox project.
