# MCP Client Integration Guide

This guide explains how to configure MCP clients (Claude Code and Continue.dev) to connect to the go-mcp-commander server running in HTTP mode, including authentication configuration.

## Security Warning

**IMPORTANT**: The commander MCP server executes shell commands. When integrating:
1. **Always use authentication** in production
2. **Restrict allowed commands** on the server side
3. **Be cautious** about what commands you allow the LLM to execute

## Authentication Overview

When running in HTTP mode with authentication enabled (via `MCP_AUTH_TOKEN` environment variable), all requests must include the `X-MCP-Auth-Token` header with the configured token value.

## Claude Code Integration

### Configuration Location

Claude Code configuration is stored in:
- **macOS/Linux**: `~/.claude/claude_code_config.json`
- **Windows**: `%USERPROFILE%\.claude\claude_code_config.json`

### HTTP Mode Configuration

```json
{
  "mcpServers": {
    "commander": {
      "type": "http",
      "url": "http://your-alb-url:3000",
      "headers": {
        "X-MCP-Auth-Token": "your-secure-auth-token"
      }
    }
  }
}
```

### Configuration with Environment Variable

```json
{
  "mcpServers": {
    "commander": {
      "type": "http",
      "url": "http://your-alb-url:3000",
      "headers": {
        "X-MCP-Auth-Token": "${MCP_COMMANDER_TOKEN}"
      }
    }
  }
}
```

### Local Development (stdio mode)

```json
{
  "mcpServers": {
    "commander": {
      "command": "/path/to/go-mcp-commander",
      "args": ["--allowed-commands", "ls,cat,grep,pwd"],
      "env": {}
    }
  }
}
```

## Continue.dev Integration

### HTTP Mode Configuration

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "commander",
        "transport": {
          "type": "http",
          "url": "http://your-alb-url:3000",
          "headers": {
            "X-MCP-Auth-Token": "your-secure-auth-token"
          }
        }
      }
    ]
  }
}
```

### Local Development (stdio mode)

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "commander",
        "transport": {
          "type": "stdio",
          "command": "/path/to/go-mcp-commander",
          "args": ["--allowed-commands", "ls,cat,grep,pwd"]
        }
      }
    ]
  }
}
```

## Testing the Connection

### Using curl

```bash
# Test health endpoint (no auth required)
curl http://your-alb-url:3000/health

# Test MCP endpoint with auth
curl -X POST http://your-alb-url:3000/ \
    -H "Content-Type: application/json" \
    -H "X-MCP-Auth-Token: your-secure-auth-token" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# Execute a command
curl -X POST http://your-alb-url:3000/ \
    -H "Content-Type: application/json" \
    -H "X-MCP-Auth-Token: your-secure-auth-token" \
    -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"execute_command","arguments":{"command":"echo hello"}},"id":2}'
```

## Security Best Practices

1. **Use HTTPS**: Always use HTTPS in production
2. **Rotate tokens**: Implement regular token rotation
3. **Audit commands**: Log and monitor all command executions
4. **Least privilege**: Only allow commands that are absolutely necessary
5. **Network isolation**: Restrict what the container can access

## Troubleshooting

### 401 Unauthorized
- Verify the `X-MCP-Auth-Token` header matches the server's `MCP_AUTH_TOKEN`

### Command Blocked
- Check the server's `MCP_ALLOWED_COMMANDS` and `MCP_BLOCKED_COMMANDS` settings
- Review logs for blocked command details

### Timeout Errors
- Increase `MCP_DEFAULT_TIMEOUT` on the server
- Check for long-running commands
