# ECS Deployment Guide

Deploy go-mcp-commander on AWS ECS (Fargate or EC2).

## Quick Reference

| Component | Purpose |
|-----------|---------|
| ECR | Container registry for Docker image |
| ECS | Container orchestration |
| ALB | Load balancer for HTTP traffic |
| Secrets Manager | Secure credential storage |
| CloudWatch | Logging and monitoring |

## Architecture

```
MCP Client --> ALB --> ECS Service --> Shell Commands
                |           |
                v           v
         Secrets Manager  CloudWatch Logs
```

## Prerequisites

1. AWS CLI configured with appropriate permissions
2. Docker installed locally
3. ECR repository created
4. VPC with subnets configured

## Deployment Steps

### Step 1: Build and Push Image

```bash
# Authenticate to ECR
aws ecr get-login-password --region YOUR_REGION | \
  docker login --username AWS --password-stdin \
  YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com

# Build
docker build -t go-mcp-commander .

# Tag
docker tag go-mcp-commander:latest \
  YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/go-mcp-commander:latest

# Push
docker push \
  YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/go-mcp-commander:latest
```

### Step 2: Create Secrets

```bash
aws secretsmanager create-secret \
  --name mcp/commander \
  --secret-string '{
    "MCP_AUTH_TOKEN": "your-secure-auth-token",
    "MCP_ALLOWED_COMMANDS": "ls,cat,grep,find,echo,pwd,whoami"
  }'
```

### Step 3: Create ECS Resources

```bash
# CloudWatch Log Group
aws logs create-log-group --log-group-name /ecs/go-mcp-commander

# Register Task Definition
aws ecs register-task-definition --cli-input-json file://ecs-task-definition.json

# Create Cluster
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

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MCP_AUTH_TOKEN` | Yes | - | HTTP authentication token |
| `MCP_ALLOWED_COMMANDS` | No | - | Comma-separated allowed command prefixes |
| `MCP_BLOCKED_COMMANDS` | No | - | Comma-separated blocked command patterns |
| `MCP_DEFAULT_TIMEOUT` | No | `30s` | Default command timeout |
| `MCP_SHELL` | No | `/bin/sh` | Shell executable |
| `MCP_SHELL_ARG` | No | `-c` | Shell argument |
| `MCP_LOG_LEVEL` | No | `info` | Log level |

### Default Blocked Commands

These commands are blocked by default:
- `rm -rf /`, `rm -rf /*`
- `mkfs`
- `dd`
- `shutdown`, `reboot`, `halt`, `poweroff`
- Fork bombs

## Task Definition Example

```json
{
  "family": "go-mcp-commander",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::ACCOUNT:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "name": "go-mcp-commander",
      "image": "ACCOUNT.dkr.ecr.REGION.amazonaws.com/go-mcp-commander:latest",
      "portMappings": [
        {"containerPort": 3000, "protocol": "tcp"}
      ],
      "secrets": [
        {
          "name": "MCP_AUTH_TOKEN",
          "valueFrom": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:mcp/commander:MCP_AUTH_TOKEN::"
        }
      ],
      "environment": [
        {"name": "MCP_ALLOWED_COMMANDS", "value": "ls,cat,grep,echo,pwd"},
        {"name": "MCP_LOG_LEVEL", "value": "info"}
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/go-mcp-commander",
          "awslogs-region": "REGION",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

## Health Check

The `/health` endpoint returns:
```json
{"status": "healthy", "server": "go-mcp-commander"}
```

ALB health check configuration:
| Setting | Value |
|---------|-------|
| Path | `/health` |
| Protocol | HTTP |
| Port | 3000 |
| Interval | 30s |
| Timeout | 5s |
| Healthy threshold | 2 |
| Unhealthy threshold | 3 |

## Security Best Practices

| Practice | Implementation |
|----------|----------------|
| Authentication | Always set `MCP_AUTH_TOKEN` |
| Command Restriction | Use `MCP_ALLOWED_COMMANDS` whitelist |
| Private Subnets | Deploy in private subnets |
| Minimal IAM | Grant only required permissions |
| Network Isolation | Restrict security group rules |
| Audit Logging | Enable CloudTrail |
| Read-Only FS | Mount container filesystem read-only |

## Monitoring

### CloudWatch Metrics

Monitor:
- CPU/Memory utilization
- Request count
- Error rate
- Response latency

### Log Queries

```
# Failed auth attempts
fields @timestamp, @message
| filter @message like /401/
| sort @timestamp desc

# Blocked commands
fields @timestamp, @message
| filter @message like /blocked/
| sort @timestamp desc

# Command executions
fields @timestamp, @message
| filter @message like /CMD_EXEC/
| sort @timestamp desc
```

## Troubleshooting

| Issue | Check |
|-------|-------|
| Commands blocked | `MCP_ALLOWED_COMMANDS` and `MCP_BLOCKED_COMMANDS` settings |
| Auth failures | `MCP_AUTH_TOKEN` matches client header |
| Timeouts | Increase `MCP_DEFAULT_TIMEOUT` |
| Container not starting | Check CloudWatch logs for errors |
| Health check failing | Verify port 3000 is exposed and accessible |

## Related Documentation

- [INTEGRATION.md](./INTEGRATION.md) - Client configuration
- [README.md](./README.md) - Server documentation
- [TESTING.md](./TESTING.md) - Testing instructions
