//go:build integration || mcp

package test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// MCPRequest represents a JSON-RPC request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents a JSON-RPC response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func getBinaryPath() string {
	if runtime.GOOS == "windows" {
		return "./go-mcp-commander.exe"
	}
	return "./go-mcp-commander"
}

func buildBinary(t *testing.T) {
	t.Helper()

	cmd := exec.Command("go", "build", "-o", getBinaryPath(), ".")
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}
}

func runMCPServer(t *testing.T, input string, timeout time.Duration) (string, error) {
	t.Helper()

	binaryPath := getBinaryPath()
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try from parent directory
		binaryPath = "../" + getBinaryPath()
	}

	cmd := exec.Command(binaryPath, "-log-level", "off")
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return "", err
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "exit status") {
			return stdout.String(), err
		}
		return stdout.String(), nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return stdout.String(), nil
	}
}

func TestMCP_Initialize(t *testing.T) {
	buildBinary(t)

	request := MCPRequest{
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

	requestData, _ := json.Marshal(request)
	input := string(requestData) + "\n"

	output, err := runMCPServer(t, input, 5*time.Second)
	if err != nil {
		t.Fatalf("Server error: %v", err)
	}

	// Parse response
	var response MCPResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &response); err != nil {
		t.Fatalf("Failed to parse response: %v\nOutput: %s", err, output)
	}

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected serverInfo in result")
	}

	if serverInfo["name"] != "go-mcp-commander" {
		t.Errorf("Expected server name 'go-mcp-commander', got %v", serverInfo["name"])
	}
}

func TestMCP_ListTools(t *testing.T) {
	buildBinary(t)

	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	requestData, _ := json.Marshal(request)
	input := string(requestData) + "\n"

	output, err := runMCPServer(t, input, 5*time.Second)
	if err != nil {
		t.Fatalf("Server error: %v", err)
	}

	var response MCPResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &response); err != nil {
		t.Fatalf("Failed to parse response: %v\nOutput: %s", err, output)
	}

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatal("Expected tools array in result")
	}

	// Should have at least the execute_command tool
	if len(tools) < 1 {
		t.Error("Expected at least 1 tool")
	}

	// Find execute_command tool
	found := false
	for _, tool := range tools {
		toolMap, ok := tool.(map[string]interface{})
		if ok && toolMap["name"] == "execute_command" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected 'execute_command' tool to be registered")
	}
}

func TestMCP_ExecuteCommand(t *testing.T) {
	buildBinary(t)

	var command string
	if runtime.GOOS == "windows" {
		command = "echo hello world"
	} else {
		command = "echo hello world"
	}

	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "execute_command",
			"arguments": map[string]interface{}{
				"command": command,
			},
		},
	}

	requestData, _ := json.Marshal(request)
	input := string(requestData) + "\n"

	output, err := runMCPServer(t, input, 10*time.Second)
	if err != nil {
		t.Fatalf("Server error: %v", err)
	}

	var response MCPResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &response); err != nil {
		t.Fatalf("Failed to parse response: %v\nOutput: %s", err, output)
	}

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("Expected content in result")
	}

	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected content item to be a map")
	}

	text, ok := contentItem["text"].(string)
	if !ok {
		t.Fatal("Expected text in content item")
	}

	if !strings.Contains(text, "hello world") {
		t.Errorf("Expected output to contain 'hello world', got %s", text)
	}
}

func TestMCP_ExecuteCommand_Blocked(t *testing.T) {
	buildBinary(t)

	var command string
	if runtime.GOOS == "windows" {
		command = "format c:"
	} else {
		command = "rm -rf /"
	}

	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "execute_command",
			"arguments": map[string]interface{}{
				"command": command,
			},
		},
	}

	requestData, _ := json.Marshal(request)
	input := string(requestData) + "\n"

	output, err := runMCPServer(t, input, 5*time.Second)
	if err != nil {
		t.Fatalf("Server error: %v", err)
	}

	var response MCPResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &response); err != nil {
		t.Fatalf("Failed to parse response: %v\nOutput: %s", err, output)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	// Should have isError = true
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("Expected command to be blocked (isError = true)")
	}
}

func TestMCP_Ping(t *testing.T) {
	buildBinary(t)

	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "ping",
	}

	requestData, _ := json.Marshal(request)
	input := string(requestData) + "\n"

	output, err := runMCPServer(t, input, 5*time.Second)
	if err != nil {
		t.Fatalf("Server error: %v", err)
	}

	var response MCPResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &response); err != nil {
		t.Fatalf("Failed to parse response: %v\nOutput: %s", err, output)
	}

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}
}

func TestMCP_GetShellInfo(t *testing.T) {
	buildBinary(t)

	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "get_shell_info",
			"arguments": map[string]interface{}{},
		},
	}

	requestData, _ := json.Marshal(request)
	input := string(requestData) + "\n"

	output, err := runMCPServer(t, input, 5*time.Second)
	if err != nil {
		t.Fatalf("Server error: %v", err)
	}

	var response MCPResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &response); err != nil {
		t.Fatalf("Failed to parse response: %v\nOutput: %s", err, output)
	}

	if response.Error != nil {
		t.Fatalf("Unexpected error: %v", response.Error)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("Expected content in result")
	}

	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected content item to be a map")
	}

	text, ok := contentItem["text"].(string)
	if !ok {
		t.Fatal("Expected text in content item")
	}

	if runtime.GOOS == "windows" {
		if !strings.Contains(text, "cmd") {
			t.Errorf("Expected 'cmd' in shell info for Windows, got %s", text)
		}
	} else {
		if !strings.Contains(text, "sh") {
			t.Errorf("Expected 'sh' in shell info for Unix, got %s", text)
		}
	}
}
