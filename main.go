package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/user/go-mcp-commander/pkg/commander"
	"github.com/user/go-mcp-commander/pkg/logging"
	"github.com/user/go-mcp-commander/pkg/mcp"
)

const (
	Version = "1.0.0"
)

var (
	// Command-line flags
	logDir              = flag.String("log-dir", "", "Directory for log files")
	logLevel            = flag.String("log-level", "info", "Log level: off|error|warn|info|access|debug")
	allowedCmds         = flag.String("allowed-commands", "", "Comma-separated list of allowed command prefixes (empty = allow all)")
	blockedCmds         = flag.String("blocked-commands", "", "Comma-separated list of blocked command patterns")
	defaultTimeout      = flag.Duration("timeout", 30*time.Second, "Default command timeout")
	shell               = flag.String("shell", "", "Shell to use for command execution (default: /bin/sh on Unix, cmd on Windows)")
	shellArg            = flag.String("shell-arg", "", "Shell argument for command execution (default: -c on Unix, /c on Windows)")
	useDefaultBlocklist = flag.Bool("use-default-blocklist", true, "Use default blocklist of dangerous commands")
	httpMode            = flag.Bool("http", false, "Run in HTTP mode instead of stdio")
	httpPort            = flag.Int("port", 3000, "HTTP port (only used with --http)")
	httpHost            = flag.String("host", "127.0.0.1", "HTTP host (only used with --http)")

	// Global variables
	logger *logging.Logger
	cmd    *commander.Commander
)

