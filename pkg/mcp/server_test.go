package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewServer(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.name != "test-server" {
		t.Errorf("Expected name 'test-server', got %s", server.name)
	}

	if server.version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", server.version)
	}
}

func TestRegisterTool(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"param1": {
					Type:        "string",
					Description: "A string parameter",
				},
			},
			Required: []string{"param1"},
		},
	}

	handler := func(args map[string]interface{}) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []ContentItem{{Type: "text", Text: "success"}},
		}, nil
	}

	server.RegisterTool(tool, handler)

	if len(server.tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(server.tools))
	}

	if _, exists := server.handlers["test_tool"]; !exists {
		t.Error("Expected handler to be registered")
	}
}

func TestHandleInitialize(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	data, _ := json.Marshal(request)
	response := server.handleMessage(data)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error != nil {
		t.Errorf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(*InitializeResult)
	if !ok {
		t.Fatal("Expected InitializeResult")
	}

	if result.ServerInfo.Name != "test-server" {
		t.Errorf("Expected server name 'test-server', got %s", result.ServerInfo.Name)
	}

	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("Expected protocol version '2024-11-05', got %s", result.ProtocolVersion)
	}
}

func TestHandleListTools(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register a test tool
	server.RegisterTool(Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: JSONSchema{Type: "object"},
	}, func(args map[string]interface{}) (*CallToolResult, error) {
		return &CallToolResult{Content: []ContentItem{{Type: "text", Text: "ok"}}}, nil
	})

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	data, _ := json.Marshal(request)
	response := server.handleMessage(data)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error != nil {
		t.Errorf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(*ListToolsResult)
	if !ok {
		t.Fatal("Expected ListToolsResult")
	}

	if len(result.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(result.Tools))
	}

	if result.Tools[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %s", result.Tools[0].Name)
	}
}

func TestHandleCallTool(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register a test tool
	server.RegisterTool(Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"message": {Type: "string"},
			},
		},
	}, func(args map[string]interface{}) (*CallToolResult, error) {
		msg, _ := args["message"].(string)
		return &CallToolResult{
			Content: []ContentItem{{Type: "text", Text: "received: " + msg}},
		}, nil
	})

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "test_tool",
			"arguments": map[string]interface{}{
				"message": "hello world",
			},
		},
	}

	data, _ := json.Marshal(request)
	response := server.handleMessage(data)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error != nil {
		t.Errorf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(*CallToolResult)
	if !ok {
		t.Fatal("Expected CallToolResult")
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(result.Content))
	}

	if result.Content[0].Text != "received: hello world" {
		t.Errorf("Unexpected result text: %s", result.Content[0].Text)
	}
}

func TestHandleCallTool_UnknownTool(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "nonexistent_tool",
			"arguments": map[string]interface{}{},
		},
	}

	data, _ := json.Marshal(request)
	response := server.handleMessage(data)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	result, ok := response.Result.(*CallToolResult)
	if !ok {
		t.Fatal("Expected CallToolResult")
	}

	if !result.IsError {
		t.Error("Expected IsError to be true for unknown tool")
	}
}

func TestHandlePing(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "ping",
	}

	data, _ := json.Marshal(request)
	response := server.handleMessage(data)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error != nil {
		t.Errorf("Unexpected error: %v", response.Error)
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	}

	data, _ := json.Marshal(request)
	response := server.handleMessage(data)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error == nil {
		t.Fatal("Expected error for unknown method")
	}

	if response.Error.Code != MethodNotFound {
		t.Errorf("Expected MethodNotFound error code, got %d", response.Error.Code)
	}
}

func TestHandleNotification(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	var stderr bytes.Buffer
	server.SetIO(nil, nil, &stderr)

	// Notification has no ID
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	data, _ := json.Marshal(request)
	response := server.handleMessage(data)

	// Notifications should not return a response
	if response != nil {
		t.Error("Expected nil response for notification")
	}
}

func TestHandleParseError(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	response := server.handleMessage([]byte("invalid json"))

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error == nil {
		t.Fatal("Expected parse error")
	}

	if response.Error.Code != ParseError {
		t.Errorf("Expected ParseError code, got %d", response.Error.Code)
	}
}

func TestServerRun(t *testing.T) {
	server := NewServer("test-server", "1.0.0")

	// Register a simple tool
	server.RegisterTool(Tool{
		Name:        "echo",
		Description: "Echo tool",
		InputSchema: JSONSchema{Type: "object"},
	}, func(args map[string]interface{}) (*CallToolResult, error) {
		return &CallToolResult{
			Content: []ContentItem{{Type: "text", Text: "echo response"}},
		}, nil
	})

	// Prepare input
	initRequest := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}
	initData, _ := json.Marshal(initRequest)

	listRequest := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	listData, _ := json.Marshal(listRequest)

	input := string(initData) + "\n" + string(listData) + "\n"

	var stdout, stderr bytes.Buffer
	server.SetIO(strings.NewReader(input), &stdout, &stderr)

	// Run server (it will exit when stdin is exhausted)
	err := server.Run()
	if err != nil {
		t.Errorf("Server.Run() returned error: %v", err)
	}

	// Check output contains responses
	output := stdout.String()
	if !strings.Contains(output, "test-server") {
		t.Error("Expected server name in output")
	}
	if !strings.Contains(output, "echo") {
		t.Error("Expected tool name in output")
	}
}
