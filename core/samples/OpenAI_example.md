This sample contains a complete example on how to integrate MCP Toolbox Go Core SDK with the OpenAI Go SDK.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	openai "github.com/openai/openai-go"
)

// ConvertToOpenAITool converts a ToolboxTool into the go-openai library's Tool format.
func ConvertToOpenAITool(toolboxTool *core.ToolboxTool) (openai.ChatCompletionToolParam, error) {
	// Get the input schema
	jsonSchemaBytes, err := toolboxTool.InputSchema()
	if err != nil {
		return openai.ChatCompletionToolParam{}, err
	}

	// Unmarshal the JSON bytes into FunctionParameters
	var paramsSchema openai.FunctionParameters
	if err := json.Unmarshal(jsonSchemaBytes, &paramsSchema); err != nil {
		return openai.ChatCompletionToolParam{}, err
	}

	// Create and return the final tool parameter struct.
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        toolboxTool.Name(),
			Description: openai.String(toolboxTool.Description()),
			Parameters:  paramsSchema,
		},
	}, nil
}

func main() {
	// Setup
	ctx := context.Background()
	toolboxURL := "http://localhost:5000"
	openAIClient := openai.NewClient()

	// Initialize the MCP Toolbox client.
	toolboxClient, err := core.NewToolboxClient(toolboxURL)
	if err != nil {
		log.Fatalf("Failed to create Toolbox client: %v", err)
	}

	// --- 2. Load and Convert the Tool ---
	toolToLoad := "search-hotels-by-name"
	log.Printf("Attempting to load tool '%s' from Toolbox at %s...\n", toolToLoad, toolboxURL)

	// Load the tool using the MCP Toolbox SDK.
	searchHotelTool, err := toolboxClient.LoadTool(toolToLoad, ctx)
	if err != nil {
		log.Fatalf("Failed to load tool '%s': %v\nMake sure your Toolbox server is running and the tool is configured.", toolToLoad, err)
	}
	log.Printf("Successfully loaded tool: %s\n", searchHotelTool.Name())

	// Convert the Toolbox tool into the OpenAI FunctionDeclaration format.
	openAITool, err := ConvertToOpenAITool(searchHotelTool)
	if err != nil {
		log.Fatal("Unable to convert to OpenAI tool", err)
	}

	question := "Find hotels in Basel with Basel in it's name "

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(question),
		},
		Tools: []openai.ChatCompletionToolParam{openAITool},
		Seed:  openai.Int(0),
		Model: openai.ChatModelGPT4o,
	}

	log.Println("start")
	// Make initial chat completion request
	completion, err := openAIClient.Chat.Completions.New(ctx, params)
	if err != nil {
		panic(err)
	}

	log.Println("start1")
	toolCalls := completion.Choices[0].Message.ToolCalls

	// Return early if there are no tool calls
	if len(toolCalls) == 0 {
		fmt.Printf("No function call")
		return
	}

	// If there is a was a function call, continue the conversation
	params.Messages = append(params.Messages, completion.Choices[0].Message.ToParam())
	for _, toolCall := range toolCalls {
		if toolCall.Function.Name == "search-hotels-by-name" {
			// Extract the location from the function call arguments
			var args map[string]interface{}
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
			if err != nil {
				panic(err)
			}

			// Invoke the tool
			result, err := searchHotelTool.Invoke(ctx, args)
			if err != nil {
				log.Fatal("Could not invoke tool", err)
			}

			params.Messages = append(params.Messages, openai.ToolMessage(result.(string), toolCall.ID))
		}
	}

	completion, err = openAIClient.Chat.Completions.New(ctx, params)
	if err != nil {
		panic(err)
	}

	fmt.Println(completion.Choices[0].Message.Content)
}
```