# Testing Guide

This document provides comprehensive instructions for testing the go-mcp-commander MCP server.

## Quick Start

```bash
# Run all tests
go test -v ./...

# Run unit tests only
go test -v ./pkg/...

# Run integration tests
go test -v -tags=integration ./test/...

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Test Structure

```
go-mcp-commander/
├── pkg/
│   ├── mcp/
│   │   └── server_test.go      # MCP server unit tests
│   ├── logging/
│   │   └── logging_test.go     # Logging unit tests
│   └── commander/
│       └── commander_test.go   # Command execution unit tests
└── test/
    └── mcp_test.go             # MCP protocol integration tests
```

## Unit Tests

### MCP Server Tests (`pkg/mcp/server_test.go`)

Tests core MCP server functionality:

- `TestNewServer` - Server initialization
- `TestRegisterTool` - Tool registration
- `TestHandleInitialize` - Initialize request handling
- `TestHandleListTools` - List tools request
- `TestHandleCallTool` - Tool invocation
- `TestHandleCallTool_UnknownTool` - Unknown tool handling
- `TestHandlePing` - Ping request
- `TestHandleUnknownMethod` - Unknown method error
- `TestHandleNotification` - Notification handling
- `TestHandleParseError` - JSON parse error handling
- `TestServerRun` - Full server run loop

```bash
go test -v ./pkg/mcp/...
```

### Commander Tests (`pkg/commander/commander_test.go`)

Tests command execution with security controls:

- `TestNewCommander` - Commander initialization
- `TestValidateCommand_AllowedEmpty` - Allow all when no allowlist
- `TestValidateCommand_AllowedList` - Allowlist enforcement
- `TestValidateCommand_BlockedList` - Blocklist enforcement
- `TestValidateCommand_CaseInsensitive` - Case-insensitive matching
- `TestExecute_SimpleCommand` - Basic command execution
- `TestExecute_Timeout` - Command timeout handling
- `TestExecute_WorkingDirectory` - Working directory support
- `TestExecute_InvalidWorkingDirectory` - Invalid directory error
- `TestExecute_EnvironmentVariables` - Environment variable passing
- `TestExecute_FailingCommand` - Non-zero exit code handling
- `TestGetCommandName` - Command name extraction
- `TestDefaultBlockedCommands` - Default blocklist population

```bash
go test -v ./pkg/commander/...
```

### Logging Tests (`pkg/logging/logging_test.go`)

Tests logging functionality:

- `TestParseLogLevel` - Log level parsing
- `TestLogLevelString` - Log level string conversion
- `TestNewLogger` - Logger initialization
- `TestLoggerLevels` - Log level filtering
- `TestLoggerCommandExec` - Command execution logging
- `TestLoggerCommandBlocked` - Blocked command logging
- `TestLoggerToolCall` - Tool call logging (security: no value logging)
- `TestLogStartupAndShutdown` - Startup/shutdown logging
- `TestDefaultLogDir` - Default log directory calculation
- `TestLoggerSetLevel` - Dynamic level change
- `TestLoggerClose` - Resource cleanup
- `TestLogFileNaming` - Log file naming convention

```bash
go test -v ./pkg/logging/...
```

## Integration Tests

### MCP Protocol Tests (`test/mcp_test.go`)

Tests full MCP protocol compliance:

- `TestMCP_Initialize` - MCP initialization handshake
- `TestMCP_ListTools` - Tool listing via protocol
- `TestMCP_ExecuteCommand` - Command execution via MCP
- `TestMCP_ExecuteCommand_Blocked` - Blocked command handling via MCP
- `TestMCP_Ping` - MCP ping request
- `TestMCP_GetShellInfo` - Shell info retrieval

Run integration tests:

```bash
# Build binary first
go build -o go-mcp-commander .

# Run integration tests
go test -v -tags=integration ./test/...

# Or run MCP-specific tests
go test -v -tags=mcp ./test/...
```

## Manual Testing

### Test with JSON-RPC

You can manually test the server using stdin/stdout:

```bash
# Build and run the server
go build -o go-mcp-commander .
./go-mcp-commander -log-level debug
```

Then send JSON-RPC requests:

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
```

```json
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
```

```json
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"echo hello"}}}
```

### Test Script

Create a test script (`test_manual.sh`):

