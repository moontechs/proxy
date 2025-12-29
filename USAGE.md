# Using proxy-nginx

Complete guide to using proxy-nginx after images are built and published to GitHub Container Registry (GHCR).

## Installation Options

### Option 1: Use Published Docker Image (Recommended)

Use a specific version:

```bash
# Exact version (immutable)
docker pull ghcr.io/moontechs/proxy:1.0.0

# Major.minor version (auto-updates for patches)
docker pull ghcr.io/moontechs/proxy:1.0

# Major version only (auto-updates for all 1.x releases)
docker pull ghcr.io/moontechs/proxy:1
```

**Version Tag Behavior:**
- When tag `1.0.0` is released, it creates Docker tags: `1.0.0`, `1.0`, `1`, `latest`
- When tag `2.1.3` is released, it creates Docker tags: `2.1.3`, `2.1`, `2`, `latest`
- See [Docker Versioning Strategy](.github/DOCKER_VERSIONING.md) for details

### Option 2: Build Locally

```bash
# Clone the repository
git clone https://github.com/moontechs/proxy.git
cd proxy

# Build using Make
make docker-build

# Or build directly
docker build -t proxy-nginx:latest .
```

## Quick Start

### 1. Create docker-compose.yml

Copy the example configuration:

```bash
cp docker-compose.example.yml docker-compose.yml
```

Edit `docker-compose.yml` and update the proxy service to use the published image:

```yaml
services:
  proxy:
    image: ghcr.io/moontechs/proxy:1.0.0  # Use specific version
    # OR
    build: .  # Build from local source

    container_name: proxy-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      LOG_LEVEL: "INFO"
    networks:
      - proxy-network
```

### 2. Start the Proxy

```bash
docker-compose up -d
```

### 3. Verify It's Running

```bash
# Check proxy logs
docker-compose logs -f proxy

# Check Nginx is running
docker exec proxy-nginx nginx -t
```

### 4. Add Your Services

Add services to `docker-compose.yml` with appropriate labels:

```yaml
services:
  myapp:
    image: myapp:latest
    labels:
      proxy.http.host: "myapp.local"
      proxy.http.port: "8080"
    networks:
      - proxy-network
```

### 5. Restart to Apply Changes

```bash
docker-compose up -d
```

The proxy automatically detects the new container and updates its configuration!

## Configuration Examples

### HTTP Hostname-Based Routing

Route traffic based on the `Host` header:

```yaml
services:
  api:
    image: my-api:latest
    labels:
      proxy.http.host: "api.example.com"  # Required
      proxy.http.port: "8080"             # Optional (default: 80)
      proxy.http.https: "false"           # Optional (default: false)
    networks:
      - proxy-network
```

**Test:**
```bash
curl -H "Host: api.example.com" http://localhost
```

**Multiple Hostnames:**
```yaml
labels:
  proxy.http.host: "app.local,www.local,admin.local"
  proxy.http.port: "3000"
```

### TCP Port-Based Routing (Layer 4)

Direct TCP port forwarding:

```yaml
services:
  postgres:
    image: postgres:16
    labels:
      proxy.tcp.ports: "5432:5432"  # Format: proxy_port:container_port
    networks:
      - proxy-network
```

**Test:**
```bash
psql -h localhost -p 5432 -U postgres
```

**Multiple TCP Ports:**
```yaml
labels:
  proxy.tcp.ports: "22:2222,80:8080,443:8443"
```

### UDP Port-Based Routing

UDP port forwarding (e.g., DNS):

```yaml
services:
  dns:
    image: adguard/adguardhome
    labels:
      proxy.udp.ports: "53:53"
      proxy.tcp.ports: "53:53"  # DNS uses both TCP and UDP
    networks:
      - proxy-network
```

**Test:**
```bash
dig @localhost example.com
```

### Mixed Configuration

Combine HTTP and TCP/UDP in the same service:

