# proxy-nginx

Dynamic Nginx configuration generator for Docker containers with automatic service discovery and intelligent routing.

## Features

- **Dual Routing Modes**:
  - **Stream Module** (Layer 4): Raw TCP/UDP port-based proxying
  - **HTTP Module** (Layer 7): Hostname-based routing with virtual hosts
- **Auto-Discovery**: Docker event monitoring for real-time container detection
- **Intelligent Routing**: HTTP hostname routing ideal for Cloudflared tunnels
- **Conflict Detection**: Validates TCP ports, UDP ports, and HTTP hostnames separately
- **Config Generation**: Template-based Nginx configuration with checksum detection
- **Zero Downtime**: Graceful Nginx reloads preserve active connections
- **Debug Output**: Full generated configs visible with LOG_LEVEL=DEBUG
- **CLI Interface**: Cobra-powered commands (generate, watch, validate)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      proxy-nginx                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Docker     │  │    Config    │  │    Nginx     │      │
│  │   Events     │→ │  Generator   │→ │   Reloader   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Generated Configs                                   │    │
│  │  • /etc/nginx/conf.d/proxy.conf (stream)            │    │
│  │  • /etc/nginx/conf.d/http-proxy.conf (HTTP)         │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### Stream Module (TCP/UDP)
- Port-based proxying for any TCP or UDP service
- Examples: SSH (22), PostgreSQL (5432), DNS (53)
- Multiple containers cannot share the same port

### HTTP Module (Hostname Routing)
- Virtual host routing using Host header
- Examples: api.example.com, web.example.com
- Multiple hostnames can route through port 80/443
- Perfect for Cloudflared tunnels

## Requirements

- Go 1.24+
- Docker daemon access
- Nginx (provided by nginx:alpine base image)

## Installation

### Using Published Docker Image (Recommended)

```bash
# Pull from GitHub Container Registry
docker pull ghcr.io/moontechs/proxy:latest

# Or use semantic versioning tags
docker pull ghcr.io/moontechs/proxy:1.0.0  # Exact version
docker pull ghcr.io/moontechs/proxy:1.0    # Auto-updates for patches
docker pull ghcr.io/moontechs/proxy:1      # Auto-updates for all 1.x
```

**Multi-Tag Strategy:** Each release (e.g., `1.0.0`) creates multiple tags: `1.0.0`, `1.0`, `1`, `latest`
See [Docker Versioning Strategy](.github/DOCKER_VERSIONING.md) for details.

### Build Binary

```bash
make build
# or
go build -o bin/proxy-nginx .
```

### Build Docker Image

```bash
make docker-build
# or
docker build -t proxy-nginx:latest .
```

## Configuration

### Environment Variables

```bash
# Logging
LOG_LEVEL=INFO                                    # DEBUG, INFO (default)
LOG_CALLER=false                                  # Show caller info

# Docker
DOCKER_HOST=unix:///var/run/docker.sock           # Docker socket

# Nginx Paths (defaults work with nginx:alpine)
STREAM_CONFIG_PATH=/etc/nginx/conf.d/proxy.conf
HTTP_CONFIG_PATH=/etc/nginx/conf.d/http-proxy.conf
NGINX_RELOAD_CMD=nginx -s reload
```

## Docker Label Schema

### Stream Routing (TCP/UDP)

```yaml
labels:
  proxy.tcp.ports: "22,80:8080,443:8443"    # TCP port mappings
  proxy.udp.ports: "53:53,5353:5300"        # UDP port mappings
```

**Port Format**:
- `80:8080` - Proxy port 80 → container port 8080
- `53` - Proxy port 53 → container port 53 (same on both sides)
- Comma-separated for multiple ports

### HTTP Routing (Hostname-based)

```yaml
labels:
  proxy.http.host: "api.example.com"        # Required: hostname(s) for routing
  proxy.http.port: "8080"                   # Optional: container port (default: 80)
  proxy.http.https: "false"                 # Optional: use HTTPS listener (default: false)
```

**Multiple Hostnames**:
```yaml
labels:
  proxy.http.host: "app.local,www.local,api.local"
  proxy.http.port: "3000"
```

### Mixed Routing (Stream + HTTP)

The same container can have both:

```yaml
labels:
  proxy.tcp.ports: "22:22"                  # SSH access via port 22
  proxy.http.host: "admin.example.com"      # Web admin via hostname
  proxy.http.port: "8080"
```

No conflict because they use different Nginx modules.

## CLI Commands

### generate

Generate Nginx configs once and exit:

```bash
proxy-nginx generate
```

### watch

Monitor Docker events and regenerate configs automatically:

```bash
proxy-nginx watch
```

This is the primary mode for production - watches for container start/stop/die events.

### validate

Validate Nginx configs without applying:

```bash
proxy-nginx validate
```

## Usage Examples

### Stream Proxying (TCP)

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16
    labels:
      proxy.tcp.ports: "5432:5432"
    networks:
      - proxy-network

  ssh-server:
    image: linuxserver/openssh-server
    labels:
      proxy.tcp.ports: "22:2222"
    networks:
      - proxy-network
```

### Stream Proxying (UDP)

```yaml
services:
  dns:
    image: adguard/adguardhome
    labels:
      proxy.tcp.ports: "53:53"     # DNS over TCP
      proxy.udp.ports: "53:53"     # DNS over UDP
    networks:
      - proxy-network
