# ECS Deployment Guide for go-mcp-commander

This guide covers deploying go-mcp-commander as an HTTP service on AWS ECS (Elastic Container Service) using either Fargate or EC2 launch types.

## Security Warning

**IMPORTANT**: The commander MCP server executes shell commands. Deploying this in a production environment requires careful security consideration:

1. **Always enable authentication** via `MCP_AUTH_TOKEN`
2. **Restrict allowed commands** via `MCP_ALLOWED_COMMANDS`
3. **Use the default blocklist** to prevent dangerous commands
4. **Run in a minimal container** with limited privileges
5. **Use network isolation** to restrict what the container can access

## Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   MCP Client    │────▶│  Load Balancer  │────▶│   ECS Service   │
│ (Claude Code/   │     │     (ALB)       │     │   (Fargate/EC2) │
│  Continue.dev)  │     └─────────────────┘     └─────────────────┘
└─────────────────┘              │                       │
                                 │                       ▼
                                 │              ┌─────────────────┐
                                 │              │   Shell         │
                                 │              │   Commands      │
                                 │              └─────────────────┘
                                 ▼
                        ┌─────────────────┐
                        │ Secrets Manager │
                        └─────────────────┘
```

## Prerequisites

1. AWS CLI configured with appropriate permissions
2. Docker installed locally for building images
3. An ECR repository created for the image
4. VPC with subnets configured for ECS

## Quick Start

### 1. Build and Push Docker Image

```bash
# Authenticate to ECR
aws ecr get-login-password --region YOUR_REGION | docker login --username AWS --password-stdin YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com

# Build the image
docker build -t go-mcp-commander .

# Tag for ECR
docker tag go-mcp-commander:latest YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/go-mcp-commander:latest

# Push to ECR
docker push YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/go-mcp-commander:latest
```

### 2. Create Secrets in AWS Secrets Manager

```bash
aws secretsmanager create-secret \
    --name mcp/commander \
    --secret-string '{
        "MCP_AUTH_TOKEN": "your-secure-auth-token",
        "MCP_ALLOWED_COMMANDS": "ls,cat,grep,find,echo,pwd,whoami"
    }'
```

### 3. Create ECS Resources

```bash
# Create CloudWatch Log Group
aws logs create-log-group --log-group-name /ecs/go-mcp-commander

# Register Task Definition
aws ecs register-task-definition --cli-input-json file://ecs-task-definition.json

# Create ECS Cluster (if not exists)
aws ecs create-cluster --cluster-name mcp-servers

# Create Service
aws ecs create-service \
    --cluster mcp-servers \
    --service-name go-mcp-commander \
    --task-definition go-mcp-commander \
    --desired-count 1 \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx],securityGroups=[sg-xxx],assignPublicIp=ENABLED}"
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `MCP_AUTH_TOKEN` | **Yes** | Token for HTTP authentication (REQUIRED for production) |
| `MCP_ALLOWED_COMMANDS` | No | Comma-separated list of allowed command prefixes |
| `MCP_BLOCKED_COMMANDS` | No | Comma-separated list of blocked command patterns |
| `MCP_DEFAULT_TIMEOUT` | No | Default command timeout (default: 30s) |
| `MCP_SHELL` | No | Shell to use (default: /bin/sh) |
| `MCP_SHELL_ARG` | No | Shell argument (default: -c) |
| `MCP_LOG_LEVEL` | No | Log level (default: info) |

### Default Blocked Commands

The following dangerous commands are blocked by default:
- `rm -rf /`
- `mkfs`
- `dd`
- `shutdown`
- `reboot`
- `halt`
- `poweroff`
- And more...

### Authentication

When `MCP_AUTH_TOKEN` is set, all HTTP requests must include the `X-MCP-Auth-Token` header.

```bash
curl -X POST http://your-alb-url:3000/ \
    -H "Content-Type: application/json" \
    -H "X-MCP-Auth-Token: your-secure-auth-token" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

## Security Best Practices

1. **Always enable authentication**: Never run without `MCP_AUTH_TOKEN` in production
2. **Restrict allowed commands**: Use `MCP_ALLOWED_COMMANDS` to whitelist only needed commands
3. **Use private subnets**: Deploy in private subnets with no direct internet access
4. **Minimal IAM permissions**: The task role should have minimal permissions
5. **Network isolation**: Use security groups to restrict outbound access
6. **Audit logging**: Enable CloudTrail and monitor command execution logs
7. **Consider read-only filesystem**: Mount the container filesystem as read-only

## Monitoring

### CloudWatch Logs

Logs are automatically sent to CloudWatch Logs at `/ecs/go-mcp-commander`.

Monitor for:
- Failed authentication attempts
- Blocked command attempts
- Command execution errors

### Health Checks

The service exposes a `/health` endpoint that returns:
```json
{"status": "healthy", "server": "go-mcp-commander"}
```

## Troubleshooting

### Common Issues

1. **Commands blocked**: Check `MCP_ALLOWED_COMMANDS` and `MCP_BLOCKED_COMMANDS`
2. **Authentication failures**: Verify `MCP_AUTH_TOKEN` matches in client and server
3. **Timeout errors**: Increase `MCP_DEFAULT_TIMEOUT` for long-running commands

See [INTEGRATION.md](./INTEGRATION.md) for configuring Claude Code and Continue.dev.