```yaml
services:
  webapp:
    image: myapp:latest
    labels:
      # HTTP routing for web UI
      proxy.http.host: "app.local"
      proxy.http.port: "3000"

      # TCP for API
      proxy.tcp.ports: "9000:9000"

      # UDP for metrics
      proxy.udp.ports: "8125:8125"
    networks:
      - proxy-network
```

## Common Use Cases

### Local Development with /etc/hosts

For local testing with custom hostnames:

1. **Add entries to `/etc/hosts`:**
   ```bash
   sudo nano /etc/hosts
   ```

   Add:
   ```
   127.0.0.1 api.local web.local admin.local
   ```

2. **Configure services:**
   ```yaml
   api:
     labels:
       proxy.http.host: "api.local"

   web:
     labels:
       proxy.http.host: "web.local"
   ```

3. **Test:**
   ```bash
   curl http://api.local
   curl http://web.local
   ```

### Cloudflare Tunnel Integration

Perfect for exposing local services via Cloudflare Tunnel:

```yaml
services:
  # Cloudflare Tunnel
  cloudflared:
    image: cloudflare/cloudflared:latest
    command: tunnel --no-autoupdate run
    environment:
      TUNNEL_TOKEN: "${CLOUDFLARE_TUNNEL_TOKEN}"
    networks:
      - proxy-network

  # Proxy routes traffic from Cloudflare to your services
  proxy:
    image: ghcr.io/moontechs/proxy:latest
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - proxy-network

  # Your services
  api:
    image: my-api:latest
    labels:
      proxy.http.host: "api.yourdomain.com"
      proxy.http.port: "8080"
    networks:
      - proxy-network

  web:
    image: my-web:latest
    labels:
      proxy.http.host: "web.yourdomain.com"
      proxy.http.port: "3000"
    networks:
      - proxy-network
```

**Cloudflare Tunnel Config:**
```yaml
ingress:
  - hostname: api.yourdomain.com
    service: http://proxy:80
  - hostname: web.yourdomain.com
    service: http://proxy:80
  - service: http_status:404
```

### Database Access

Expose databases for external tools:

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

  mysql:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: secret
    labels:
      proxy.tcp.ports: "3306:3306"
    networks:
      - proxy-network

  redis:
    image: redis:7
    labels:
      proxy.tcp.ports: "6379:6379"
    networks:
      - proxy-network
```

**Connect:**
```bash
psql -h localhost -p 5432 -U postgres
mysql -h localhost -P 3306 -u root -p
redis-cli -h localhost -p 6379
```

### SSH Jump Host

Use the proxy as an SSH bastion:

```yaml
services:
  ssh-server:
    image: linuxserver/openssh-server
    environment:
      PASSWORD_ACCESS: "true"
      USER_PASSWORD: "secure-password"
    labels:
      proxy.tcp.ports: "2222:2222"
    networks:
      - proxy-network
```

**Connect:**
```bash
ssh -p 2222 user@localhost
```

## Environment Variables

### Proxy Configuration

```yaml
environment:
  # Logging
  LOG_LEVEL: "INFO"              # DEBUG, INFO (default)
  LOG_CALLER: "false"            # Show caller info in logs

  # Docker
  DOCKER_HOST: "unix:///var/run/docker.sock"

  # Nginx Paths (defaults)
  STREAM_CONFIG_PATH: "/etc/nginx/conf.d/proxy.conf"
  HTTP_CONFIG_PATH: "/etc/nginx/conf.d/http-proxy.conf"
  NGINX_RELOAD_CMD: "nginx -s reload"
```

### Debug Mode

Enable detailed logging to see generated Nginx configs:

```yaml
environment:
  LOG_LEVEL: "DEBUG"
  LOG_CALLER: "true"
```

View logs:
```bash
docker-compose logs -f proxy | grep -E "stream config|http config"
```

## Maintenance

### Update to Latest Version

```bash
# Pull latest image
docker-compose pull proxy

# Restart with new image
docker-compose up -d proxy
```

### View Generated Nginx Configs

```bash
# Stream config (TCP/UDP)
docker exec proxy-nginx cat /etc/nginx/conf.d/proxy.conf

