# SWARM Deployment Guide

This document provides comprehensive guidance for deploying SWARM in various environments.

## Table of Contents

- [Deployment Overview](#deployment-overview)
- [Local Development](#local-development)
- [Production Deployment](#production-deployment)
- [Cloud Deployment](#cloud-deployment)
- [Docker Deployment](#docker-deployment)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Security](#security)
- [Troubleshooting](#troubleshooting)

## Deployment Overview

SWARM can be deployed in multiple configurations:

1. **Local Development**: Single instance for development/testing
2. **Production Server**: Dedicated server with authentication
3. **Distributed**: Multiple SWARM instances working together
4. **Containerized**: Docker/Kubernetes deployment

## Local Development

### Quick Start

```bash
# Clone repository
git clone https://github.com/mojomast/clanker01.git
cd clanker01

# Build and run
go build -o swarm ./cmd/swarm
./swarm
```

### Development Configuration

Create `~/.config/swarm/config.yaml`:

```yaml
version: "1.0"
project_name: "swarm-dev"

server:
  host: "127.0.0.1"
  port: 8080
  enable_tls: false

logging:
  level: "debug"
  format: "text"  # Easier to read locally
  file: ""  # Output to stdout in development
```

### Development Tips

- Use `debug` log level for detailed output
- Disable TLS for easier development
- Use local LLM providers (e.g., Ollama) to avoid API costs
- Run tests frequently: `go test ./...`

## Production Deployment

### Prerequisites

- Server with Linux/macOS
- Go 1.24+ (or compile binary)
- 4GB+ RAM minimum, 8GB+ recommended
- Database (PostgreSQL) for cold storage (optional but recommended)
- Redis for warm storage (optional but recommended)

### System Requirements

**Minimum**:
- CPU: 2 cores
- RAM: 4GB
- Disk: 10GB
- Network: Stable internet connection

**Recommended**:
- CPU: 4+ cores
- RAM: 8-16GB
- Disk: 50GB+ SSD
- Network: High bandwidth for remote agents

### Installation

```bash
# Download binary
wget https://github.com/mojomast/clanker01/releases/latest/download/swarm-linux-amd64
chmod +x swarm-linux-amd64

# Create user
useradd -r -s /bin/bash swarm

# Create directories
sudo mkdir -p /opt/swarm
sudo chown swarm:swarm /opt/swarm

# Install
sudo mv swarm-linux-amd64 /opt/swarm/swarm
```

### Production Configuration

Create `/opt/swarm/config/config.yaml`:

```yaml
version: "1.0"
project_name: "swarm-prod"

server:
  host: "0.0.0.0"
  port: 8080
  enable_tls: true
  tls_cert: "/etc/swarm/tls/cert.pem"
  tls_key: "/etc/swarm/tls/key.pem"

logging:
  level: "info"
  format: "json"
  file: "/var/log/swarm/swarm.log"

providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"  # Use env vars for secrets
    models:
      - id: "claude-3-sonnet-20240229"
        alias: "claude-3-sonnet"
        max_tokens: 200000

agents:
  architect:
    model: "claude-3-sonnet"
    max_concurrent: 5  # Scale based on CPU/RAM
  coder:
    model: "claude-3-sonnet"
    max_concurrent: 10
```

### Systemd Service

Create `/etc/systemd/system/swarm.service`:

```ini
[Unit]
Description=SWARM Multi-Agent AI Platform
After=network.target

[Service]
Type=simple
User=swarm
WorkingDirectory=/opt/swarm
ExecStart=/opt/swarm/swarm serve
ExecStop=/bin/kill -s SIGTERM $MAINPID
Restart=on-failure
RestartSec=10
Environment="GH_CONFIG_DIR=/opt/swarm/config"

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable swarm
sudo systemctl start swarm
sudo systemctl status swarm
```

### Nginx Reverse Proxy

Configure `/etc/nginx/sites-available/swarm.conf`:

```nginx
upstream swarm {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name swarm.example.com;

    location /api/ {
        proxy_pass http://swarm/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /ws {
        proxy_pass http://swarm/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Cloud Deployment

### AWS

Deploy to EC2 with Elastic Beanstalk:

```bash
# Create Elastic Beanstalk application
eb init -p go -r us-west-2

# Deploy
eb create swarm-production
```

### Google Cloud Platform

Deploy to Cloud Run:

```bash
# Build container
gcloud builds submit --tag gcr.io/mojomast/swarm

# Deploy
gcloud run deploy --image gcr.io/mojomast/swarm \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

### Azure

Deploy to App Service:

```bash
# Create web app
az webapp create -g swarm -p F1Free

# Deploy
az webapp up -n swarm -g swarm
```

### DigitalOcean

Deploy to App Platform:

```bash
# Create app
doctl apps create swarm --spec swarm-spec.yaml

# Deploy
doctl apps deploy swarm
```

## Docker Deployment

### Dockerfile

Create `Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o swarm ./cmd/swarm

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/swarm .
COPY --from=builder /app/config ./config/

EXPOSE 8080

CMD ["./swarm", "serve"]
```

### docker-compose.yml

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  swarm:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - SWARM_LOG_LEVEL=info
    volumes:
      - ./config:/root/.config/swarm
      - ./logs:/var/log/swarm
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=swarm
      - POSTGRES_USER=swarm
      - POSTGRES_PASSWORD=swarm_pass
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### Build and Run

```bash
# Build image
docker build -t mojomast/swarm:latest .

# Run with docker-compose
docker-compose up -d

# Run standalone
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config:/root/.config/swarm \
  -e ANTHROPIC_API_KEY=your_key \
  mojomast/swarm:latest
```

## Kubernetes Deployment

### Deployment Manifest

Create `k8s/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: swarm
spec:
  replicas: 3
  selector:
    matchLabels:
      app: swarm
  template:
    metadata:
      labels:
        app: swarm
    spec:
      containers:
      - name: swarm
        image: mojomast/swarm:latest
        ports:
        - containerPort: 8080
        env:
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: swarm-secrets
              key: anthropic-api-key
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: swarm-service
spec:
  selector:
    app: swarm
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
---
apiVersion: v1
kind: Secret
metadata:
  name: swarm-secrets
type: Opaque
data:
  anthropic-api-key: <base64-encoded-key>
```

### Deploy

```bash
# Apply manifests
kubectl apply -f k8s/deployment.yaml

# Check status
kubectl get pods -l app=swarm

# Get logs
kubectl logs -f deployment/swarm
```

### Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: swarm-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: swarm
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Configuration

### Environment Variables

- `SWARM_CONFIG`: Path to config file
- `SWARM_LOG_LEVEL`: Log level (debug, info, warn, error)
- `SWARM_LOG_FORMAT`: Log format (text, json)
- `ANTHROPIC_API_KEY`: Anthropic API key
- `OPENAI_API_KEY`: OpenAI API key
- `GOOGLE_API_KEY`: Google AI API key
- `POSTGRES_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string

### Config File Locations

- Linux/macOS: `~/.config/swarm/config.yaml`
- Windows: `%APPDATA%\swarm\config.yaml`
- Custom: Specified via `--config` flag

## Monitoring

### Health Checks

SWARM provides health endpoints:

```bash
# HTTP health check
curl http://localhost:8080/health

# Expected response
{"status":"healthy","version":"1.0.0","uptime":"3600"}
```

### Metrics Export

Prometheus metrics endpoint:

```bash
# Metrics available at /metrics
curl http://localhost:8080/metrics

# Example metrics
swarm_agents_total{status="running"} 5
swarm_tasks_total{status="completed"} 123
swarm_llm_tokens_total{provider="anthropic"} 45678
```

### Logging

Log locations:

- **Development**: Stdout
- **Production**: `/var/log/swarm/swarm.log`
- **Container**: `/var/log/swarm/` (volume mount)
- **Cloud**: Cloud provider logging service

Log rotation:

```bash
# Configure logrotate for production
cat <<EOF > /etc/logrotate.d/swarm
/var/log/swarm/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 swarm swarm
}
EOF
```

### Distributed Tracing

Enable OpenTelemetry tracing:

```yaml
monitoring:
  tracing:
    enabled: true
    exporter: "stdout"  # or "jaeger", "otlp"
    sampling: 0.1  # Sample 10% of traces
```

## Security

### TLS Configuration

Generate certificates:

```bash
# Self-signed for testing
openssl req -x509 -newkey rsa:4096 \
  -keyout key.pem -out cert.pem -days 365 -nodes

# Let's Encrypt for production
certbot certonly --standalone -d swarm.example.com
```

Configure SWARM:

```yaml
server:
  enable_tls: true
  tls_cert: "/etc/swarm/tls/cert.pem"
  tls_key: "/etc/swarm/tls/key.pem"
  min_tls_version: "1.2"
```

### Firewall Rules

```bash
# UFW
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 8080/tcp

# iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
```

### Rate Limiting

Configure API rate limits:

```yaml
server:
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst: 10
```

### RBAC Configuration

Define roles and permissions:

```yaml
rbac:
  roles:
    admin:
      permissions: ["*"]  # All permissions
    user:
      permissions: [
        "agents:read",
        "agents:execute",
        "tasks:create",
        "tasks:read"
      ]
    readonly:
      permissions: [
        "agents:read",
        "tasks:read",
        "skills:read"
      ]
```

## Troubleshooting

### Common Issues

#### SWARM Won't Start

**Symptoms**: Binary exits immediately

**Solutions**:
```bash
# Check Go version
go version  # Must be 1.24+

# Check configuration
swarm --validate-config

# Check logs
tail -f /var/log/swarm/swarm.log

# Check ports
netstat -tuln | grep 8080
```

#### Agent Not Responding

**Symptoms**: Agent hangs or times out

**Solutions**:
- Check LLM provider connectivity
- Verify API key is valid
- Check agent pool capacity
- Increase timeout in config
- Check memory usage: `top`

#### High Memory Usage

**Symptoms**: Out of memory errors

**Solutions**:
```yaml
# Reduce agent pool sizes
agents:
  architect:
    max_concurrent: 2  # Reduce from 5

# Enable memory limits
server:
  max_memory_mb: 4096

# Increase tiered storage size
context:
  hot_store:
    max_size_mb: 256  # Reduce LRU size
```

#### Database Connection Issues

**Symptoms**: Connection refused to PostgreSQL/Redis

**Solutions**:
```bash
# Check if database is running
systemctl status postgresql
systemctl status redis

# Check connection string
psql -h localhost -U swarm -d swarm -c "SELECT 1;"

# Check logs
tail -f /var/log/postgresql/postgresql.log
```

### Performance Issues

#### Slow Response Times

**Solutions**:
```yaml
# Enable caching
providers:
  enable_cache: true
  cache_ttl: 3600

# Increase agent pools
agents:
  coder:
    max_concurrent: 10  # Increase parallelism

# Use faster models
providers:
  openai:
    models:
      - id: "gpt-4-turbo"  # Faster than gpt-4
```

#### High API Costs

**Solutions**:
```yaml
# Enable cost tracking
monitoring:
  cost_alerts:
    enabled: true
    threshold_usd: 100  # Alert at $100/day

# Use semantic caching
providers:
  cache:
    enabled: true
    similarity_threshold: 0.85  # Cache 85%+ similar prompts

# Set token limits
agents:
  coder:
    max_tokens_per_task: 4000  # Limit output size
```

### Debug Mode

Enable detailed debugging:

```bash
# Run with debug logging
SWARM_LOG_LEVEL=debug ./swarm

# Enable request/response logging
providers:
  log_requests: true
  log_responses: true

# Enable profiling
./swarm server start --profile-cpu --profile-mem
```

### Getting Help

If issues persist:

1. Check logs: `journalctl -u swarm -f`
2. Review [ARCHITECTURE.md](ARCHITECTURE.md)
3. Search [GitHub Issues](https://github.com/mojomast/clanker01/issues)
4. Join [Discord](https://discord.gg/swarm)
5. Create new issue with full context

## Backup and Recovery

### Database Backups

```bash
# Automated backups
0 2 * * * * postgres-user=swarm pg_dump swarm > /backup/swarm-$(date +\%Y\%m\%d).sql

# Restore
psql -U swarm -d swarm < /backup/swarm-2024-03-21.sql
```

### Configuration Backups

```bash
# Backup config
cp ~/.config/swarm/config.yaml ~/.config/swarm/config.yaml.backup

# Restore
cp ~/.config/swarm/config.yaml.backup ~/.config/swarm/config.yaml
```

### Session Recovery

SWARM auto-saves sessions. To recover:

```bash
# List available sessions
swarm session list

# Restore session
swarm session restore --session-id abc123
```

## Upgrades

### Upgrade Procedure

```bash
# 1. Backup current installation
sudo systemctl stop swarm
cp -r /opt/swarm /opt/swarm.backup

# 2. Download new version
wget https://github.com/mojomast/clanker01/releases/latest/download/swarm-linux-amd64

# 3. Install
sudo mv swarm-linux-amd64 /opt/swarm/swarm

# 4. Test new version
sudo systemctl start swarm
./swarm --version

# 5. Rollback if needed
sudo systemctl stop swarm
cp -r /opt/swarm.backup/* /opt/swarm/
sudo systemctl start swarm
```

---

For additional support, see [README.md](README.md#-support).
