# Claude Code Guidelines for go-mcp-commander

This document provides guidelines for LLMs (Claude Code, etc.) working with this repository.

## Repository Overview

**Purpose**: MCP server for secure shell command execution
**Language**: Go 1.21+
**Protocol**: Model Context Protocol (MCP) over stdio or HTTP

## Quick Reference

### Build Commands
```bash
go build -o go-mcp-commander .        # Build binary
go test -v ./...                       # Run all tests
go test -v ./pkg/commander/...         # Test commander only
go test -race -coverprofile=c.out ./...  # Tests with coverage
```

### Run Commands
```bash
./go-mcp-commander                     # Default settings
./go-mcp-commander -log-level debug    # Debug logging
./go-mcp-commander -allowed-commands "git,npm"  # Restricted mode
```

## Project Structure

```
go-mcp-commander/
  main.go              # Entry point, tool registration
  pkg/
    mcp/               # MCP protocol implementation
      server.go        # Server logic
      types.go         # Protocol types
    commander/         # Command execution engine
      commander.go     # Execute, validate commands
    logging/           # Logging system
      logging.go       # Log levels, file output
  test/
    mcp_test.go        # Integration tests
```

## Code Conventions

### Tool Registration Pattern
Tools are registered in `main.go` using:
```go
server.RegisterTool(mcp.Tool{
    Name:        "tool_name",
    Description: "What the tool does",
    InputSchema: json.RawMessage(`{...}`),
    Handler:     handlerFunction,
})
```

### Error Handling Pattern
```go
if err != nil {
    return nil, &mcp.Error{
        Code:    mcp.InternalError,
        Message: fmt.Sprintf("descriptive error: %v", err),
    }
}
```

### Test Pattern
```go
func TestFunctionName(t *testing.T) {
    // Setup
    cmd := NewCommander(Config{...})

    // Execute
    result, err := cmd.Execute(...)

    // Verify
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

## Common Tasks

### Adding a New Tool
1. Define handler function in `main.go`
2. Register with `server.RegisterTool()`
3. Add tests in `pkg/mcp/server_test.go`
4. Update README.md Tool Reference section

### Modifying Command Validation
1. Edit `pkg/commander/commander.go`
2. Update `ValidateCommand()` method
3. Add test cases in `pkg/commander/commander_test.go`

### Changing Log Output
1. Edit `pkg/logging/logging.go`
2. Add new log level or format
3. Test in `pkg/logging/logging_test.go`

## LLM Usability Checklist

Before committing changes, verify:

### Tool Definitions
- [ ] Tool has clear description explaining purpose
- [ ] Parameter descriptions include format examples
- [ ] Numeric parameters have min/max constraints
- [ ] Boolean parameters document default values

### Documentation
- [ ] README Tool Reference is updated
- [ ] Common Workflows section covers new features
- [ ] Error Handling documents new error cases
- [ ] Parameter Formats documents new formats

### Code Quality
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `go fmt ./...` shows no changes
- [ ] No commented-out code

## AWS Deployment Policy

**CRITICAL**: All infrastructure changes must go through GitHub Actions pipelines.

### Prohibited Actions
- Direct `aws` CLI deployments
- Direct `sam deploy` commands
- Manual CloudFormation stack operations
- Direct ECS service updates

### Required Process
1. Make code/infrastructure changes
2. Commit and push to branch
3. Create pull request
4. Pipeline deploys on merge

### Pipeline Failures
If GitHub Actions pipeline fails:
1. Use `/fix-pipeline` skill for automated diagnosis
2. Review logs in Actions tab
3. Fix identified issues
4. Push fixes and re-trigger pipeline

## Security Considerations

### When Modifying Blocklist
- Test all existing blocked commands still blocked
- Verify new patterns don't create bypasses
- Run security tests: `go test -v ./pkg/commander/... -run Block`

### When Adding Commands
- Consider if command could be dangerous
- Add to default blocklist if necessary
- Document security implications

## File Locations

| File | Purpose |
|------|---------|
| `main.go` | Server setup, tool registration |
| `pkg/mcp/server.go` | MCP protocol handling |
| `pkg/mcp/types.go` | Request/response types |
| `pkg/commander/commander.go` | Command execution |
| `pkg/logging/logging.go` | Log management |
| `.mcp.json` | Local MCP configuration |
| `README.md` | User documentation |
| `TESTING.md` | Test instructions |
| `INTEGRATION.md` | Client setup guides |
| `ECS.md` | AWS deployment guide |
