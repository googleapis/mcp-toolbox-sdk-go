package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/invopop/jsonschema"
)

// NewGenkitToolFromToolbox converts a custom ToolboxTool into a genkit ai.Tool
func NewGenkitToolFromToolbox(tool *core.ToolboxTool, g *genkit.Genkit) ai.Tool {
	jsonBytes, err := tool.InputSchema()
	if err != nil {
		return nil
	}
	var schema *jsonschema.Schema
	if err := json.Unmarshal(jsonBytes, &schema); err != nil {
		return nil
	}

	// Create a wrapper execution function with the necessary signature
	executeFn := func(ctx *ai.ToolContext, input any) (string, error) {
		result, err := tool.Invoke(ctx, input.(map[string]any))
		if err != nil {
			// Propagate errors from the tool invocation.
			return "", err
		}

		return result.(string), nil
	}

	// Create a Genkit Tool
	return genkit.DefineToolWithInputSchema(
		g,
		tool.Name(),
		tool.Description(),
		schema,
		executeFn,
	)
}

func main() {
	ctx := context.Background()
	toolboxClient, err := core.NewToolboxClient("http://127.0.0.1:5000")
	if err != nil {
		log.Fatalf("Failed to create Toolbox client: %v", err)
	}

	// Load the tool using the MCP Toolbox SDK.
	searchHotelTool, err := toolboxClient.LoadTool("search-hotels-by-name", ctx)
	if err != nil {
		log.Fatalf("Failed to load tools: %v\nMake sure your Toolbox server is running and the tool is configured.", err)
	}

	g, err := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}),
		genkit.WithDefaultModel("googleai/gemini-1.5-flash"), // Updated model name
	)
	if err != nil {
		log.Fatalf("Failed to init genkit: %v\n", err)
	}

	// Convert your tool to a Genkit tool.
	genkitTool := NewGenkitToolFromToolbox(searchHotelTool, g)

	// Generate llm response using prompts and tools.
	resp, err := genkit.Generate(ctx, g,
		ai.WithPrompt("Find hotels in Basel with Basel in it's name."),
		ai.WithTools(genkitTool),
	)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	fmt.Println(resp.Text())
}
