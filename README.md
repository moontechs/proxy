# proxy

Automatic reverse proxy for Docker containers - just add labels to your containers and they're instantly accessible.

## What It Does

Route traffic to your Docker containers without manually editing config files:

- **HTTP Routing**: Access containers by hostname (api.example.com, web.local)
- **TCP/UDP Proxying**: Expose databases, SSH, DNS, or any TCP/UDP service
- **Zero Configuration**: Containers auto-register when they start
- **No Downtime**: Config updates happen automatically without interrupting traffic
- **Perfect for Cloudflare Tunnels**: Route multiple services through a single tunnel

## Quick Start

1. **Create the proxy network**:
   ```bash
   docker network create --driver bridge proxy-network
   ```

2. **Pull the image**:
   ```bash
   docker pull ghcr.io/moontechs/proxy:latest
   ```

3. **Create a docker-compose.yml**:
   ```yaml
   version: '3.8'

   services:
     proxy:
       image: ghcr.io/moontechs/proxy:latest
       restart: unless-stopped
       ports:
         - "80:80"
         - "443:443"
       volumes:
         - /var/run/docker.sock:/var/run/docker.sock:ro
       networks:
         - proxy-network

     # Your application - accessed at http://app.local
     webapp:
       image: nginx:alpine
       labels:
         proxy.http.host: "app.local"
         proxy.http.port: "80"
       networks:
         - proxy-network

   networks:
     proxy-network:
       name: proxy-network
       external: true  # Use the network created in step 1
   ```

4. **Start it**:
   ```bash
   docker-compose up -d
   ```

5. **Add to /etc/hosts** (for local testing):
   ```bash
   echo "127.0.0.1 app.local" | sudo tee -a /etc/hosts
   ```

6. **Visit http://app.local** - your container is live!

## Installation

### Using Docker (Recommended)

```bash
# Latest stable version
docker pull ghcr.io/moontechs/proxy:latest

# Specific version
docker pull ghcr.io/moontechs/proxy:1.0.0
```

### Version Tags
- `latest` - Most recent release
- `1.0.0` - Specific version
- `1.0` - Latest patch for 1.0.x
- `1` - Latest minor for 1.x

## How to Use

### HTTP Routing (Hostname-based)

Access containers by domain name - perfect for web services and APIs:

```yaml
services:
  api:
    image: my-api:latest
    labels:
      proxy.http.host: "api.example.com"
      proxy.http.port: "8080"              # Optional, defaults to 80
    networks:
      - proxy-network

  web:
    image: my-web:latest
    labels:
      proxy.http.host: "web.example.com,www.example.com"  # Multiple domains
      proxy.http.port: "3000"
    networks:
      - proxy-network
```

Traffic to `api.example.com` → your API container
Traffic to `web.example.com` → your web container

### TCP Proxying (Port-based)

Expose databases, SSH servers, or any TCP service:

```yaml
services:
  postgres:
    image: postgres:16
    labels:
      proxy.tcp.ports: "5432:5432"         # Host port:Container port
    networks:
      - proxy-network

  ssh-server:
    image: linuxserver/openssh-server
    labels:
      proxy.tcp.ports: "22:2222"           # Expose container's 2222 as host's 22
    networks:
      - proxy-network
```

Connect to PostgreSQL: `psql -h localhost -p 5432`
Connect to SSH: `ssh user@localhost -p 22`

### UDP Proxying

For DNS, VPN, or other UDP services:

```yaml
services:
  dns:
    image: adguard/adguardhome
    labels:
      proxy.tcp.ports: "53:53"             # DNS over TCP
      proxy.udp.ports: "53:53"             # DNS over UDP
    networks:
      - proxy-network
```

### Mix TCP and HTTP

Same container can be accessed both ways:

```yaml
services:
  dev-server:
    image: my-dev-env
    labels:
      proxy.tcp.ports: "22:22"             # SSH access
      proxy.http.host: "dev.local"         # Web interface
      proxy.http.port: "8080"
    networks:
      - proxy-network
```

### Cloudflare Tunnel Integration

Run multiple services through a single Cloudflare tunnel:

```yaml
# Cloudflare tunnel config
ingress:
  - hostname: api.example.com
    service: http://proxy:80
  - hostname: web.example.com
    service: http://proxy:80
  - service: http_status:404

# Your containers
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

The proxy inspects the `Host` header and routes to the correct container.

## Network Architecture Best Practices

### Use Multiple Networks for Service Isolation

**Best Practice**: Services should use **both** an external proxy network AND their own local network for internal components.

**Pattern**:
- **External Network** (`proxy-network`): For proxy communication
- **Local Network** (`your-service-local-network`): For service-specific components (databases, cache, etc.)

This keeps your databases and internal services isolated from the proxy while allowing your web application to be accessible.

**Example with database isolation**:

```yaml
services:
  # Web application - accessible via proxy
  webapp:
    image: my-webapp
    labels:
      proxy.http.host: "app.example.com"
      proxy.http.port: "3000"
    networks:
      - proxy-network        # External - proxy can reach this
      - webapp-local         # Local - for database access
    environment:
      - DATABASE_URL=postgresql://postgres:5432/myapp

  # Database - NOT accessible via proxy
  postgres:
    image: postgres:16
    networks:
      - webapp-local         # Only on local network
    # No proxy labels - isolated from external access

networks:
  proxy-network:
    name: proxy-network
    external: true
  webapp-local:
    name: webapp-local-network
    driver: bridge
```

**Why this works**:
- `webapp` is on both networks, so it can:
  - Receive traffic from the proxy (via `proxy-network`)
  - Connect to the database (via `webapp-local`)
- `postgres` is only on `webapp-local`, so it's isolated from the proxy

**Real-world example with Cloudflare Tunnel**:

```yaml
# Proxy with Cloudflare
services:
  proxy:
    image: ghcr.io/moontechs/proxy:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - "80:80"
      - "443:443"
    networks:
      - proxy-network
      - proxy-local        # For cloudflared

  cloudflared:
    image: cloudflare/cloudflared
    command: tunnel run
    environment:
      - TUNNEL_TOKEN=your-token-here
    networks:
      - proxy-local        # Only needs to reach proxy, not services

networks:
  proxy-network:
    name: proxy-network
    external: true
  proxy-local:
    driver: bridge

---

# Your application (separate docker-compose.yml)
services:
  app:
    image: my-app
    labels:
      proxy.http.host: "app.example.com"
      proxy.http.port: "8080"
    networks:
      - proxy-network      # External - receives traffic from proxy
      - app-local          # Local - for postgres
    environment:
      - DATABASE_URL=postgresql://postgres:5432/app

  postgres:
    image: postgres:16
    networks:
      - app-local          # Isolated from proxy

networks:
  proxy-network:
    name: proxy-network
    external: true
  app-local:
    driver: bridge
```

**Security benefits**:
- Databases are never exposed to the proxy network
- Each service has its own isolated network for internal components
- Proxy can only reach services that explicitly join `proxy-network`
- No need to expose database ports on the host

## Configuration

### Environment Variables

```bash
# Logging
LOG_LEVEL=INFO                             # DEBUG or INFO
LOG_CALLER=false                           # Show source code location in logs

# Docker
DOCKER_HOST=unix:///var/run/docker.sock    # Docker socket path
PROXY_NETWORK=proxy-network                # Docker network name

# Nginx
NGINX_WORKER_CONNECTIONS=1000              # Max connections per worker
```

### Label Reference

**HTTP Routing**:
```yaml
labels:
  proxy.http.host: "example.com"           # Required: hostname(s) - comma-separated for multiple
  proxy.http.port: "80"                    # Optional: container port (default: 80)
  proxy.http.https: "false"                # Optional: use port 443 (default: false)
```

**TCP Proxying**:
```yaml
labels:
  proxy.tcp.ports: "80:8080,443:8443"      # host_port:container_port - comma-separated
```

**UDP Proxying**:
```yaml
labels:
  proxy.udp.ports: "53:53,5353:5300"       # host_port:container_port - comma-separated
```

**Port format**:
- `80:8080` - Different ports (host 80 → container 8080)
- `53` - Same port on both sides (shorthand for `53:53`)

## Troubleshooting

### Container not accessible?

**Check container labels**:
```bash
docker inspect <container-name> | jq '.[0].Config.Labels'
```

Make sure the container has proxy labels and is on the `proxy-network`.

**Verify container is on proxy-network**:
```bash
# List all containers on proxy-network
docker ps --format "{{.Names}}\t{{.Networks}}" | grep proxy-network
```

If your container is missing from this list, add it:
```yaml
# In your docker-compose.yml
services:
  your-service:
    networks:
      - proxy-network  # Add this!