func main() {
	// Load environment variables from ~/.mcp_env if it exists
	// This must happen before flag parsing so env vars are available for defaults
	logging.LoadEnvFile()

	flag.Parse()

	// Resolve configuration with priority: flags > env vars > defaults
	resolvedLogDir := resolvePriority(*logDir, os.Getenv("MCP_LOG_DIR"), "")
	resolvedLogLevel := resolvePriority(*logLevel, os.Getenv("MCP_LOG_LEVEL"), "info")
	resolvedAllowedCmds := resolvePriority(*allowedCmds, os.Getenv("MCP_ALLOWED_COMMANDS"), "")
	resolvedBlockedCmds := resolvePriority(*blockedCmds, os.Getenv("MCP_BLOCKED_COMMANDS"), "")
	resolvedTimeout := *defaultTimeout
	if envTimeout := os.Getenv("MCP_DEFAULT_TIMEOUT"); envTimeout != "" {
		if parsed, err := time.ParseDuration(envTimeout); err == nil {
			resolvedTimeout = parsed
		}
	}
	resolvedShell := resolvePriority(*shell, os.Getenv("MCP_SHELL"), "")
	resolvedShellArg := resolvePriority(*shellArg, os.Getenv("MCP_SHELL_ARG"), "")

	// Determine if we should add app subfolder (when log dir was specified by user)
	addAppSubfolder := *logDir != "" || os.Getenv("MCP_LOG_DIR") != ""

	// Initialize logger
	var err error
	logger, err = logging.NewLogger(logging.Config{
		LogDir:          resolvedLogDir,
		AppName:         "go-mcp-commander",
		Level:           logging.ParseLogLevel(resolvedLogLevel),
		AddAppSubfolder: addAppSubfolder,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Parse allowed/blocked commands
	var allowedList, blockedList []string
	if resolvedAllowedCmds != "" {
		allowedList = parseCommandList(resolvedAllowedCmds)
	}
	if resolvedBlockedCmds != "" {
		blockedList = parseCommandList(resolvedBlockedCmds)
	}
	if *useDefaultBlocklist {
		blockedList = append(blockedList, commander.DefaultBlockedCommands()...)
	}

	// Initialize commander
	cmdConfig := commander.Config{
		AllowedCommands: allowedList,
		BlockedCommands: blockedList,
		DefaultTimeout:  resolvedTimeout,
		Shell:           resolvedShell,
		ShellArg:        resolvedShellArg,
	}
	cmd = commander.NewCommander(cmdConfig)

	// Get shell info for logging
	shellInfo, shellArgInfo := cmd.GetShellInfo()

	// Log startup information
	startupInfo := logging.GetStartupInfo(
		Version,
		getConfigValue(resolvedLogDir, *logDir, os.Getenv("MCP_LOG_DIR")),
		getConfigValue(resolvedLogLevel, *logLevel, os.Getenv("MCP_LOG_LEVEL")),
		getConfigValue(resolvedAllowedCmds, *allowedCmds, os.Getenv("MCP_ALLOWED_COMMANDS")),
		getConfigValue(strings.Join(blockedList, ","), *blockedCmds, os.Getenv("MCP_BLOCKED_COMMANDS")),
		getConfigValue(resolvedTimeout.String(), defaultTimeout.String(), os.Getenv("MCP_DEFAULT_TIMEOUT")),
		getConfigValue(shellInfo+" "+shellArgInfo, *shell, os.Getenv("MCP_SHELL")),
	)
	logger.LogStartup(startupInfo)

	// Create MCP server
	server := mcp.NewServer("go-mcp-commander", Version)

	// Register tools
	registerTools(server)

	// Run server
	logger.Info("MCP server starting...")
	if *httpMode {
		addr := fmt.Sprintf("%s:%d", *httpHost, *httpPort)
		logger.Info("Starting HTTP server on %s", addr)
		if err := server.RunHTTP(addr); err != nil {
			logger.Error("HTTP server error: %v", err)
			logger.LogShutdown(fmt.Sprintf("error: %v", err))
			os.Exit(1)
		}
	} else {
		if err := server.Run(); err != nil {
			logger.Error("Server error: %v", err)
			logger.LogShutdown(fmt.Sprintf("error: %v", err))
			os.Exit(1)
		}
	}

	logger.LogShutdown("normal")
}

func registerTools(server *mcp.Server) {
	// Helper for creating bool pointers
	boolPtr := func(b bool) *bool { return &b }
	// Helper for creating int pointers
	intPtr := func(i int) *int { return &i }

	// Register execute_command tool
	server.RegisterTool(mcp.Tool{
		Name:        "execute_command",
		Description: "Execute a system command and return its output. Commands are validated against allow/block lists before execution - use list_allowed_commands and list_blocked_commands to check what's permitted. Supports timeout, working directory, and environment variables.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"command": {
					Type:        "string",
					Description: "The command to execute. Will be validated against configured allow/block lists before execution.",
				},
				"working_directory": {
					Type:        "string",
					Description: "Working directory for command execution. If relative, resolved relative to server's current working directory. If not specified, uses the server's current working directory.",
				},
				"timeout": {
					Type:        "string",
					Description: "Timeout duration in Go duration format. Valid examples: '30s' (30 seconds), '1m' (1 minute), '5m' (5 minutes), '1h' (1 hour), '1m30s' (1 minute 30 seconds). Default is 30s. Maximum recommended: 1h.",
				},
				"env": {
					Type:        "object",
					Description: "Environment variables as key-value pairs (e.g., {\"NODE_ENV\": \"production\", \"DEBUG\": \"true\"}). These are added to the command's environment, supplementing (not replacing) existing environment variables.",
					Properties:  map[string]mcp.Property{},
				},
			},
			Required: []string{"command"},
		},
		Annotations: &mcp.ToolAnnotations{
			Title:           "Execute Command",
			ReadOnlyHint:    boolPtr(false),
			DestructiveHint: boolPtr(true),
			IdempotentHint:  boolPtr(false),
			OpenWorldHint:   boolPtr(true),
		},
	}, handleExecuteCommand)

	// Register list_allowed_commands tool
	server.RegisterTool(mcp.Tool{
		Name:        "list_allowed_commands",
		Description: "List all allowed command patterns configured for this server. Use this tool before execute_command to verify if a command will be permitted. If the list is empty, all commands are allowed (except those matching blocked patterns). Commands must match at least one allowed pattern (prefix match) to execute.",
		InputSchema: mcp.JSONSchema{
			Type:       "object",
			Properties: map[string]mcp.Property{},
		},
		Annotations: &mcp.ToolAnnotations{
			Title:          "List Allowed Commands",
			ReadOnlyHint:   boolPtr(true),
			IdempotentHint: boolPtr(true),
		},
	}, handleListAllowedCommands)

	// Register list_blocked_commands tool
	server.RegisterTool(mcp.Tool{
		Name:        "list_blocked_commands",
		Description: "List all blocked command patterns configured for this server. Commands matching any blocked pattern will be rejected with an error, even if they match an allowed pattern. Blocked patterns take precedence over allowed patterns. Use this to understand what commands are prohibited before attempting execution.",
		InputSchema: mcp.JSONSchema{
			Type:       "object",
			Properties: map[string]mcp.Property{},
		},
		Annotations: &mcp.ToolAnnotations{
			Title:          "List Blocked Commands",
			ReadOnlyHint:   boolPtr(true),
			IdempotentHint: boolPtr(true),
		},
	}, handleListBlockedCommands)

	// Register get_shell_info tool
	server.RegisterTool(mcp.Tool{
		Name:        "get_shell_info",
		Description: "Get information about the shell used for command execution, including the shell path, shell argument, and default timeout. Useful for understanding how commands will be interpreted and executed.",
		InputSchema: mcp.JSONSchema{
			Type:       "object",
			Properties: map[string]mcp.Property{},
		},
		Annotations: &mcp.ToolAnnotations{
			Title:          "Get Shell Info",
			ReadOnlyHint:   boolPtr(true),
			IdempotentHint: boolPtr(true),
		},
	}, handleGetShellInfo)

	// Register web_fetch tool
	server.RegisterTool(mcp.Tool{
		Name:        "web_fetch",
		Description: "Fetch content from a URL and return the response body. Supports HTTP/HTTPS. Returns raw HTML/text content. Use for retrieving web pages, APIs, or any HTTP resource. Timeout defaults to 30s.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"url": {
					Type:        "string",
					Description: "URL to fetch (e.g., 'https://example.com', 'https://api.github.com/users/octocat'). Must include protocol (http:// or https://).",
				},
				"method": {
					Type:        "string",
					Description: "HTTP method (default: 'GET'). Supported: GET, POST, PUT, DELETE, HEAD, OPTIONS.",
					Default:     "GET",
					Enum:        []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"},
				},
				"headers": {
					Type:        "object",
					Description: "HTTP headers as key-value pairs (e.g., {\"Authorization\": \"Bearer token\", \"Accept\": \"application/json\"}).",
					Properties:  map[string]mcp.Property{},
				},
				"body": {
					Type:        "string",
					Description: "Request body for POST/PUT requests. Use with appropriate Content-Type header.",
				},
				"timeout": {
					Type:        "string",
					Description: "Request timeout in Go duration format (e.g., '30s', '1m', '5m'). Default: 30s, max: 5m.",
					Default:     "30s",
				},
				"max_size": {
					Type:        "integer",
					Description: "Maximum response body size in bytes. Default: 1MB (1048576). Prevents memory issues with large responses.",
					Default:     1048576,
					Minimum:     intPtr(1024),
					Maximum:     intPtr(10485760),
				},
			},
			Required: []string{"url"},
		},
		Annotations: &mcp.ToolAnnotations{
			Title:          "Web Fetch",
			ReadOnlyHint:   boolPtr(false),
			IdempotentHint: boolPtr(false),
			OpenWorldHint:  boolPtr(true),
		},
	}, handleWebFetch)

	// Register google_search tool
	server.RegisterTool(mcp.Tool{
		Name:        "google_search",
		Description: "Perform a Google search and return the search results page HTML. Results can be parsed to extract links, snippets, and titles. For structured results, consider using the Google Custom Search API instead.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"query": {
					Type:        "string",
					Description: "Search query string (e.g., 'golang mcp server', 'site:github.com kubernetes').",
				},
				"num_results": {
					Type:        "integer",
					Description: "Number of results to request (10-100). Google may return fewer. Default: 10.",
					Default:     10,
					Minimum:     intPtr(10),
					Maximum:     intPtr(100),
				},
				"language": {
					Type:        "string",
					Description: "Language code for results (e.g., 'en', 'es', 'fr', 'de'). Default: 'en'.",
					Default:     "en",
				},
				"safe_search": {
					Type:        "string",
					Description: "Safe search filter level. Default: 'moderate'.",
					Default:     "moderate",
					Enum:        []string{"off", "moderate", "strict"},
				},
				"timeout": {
					Type:        "string",
					Description: "Request timeout in Go duration format (e.g., '30s', '1m'). Default: 30s.",
					Default:     "30s",
				},
			},
			Required: []string{"query"},
		},
		Annotations: &mcp.ToolAnnotations{
			Title:          "Google Search",
			ReadOnlyHint:   boolPtr(true),
			IdempotentHint: boolPtr(true),
			OpenWorldHint:  boolPtr(true),
		},
	}, handleGoogleSearch)
}

