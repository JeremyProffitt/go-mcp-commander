package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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
	// Register execute_command tool
	server.RegisterTool(mcp.Tool{
		Name:        "execute_command",
		Description: "Execute a system command and return its output. Supports timeout, working directory, and environment variables.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"command": {
					Type:        "string",
					Description: "The command to execute",
				},
				"working_directory": {
					Type:        "string",
					Description: "Working directory for command execution (optional)",
				},
				"timeout": {
					Type:        "string",
					Description: "Timeout duration (e.g., '30s', '5m'). Default is 30s",
				},
				"env": {
					Type:        "object",
					Description: "Environment variables to set for the command (optional)",
				},
			},
			Required: []string{"command"},
		},
	}, handleExecuteCommand)

	// Register list_allowed_commands tool
	server.RegisterTool(mcp.Tool{
		Name:        "list_allowed_commands",
		Description: "List all allowed command patterns. If empty, all commands are allowed (except blocked ones).",
		InputSchema: mcp.JSONSchema{
			Type:       "object",
			Properties: map[string]mcp.Property{},
		},
	}, handleListAllowedCommands)

	// Register list_blocked_commands tool
	server.RegisterTool(mcp.Tool{
		Name:        "list_blocked_commands",
		Description: "List all blocked command patterns.",
		InputSchema: mcp.JSONSchema{
			Type:       "object",
			Properties: map[string]mcp.Property{},
		},
	}, handleListBlockedCommands)

	// Register get_shell_info tool
	server.RegisterTool(mcp.Tool{
		Name:        "get_shell_info",
		Description: "Get information about the shell used for command execution.",
		InputSchema: mcp.JSONSchema{
			Type:       "object",
			Properties: map[string]mcp.Property{},
		},
	}, handleGetShellInfo)
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
