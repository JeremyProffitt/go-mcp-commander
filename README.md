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

When `MCP_LOG_DIR` is set or `-log-dir` flag is used, logs are automatically placed in a subfolder named after the binary. This allows multiple MCP servers to share the same log directory:

```
MCP_LOG_DIR=/var/log/mcp
  └── go-mcp-commander/
      └── go-mcp-commander-2025-01-15.log
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

## Tool Reference

This section provides detailed information about each MCP tool for LLM consumption.

### execute_command

**Purpose**: Execute shell commands on the host system and retrieve output.

**When to Use**:
- Running build commands (npm, go, make)
- File system operations (ls, cat, mkdir)
- Git operations (git status, git commit)
- System information queries (whoami, pwd, uname)
- Running scripts and automation tasks

**Parameters**:
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `command` | string | Yes | - | Shell command to execute. Supports pipes, redirects, and chaining |
| `working_directory` | string | No | Server CWD | Absolute path to execute command from |
| `timeout` | string | No | `30s` | Duration string (e.g., `10s`, `2m`, `1h`) |
| `env` | object | No | `{}` | Key-value pairs of environment variables |

**Return Fields**:
| Field | Type | Description |
|-------|------|-------------|
| `stdout` | string | Standard output from command |
| `stderr` | string | Standard error from command |
| `exit_code` | integer | Exit code (0 = success) |
| `duration` | string | Execution time |

**Example Request**:
```json
{
  "name": "execute_command",
  "arguments": {
    "command": "git status --porcelain",
    "working_directory": "/home/user/project",
    "timeout": "5s"
  }
}
```

**Example Response**:
```json
{
  "stdout": "M README.md\n?? newfile.txt\n",
  "stderr": "",
  "exit_code": 0,
  "duration": "23ms"
}
```

### list_allowed_commands

**Purpose**: Query the server's command allowlist configuration.

**When to Use**:
- Before executing commands to verify they will be accepted
- Debugging command rejection issues
- Understanding server security configuration

**Parameters**: None

**Return Fields**:
| Field | Type | Description |
|-------|------|-------------|
| `allowed_commands` | array | List of allowed command prefixes |
| `allow_all` | boolean | `true` if no allowlist configured (all commands allowed) |

**Example Response** (restricted):
```json
{
  "allowed_commands": ["git", "npm", "go", "docker"],
  "allow_all": false
}
```

**Example Response** (unrestricted):
```json
{
  "allowed_commands": [],
  "allow_all": true
}
```

### list_blocked_commands

**Purpose**: Query the server's command blocklist configuration.

**When to Use**:
- Understanding which commands are blocked
- Debugging unexpected command rejections
- Verifying security configuration

**Parameters**: None

**Return Fields**:
| Field | Type | Description |
|-------|------|-------------|
| `blocked_commands` | array | List of blocked command patterns |
| `using_default_blocklist` | boolean | `true` if default dangerous commands are blocked |

**Example Response**:
```json
{
  "blocked_commands": ["rm -rf /", "mkfs", "dd if=", "shutdown"],
  "using_default_blocklist": true
}
```

### get_shell_info

**Purpose**: Query the shell configuration used for command execution.

**When to Use**:
- Determining shell syntax to use (bash vs cmd vs powershell)
- Understanding timeout defaults
- Cross-platform compatibility checks

**Parameters**: None

**Return Fields**:
| Field | Type | Description |
|-------|------|-------------|
| `shell` | string | Shell executable path |
| `shell_arg` | string | Argument used to pass commands |
| `default_timeout` | string | Default timeout duration |

**Example Response** (Unix):
```json
{
  "shell": "/bin/sh",
  "shell_arg": "-c",
  "default_timeout": "30s"
}
```

**Example Response** (Windows):
```json
{
  "shell": "cmd",
  "shell_arg": "/c",
  "default_timeout": "30s"
}
```

## Common Workflows

### Workflow 1: Project Build and Test

Execute a typical build workflow for a Go project:

```
Step 1: Check current directory
  Tool: execute_command
  Args: {"command": "pwd"}

Step 2: Run tests
  Tool: execute_command
  Args: {"command": "go test -v ./...", "timeout": "5m"}