func handleExecuteCommand(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("execute_command", args)

	// Extract command
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return errorResult("command is required")
	}

	// Validate command
	if err := cmd.ValidateCommand(command); err != nil {
		logger.CommandBlocked(command, err.Error())
		return errorResult(fmt.Sprintf("Command validation failed: %s", err.Error()))
	}

	// Extract optional parameters
	workDir := getString(args, "working_directory", "")
	timeoutStr := getString(args, "timeout", "")
	envMap := getStringMap(args, "env")

	// Parse timeout
	var timeout time.Duration
	if timeoutStr != "" {
		var err error
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return errorResult(fmt.Sprintf("Invalid timeout format: %s", err.Error()))
		}
	}

	// Execute command
	result := cmd.Execute(context.Background(), command, workDir, timeout, envMap)

	// Log execution
	logger.CommandExec(command, workDir, result.ExitCode, result.Duration, result.Error)

	// Format response
	response := map[string]interface{}{
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"exit_code": result.ExitCode,
		"duration":  result.Duration.String(),
	}
	if result.Error != nil {
		response["error"] = result.Error.Error()
	}

	data, _ := json.MarshalIndent(response, "", "  ")

	// Return error result if command failed
	if result.ExitCode != 0 {
		return &mcp.CallToolResult{
			Content: []mcp.ContentItem{{Type: "text", Text: string(data)}},
			IsError: true,
		}, nil
	}

	return textResult(string(data))
}

