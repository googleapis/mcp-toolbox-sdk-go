![MCP Toolbox
Logo](https://raw.githubusercontent.com/googleapis/mcp-toolbox/main/logo.png)

# MCP Toolbox SDKs for Go

[![License: Apache
2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Docs](https://img.shields.io/badge/Docs-MCP_Toolbox-blue)](https://mcp-toolbox.dev/)
[![Discord](https://img.shields.io/badge/Discord-%235865F2.svg?style=flat&logo=discord&logoColor=white)](https://discord.gg/Dmm69peqjh)
[![Medium](https://img.shields.io/badge/Medium-12100E?style=flat&logo=medium&logoColor=white)](https://medium.com/@mcp_toolbox)

**Core Module :**
[![Go Report Card](https://goreportcard.com/badge/github.com/googleapis/mcp-toolbox-sdk-go/core)](https://goreportcard.com/report/github.com/googleapis/mcp-toolbox-sdk-go/core)
[![Module Version](https://img.shields.io/github/v/release/googleapis/mcp-toolbox-sdk-go?filter=core/v*)](https://img.shields.io/github/v/release/googleapis/mcp-toolbox-sdk-go?filter=core/v*)
[![Go Version](https://img.shields.io/github/go-mod/go-version/googleapis/mcp-toolbox-sdk-go?filename=core/go.mod)]([https://img.shields.io/github/go-mod/go-version/googleapis/mcp-toolbox-sdk-go](https://img.shields.io/github/go-mod/go-version/googleapis/mcp-toolbox-sdk-go?filename=core/go.mod))

**TBADK Module :**
[![Go Report Card](https://goreportcard.com/badge/github.com/googleapis/mcp-toolbox-sdk-go/tbadk)](https://goreportcard.com/report/github.com/googleapis/mcp-toolbox-sdk-go/tbadk)
[![Module Version](https://img.shields.io/github/v/release/googleapis/mcp-toolbox-sdk-go?filter=tbadk/v*)](https://img.shields.io/github/v/release/googleapis/mcp-toolbox-sdk-go?filter=tbadk/v*)
[![Go Version](https://img.shields.io/github/go-mod/go-version/googleapis/mcp-toolbox-sdk-go?filename=tbadk/go.mod)]([https://img.shields.io/github/go-mod/go-version/googleapis/mcp-toolbox-sdk-go](https://img.shields.io/github/go-mod/go-version/googleapis/mcp-toolbox-sdk-go?filename=tbadk/go.mod))

**TBGenkit Module :**
[![Go Report Card](https://goreportcard.com/badge/github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit)](https://goreportcard.com/report/github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit)
[![Module Version](https://img.shields.io/github/v/release/googleapis/mcp-toolbox-sdk-go?filter=tbgenkit/v*)](https://img.shields.io/github/v/release/googleapis/mcp-toolbox-sdk-go?filter=tbgenkit/v*)
[![Go Version](https://img.shields.io/github/go-mod/go-version/googleapis/mcp-toolbox-sdk-go?filename=tbgenkit/go.mod)](https://img.shields.io/github/go-mod/go-version/googleapis/mcp-toolbox-sdk-go?filename=tbgenkit/go.mod)

 > [!IMPORTANT]
> **Breaking Change Notice**: As of version `0.6.0`, this repository has transitioned to a multi-module structure.
> *   **For new versions (`v0.6.0`+)**: You must import specific modules (e.g., `go get github.com/googleapis/mcp-toolbox-sdk-go/core`).
> *   **For older versions (`v0.5.1` and below)**: The repository remains a single-module library (`go get github.com/googleapis/mcp-toolbox-sdk-go`).
> *   Please update your imports and `go.mod` accordingly when upgrading.

This repository contains the Go SDKs for [MCP Toolbox](https://github.com/googleapis/mcp-toolbox). These SDKs allow you to load and use tools defined in your MCP Toolbox server as standard Go structs within your Agentic applications.

For comprehensive guides and advanced configuration, visit the [Main Documentation Site](https://mcp-toolbox.dev/).

<!-- TOC -->

- [Overview](#overview)
- [Available Packages](#available-packages)
- [Quick Start](#quick-start)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

<!-- /TOC -->

## Overview

The MCP Toolbox service provides a centralized way to manage and expose tools
(like API connectors, database query tools, etc.) for use by GenAI applications.

The Go SDK act as clients for that service. They handle the communication needed to:

* Fetch tool definitions from your running Toolbox instance.
* Provide convenient Go structs representing those tools.
* Invoke the tools (calling the underlying APIs/services configured in Toolbox).
* Handle authentication and parameter binding as needed.

By using the SDK, you can easily leverage your MCP Toolbox-managed tools directly
within your Go applications or AI orchestration frameworks.

## Available Packages

This repository hosts the following Go packages. See the package-specific
README for detailed installation and usage instructions:

| Package | Target Use Case | Path | Documentation |
| :------ | :----------| :--- | :---------- |
| `core` | Framework-agnostic / Custom apps | `core/` | [Go SDK Core Guide](https://mcp-toolbox.dev/documentation/connect-to/toolbox-sdks/go-sdk/core/) |
| `tbadk` | ADK Go Integration | `tbadk/` | [ADK Package Guide](https://mcp-toolbox.dev/documentation/connect-to/toolbox-sdks/go-sdk/tbadk/) |
| `tbgenkit` | Genkit Go Integration | `tbgenkit/` | [Genkit Package Guide](https://mcp-toolbox.dev/documentation/connect-to/toolbox-sdks/go-sdk/tbgenkit/) |

## Quick Start

1.  **Set up the Toolbox Service**: Ensure you have a running MCP Toolbox server. Follow the [MCP Toolbox Server Quickstart](https://mcp-toolbox.dev/documentation/introduction/).
2.  **Install the Appropriate SDK**:
    ```bash
    # For the core, framework-agnostic SDK
    go get github.com/googleapis/mcp-toolbox-sdk-go/core

    # For ADK Go
    go get github.com/googleapis/mcp-toolbox-sdk-go/tbadk

    # For Genkit Go
    go get github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit
    ```
3.  **Explore Tutorials**: Check out the [Go Quickstart Tutorial](https://mcp-toolbox.dev/documentation/connect-to/toolbox-sdks/go-sdk/) for a full walkthrough.

## Contributing

Contributions are welcome! Please refer to the
[`CONTRIBUTING.md`](https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/CONTRIBUTING.md)
to get started.

## License

This project is licensed under the Apache License 2.0. See the
[LICENSE](https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/LICENSE) file
for details.

## Support

If you encounter issues or have questions, please check the existing [GitHub
Issues](https://github.com/googleapis/mcp-toolbox/issues) for the main Toolbox
project. If your issue is specific to one of the SDKs, please look for existing
issues [here](https://github.com/googleapis/mcp-toolbox-sdk-go/issues) or
open a new issue in this repository.