# HTTP config
docker exec proxy-nginx cat /etc/nginx/conf.d/http-proxy.conf
```

### Test Nginx Configuration

```bash
docker exec proxy-nginx nginx -t
```

### Reload Nginx Manually

```bash
docker exec proxy-nginx nginx -s reload
```

### View Nginx Access Logs

```bash
docker exec proxy-nginx tail -f /var/log/nginx/access.log
```

### View Nginx Error Logs

```bash
docker exec proxy-nginx tail -f /var/log/nginx/error.log
```

## Troubleshooting

### Container Not Proxied

**Check labels:**
```bash
docker inspect <container> | jq '.[0].Config.Labels'
```

**Verify container is on the same network:**
```bash
docker network inspect proxy-network
```

### Port Already in Use

```bash
# Check what's using the port
sudo lsof -i :80
sudo lsof -i :443

# Or use netstat
sudo netstat -tulpn | grep :80
```

### Hostname Not Resolving

For local testing, add to `/etc/hosts`:
```bash
echo "127.0.0.1 myapp.local api.local" | sudo tee -a /etc/hosts
```

### Nginx Reload Failed

```bash
# Check Nginx syntax
docker exec proxy-nginx nginx -t

# View error logs
docker-compose logs proxy

# Restart proxy
docker-compose restart proxy
```

### Config Not Updating

1. **Enable DEBUG logging:**
   ```yaml
   environment:
     LOG_LEVEL: "DEBUG"
   ```

2. **Restart proxy:**
   ```bash
   docker-compose restart proxy
   ```

3. **Check Docker events:**
   ```bash
   docker events
   ```

### Port Conflicts

The proxy will detect and report conflicts:

```
ERROR: TCP port conflict: port 80 claimed by both webapp and api
ERROR: UDP port conflict: port 53 claimed by both dns1 and dns2
ERROR: HTTP hostname conflict: api.local claimed by both api-v1 and api-v2
```

Fix by ensuring unique ports or hostnames.

## Production Deployment

### Security Best Practices

1. **Use specific version tags:**
   ```yaml
   image: ghcr.io/moontechs/proxy:1.0.0  # Not :latest
   ```

2. **Read-only Docker socket:**
   ```yaml
   volumes:
     - /var/run/docker.sock:/var/run/docker.sock:ro
   ```

3. **Resource limits:**
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '2'
         memory: 1G
       reservations:
         cpus: '0.5'
         memory: 256M
   ```

4. **Health checks:**
   ```yaml
   healthcheck:
     test: ["CMD", "nginx", "-t"]
     interval: 30s
     timeout: 5s
     retries: 3
   ```

### High Availability

For production, consider:

1. **Multiple proxy instances** (with load balancer)
2. **Monitoring** (Prometheus + Grafana)
3. **Logging** (ELK stack or similar)
4. **Backups** of configurations
5. **Alerting** on failures

## CI/CD Integration

### Automated Deployment

```yaml
# In your CI/CD pipeline
deploy:
  script:
    - docker-compose pull proxy
    - docker-compose up -d proxy
    - docker-compose ps
```

### Rolling Updates

```bash
# Pull new image
docker-compose pull proxy

# Stop old, start new
docker-compose up -d --no-deps proxy

# Verify
docker-compose ps
```

## Additional Resources

- [README.md](README.md) - Feature overview
- [QUICKSTART.md](QUICKSTART.md) - Quick start guide
- [DEBUG_OUTPUT.md](DEBUG_OUTPUT.md) - Debug logging examples
- [docker-compose.example.yml](docker-compose.example.yml) - Full example
- [Examples Directory](examples/) - Working examples

## Getting Help

- **Check logs:** `docker-compose logs -f proxy`
- **Enable debug:** Set `LOG_LEVEL=DEBUG`
- **Test config:** `docker exec proxy-nginx nginx -t`
- **GitHub Issues:** Report bugs or request features

## License

MIT
