# Production Example with Cloudflare Tunnel

This example demonstrates a **production-ready setup** with:
- Cloudflare Tunnel for secure external access
- Multi-network architecture for database isolation
- Real application (taxhacker) with PostgreSQL database
- Proper logging and restart policies

## Architecture

The setup uses three Docker networks for security isolation:

1. **proxy-local-network** (Internal: proxy ↔ cloudflared)
   - proxy
   - cloudflared

2. **proxy-network** (External: proxy ↔ services)
   - proxy
   - taxhacker

3. **taxhacker-local-network** (Internal: app ↔ database)
   - taxhacker
   - postgres (ISOLATED from proxy)

**Security Model**:
- `postgres` is ONLY on `taxhacker-local-network` - no external access
- `taxhacker` is on BOTH networks - receives traffic from proxy, connects to database
- `cloudflared` is ONLY on `proxy-local-network` - isolated from services

## Multi-Network Pattern Explained

### Why Multiple Networks?

**Security Isolation**: Each component only has access to what it needs.

```yaml
# Proxy setup (proxy/docker-compose.yml)
services:
  proxy:
    networks:
      - proxy-network         # External - for service communication
      - proxy-local-network   # Internal - for cloudflared only

  cloudflared:
    networks:
      - proxy-local-network   # Can reach proxy, but NOT services
```

```yaml
# Service setup (taxhacker/docker-compose.yml)
services:
  taxhacker:
    networks:
      - proxy-network              # External - receives traffic from proxy
      - taxhacker-local-network    # Internal - connects to database
    labels:
      proxy.http.host: "taxhacker.example.com"
      proxy.http.port: "7331"

  postgres:
    networks:
      - taxhacker-local-network    # ONLY on local network - fully isolated
```

**Result**:
- ✅ Proxy can route to taxhacker (both on `proxy-network`)
- ✅ Taxhacker can connect to postgres (both on `taxhacker-local-network`)
- ❌ Proxy CANNOT access postgres (on different networks)
- ❌ Cloudflared CANNOT access services (on different networks)

## Setup Instructions

### 1. Create External Network

```bash
docker network create --driver bridge proxy-network
```

### 2. Configure Cloudflare Tunnel

Get your tunnel token from [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/):

1. Create a tunnel
2. Configure public hostname:
   ```
   Public hostname: taxhacker.example.com
   Service: http://proxy:80
   ```
3. Copy the tunnel token

### 3. Update Proxy Configuration

Edit `proxy/docker-compose.yml`:

```yaml
cloudflared:
  environment:
    - TUNNEL_TOKEN=your-actual-token-here  # Replace this!
```

### 4. Update Service Configuration

Edit `taxhacker/docker-compose.yml`:

```yaml
labels:
  proxy.http.host: "taxhacker.yourdomain.com"  # Your actual domain
```

### 5. Start Proxy with Cloudflare

```bash
cd proxy
docker compose up -d
```

### 6. Start Application

```bash
cd ../taxhacker
docker compose up -d
```

### 7. Verify

Check all containers are running:
```bash
docker ps
# Should show: proxy, cloudflared, taxhacker, postgres
```

Check networks:
```bash
docker network inspect proxy-network
# Should show: proxy, taxhacker (NOT postgres, NOT cloudflared)

docker network inspect taxhacker-local-network
# Should show: taxhacker, postgres (NOT proxy)
```

Access your application:
```bash
curl https://taxhacker.yourdomain.com
# Should work if Cloudflare DNS is configured
```

## Configuration Options

### Environment Variables (Proxy)

```yaml
# proxy/docker-compose.yml
environment:
  LOG_LEVEL: "DEBUG"          # DEBUG or INFO
  LOG_CALLER: "false"         # Show source code location
```

### Environment Variables (Application)

```yaml
# taxhacker/docker-compose.yml
environment:
  - NODE_ENV=production
  - SELF_HOSTED_MODE=true
  - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/taxhacker
```