```

### HTTP Hostname Routing

```yaml
services:
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
      proxy.http.host: "web.example.com,www.example.com"
      proxy.http.port: "3000"
    networks:
      - proxy-network
```

Nginx generates:
- `api.example.com` → container IP:8080
- `web.example.com` → container IP:3000
- `www.example.com` → container IP:3000

### Cloudflared Integration

Perfect for Cloudflared tunnels where all traffic comes through ports 80/443:

```yaml
# Cloudflared tunnel config
ingress:
  - hostname: api.example.com
    service: http://proxy:80
  - hostname: web.example.com
    service: http://proxy:80
  - service: http_status:404

# Your services
services:
  api:
    labels:
      proxy.http.host: "api.example.com"
      proxy.http.port: "8080"

  web:
    labels:
      proxy.http.host: "web.example.com"
      proxy.http.port: "3000"
```

Nginx routes by Host header to the correct backend.

## Complete Docker Compose Example

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
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      LOG_LEVEL: "INFO"
      LOG_CALLER: "false"
      DOCKER_HOST: "unix:///var/run/docker.sock"
    networks:
      - proxy-network

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

  database:
    image: postgres:16
    labels:
      proxy.tcp.ports: "5432:5432"
    networks:
      - proxy-network

networks:
  proxy-network:
    driver: bridge
```

## Conflict Detection

### TCP Port Conflicts
```
ERROR: TCP port conflict: port 80 claimed by both webapp and api-server
```

### UDP Port Conflicts
```
ERROR: UDP port conflict: port 53 claimed by both dns1 and dns2
```

### HTTP Hostname Conflicts
```
ERROR: HTTP hostname conflict: api.example.com claimed by both api-v1 and api-v2
```

**No Conflict**: HTTP + TCP on same port (different modules):
```yaml
labels:
  proxy.tcp.ports: "22:22"              # Stream module
  proxy.http.host: "admin.local"        # HTTP module
```

## Debug Output

Enable DEBUG logging to see generated Nginx configs:

```bash
LOG_LEVEL=DEBUG proxy-nginx watch
```

Output shows:
```
[DEBUG] [Generator] stream config generated:
# Auto-generated by proxy-nginx at 2025-12-28T19:28:27+01:00

upstream tcp_5432 {
    server 172.17.0.2:5432;
}

server {
    listen 5432;
    proxy_pass tcp_5432;
    proxy_connect_timeout 10s;
    proxy_timeout 5m;
}
```

See [DEBUG_OUTPUT.md](DEBUG_OUTPUT.md) for complete guide.

## Generated Nginx Configs

### Stream Config (/etc/nginx/conf.d/proxy.conf)

```nginx
upstream tcp_80 {
    server 172.17.0.2:8080;
}

server {
    listen 80;
    proxy_pass tcp_80;
    proxy_connect_timeout 10s;
    proxy_timeout 5m;
}

upstream udp_53 {
    server 172.17.0.2:53;
}

server {
    listen 53 udp;
    proxy_pass udp_53;
    proxy_timeout 30s;
    proxy_responses 1;
}
```

### HTTP Config (/etc/nginx/conf.d/http-proxy.conf)

```nginx
upstream http_api_example_com {
    server 172.17.0.3:8080;
}

server {
    listen 80;
    server_name api.example.com;

    location / {
        proxy_pass http://http_api_example_com;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Development

### Using Make

```bash
# Show all commands
make help

# Install tools
make install-tools

# Run checks
make check

# Build
make build

# Run tests
make test

# Build Docker image
make docker-build
```

### Running Tests

```bash
make test
# or
go test -v ./...
```

### Coverage

```bash
make test
```

Current coverage:
- `config`: 100%
- `nginx`: 68.8%
- `docker`: 18.1%

## Project Structure

```
.
├── cmd/                    # CLI commands (Cobra)
│   ├── generate.go        # One-shot config generation
│   ├── watch.go           # Docker event monitoring
│   └── validate.go        # Config validation
├── config/                # Configuration management
├── docker/                # Docker client and event handling
├── nginx/                 # Nginx config generation
│   ├── generator.go       # Template execution and file writing
│   ├── templates.go       # Embedded Nginx templates
│   ├── reloader.go        # Nginx reload orchestration
│   └── validator.go       # Config validation via nginx -t
├── examples/              # Example setups
│   ├── http-services/     # HTTP hostname routing example
│   └── stream-services/   # TCP/UDP proxying example
└── main.go               # CLI entry point
```

## Examples

Complete working examples with documentation:

- [HTTP Services](examples/http-services/) - Hostname-based routing
- See [examples/](examples/) for more

## Documentation

- [QUICKSTART.md](QUICKSTART.md) - Quick start guide
- [DEBUG_OUTPUT.md](DEBUG_OUTPUT.md) - Debug logging guide
- [PRD.md](PRD.md) - Product requirements
- [IMPLEMENTATION.md](IMPLEMENTATION.md) - Implementation details

## Troubleshooting

### Container not proxied?

Check labels:
```bash
docker inspect <container> | jq '.[0].Config.Labels'
```

### Nginx reload failed?

Check Nginx logs:
```bash
docker logs proxy-nginx
docker exec proxy-nginx nginx -t
```

### Config not updating?

Enable DEBUG:
```bash
LOG_LEVEL=DEBUG docker-compose up
```

### Hostname not resolving?

For local testing, add to `/etc/hosts`:
```bash
127.0.0.1 api.local web.local
```

## License

MIT
