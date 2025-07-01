![MCP Toolbox Logo](https://raw.githubusercontent.com/googleapis/genai-toolbox/main/logo.png)

# MCP Toolbox Core SDK

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

This SDK allows you to seamlessly integrate the functionalities of
[Toolbox](https://github.com/googleapis/genai-toolbox) allowing you to load and
use tools defined in the service as standard Go structs within your GenAI
applications.

This simplifies integrating external functionalities (like APIs, databases, or
custom logic) managed by the Toolbox into your workflows, especially those
involving Large Language Models (LLMs).

<!-- TOC ignore:true -->
<!-- TOC -->

- [Installation](#installation)
- [Quickstart](#quickstart)
- [Usage](#usage)
- [Loading Tools](#loading-tools)
    - [Load a toolset](#load-a-toolset)
    - [Load a single tool](#load-a-single-tool)
- [Invoking Tools](#invoking-tools)
- [Synchronous Usage](#synchronous-usage)
- [Use with LangGraph](#use-with-langgraph)
- [Client to Server Authentication](#client-to-server-authentication)
    - [When is Client-to-Server Authentication Needed?](#when-is-client-to-server-authentication-needed)
    - [How it works](#how-it-works)
    - [Configuration](#configuration)
    - [Authenticating with Google Cloud Servers](#authenticating-with-google-cloud-servers)
    - [Step by Step Guide for Cloud Run](#step-by-step-guide-for-cloud-run)
- [Authenticating Tools](#authenticating-tools)
    - [When is Authentication Needed?](#when-is-authentication-needed)
    - [Supported Authentication Mechanisms](#supported-authentication-mechanisms)
    - [Step 1: Configure Tools in Toolbox Service](#step-1-configure-tools-in-toolbox-service)
    - [Step 2: Configure SDK Client](#step-2-configure-sdk-client)
        - [Provide an ID Token Retriever Function](#provide-an-id-token-retriever-function)
        - [Option A: Add Authentication to a Loaded Tool](#option-a-add-authentication-to-a-loaded-tool)
        - [Option B: Add Authentication While Loading Tools](#option-b-add-authentication-while-loading-tools)
    - [Complete Authentication Example](#complete-authentication-example)
- [Binding Parameter Values](#binding-parameter-values)
    - [Why Bind Parameters?](#why-bind-parameters)
    - [Option A: Binding Parameters to a Loaded Tool](#option-a-binding-parameters-to-a-loaded-tool)
    - [Option B: Binding Parameters While Loading Tools](#option-b-binding-parameters-while-loading-tools)
    - [Binding Dynamic Values](#binding-dynamic-values)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

<!-- /TOC -->

## Installation

```bash
go get github.com/googleapis/mcp-toolbox-sdk-go
```
This SDK is supported on Go version 1.24.2 and higher.

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
	"github.com/googleapis/mcp-toolbox-sdk-go/core"
)

func quickstart() string {
	ctx := context.Background()
	inputs := map[string]any{"location": "London"}
	client, err := core.NewToolboxClient("http://localhost:5000")
	if err != nil {
		return fmt.Sprintln("Could not start Toolbox Client", err)
	}
	tool, err := client.LoadTool("get_weather", ctx)
	if err != nil {
		return fmt.Sprintln("Could not load Toolbox Tool", err)
	}
	result, err := tool.Invoke(ctx, inputs)
	if err != nil {
		return fmt.Sprintln("Could not invoke tool", err)
	}
	return fmt.Sprintln(result)
}

func main() {
	fmt.Println(quickstart())
}
```

## Usage

Import and initialize a Toolbox client, pointing it to the URL of your running
Toolbox service.

```go
import "github.com/googleapis/mcp-toolbox-sdk-go/core"

client, err := core.NewToolboxClient("http://localhost:5000")
```

All interactions for loading and invoking tools happen through this client.

> [!NOTE]
> For advanced use cases, you can provide an external custom `http.Client`
> during initialization (e.g., `core.NewToolboxClient(URL, core.WithHTTPClient(myClient)`). If you
> provide your own session, you are responsible for managing its lifecycle;
> `ToolboxClient` *will not* close it.

> [!IMPORTANT]
> Closing the `ToolboxClient` also closes the underlying network session shared by
> all tools loaded from that client. As a result, any tool instances you have
> loaded will cease to function and will raise an error if you attempt to invoke
> them after the client is closed.

## Loading Tools

You can load tools individually or in groups (toolsets) as defined in your
Toolbox service configuration. Loading a toolset is convenient when working with
multiple related functions, while loading a single tool offers more granular
control.

### Load a toolset

A toolset is a collection of related tools. You can load all tools in a toolset
or a specific one:

```go
// Load default toolset
tools, err := client.LoadToolset(ctx)

// Load a specific toolset
tools, err := client.LoadToolset(ctx, core.WithName("my-toolset"))
```


### Load a single tool

Loads a specific tool by its unique name. This provides fine-grained control.

```go
tool, err = client.LoadTool("my-tool", ctx)
```

## Invoking Tools

Once loaded, tools behave like Go structs. You invoke them using `Invoke` method
by pass arguments corresponding to the parameters defined in the tool's
configuration within the Toolbox service as a map of parameter name to value.

```go
tool, err = client.LoadTool("my-tool", ctx)
inputs := map[string]any{"location": "London"}
result, err := tool.Invoke(ctx, inputs)
```

> [!TIP]
> For a more comprehensive guide on setting up the Toolbox service itself, which
> you'll need running to use this SDK, please refer to the [Toolbox Quickstart
> Guide](https://googleapis.github.io/genai-toolbox/getting-started/local_quickstart).

## Client to Server Authentication

This section describes how to authenticate the ToolboxClient itself when
connecting to a Toolbox server instance that requires authentication. This is
crucial for securing your Toolbox server endpoint, especially when deployed on
platforms like Cloud Run, GKE,  or any environment where unauthenticated access is restricted.

This client-to-server authentication ensures that the Toolbox server can verify
the identity of the client making the request before any tool is loaded or
called. It is different from [Authenticating Tools](#authenticating-tools),
which deals with providing credentials for specific tools within an already
connected Toolbox session.

### When is Client-to-Server Authentication Needed?

You'll need this type of authentication if your Toolbox server is configured to
deny unauthenticated requests. For example:

- Your Toolbox server is deployed on Cloud Run and configured to "Require authentication."
- Your server is behind an Identity-Aware Proxy (IAP) or a similar
  authentication layer.
- You have custom authentication middleware on your self-hosted Toolbox server.

Without proper client authentication in these scenarios, attempts to connect or
make calls (like `LoadTool`) will likely fail with `Unauthorized` errors.

### How it works

The `ToolboxClient` allows you to specify TokenSources that dynamically generate HTTP headers for
every request sent to the Toolbox server. The most common use case is to add an
Authorization header with a bearer token (e.g., a Google ID token).

These header-generating functions are called just before each request, ensuring
that fresh credentials or header values can be used.

### Configuration

You can configure these dynamic headers as seen below:


```go
import "github.com/googleapis/mcp-toolbox-sdk-go/core"

tokenProvider := func() string {
	return "header3_value"
}

staticTokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "header2_value"})
dynamicTokenSource := core.NewCustomTokenSource(tokenProvider)

client, err := core.NewToolboxClient(
  "toolbox-url",
  core.WithClientHeaderString("header1", "header1_value"),
  core.WithClientHeaderTokenSource("header2", staticTokenSource),
  core.WithClientHeaderTokenSource("header3", dynamicTokenSource),
)
```

### Authenticating with Google Cloud Servers

For Toolbox servers hosted on Google Cloud (e.g., Cloud Run) and requiring
`Google ID token` authentication, the helper module
[auth_methods](/core/auth_methods.go) provides utility functions.

### Step by Step Guide for Cloud Run

1. **Configure Permissions**: [Grant](https://cloud.google.com/run/docs/securing/managing-access#service-add-principals) the `roles/run.invoker` IAM role on the Cloud
   Run service to the principal. This could be your `user account email` or a
   `service account`.
2. **Configure Credentials**
    - Local Development: Set up
   [ADC](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment).
    - Google Cloud Environments: When running within Google Cloud (e.g., Compute
      Engine, GKE, another Cloud Run service, Cloud Functions), ADC is typically
      configured automatically, using the environment's default service account.
3. **Connect to the Toolbox Server**

    - TODO

## Authenticating Tools

> [!WARNING]
> **Always use HTTPS** to connect your application with the Toolbox service,
> especially in **production environments** or whenever the communication
> involves **sensitive data** (including scenarios where tools require
> authentication tokens). Using plain HTTP lacks encryption and exposes your
> application and data to significant security risks, such as eavesdropping and
> tampering.

Tools can be configured within the Toolbox service to require authentication,
ensuring only authorized users or applications can invoke them, especially when
accessing sensitive data.

### When is Authentication Needed?

Authentication is configured per-tool within the Toolbox service itself. If a
tool you intend to use is marked as requiring authentication in the service, you
must configure the SDK client to provide the necessary credentials (currently
Oauth2 tokens) when invoking that specific tool.

### Supported Authentication Mechanisms

The Toolbox service enables secure tool usage through **Authenticated Parameters**. For detailed information on how these mechanisms work within the Toolbox service and how to configure them, please refer to [Toolbox Service Documentation - Authenticated Parameters](https://googleapis.github.io/genai-toolbox/resources/tools/#authenticated-parameters)

### Step 1: Configure Tools in Toolbox Service

First, ensure the target tool(s) are configured correctly in the Toolbox service
to require authentication. Refer to the [Toolbox Service Documentation -
Authenticated
Parameters](https://googleapis.github.io/genai-toolbox/resources/tools/#authenticated-parameters)
for instructions.

### Step 2: Configure SDK Client

Your application needs a way to obtain the required Oauth2 token for the
authenticated user. The SDK requires you to provide a function capable of
retrieving this token *when the tool is invoked*.

#### Provide an ID Token Retriever Function

You must provide the SDK with a function that returns the
necessary token when called. The implementation depends on your application's
authentication flow (e.g., retrieving a stored token, initiating an OAuth flow).

> [!IMPORTANT]
> The name used when registering the getter function with the SDK (e.g.,
> `"my_api_token"`) must exactly match the `name` of the corresponding
> `authServices` defined in the tool's configuration within the Toolbox service.

```go
func getAuthToken() string {
  // ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
  // This example just returns a placeholder. Replace with your actual token retrieval.
	return "YOUR_ID_TOKEN" // Placeholder
}
```

> [!TIP]
> Your token retriever function is invoked every time an authenticated parameter
> requires a token for a tool call. Consider implementing caching logic within
> this function to avoid redundant token fetching or generation, especially for
> tokens with longer validity periods or if the retrieval process is
> resource-intensive.

#### Option A: Add Authentication to a Loaded Tool

You can add the token retriever function to a tool object *after* it has been
loaded. This modifies the specific tool instance.

```go

ctx := context.Background()

client, err := core.NewToolboxClient("http://127.0.0.1:5000")

tool, err := client.LoadTool("my-tool", ctx)

AuthTool, err := tool.ToolFrom(
  core.WithAuthTokenSource("my-auth", headerTokenSource),
  core.WithAuthTokenString("my-auth-1", "value"),
  )
```

#### Option B: Add Authentication While Loading Tools

You can provide the token retriever(s) directly during the `LoadTool` or
`LoadToolset` calls. This applies the authentication configuration only to the
tools loaded in that specific call, without modifying the original tool objects
if they were loaded previously.

```go
AuthTool, err := client.LoadTool("my-tool", ctx, core.WithAuthTokenString("my-auth-1", "value"))

// or

AuthTools, err := client.LoadToolset(
  ctx,
  core.WithName("my-toolset"),
  core.WithAuthTokenString("my-auth-1", "value"),
)
```

> [!NOTE]
> Adding auth tokens during loading only affect the tools loaded within that
> call.

### Complete Authentication Example

```go
import "github.com/googleapis/mcp-toolbox-sdk-go/core"
import "fmt"

func getAuthToken() string {
  // ... Logic to retrieve ID token (e.g., from local storage, OAuth flow)
  // This example just returns a placeholder. Replace with your actual token retrieval.
	return "YOUR_ID_TOKEN" // Placeholder
}

func main() {
  ctx := context.Background()
  inputs := map[string]any{"input": "some input"}

  dynamicTokenSource := core.NewCustomTokenSource(getAuthToken)

  client, err := core.NewToolboxClient("http://127.0.0.1:5000")
  tool, err := client.LoadTool("my-tool", ctx)
  AuthTool, err := tool.ToolFrom(core.WithAuthTokenSource("my_auth", dynamicTokenSource))

  result, err := AuthTool.Invoke(ctx, inputs)

  fmt.Println(result)
}
```

> [!NOTE]
> An auth token getter for a specific name (e.g., "GOOGLE_ID") will replace any
> client header with the same name followed by "_token" (e.g.,
> "GOOGLE_ID_token").

## Binding Parameter Values

The SDK allows you to pre-set, or "bind", values for specific tool parameters
before the tool is invoked or even passed to an LLM. These bound values are
fixed and will not be requested or modified by the LLM during tool use.

### Why Bind Parameters?

- **Protecting sensitive information:**  API keys, secrets, etc.
- **Enforcing consistency:** Ensuring specific values for certain parameters.
- **Pre-filling known data:**  Providing defaults or context.

> [!IMPORTANT]
> The parameter names used for binding (e.g., `"api_key"`) must exactly match the
> parameter names defined in the tool's configuration within the Toolbox
> service.

> [!NOTE]
> You do not need to modify the tool's configuration in the Toolbox service to
> bind parameter values using the SDK.

### Option A: Binding Parameters to a Loaded Tool

Bind values to a tool object *after* it has been loaded. This modifies the
specific tool instance.

```go
client, err := core.NewToolboxClient("http://127.0.0.1:5000")

tool, err := client.LoadTool("my-tool", ctx)

boundTool, err := tool.ToolFrom(
  core.WithBindParamString("param1", "value"),
  core.WithBindParamString("param2", "value")
  )
```

### Option B: Binding Parameters While Loading Tools

Specify bound parameters directly when loading tools. This applies the binding
only to the tools loaded in that specific call.

```go
boundTool, err := client.LoadTool("my-tool", ctx, core.WithBindParamString("param", "value"))

// OR

boundTool, err := client.LoadToolset(ctx, core.WithBindParamString("param", "value"))
```

> [!NOTE]
> Bound values during loading only affect the tools loaded in that call.

### Binding Dynamic Values

Instead of a static value, you can bind a parameter to a synchronous or
asynchronous function. This function will be called *each time* the tool is
invoked to dynamically determine the parameter's value at runtime.
Functions with the return type (data_type, error) can be provided.

```go
getDynamicValue := func() (string, error) { return "req-123", nil }

dynamicBoundTool, err := tool.ToolFrom(core.WithBindParamStringFunc("param", getDynamicValue))
```

> [!IMPORTANT]
> You don't need to modify tool configurations to bind parameter values.

# Contributing

Contributions are welcome! Please refer to the [DEVELOPER.md](./DEVELOPER.md)
file for guidelines on how to set up a development environment and run tests.

# License

This project is licensed under the Apache License 2.0. See the
[LICENSE](https://github.com/googleapis/mcp-toolbox-sdk-go/blob/main/LICENSE) file for details.

# Support

If you encounter issues or have questions, check the existing [GitHub Issues](https://github.com/googleapis/genai-toolbox/issues) for the main Toolbox project.