### Volumes

The example includes a volume mount for persistent data:

```yaml
volumes:
  - /mnt/drive/data/taxhacker/storage:/app/data
```

**Update this path** to match your server's storage location.

### Logging

Both compose files include production logging configuration:

```yaml
logging:
  driver: "local"
  options:
    max-size: "100M"    # Maximum log file size
    max-file: "3"       # Keep 3 rotated files
```

This prevents logs from filling up disk space.

## Cloudflare Tunnel Configuration

In your Cloudflare dashboard, configure the tunnel ingress:

```yaml
# Example tunnel config.yml
tunnel: <your-tunnel-id>
credentials-file: /etc/cloudflared/credentials.json

ingress:
  - hostname: taxhacker.yourdomain.com
    service: http://proxy:80
  - hostname: anotherapp.yourdomain.com
    service: http://proxy:80
  - service: http_status:404
```

**Multiple services through one tunnel**: All hostnames point to `http://proxy:80`, and the proxy uses the `Host` header to route to the correct container.

## Troubleshooting

### Cloudflare 502 Bad Gateway

**Check tunnel is connected:**
```bash
docker logs cloudflared
# Should show: Connection established
```

**Check proxy can reach service:**
```bash
docker exec proxy ping taxhacker
# Should work if both on proxy-network
```

### Database Connection Failed

**Check postgres is running:**
```bash
docker logs postgres
# Should show: database system is ready to accept connections
```

**Check taxhacker can reach postgres:**
```bash
docker exec taxhacker ping postgres
# Should work if both on taxhacker-local-network
```

**Verify DATABASE_URL is correct:**
```bash
docker exec taxhacker env | grep DATABASE_URL
```

### Service Not Accessible

**Verify labels are correct:**
```bash
docker inspect taxhacker | jq '.[0].Config.Labels'
# Should show: proxy.http.host and proxy.http.port
```

**Check proxy detected the service:**
```bash
docker logs proxy | grep taxhacker
# Should show: [INFO] registered_container name=taxhacker
```

## Adding More Services

To add another service (e.g., an admin panel):

```yaml
# admin/docker-compose.yml
services:
  admin:
    image: your-admin-app
    labels:
      proxy.http.host: "admin.yourdomain.com"
      proxy.http.port: "3000"
    networks:
      - proxy-network        # Connect to proxy
      - admin-local          # For admin's database

  admin-db:
    image: postgres:17-alpine
    networks:
      - admin-local          # Isolated from everything else

networks:
  proxy-network:
    name: proxy-network
    external: true
  admin-local:
    name: admin-local-network
    driver: bridge
```

**Update Cloudflare tunnel** to add the new hostname pointing to `http://proxy:80`.

## Production Checklist

Before deploying to production:

- [ ] Replace `TUNNEL_TOKEN` with real token
- [ ] Update `proxy.http.host` to your actual domain
- [ ] Configure Cloudflare DNS for your domain
- [ ] Update volume paths for your server
- [ ] Review and adjust logging settings
- [ ] Set strong database passwords (not the example ones!)
- [ ] Configure firewall rules (only allow Cloudflare IPs)
- [ ] Set up monitoring and alerting
- [ ] Configure backup strategy for volumes
- [ ] Review resource limits for containers

## Benefits of This Architecture

✅ **Security**: Database never exposed to proxy or internet
✅ **Isolation**: Each service has its own local network for internal components
✅ **Scalability**: Add services without changing existing ones
✅ **Zero Trust**: Cloudflare Tunnel eliminates need to expose ports
✅ **Observability**: Centralized logging with rotation
✅ **Resilience**: Automatic restarts with `unless-stopped`
✅ **Simplicity**: No manual nginx configuration needed

## Related Documentation

- [Main README](../../README.md) - Full proxy documentation
- [Network Architecture Best Practices](../../README.md#network-architecture-best-practices)
- [Cloudflare Tunnel Setup](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/)