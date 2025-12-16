package v20241105

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      any    `json:"id,omitempty"`     // string or int
	Params  any    `json:"params,omitempty"` // map or struct
}

type jsonRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type clientCapabilities map[string]any

type serverCapabilities struct {
	Prompts map[string]any `json:"prompts,omitempty"`
	Tools   map[string]any `json:"tools,omitempty"`
}

type initializeRequestParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    clientCapabilities `json:"capabilities"`
	ClientInfo      implementation     `json:"clientInfo"`
}

type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    serverCapabilities `json:"capabilities"`
	ServerInfo      implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

type listToolsResult struct {
	Tools []map[string]any `json:"tools"`
}

type callToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callToolResult struct {
	Content []textContent `json:"content"`
	IsError bool          `json:"isError"`
}
