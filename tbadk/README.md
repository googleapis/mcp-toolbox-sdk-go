![MCP Toolbox Logo](https://raw.githubusercontent.com/googleapis/genai-toolbox/main/logo.png)

# MCP Toolbox tbadk SDK

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

> [!IMPORTANT]
> **Breaking Change Notice**: As of version `0.6.0`, this repository has transitioned to a multi-module structure.
> *   **For new versions (`v0.6.0`+)**: You must import specific modules (e.g., `go get github.com/googleapis/mcp-toolbox-sdk-go/tbadk`).
> *   **For older versions (`v0.5.1` and below)**: The repository remains a single-module library (`go get github.com/googleapis/mcp-toolbox-sdk-go`).
> *   Please update your imports and `go.mod` accordingly when upgrading.

This SDK allows you to seamlessly integrate the functionalities of
[Toolbox](https://github.com/googleapis/genai-toolbox) allowing you to load and
use tools defined in the service as standard Go structs within your ADK Go
applications.

This simplifies integrating external functionalities (like APIs, databases, or
custom logic) managed by the Toolbox into your workflows, especially those
involving Large Language Models (LLMs).

<!-- TOC ignore:true -->
<!-- TOC -->

- [MCP Toolbox tbadk SDK](#mcp-toolbox-tbadk-sdk)
  - [Installation](#installation)
  - [Quickstart](#quickstart)
  - [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

<!-- /TOC -->

## Installation

```bash
go get github.com/googleapis/mcp-toolbox-sdk-go/tbadk
```
This SDK is supported on Go version 1.24.4 and higher.

> [!NOTE]
>
> - While the SDK itself is synchronous, you can execute its functions within goroutines to achieve asynchronous behavior.


## Quickstart

Here's a minimal example to get you started. Ensure your Toolbox service is
running and accessible.

```go
package main

import (
  "context"
  "fmt"
  "github.com/googleapis/mcp-toolbox-sdk-go/tbadk"
)

func quickstart() string {
  inputs := map[string]any{"location": "London"}
  client, err := tbadk.NewToolboxClient("http://localhost:5000")
  if err != nil {
    return fmt.Sprintln("Could not start Toolbox Client", err)
  }
  tool, err := client.LoadTool("get_weather", ctx)
  if err != nil {
    return fmt.Sprintln("Could not load Toolbox Tool", err)
  }
  // pass the tool.Context as ctx into the Run() method
  result, err := tool.Run(ctx, inputs)
  if err != nil {
    return fmt.Sprintln("Could not invoke tool", err)
  }
  return fmt.Sprintln(result["output"])
}

func main() {
  fmt.Println(quickstart())
}
```

## Usage

The `tbadk` package provides a framework-agnostic way to interact with your MCP Toolbox server. For detailed guides and advanced configuration, please visit the following sections on our [Documentation Site](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbadk):

- [Transport Protocols](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbadk/#transport-protocols)
- [Loading Tools](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbadk/#loading-tools)
- [Invoking Tools](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbadk/#invoking-tools)
- [Client to Server Authentication](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbadk/#client-to-server-authentication)
- [Authenticating Tools](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbadk/#authenticating-tools)
- [Binding Parameter Values](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbadk/#binding-parameter-values)
- [Default Parameters](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbadk/#default-parameters)
- [Using with Adk Go](https://googleapis.github.io/genai-toolbox/sdks/go-sdk/tbadk/#using-with-adk-go)

# Contributing

Contributions are welcome! Please refer to the [DEVELOPER.md](/DEVELOPER.md)
file for guidelines on how to set up a development environment and run tests.

# License

This project is licensed under the Apache License 2.0. See the
[LICENSE](https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/LICENSE) file for details.

# Support

If you encounter issues or have questions, check the existing [GitHub Issues](https://github.com/googleapis/genai-toolbox/issues) for the main Toolbox project.
