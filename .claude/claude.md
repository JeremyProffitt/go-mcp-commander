# Claude Code Project Instructions

## Deployment with GitHub Actions

This project uses GitHub Actions for CI/CD. Follow these rules when deploying:

### Deployment Process

1. **Commit and Push**: Push changes to the `main` branch to trigger the CI/CD pipeline
2. **Monitor Workflows**: Check GitHub Actions for build status
3. **Fix Failures**: If workflows fail, analyze the error, fix the issue, and push again
4. **Iterate**: Continue fixing and pushing until all workflows pass

### GitHub Actions Workflows

- **CI (`.github/workflows/ci.yml`)**: Runs on all pushes and PRs
  - Linting (go vet, go fmt)
  - Unit tests across multiple platforms (Ubuntu, macOS, Windows)
  - Integration tests
  - MCP protocol validation

- **Release (`.github/workflows/release.yml`)**: Runs on main branch pushes
  - Tests across all platforms
  - Cross-platform builds (Darwin, Linux, Windows)
  - Creates GitHub Release with binaries

### Commit Message Guidelines

Use conventional commits with the robot footer:

```
<type>: <description>

[optional body]

Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`

### Fixing CI Failures

When GitHub Actions fail:

1. **Read the error logs** from the failed workflow run
2. **Identify the issue**: lint error, test failure, build error
3. **Fix locally** and verify with:
   - `go fmt ./...` - Format code
   - `go vet ./...` - Check for issues
   - `go test -v ./pkg/...` - Run unit tests
   - `go test -v -tags=integration ./test/...` - Run integration tests
   - `go build .` - Verify build
4. **Commit and push** the fix
5. **Repeat** until all workflows pass

### Pre-Push Checklist

Before pushing, ensure:

- [ ] Code is formatted: `go fmt ./...`
- [ ] No vet errors: `go vet ./...`
- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build .`
