# Quick Start Guide

Get started with proxy-nginx in minutes.

## Prerequisites

- Docker daemon running
- Access to Docker socket (`/var/run/docker.sock`)
- Go 1.24+ (for building from source)

## Quick Setup (Docker)

### 1. Clone and Build

```bash
git clone https://github.com/moontechs/proxy.git
cd proxy
docker build -t proxy-nginx:latest .
```

### 2. Create docker-compose.yml

```yaml
version: '3.8'

services:
  proxy:
    image: proxy-nginx:latest
    container_name: proxy-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      LOG_LEVEL: "INFO"
      DOCKER_HOST: "unix:///var/run/docker.sock"
    networks:
      - proxy-network

  # Example web application
  webapp:
    image: ghcr.io/umputun/echo-http
    command: --message="Hello from webapp"
    labels:
      proxy.http.host: "app.local"
      proxy.http.port: "8080"
    networks:
      - proxy-network

networks:
  proxy-network:
    driver: bridge
```

### 3. Add Hostname to /etc/hosts

```bash
echo "127.0.0.1 app.local" | sudo tee -a /etc/hosts
```

### 4. Start Services

```bash
docker-compose up -d
```

### 5. Test

```bash
curl http://app.local
# Output: Hello from webapp
```

## Usage Scenarios

### Scenario 1: HTTP Hostname Routing

Perfect for hosting multiple web applications on ports 80/443:

```yaml
services:
  api:
    image: my-api:latest
    labels:
      proxy.http.host: "api.local"
      proxy.http.port: "8080"
    networks:
      - proxy-network

  web:
    image: my-web:latest
    labels:
      proxy.http.host: "web.local"
      proxy.http.port: "3000"
    networks:
      - proxy-network
```

Add to `/etc/hosts`:
```bash
127.0.0.1 api.local web.local
```

Test:
```bash
curl http://api.local
curl http://web.local
```

### Scenario 2: TCP Port Proxying

For services that need dedicated ports (databases, SSH, etc.):

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: secret
    labels:
      proxy.tcp.ports: "5432:5432"
    networks:
      - proxy-network

  ssh:
    image: linuxserver/openssh-server
    labels:
      proxy.tcp.ports: "22:2222"
    networks:
      - proxy-network
```

Don't forget to expose ports in proxy service:
```yaml
  proxy:
    ports:
      - "22:22"
      - "5432:5432"
```

Test:
```bash
psql -h localhost -U postgres
ssh user@localhost
```

### Scenario 3: UDP Proxying (DNS)

For UDP services like DNS servers:

```yaml
services:
  dns:
    image: adguard/adguardhome
    labels:
      proxy.tcp.ports: "53:53"      # DNS over TCP
      proxy.udp.ports: "53:53"      # DNS over UDP
    networks:
      - proxy-network
```

Expose UDP port:
```yaml
  proxy:
    ports:
      - "53:53/udp"
      - "53:53"
```

Test:
```bash
dig @localhost example.com
```

### Scenario 4: Mixed Routing

Same container with both stream (port-based) and HTTP (hostname) routing:

```yaml
services:
  admin:
    image: my-admin-app
    labels:
      proxy.tcp.ports: "22:22"              # SSH access
      proxy.http.host: "admin.local"        # Web interface
      proxy.http.port: "8080"
    networks:
      - proxy-network
```

Access via:
- SSH: `ssh admin@localhost`
- Web: `http://admin.local`

### Scenario 5: Cloudflared Tunnel

Perfect for Cloudflared tunnels where all traffic arrives on port 80/443:

**Cloudflared config (config.yml):**
```yaml
tunnel: your-tunnel-id
credentials-file: /path/to/credentials.json

ingress:
  - hostname: api.example.com
    service: http://proxy:80
  - hostname: web.example.com
    service: http://proxy:80
  - service: http_status:404
```

**Docker Compose:**
```yaml
services:
  cloudflared:
    image: cloudflare/cloudflared:latest
    command: tunnel run
    volumes:
      - ./config.yml:/etc/cloudflared/config.yml:ro
      - ./credentials.json:/etc/cloudflared/credentials.json:ro
    networks:
      - proxy-network

  api:
    image: my-api:latest
    labels:
      proxy.http.host: "api.example.com"
      proxy.http.port: "8080"
    networks:
      - proxy-network

  web:
    image: my-web:latest
    labels:
      proxy.http.host: "web.example.com"
      proxy.http.port: "3000"
    networks:
      - proxy-network
```

