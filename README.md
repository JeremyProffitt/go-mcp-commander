# go-mcp-commander

A secure, cross-platform Model Context Protocol (MCP) server for executing system commands. Written in Go, it provides command execution capabilities with configurable security controls including command allowlists, blocklists, and timeout management.

## Features

- **Cross-Platform**: Supports Windows, macOS, and Linux
- **Security Controls**: Configurable command allowlists and blocklists
- **Default Blocklist**: Built-in protection against dangerous commands
- **Timeout Management**: Configurable command timeouts
- **Working Directory**: Execute commands in specific directories
- **Environment Variables**: Pass custom environment variables to commands
- **Comprehensive Logging**: Detailed logging with configurable levels
- **MCP Protocol Compliant**: Full JSON-RPC 2.0 and MCP protocol support

## Installation

### From Binary Releases

Download the latest binary for your platform from the [Releases](https://github.com/user/go-mcp-commander/releases) page:

| Platform | Architecture | File |
|----------|--------------|------|
| macOS | Universal (Intel + Apple Silicon) | go-mcp-commander-darwin-universal |
| Linux | x64 | go-mcp-commander-linux-amd64 |
| Linux | ARM64 | go-mcp-commander-linux-arm64 |
| Windows | x64 | go-mcp-commander-windows-amd64.exe |

### From Source

```bash
git clone https://github.com/user/go-mcp-commander.git
cd go-mcp-commander
go build -o go-mcp-commander .
```

## Usage

### Command Line Options

```bash
go-mcp-commander [options]
```

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `-log-dir` | `MCP_LOG_DIR` | `~/<app>/logs` | Directory for log files |
| `-log-level` | `MCP_LOG_LEVEL` | `info` | Log level: off\|error\|warn\|info\|access\|debug |
| `-allowed-commands` | `MCP_ALLOWED_COMMANDS` | (empty = allow all) | Comma-separated list of allowed command prefixes |
| `-blocked-commands` | `MCP_BLOCKED_COMMANDS` | (empty) | Comma-separated list of blocked command patterns |
| `-timeout` | `MCP_DEFAULT_TIMEOUT` | `30s` | Default command timeout |
| `-shell` | `MCP_SHELL` | OS-dependent | Shell to use for command execution |
| `-shell-arg` | `MCP_SHELL_ARG` | OS-dependent | Shell argument for command execution |
| `-use-default-blocklist` | - | `true` | Use default blocklist of dangerous commands |

### Configuration Priority

Configuration values are resolved in the following priority order:
1. Command-line flags (highest priority)
2. Environment variables
3. Default values (lowest priority)

## MCP Tools

### execute_command

Execute a system command and return its output.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `command` | string | Yes | The command to execute |
| `working_directory` | string | No | Working directory for command execution |
| `timeout` | string | No | Timeout duration (e.g., '30s', '5m') |
| `env` | object | No | Environment variables to set |

**Example:**
```json
{
  "name": "execute_command",
  "arguments": {
    "command": "ls -la",
    "working_directory": "/tmp",
    "timeout": "10s",
    "env": {
      "MY_VAR": "value"
    }
  }
}
```

**Response:**
```json
{
  "stdout": "...",
  "stderr": "...",
  "exit_code": 0,
  "duration": "50ms"
}
```

### list_allowed_commands

List all allowed command patterns.

**Response:**
```json
{
  "allowed_commands": ["git", "npm", "docker"],
  "allow_all": false
}
```

### list_blocked_commands

List all blocked command patterns.

**Response:**
```json
{
  "blocked_commands": ["rm -rf /", "mkfs", ...],
  "using_default_blocklist": true
}
```

### get_shell_info

Get information about the shell used for command execution.

**Response:**
```json
{
  "shell": "/bin/sh",
  "shell_arg": "-c",
  "default_timeout": "30s"
}
```

## Integration

### Claude Desktop

Add to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "commander": {
      "command": "/path/to/go-mcp-commander",
      "args": ["-log-level", "info"]
    }
  }
}
```

### Claude Code

Create a `.mcp.json` file in your project root:

```json
{
  "mcpServers": {
    "commander": {
      "command": "/path/to/go-mcp-commander",
      "args": ["-log-level", "info"]
    }
  }
}
```

Or create `.claude/mcp.json` for workspace-specific configuration:

```json
{
  "mcpServers": {
    "commander": {
      "command": "${workspaceFolder}/go-mcp-commander",
      "args": [
        "-log-dir", "${workspaceFolder}/logs",
        "-log-level", "info"
      ]
    }
  }
}
```

### Continue.dev

Create a `.continue/config.json` file:

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "go-mcp-commander",
        "transport": {
          "type": "stdio",
          "command": "/path/to/go-mcp-commander",
          "args": ["-log-level", "info"]
        }
      }
    ]
  }
}
```

