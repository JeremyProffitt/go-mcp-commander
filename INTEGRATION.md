# MCP Client Integration Guide

This guide explains how to connect MCP clients to go-mcp-commander.

## Quick Reference

| Client | Config File | Transport |
|--------|-------------|-----------|
| Claude Desktop | `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) | stdio |
| Claude Code | `.mcp.json` or `.claude/mcp.json` | stdio |
| Continue.dev | `.continue/config.json` | stdio or HTTP |

## Security Warning

go-mcp-commander executes shell commands. Always:
1. Enable authentication in production (HTTP mode)
2. Restrict allowed commands via `-allowed-commands`
3. Review the default blocklist

## Claude Desktop

### Configuration Location
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%/Claude/claude_desktop_config.json`

### stdio Mode (Recommended)
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

### With Restricted Commands
```json
{
  "mcpServers": {
    "commander": {
      "command": "/path/to/go-mcp-commander",
      "args": [
        "-allowed-commands", "git,npm,go,docker",
        "-log-level", "info"
      ]
    }
  }
}
```

## Claude Code

### Project-Level Configuration

**File**: `.mcp.json` in project root
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

### Workspace-Specific Configuration

**File**: `.claude/mcp.json`
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

**Variables**:
| Variable | Meaning |
|----------|---------|
| `${workspaceFolder}` | Project root directory |

## Continue.dev

### stdio Mode
```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "commander",
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

### HTTP Mode (Remote Server)
```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "commander",
        "transport": {
          "type": "http",
          "url": "http://your-server:3000",
          "headers": {
            "X-MCP-Auth-Token": "your-auth-token"
          }
        }
      }
    ]
  }
}
```

## HTTP Mode Setup

### Server Configuration

Start server with authentication:
```bash
MCP_AUTH_TOKEN=your-secure-token ./go-mcp-commander -http -port 3000
```

Environment variables:
| Variable | Purpose |
|----------|---------|
| `MCP_AUTH_TOKEN` | Required auth token for HTTP mode |
| `MCP_HTTP_PORT` | HTTP port (default: 3000) |

### Client Authentication

All HTTP requests must include:
```
X-MCP-Auth-Token: your-secure-token
```

### Testing HTTP Connection

```bash
# Health check (no auth required)
curl http://localhost:3000/health

# List tools (auth required)
curl -X POST http://localhost:3000/ \
  -H "Content-Type: application/json" \
  -H "X-MCP-Auth-Token: your-secure-token" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# Execute command
curl -X POST http://localhost:3000/ \
  -H "Content-Type: application/json" \
  -H "X-MCP-Auth-Token: your-secure-token" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"execute_command","arguments":{"command":"echo hello"}},"id":2}'
```

## Troubleshooting

| Error | Cause | Solution |
|-------|-------|----------|
| 401 Unauthorized | Invalid or missing auth token | Verify `X-MCP-Auth-Token` matches server's `MCP_AUTH_TOKEN` |
| Command Blocked | Command not in allowlist | Check `-allowed-commands` setting |
| Command Blocked | Command in blocklist | Check `-blocked-commands` and default blocklist |
| Timeout | Command took too long | Increase `-timeout` or command timeout parameter |
| Connection Refused | Server not running | Start server with `-http` flag |

### Debug Logging

Enable verbose logging:
```bash
./go-mcp-commander -log-level debug
```

Log file location:
- **Unix**: `~/go-mcp-commander/logs/`
- **Windows**: `%USERPROFILE%\go-mcp-commander\logs\`

## Security Best Practices

### Production Checklist

1. **Authentication**: Always set `MCP_AUTH_TOKEN`
2. **Command Restriction**: Use `-allowed-commands` whitelist
3. **HTTPS**: Use TLS termination (ALB, nginx)
4. **Token Rotation**: Rotate auth tokens regularly
5. **Audit Logging**: Set `-log-level access` for command logging
6. **Network Isolation**: Restrict server network access