Step 3: Build binary
  Tool: execute_command
  Args: {"command": "go build -o app .", "working_directory": "/project"}
```

### Workflow 2: Git Operations

Perform a git commit workflow:

```
Step 1: Check status
  Tool: execute_command
  Args: {"command": "git status --porcelain"}

Step 2: Stage changes
  Tool: execute_command
  Args: {"command": "git add -A"}

Step 3: Commit
  Tool: execute_command
  Args: {"command": "git commit -m \"feat: add new feature\""}

Step 4: Verify
  Tool: execute_command
  Args: {"command": "git log -1 --oneline"}
```

### Workflow 3: Security Check Before Execution

Verify command will be accepted before running:

```
Step 1: Check allowlist
  Tool: list_allowed_commands
  Result: Check if your command prefix is in allowed_commands or allow_all is true

Step 2: Check blocklist
  Tool: list_blocked_commands
  Result: Verify your command doesn't match any blocked patterns

Step 3: Execute if safe
  Tool: execute_command
  Args: {"command": "your-command-here"}
```

### Workflow 4: Cross-Platform Command Execution

Adapt commands based on shell:

```
Step 1: Get shell info
  Tool: get_shell_info
  Result: Check shell type (sh, cmd, powershell)

Step 2: Execute platform-appropriate command
  If shell contains "cmd":
    Tool: execute_command
    Args: {"command": "dir /b"}
  Else:
    Tool: execute_command
    Args: {"command": "ls -la"}
```

### Workflow 5: Long-Running Process with Custom Timeout

Execute a command that may take longer than default timeout:

```
Step 1: Execute with extended timeout
  Tool: execute_command
  Args: {
    "command": "npm install",
    "working_directory": "/project",
    "timeout": "10m"
  }

Step 2: Verify installation
  Tool: execute_command
  Args: {"command": "npm list --depth=0", "working_directory": "/project"}