## Security

### Default Blocklist

The following commands are blocked by default on Unix systems:
- `rm -rf /` and `rm -rf /*`
- `mkfs`
- `dd if=`
- Fork bombs (`:(){:|:&};:`)
- `chmod -R 777 /`
- `chown -R`
- `> /dev/sda`
- `shutdown`, `reboot`, `halt`, `poweroff`
- `init 0`, `init 6`

On Windows:
- `format`
- `del /s`, `rd /s`, `rmdir /s`
- `reg delete`
- `net user`, `net localgroup`
- `shutdown`, `restart`

### Custom Allowlist

To restrict commands to a specific set, use the `-allowed-commands` flag:

```bash
go-mcp-commander -allowed-commands "git,npm,docker,kubectl"
```

Only commands starting with these prefixes will be allowed.

### Custom Blocklist

To add additional blocked commands:

```bash
go-mcp-commander -blocked-commands "curl,wget,ssh"
```

### Disable Default Blocklist

To disable the default blocklist (not recommended):

```bash
go-mcp-commander -use-default-blocklist=false
```

## Global Environment File

All go-mcp servers support loading environment variables from `~/.mcp_env`. This provides a central location to configure credentials and settings, especially useful on macOS where GUI applications don't inherit shell environment variables from `.zshrc` or `.bashrc`.

### File Format

Create `~/.mcp_env` with KEY=VALUE pairs:

```bash
# ~/.mcp_env - MCP Server Environment Variables

# Commander Configuration
MCP_ALLOWED_COMMANDS=git,npm,docker,kubectl
MCP_BLOCKED_COMMANDS=curl,wget
MCP_DEFAULT_TIMEOUT=60s

# Logging
MCP_LOG_DIR=~/mcp-logs
MCP_LOG_LEVEL=info
```

### Features

- Lines starting with `#` are treated as comments
- Empty lines are ignored
- Values can be quoted with single or double quotes
- **Existing environment variables are NOT overwritten** (env vars take precedence)
- Paths with `~` are automatically expanded to your home directory

### Path Expansion

All path-related settings support `~` expansion:

```bash
MCP_LOG_DIR=~/logs/commander
```

This works in the `~/.mcp_env` file, environment variables, and command-line flags.

## Logging

Logs are written to date-stamped files in the log directory:

```
~/go-mcp-commander/logs/go-mcp-commander-2025-01-15.log
```

### Log Levels

| Level | Description |
|-------|-------------|
| `off` | No logging |
| `error` | Errors only |
| `warn` | Warnings and errors |
| `info` | General information (default) |
| `access` | Command execution details |
| `debug` | Detailed debugging information |

### Log Format

```
[2025-01-15T10:30:45.123Z] [INFO] TOOL_CALL tool="execute_command" args=[command, working_directory]
[2025-01-15T10:30:45.150Z] [ACCESS] CMD_EXEC command="ls -la" workdir="/tmp" exit_code=0 duration=27ms
```

**Security Note**: Command output is never logged to prevent sensitive data exposure.

## Development

### Prerequisites

- Go 1.21 or later

### Building

```bash
# Build for current platform
go build -o go-mcp-commander .

# Build for all platforms
make build
```

### Testing

```bash
# Run unit tests
go test -v ./pkg/...

# Run integration tests
go test -v -tags=integration ./test/...

# Run all tests with coverage
go test -v -race -coverprofile=coverage.out ./...
```

### Project Structure

```
go-mcp-commander/
├── main.go                    # Entry point, tool registration
├── go.mod                     # Go module definition
├── pkg/
│   ├── mcp/
│   │   ├── server.go          # MCP server implementation
│   │   ├── server_test.go     # Server tests
│   │   └── types.go           # MCP protocol types
│   ├── logging/
│   │   ├── logging.go         # Logging implementation
│   │   └── logging_test.go    # Logging tests
│   └── commander/
│       ├── commander.go       # Command execution
│       └── commander_test.go  # Commander tests
├── test/
│   └── mcp_test.go            # MCP integration tests
├── .github/workflows/
│   ├── ci.yml                 # CI workflow
│   └── release.yml            # Release workflow
├── .mcp.json                  # MCP configuration
├── .claude/mcp.json           # Claude workspace config
├── .continue/config.json      # Continue.dev config
├── README.md                  # This file
└── TESTING.md                 # Testing documentation
```

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- Inspired by [mcp-local-command-server](https://github.com/kentaro/mcp-local-command-server)
- Built following patterns from [go-mcp-file-context-server](https://github.com/user/go-mcp-file-context-server)
- Implements the [Model Context Protocol](https://modelcontextprotocol.io/)