func handleListAllowedCommands(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("list_allowed_commands", args)

	allowedStr := *allowedCmds
	if allowedStr == "" {
		allowedStr = os.Getenv("MCP_ALLOWED_COMMANDS")
	}

	var allowed []string
	if allowedStr != "" {
		allowed = parseCommandList(allowedStr)
	}

	response := map[string]interface{}{
		"allowed_commands": allowed,
		"allow_all":        len(allowed) == 0,
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return textResult(string(data))
}

func handleListBlockedCommands(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("list_blocked_commands", args)

	blockedStr := *blockedCmds
	if blockedStr == "" {
		blockedStr = os.Getenv("MCP_BLOCKED_COMMANDS")
	}

	var blocked []string
	if blockedStr != "" {
		blocked = parseCommandList(blockedStr)
	}
	if *useDefaultBlocklist {
		blocked = append(blocked, commander.DefaultBlockedCommands()...)
	}

	response := map[string]interface{}{
		"blocked_commands":        blocked,
		"using_default_blocklist": *useDefaultBlocklist,
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return textResult(string(data))
}

func handleGetShellInfo(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("get_shell_info", args)

	shell, shellArg := cmd.GetShellInfo()

	response := map[string]interface{}{
		"shell":           shell,
		"shell_arg":       shellArg,
		"default_timeout": cmd.GetDefaultTimeout().String(),
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return textResult(string(data))
}

func handleWebFetch(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("web_fetch", args)

	// Extract URL (required)
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return errorResult("url is required")
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid URL: %s", err.Error()))
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errorResult("URL must use http:// or https:// protocol")
	}

	// Extract optional parameters
	method := getString(args, "method", "GET")
	body := getString(args, "body", "")
	timeoutStr := getString(args, "timeout", "30s")
	maxSize := getInt(args, "max_size", 1048576) // 1MB default
	headers := getStringMap(args, "headers")

	// Parse timeout
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid timeout format: %s", err.Error()))
	}
	if timeout > 5*time.Minute {
		timeout = 5 * time.Minute
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, urlStr, reqBody)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to create request: %s", err.Error()))
	}

	// Set User-Agent to identify as bot
	req.Header.Set("User-Agent", "go-mcp-commander/1.0")

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, int64(maxSize))
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to read response: %s", err.Error()))
	}

	duration := time.Since(startTime)

	// Build response
	response := map[string]interface{}{
		"status_code":    resp.StatusCode,
		"status":         resp.Status,
		"content_length": len(respBody),
		"content_type":   resp.Header.Get("Content-Type"),
		"duration":       duration.String(),
		"body":           string(respBody),
	}

	// Add response headers
	respHeaders := make(map[string]string)
	for key := range resp.Header {
		respHeaders[key] = resp.Header.Get(key)
	}
	response["headers"] = respHeaders

	logger.Info("web_fetch: %s %s -> %d (%d bytes, %s)", method, urlStr, resp.StatusCode, len(respBody), duration)

	data, _ := json.MarshalIndent(response, "", "  ")

	// Return error result for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &mcp.CallToolResult{
			Content: []mcp.ContentItem{{Type: "text", Text: string(data)}},
			IsError: true,
		}, nil
	}

	return textResult(string(data))
}

