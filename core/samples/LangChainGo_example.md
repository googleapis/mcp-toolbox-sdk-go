package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
)

// ConvertToLangchainTool converts a generic core.ToolboxTool into a LangChainGo llms.Tool.
func ConvertToLangchainTool(toolboxTool *core.ToolboxTool) llms.Tool {

	// Fetch the tool's input schema
	inputschema, err := toolboxTool.InputSchema()
	if err != nil {
		return llms.Tool{}
	}

	var paramsSchema map[string]any
	_ = json.Unmarshal(inputschema, &paramsSchema)

	// Convert into LangChain's llms.Tool
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        toolboxTool.Name(),
			Description: toolboxTool.Description(),
			Parameters:  paramsSchema,
		},
	}
}

func main() {
	genaiKey := os.Getenv("GOOGLE_API_KEY")
	toolboxURL := "http://localhost:5000"
	ctx := context.Background()

	// Initialize the Google AI client (LLM).
	llm, err := googleai.New(ctx, googleai.WithAPIKey(genaiKey), googleai.WithDefaultModel("gemini-1.5-flash"))
	if err != nil {
		log.Fatalf("Failed to create Google AI client: %v", err)
	}

	// Initialize the MCP Toolbox client.
	toolboxClient, err := core.NewToolboxClient(toolboxURL)
	if err != nil {
		log.Fatalf("Failed to create Toolbox client: %v", err)
	}

	// Load the tool using the MCP Toolbox SDK.
	searchHotelTool, err := toolboxClient.LoadTool("search-hotels-by-name", ctx)
	if err != nil {
		log.Fatalf("Failed to load tools: %v\nMake sure your Toolbox server is running and the tool is configured.", err)
	}
	log.Printf("Successfully loaded tool: %s\n", searchHotelTool.Name())

	// Convert the loaded ToolboxTool into the format LangChainGo requires.
	langchainFormattedTool := ConvertToLangchainTool(searchHotelTool)

	// Start the conversation history.
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "Find hotels in Basel with Basel in it's name."),
	}

	log.Println("Asking the LLM with the provided tool...")

	// Make the first call to the LLM, making it aware of the tool.
	resp, err := llm.GenerateContent(ctx, messageHistory, llms.WithTools([]llms.Tool{langchainFormattedTool}))
	if err != nil {
		log.Fatalf("LLM call failed: %v", err)
	}

	// Add the model's response (which should be a tool call) to the history.
	respChoice := resp.Choices[0]
	assistantResponse := llms.TextParts(llms.ChatMessageTypeAI, respChoice.Content)
	for _, tc := range respChoice.ToolCalls {
		assistantResponse.Parts = append(assistantResponse.Parts, tc)
	}
	messageHistory = append(messageHistory, assistantResponse)

	// Process each tool call requested by the model.
	for _, tc := range respChoice.ToolCalls {
		toolName := tc.FunctionCall.Name

		switch tc.FunctionCall.Name {
		case "search-hotels-by-name":
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
				log.Fatalf("Failed to unmarshal arguments for tool '%s': %v", toolName, err)
			}
			toolResult, err := searchHotelTool.Invoke(ctx, args)
			if err != nil {
				log.Fatalf("Failed to execute tool '%s': %v", toolName, err)
			}

			// Create the tool call response message and add it to the history.
			toolResponse := llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{
					llms.ToolCallResponse{
						Name:    toolName,
						Content: fmt.Sprintf("%v", toolResult),
					},
				},
			}
			messageHistory = append(messageHistory, toolResponse)
		default:
			log.Fatalf("got unexpected function call: %v", tc.FunctionCall.Name)
		}
	}

	// Final LLM Call for Natural Language Response
	log.Println("Sending tool response back to LLM for a final answer...")

	// Call the LLM again with the updated history, which now includes the tool's result.
	finalResp, err := llm.GenerateContent(ctx, messageHistory)
	if err != nil {
		log.Fatalf("Final LLM call failed: %v", err)
	}

	// Display the Result
	fmt.Println("\n======================================")
	fmt.Println("Final Response from LLM:")
	fmt.Println(finalResp.Choices[0].Content)
	fmt.Println("======================================")
}