Cloudflared sends traffic to proxy:80, Nginx routes by Host header to the correct backend.

## CLI Commands

### Watch Mode (Production)

Monitor Docker events and auto-regenerate configs:

```bash
# In Docker (default)
docker-compose up

# Standalone
./bin/proxy-nginx watch
```

### Generate Once

Generate configs once and exit:

```bash
./bin/proxy-nginx generate
```

### Validate

Validate Nginx configs without applying:

```bash
./bin/proxy-nginx validate
```

## Environment Variables

```bash
# Logging
LOG_LEVEL=DEBUG          # DEBUG, INFO (default)
LOG_CALLER=true          # Show caller info (default: false)

# Docker
DOCKER_HOST=unix:///var/run/docker.sock

# Nginx Config Paths (defaults)
STREAM_CONFIG_PATH=/etc/nginx/conf.d/proxy.conf
HTTP_CONFIG_PATH=/etc/nginx/conf.d/http-proxy.conf
NGINX_RELOAD_CMD=nginx -s reload
```

## Label Reference

### Stream Routing (TCP/UDP)

```yaml
labels:
  proxy.tcp.ports: "80:8080,443:8443"    # TCP: proxy_port:container_port
  proxy.udp.ports: "53:53,5353:5300"     # UDP: proxy_port:container_port
```

**Format**:
- `80:8080` - Proxy port 80 → container port 8080
- `53` - Same port on both sides (53 → 53)
- Comma-separated for multiple ports

### HTTP Routing (Hostname)

```yaml
labels:
  proxy.http.host: "api.local"           # Required: hostname(s)
  proxy.http.port: "8080"                # Optional: container port (default: 80)
  proxy.http.https: "false"              # Optional: HTTPS listener (default: false)
```

**Multiple Hostnames**:
```yaml
labels:
  proxy.http.host: "app.local,www.local,api.local"
  proxy.http.port: "3000"
```

## Debugging

### Enable Debug Logging

```bash
LOG_LEVEL=DEBUG docker-compose up
```

You'll see full generated Nginx configs:

```
[DEBUG] [Generator] stream config generated:
upstream tcp_5432 {
    server 172.17.0.2:5432;
}

server {
    listen 5432;
    proxy_pass tcp_5432;
}
```

### Check Generated Configs

```bash
# Stream config (TCP/UDP)
docker exec proxy-nginx cat /etc/nginx/conf.d/proxy.conf

# HTTP config (hostname routing)
docker exec proxy-nginx cat /etc/nginx/conf.d/http-proxy.conf
```

### Test Nginx Configuration

```bash
docker exec proxy-nginx nginx -t
```

### View Nginx Logs

```bash
# Access logs
docker exec proxy-nginx tail -f /var/log/nginx/access.log

# Stream logs
docker exec proxy-nginx tail -f /var/log/nginx/stream.log

# Error logs
docker exec proxy-nginx tail -f /var/log/nginx/error.log
```

### Watch Docker Events

```bash
docker events --filter 'event=start' --filter 'event=stop' --filter 'event=die'
```

## Common Issues

### Container Not Proxied

**Check labels:**
```bash
docker inspect <container> | jq '.[0].Config.Labels'
```

**Required labels:**
- For TCP/UDP: `proxy.tcp.ports` and/or `proxy.udp.ports`
- For HTTP: `proxy.http.host`

**Network requirement:**
- Container must be on same network as proxy

### Hostname Not Resolving (Local Testing)

Add to `/etc/hosts`:
```bash
sudo bash -c 'cat >> /etc/hosts <<EOF
127.0.0.1 api.local
127.0.0.1 web.local
127.0.0.1 admin.local
EOF'
```

### Port Conflict Errors

**Error:**
```
ERROR: TCP port conflict: port 80 claimed by both webapp and api-server
```

**Solution:**
- Use different ports for stream routing
- OR use HTTP hostname routing instead (multiple hostnames can share port 80/443)

**Example - Change from:**
```yaml
webapp:
  labels:
    proxy.tcp.ports: "80:8080"
api:
  labels:
    proxy.tcp.ports: "80:3000"    # ❌ Conflict!
```