func handleGoogleSearch(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("google_search", args)

	// Extract query (required)
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return errorResult("query is required")
	}

	// Extract optional parameters
	numResults := getInt(args, "num_results", 10)
	language := getString(args, "language", "en")
	safeSearch := getString(args, "safe_search", "moderate")
	timeoutStr := getString(args, "timeout", "30s")

	// Clamp num_results
	if numResults < 10 {
		numResults = 10
	}
	if numResults > 100 {
		numResults = 100
	}

	// Parse timeout
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		timeout = 30 * time.Second
	}

	// Build Google search URL
	searchURL := fmt.Sprintf("https://www.google.com/search?q=%s&num=%d&hl=%s&safe=%s",
		url.QueryEscape(query),
		numResults,
		url.QueryEscape(language),
		url.QueryEscape(safeSearch),
	)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to create request: %s", err.Error()))
	}

	// Set headers to appear as regular browser (Google blocks obvious bots)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", language+",en;q=0.5")

	// Execute request
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("Search request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	// Read response body (limit to 2MB for search results)
	limitedReader := io.LimitReader(resp.Body, 2*1024*1024)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return errorResult(fmt.Sprintf("Failed to read response: %s", err.Error()))
	}

	duration := time.Since(startTime)

	// Build response
	response := map[string]interface{}{
		"query":          query,
		"status_code":    resp.StatusCode,
		"content_length": len(respBody),
		"duration":       duration.String(),
		"search_url":     searchURL,
		"body":           string(respBody),
	}

	logger.Info("google_search: query=%q -> %d (%d bytes, %s)", query, resp.StatusCode, len(respBody), duration)

	data, _ := json.MarshalIndent(response, "", "  ")

	// Return error result for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &mcp.CallToolResult{
			Content: []mcp.ContentItem{{Type: "text", Text: string(data)}},
			IsError: true,
		}, nil
	}

	return textResult(string(data))
}

// Helper functions

func resolvePriority(flagVal, envVal, defaultVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if envVal != "" {
		return envVal
	}
	return defaultVal
}

func getConfigValue(resolved, flagVal, envVal string) logging.ConfigValue {
	if flagVal != "" && flagVal == resolved {
		return logging.ConfigValue{Value: resolved, Source: logging.SourceFlag}
	}
	if envVal != "" && envVal == resolved {
		return logging.ConfigValue{Value: resolved, Source: logging.SourceEnvironment}
	}
	return logging.ConfigValue{Value: resolved, Source: logging.SourceDefault}
}

func parseCommandList(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func getString(args map[string]interface{}, key, defaultVal string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return defaultVal
}

func getInt(args map[string]interface{}, key string, defaultVal int) int {
	if val, ok := args[key].(float64); ok {
		return int(val)
	}
	if val, ok := args[key].(int); ok {
		return val
	}
	return defaultVal
}

func getStringMap(args map[string]interface{}, key string) map[string]string {
	result := make(map[string]string)
	if val, ok := args[key].(map[string]interface{}); ok {
		for k, v := range val {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
	}
	return result
}

func textResult(text string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.ContentItem{{Type: "text", Text: text}},
	}, nil
}

func errorResult(message string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.ContentItem{{Type: "text", Text: message}},
		IsError: true,
	}, nil
}