networks:
  proxy-network:
    name: proxy-network
    external: true
```

### "Connection refused" or "502 Bad Gateway"?

**Verify container is running**:
```bash
docker ps | grep <container-name>
```

**Check proxy logs**:
```bash
docker logs proxy
```

**Verify Nginx config**:
```bash
docker exec proxy nginx -t
```

### Hostname not resolving?

For local testing, add to `/etc/hosts`:
```bash
sudo sh -c 'echo "127.0.0.1 app.local api.local" >> /etc/hosts'
```

For production, configure DNS to point to your proxy server.

### Config not updating?

Enable debug logging to see what's happening:
```bash
docker-compose down
LOG_LEVEL=DEBUG docker-compose up
```

Look for logs like:
```
[INFO] registered_container name=webapp tcp_ports=0 udp_ports=0 http_hosts=1
[INFO] Nginx reload successful
```

### Port conflicts?

You'll see errors like:
```
ERROR: TCP port conflict: port 80 claimed by both webapp and api-server
```

Each TCP/UDP port can only be used by one container. For HTTP, use different hostnames instead.

### Need more help?

See detailed guides:
- [DEBUG_OUTPUT.md](DEBUG_OUTPUT.md) - Understanding debug logs
- [examples/](examples/) - Working examples

## Advanced Usage

### Multiple Proxy Networks

Run proxy in multiple separate docker-compose projects:

```yaml
# proxy/docker-compose.yml
services:
  proxy:
    image: ghcr.io/moontechs/proxy:latest
    environment:
      PROXY_NETWORK: "proxy-network"
    networks:
      - proxy-network

networks:
  proxy-network:
    name: proxy-network
    external: true  # Created automatically by proxy

# services/docker-compose.yml (separate directory)
services:
  app:
    image: my-app
    labels:
      proxy.http.host: "app.local"
    networks:
      - proxy-network

networks:
  proxy-network:
    name: proxy-network
    external: true
```

The proxy automatically creates the network on startup.

### Error Messages

**Conflict Detection**:
- `TCP port conflict: port 80 claimed by both webapp and api-server` - Two containers trying to use same TCP port
- `UDP port conflict: port 53 claimed by both dns1 and dns2` - Two containers trying to use same UDP port
- `HTTP hostname conflict: api.local claimed by both api-v1 and api-v2` - Two containers using same hostname

**No conflict**: HTTP + TCP on the same port is allowed (different Nginx modules):
```yaml
labels:
  proxy.tcp.ports: "22:22"                 # SSH via port 22
  proxy.http.host: "admin.local"           # Web via hostname
```

## Documentation

- [QUICKSTART.md](QUICKSTART.md) - Detailed quick start guide
- [DEBUG_OUTPUT.md](DEBUG_OUTPUT.md) - Understanding logs and debugging
- [examples/](examples/) - Working example configurations

---

## For Developers

### Building from Source

**Requirements**:
- Go 1.24+
- Docker
- Make

**Build binary**:
```bash
make build
# or
go build -o bin/proxy .
```

**Build Docker image**:
```bash
make docker-build
# or
docker build -t proxy:latest .
```

**Run tests**:
```bash
make test
# or
go test -v ./...
```

### Development Commands

```bash
make help            # Show all commands
make check           # Run linters and tests
make install-tools   # Install development tools
```

### How It Works

**Process**:
1. On container startup, scans Docker for running containers
2. Reads container labels for proxy configuration
3. Generates Nginx config files
4. Starts Nginx with generated configs

**Two routing modes**:
- **Stream Module** (Layer 4): Raw TCP/UDP port forwarding
- **HTTP Module** (Layer 7): Virtual host routing by hostname

### Project Structure

Key components:
- **cmd/** - CLI commands (generate.go, root.go)
- **config/** - Configuration management
- **docker/** - Docker client and container scanning
- **nginx/** - Nginx config generation (generator.go, templates.go, reloader.go)
- **examples/** - Example setups
- **main.go** - CLI entry point

### CLI Commands

**`proxy generate`**:
- Scans running containers and generates Nginx configs
- Used during container startup via entrypoint

### Generated Nginx Configs

**Stream config** (`/etc/nginx/conf.d/proxy.conf`):
```nginx
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

**HTTP config** (`/etc/nginx/conf.d/http-proxy.conf`):
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

## License

MIT