**To:**
```yaml
webapp:
  labels:
    proxy.http.host: "webapp.local"
    proxy.http.port: "8080"
api:
  labels:
    proxy.http.host: "api.local"
    proxy.http.port: "3000"        # ✅ No conflict
```

### Nginx Reload Failed

**Check proxy logs:**
```bash
docker logs proxy-nginx
```

**Test config manually:**
```bash
docker exec proxy-nginx nginx -t
```

**Common causes:**
- Invalid container IP (container stopped)
- Port already in use by another process
- Syntax error in generated config (bug - please report)

### Config Not Updating

**Enable DEBUG logging:**
```bash
LOG_LEVEL=DEBUG docker-compose up
```

**Check if events are detected:**
```bash
docker logs -f proxy-nginx | grep "event"
```

**Manually trigger regeneration:**
```bash
docker exec proxy-nginx /usr/local/bin/proxy-nginx generate
docker exec proxy-nginx nginx -s reload
```

## Building from Source

### Local Build

```bash
make build
./bin/proxy-nginx watch
```

### Docker Build

```bash
make docker-build
docker run -d \
  --name proxy-nginx \
  -p 80:80 -p 443:443 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  proxy-nginx:latest
```

### Development

```bash
# Install tools
make install-tools

# Run checks (fmt, vet, lint, test)
make check

# Run tests
make test

# Show all commands
make help
```

## Production Deployment

### 1. Security

```yaml
services:
  proxy:
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro  # Read-only
    environment:
      LOG_LEVEL: "INFO"                                # Not DEBUG
    restart: unless-stopped
```

### 2. Monitoring

```bash
# Check proxy health
docker ps -f name=proxy-nginx

# Monitor logs
docker logs -f proxy-nginx

# Watch for errors
docker logs proxy-nginx 2>&1 | grep ERROR
```

### 3. Resource Limits

```yaml
services:
  proxy:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

### 4. Health Checks

```yaml
services:
  proxy:
    healthcheck:
      test: ["CMD", "nginx", "-t"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

## Next Steps

- Read [README.md](README.md) for complete documentation
- See [examples/http-services/](examples/http-services/) for working example
- Check [DEBUG_OUTPUT.md](DEBUG_OUTPUT.md) for debug logging details
- Review [PRD.md](PRD.md) for design decisions

## Complete Example

Full working setup with all features:

```yaml
version: '3.8'

services:
  proxy:
    build: .
    container_name: proxy-nginx
    restart: unless-stopped
    ports:
      - "22:22"          # SSH
      - "80:80"          # HTTP
      - "443:443"        # HTTPS
      - "53:53/udp"      # DNS
      - "5432:5432"      # PostgreSQL
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      LOG_LEVEL: "INFO"
      DOCKER_HOST: "unix:///var/run/docker.sock"
    networks:
      - proxy-network
    healthcheck:
      test: ["CMD", "nginx", "-t"]
      interval: 30s
      timeout: 10s
      retries: 3

  # HTTP hostname routing
  webapp:
    image: nginx:alpine
    labels:
      proxy.http.host: "app.local"
      proxy.http.port: "80"
    networks:
      - proxy-network

  api:
    image: my-api:latest
    labels:
      proxy.http.host: "api.local"
      proxy.http.port: "8080"
    networks:
      - proxy-network

  # TCP stream routing
  database:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: secret
    labels:
      proxy.tcp.ports: "5432:5432"
    networks:
      - proxy-network

  ssh:
    image: linuxserver/openssh-server
    labels:
      proxy.tcp.ports: "22:2222"
    networks:
      - proxy-network

  # Mixed routing
  admin:
    image: my-admin:latest
    labels:
      proxy.tcp.ports: "2222:22"              # SSH on port 2222
      proxy.http.host: "admin.local"          # Web interface
      proxy.http.port: "8080"
    networks:
      - proxy-network

networks:
  proxy-network:
    driver: bridge
```

Add to `/etc/hosts`:
```bash
sudo bash -c 'cat >> /etc/hosts <<EOF
127.0.0.1 app.local api.local admin.local
EOF'
```

Start and test:
```bash
docker-compose up -d
curl http://app.local
curl http://api.local
psql -h localhost -U postgres
ssh admin@localhost -p 2222
```