```bash
#!/bin/bash

echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./go-mcp-commander -log-level off

echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./go-mcp-commander -log-level off

echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"echo hello world"}}}' | ./go-mcp-commander -log-level off
```

## Claude Code Validation

### Setup

1. Build the binary:
```bash
go build -o go-mcp-commander.exe .
```

2. Ensure `.mcp.json` is configured:
```json
{
  "mcpServers": {
    "commander": {
      "command": "C:\\dev\\go-mcp-commander\\go-mcp-commander.exe",
      "args": ["-log-level", "info"]
    }
  }
}
```

### Test with Claude Code

1. Open a new Claude Code session in the project directory:
```bash
cd C:\dev\go-mcp-commander
claude
```

2. Ask Claude to use the commander tool:
```
Use the commander MCP server to run "echo hello from MCP"
```

3. Expected behavior:
   - Claude should recognize the `execute_command` tool
   - Execute the echo command
   - Return the output

### Automated Claude Code Test

Create a test prompt file (`test_claude_code.txt`):
```
Please test the go-mcp-commander MCP server by:
1. Listing available tools
2. Getting shell info
3. Executing "echo test"
4. Verifying the blocked command "rm -rf /" is rejected
```

Run with Claude Code:
```bash
claude -p "$(cat test_claude_code.txt)"
```

## Continue.dev Validation

### Setup

1. Ensure the binary is built
2. Configuration in `.continue/config.json` should be present

### Test with Continue.dev CLI (cn)

```bash
# List MCP servers
cn mcp list

# Test the commander tool
cn mcp call go-mcp-commander execute_command --command "echo hello"

# Test shell info
cn mcp call go-mcp-commander get_shell_info

# Test blocked commands
cn mcp call go-mcp-commander list_blocked_commands
```

## CI/CD Testing

The GitHub Actions workflow runs:

1. **Lint** (`ci.yml`):
   - `go vet ./...`
   - `gofmt` check

2. **Unit Tests** (`ci.yml`):
   - Runs on Ubuntu, macOS, Windows
   - Go versions 1.21 and 1.22
   - Race detection enabled
   - Coverage reporting

3. **Integration Tests** (`ci.yml`):
   - MCP protocol validation
   - Cross-platform binary testing

4. **Build** (`release.yml`):
   - Multi-platform binaries
   - Checksum generation

## Security Testing

### Test Blocklist

```bash
# These should all be blocked
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"rm -rf /"}}}' | ./go-mcp-commander -log-level off

echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"mkfs.ext4 /dev/sda"}}}' | ./go-mcp-commander -log-level off
```

### Test Allowlist

```bash
# Start with allowlist
./go-mcp-commander -allowed-commands "echo,ls"

# This should work
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"echo hello"}}}' | ./go-mcp-commander -allowed-commands "echo,ls" -log-level off

# This should be blocked (not in allowlist)
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"cat /etc/passwd"}}}' | ./go-mcp-commander -allowed-commands "echo,ls" -log-level off
```

## Performance Testing

### Benchmark Tests

Add benchmark tests in `*_test.go` files:

```go
func BenchmarkExecuteCommand(b *testing.B) {
    cmd := NewCommander(Config{})
    ctx := context.Background()

    for i := 0; i < b.N; i++ {
        cmd.Execute(ctx, "echo test", "", 0, nil)
    }
}
```

Run benchmarks:

```bash
go test -bench=. -benchmem ./pkg/commander/...
```

## Troubleshooting

### Common Issues

1. **Binary not found**: Ensure you've built the binary before running integration tests
   ```bash
   go build -o go-mcp-commander .
   ```

2. **Permission denied**: Make binary executable (Unix)
   ```bash
   chmod +x ./go-mcp-commander
   ```

3. **Tests timeout**: Increase timeout
   ```bash
   go test -v -timeout 120s ./...
   ```

4. **Integration tests fail on Windows**: Use `.exe` extension
   ```bash
   go build -o go-mcp-commander.exe .
   ```

### Debug Mode

Enable debug logging:

```bash
./go-mcp-commander -log-level debug
```

Check log files:
```bash
cat ~/go-mcp-commander/logs/go-mcp-commander-*.log
```

## Test Coverage Goals

| Package | Target Coverage |
|---------|-----------------|
| pkg/mcp | 80%+ |
| pkg/commander | 90%+ |
| pkg/logging | 75%+ |

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```
