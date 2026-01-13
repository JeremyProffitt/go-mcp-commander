# Testing Guide

This document provides comprehensive testing instructions for go-mcp-commander.

## Quick Start

| Command | Purpose |
|---------|---------|
| `go test -v ./...` | Run all tests |
| `go test -v ./pkg/...` | Unit tests only |
| `go test -v -tags=integration ./test/...` | Integration tests |
| `go test -race -coverprofile=coverage.out ./...` | Tests with race detection and coverage |

## Test Structure

```
go-mcp-commander/
  pkg/
    mcp/server_test.go          # MCP server unit tests
    logging/logging_test.go     # Logging unit tests
    commander/commander_test.go # Command execution unit tests
  test/
    mcp_test.go                 # MCP protocol integration tests
```

## Unit Tests

### MCP Server Tests

**File**: `pkg/mcp/server_test.go`

| Test | Purpose |
|------|---------|
| `TestNewServer` | Server initialization |
| `TestRegisterTool` | Tool registration |
| `TestHandleInitialize` | MCP initialize handshake |
| `TestHandleListTools` | List tools response |
| `TestHandleCallTool` | Tool invocation |
| `TestHandleCallTool_UnknownTool` | Unknown tool error |
| `TestHandlePing` | Ping request handling |
| `TestHandleUnknownMethod` | Method not found error |
| `TestHandleNotification` | Notification handling |
| `TestHandleParseError` | JSON parse error |
| `TestServerRun` | Full server loop |

**Run**:
```bash
go test -v ./pkg/mcp/...
```

### Commander Tests

**File**: `pkg/commander/commander_test.go`

| Test | Purpose |
|------|---------|
| `TestNewCommander` | Commander initialization |
| `TestValidateCommand_AllowedEmpty` | Allow all when no allowlist |
| `TestValidateCommand_AllowedList` | Allowlist enforcement |
| `TestValidateCommand_BlockedList` | Blocklist enforcement |
| `TestValidateCommand_CaseInsensitive` | Case-insensitive matching |
| `TestExecute_SimpleCommand` | Basic command execution |
| `TestExecute_Timeout` | Timeout handling |
| `TestExecute_WorkingDirectory` | Working directory support |
| `TestExecute_InvalidWorkingDirectory` | Invalid directory error |
| `TestExecute_EnvironmentVariables` | Env var passing |
| `TestExecute_FailingCommand` | Non-zero exit handling |
| `TestGetCommandName` | Command name extraction |
| `TestDefaultBlockedCommands` | Default blocklist |

**Run**:
```bash
go test -v ./pkg/commander/...
```

### Logging Tests

**File**: `pkg/logging/logging_test.go`

| Test | Purpose |
|------|---------|
| `TestParseLogLevel` | Log level parsing |
| `TestLogLevelString` | Level to string |
| `TestNewLogger` | Logger initialization |
| `TestLoggerLevels` | Level filtering |
| `TestLoggerCommandExec` | Command execution logging |
| `TestLoggerCommandBlocked` | Blocked command logging |
| `TestLoggerToolCall` | Tool call logging |
| `TestLogStartupAndShutdown` | Startup/shutdown logging |
| `TestDefaultLogDir` | Default log directory |
| `TestLoggerSetLevel` | Dynamic level change |
| `TestLoggerClose` | Resource cleanup |
| `TestLogFileNaming` | Log file naming |

**Run**:
```bash
go test -v ./pkg/logging/...
```

## Integration Tests

**File**: `test/mcp_test.go`

| Test | Purpose |
|------|---------|
| `TestMCP_Initialize` | MCP initialization handshake |
| `TestMCP_ListTools` | Tool listing via protocol |
| `TestMCP_ExecuteCommand` | Command execution via MCP |
| `TestMCP_ExecuteCommand_Blocked` | Blocked command via MCP |
| `TestMCP_Ping` | MCP ping request |
| `TestMCP_GetShellInfo` | Shell info via MCP |

**Run**:
```bash
# Build binary first
go build -o go-mcp-commander .

# Run integration tests
go test -v -tags=integration ./test/...
```

## Manual Testing

### JSON-RPC Protocol Testing

Start the server:
```bash
./go-mcp-commander -log-level debug
```

Send JSON-RPC requests via stdin:

**Initialize**:
```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
```

**List Tools**:
```json
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
```

**Execute Command**:
```json
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"echo hello"}}}
```

**Get Shell Info**:
```json
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_shell_info","arguments":{}}}
```

### Claude Code Testing

1. Build the binary:
```bash
go build -o go-mcp-commander.exe .  # Windows
go build -o go-mcp-commander .      # Unix
```

2. Configure `.mcp.json`:
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

3. Test prompts:
```
Use the commander MCP server to run "echo hello"
List the available commander tools
Get shell info from commander
```

## Security Testing

### Test Blocklist

Commands that should be blocked:
```bash
# These should return "command blocked" errors
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"rm -rf /"}}}' | ./go-mcp-commander

echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"mkfs.ext4 /dev/sda"}}}' | ./go-mcp-commander

echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"shutdown now"}}}' | ./go-mcp-commander
```

### Test Allowlist

Start server with allowlist:
```bash
./go-mcp-commander -allowed-commands "echo,ls"
```

Test allowed command (should work):
```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"echo hello"}}}
```

Test blocked command (should fail):
```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_command","arguments":{"command":"cat /etc/passwd"}}}
```

## CI/CD Testing

GitHub Actions runs:

| Stage | Commands |
|-------|----------|
| Lint | `go vet ./...`, `gofmt` check |
| Unit Tests | `go test -v -race ./pkg/...` (Ubuntu, macOS, Windows) |
| Integration Tests | `go test -v -tags=integration ./test/...` |
| Build | Multi-platform binary compilation |

## Coverage

Generate coverage report:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out          # Summary
go tool cover -html=coverage.out -o coverage.html  # HTML report
```

**Coverage Targets**:
| Package | Target |
|---------|--------|
| `pkg/mcp` | 80%+ |
| `pkg/commander` | 90%+ |
| `pkg/logging` | 75%+ |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Binary not found | Run `go build -o go-mcp-commander .` |
| Permission denied (Unix) | Run `chmod +x ./go-mcp-commander` |
| Tests timeout | Use `go test -v -timeout 120s ./...` |
| Windows integration tests fail | Use `.exe` extension |

### Debug Mode

Enable debug logging:
```bash
./go-mcp-commander -log-level debug
```

View logs:
```bash
# Unix
cat ~/go-mcp-commander/logs/go-mcp-commander-*.log

# Windows
type %USERPROFILE%\go-mcp-commander\logs\go-mcp-commander-*.log
```

## Benchmark Tests

Add benchmarks to `*_test.go` files:
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