```

## Error Handling

### Error: Command Blocked by Allowlist

**Error Message**: `command not allowed: 'wget' - not in allowed commands list`

**Cause**: Server is configured with an allowlist and the command prefix doesn't match.

**Solution**:
1. Use `list_allowed_commands` to see permitted commands
2. Use an allowed command alternative
3. Request server administrator to add command to allowlist

**Example**:
```json
{
  "isError": true,
  "content": [{
    "type": "text",
    "text": "command not allowed: 'wget' - not in allowed commands list"
  }]
}
```

### Error: Command Blocked by Blocklist

**Error Message**: `command blocked: 'rm -rf /' matches blocked pattern`

**Cause**: Command matches a blocked pattern (either default or custom blocklist).

**Solution**:
1. Use `list_blocked_commands` to see blocked patterns
2. Use a safer alternative command
3. Break the operation into smaller, safer steps

**Example**:
```json
{
  "isError": true,
  "content": [{
    "type": "text",
    "text": "command blocked: 'rm -rf /' matches blocked pattern"
  }]
}
```

### Error: Command Timeout

**Error Message**: `command timed out after 30s`

**Cause**: Command execution exceeded the timeout duration.

**Solution**:
1. Increase timeout: `{"timeout": "5m"}`
2. Break long operations into smaller steps
3. Check if command is hanging (infinite loop, waiting for input)

**Example**:
```json
{
  "isError": true,
  "content": [{
    "type": "text",
    "text": "command timed out after 30s"
  }]
}
```

### Error: Invalid Working Directory

**Error Message**: `working directory does not exist: '/nonexistent/path'`

**Cause**: The specified `working_directory` path doesn't exist.

**Solution**:
1. Verify the directory exists: `{"command": "ls -la /path/to/dir"}`
2. Create the directory first: `{"command": "mkdir -p /path/to/dir"}`
3. Use an absolute path instead of relative

**Example**:
```json
{
  "isError": true,
  "content": [{
    "type": "text",
    "text": "working directory does not exist: '/nonexistent/path'"
  }]
}
```

### Error: Command Not Found

**Error Message**: `exit_code: 127` with `stderr: "command not found"`

**Cause**: The command executable doesn't exist in PATH.

**Solution**:
1. Use full path to executable
2. Verify command is installed: `{"command": "which commandname"}`
3. Check PATH: `{"command": "echo $PATH"}`

**Example**:
```json
{
  "stdout": "",
  "stderr": "/bin/sh: mycommand: command not found",
  "exit_code": 127,
  "duration": "5ms"
}
```

### Error: Permission Denied

**Error Message**: `exit_code: 1` with `stderr: "Permission denied"`

**Cause**: Insufficient permissions to execute command or access file.

**Solution**:
1. Check file permissions: `{"command": "ls -la /path/to/file"}`
2. Use appropriate user context
3. Avoid operations requiring elevated privileges

### Error: HTTP Authentication Failed (HTTP Mode)

**Error Message**: HTTP 401 Unauthorized

**Cause**: Missing or invalid `X-MCP-Auth-Token` header.

**Solution**:
1. Ensure `X-MCP-Auth-Token` header is included in request
2. Verify token matches server's `MCP_AUTH_TOKEN` configuration
3. Check for typos in token value

## Parameter Formats

### Timeout Format

The `timeout` parameter accepts Go duration strings.

**Format**: `<number><unit>`

**Valid Units**:
| Unit | Meaning | Example |
|------|---------|---------|
| `s` | seconds | `30s` = 30 seconds |
| `m` | minutes | `5m` = 5 minutes |
| `h` | hours | `1h` = 1 hour |

**Combinations**: Units can be combined: `1h30m`, `2m30s`

**Examples**:
- `"10s"` - 10 seconds
- `"2m"` - 2 minutes
- `"1h"` - 1 hour
- `"1m30s"` - 1 minute 30 seconds

**Default**: `30s` if not specified

### Environment Variables Format

The `env` parameter accepts a JSON object with string key-value pairs.

**Format**: `{"VAR_NAME": "value", "ANOTHER_VAR": "value2"}`

**Rules**:
- Keys must be valid environment variable names (uppercase recommended)
- Values must be strings
- Existing environment variables are NOT overwritten
- Variables are only set for the current command execution

**Examples**:
```json
{
  "env": {
    "NODE_ENV": "production",
    "DEBUG": "true",
    "API_URL": "https://api.example.com"
  }
}
```

### Working Directory Format

The `working_directory` parameter accepts an absolute path string.

**Format**: Absolute path to existing directory

**Platform-Specific**:
- Unix: `/home/user/project`
- Windows: `C:\\Users\\user\\project` or `C:/Users/user/project`

**Rules**:
- Path must be absolute (not relative)
- Directory must exist before command execution
- Use forward slashes on Windows for JSON compatibility

**Examples**:
- Unix: `"/home/user/myproject"`
- Windows: `"C:/dev/myproject"`

### Command Format

The `command` parameter accepts shell command strings.

**Format**: Any valid shell command for the configured shell

**Supported Features**:
- Pipes: `ls -la | grep .txt`
- Redirects: `echo hello > file.txt`
- Command chaining: `cd /tmp && ls`
- Environment variable expansion: `echo $HOME`
- Glob patterns: `ls *.txt`

**Platform Considerations**:
- Unix (sh/bash): Standard Unix command syntax
- Windows (cmd): Windows command syntax (`dir`, `type`, etc.)
- Windows (powershell): PowerShell cmdlet syntax

**Examples**:
```json
{"command": "ls -la"}
{"command": "git status && git diff"}
{"command": "cat file.txt | grep pattern | head -10"}
{"command": "NODE_ENV=test npm run test"}
```

### Allowed/Blocked Commands Format

Server configuration uses comma-separated command prefixes.

**Format**: `"cmd1,cmd2,cmd3"`

**Matching Rules**:
- Commands are matched by prefix
- Matching is case-insensitive
- Spaces around commas are trimmed

**Examples**:
- `-allowed-commands "git,npm,go"` - Allows `git`, `npm`, `go` and any commands starting with these
- `-blocked-commands "curl,wget"` - Blocks `curl`, `wget` commands

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- Inspired by [mcp-local-command-server](https://github.com/kentaro/mcp-local-command-server)
- Built following patterns from [go-mcp-file-context-server](https://github.com/user/go-mcp-file-context-server)
- Implements the [Model Context Protocol](https://modelcontextprotocol.io/)
