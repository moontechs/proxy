# Separate Compose Files Example

This example demonstrates running the proxy and services in **separate docker-compose projects**, which is the recommended pattern for production deployments.

## Architecture

The example contains two separate docker-compose projects:
- `proxy/docker-compose.yml` - Proxy service only
- `http-services/docker-compose.yml` - Application services only

**Key Benefit**: Services can be managed independently - restart apps without touching the proxy, or update proxy without affecting services.

## How It Works

Both compose files connect to the same external `proxy-network`:

```yaml
# In both files
networks:
  proxy-network:
    name: proxy-network
    external: true  # Network must exist before starting
```

The proxy watches for containers on this network and auto-configures routing.

## Usage

### 1. Create the External Network

```bash
docker network create --driver bridge proxy-network
```

This network is shared by all compose projects.

### 2. Start the Proxy

```bash
cd proxy
docker compose up -d
```

The proxy starts and watches for containers joining `proxy-network`.

### 3. Start Your Services

```bash
cd ../http-services
docker compose up -d
```

Services automatically register with the proxy when they start.

### 4. Add Hostnames to /etc/hosts

```bash
sudo bash -c 'cat >> /etc/hosts <<EOF
127.0.0.1 svc1.local
127.0.0.1 svc2.local
EOF'
```

### 5. Test

```bash
curl http://svc1.local
# Output: hello world from svc1

curl http://svc2.local
# Output: hello world from svc2
```

## Verify Setup

**Check the proxy is running:**
```bash
docker ps | grep proxy
```

**Check services are on the network:**
```bash
docker network inspect proxy-network
# Should show proxy, svc1, and svc2
```

**Check proxy logs:**
```bash
docker logs proxy
# Should show: [INFO] registered_container name=svc1 http_hosts=1
# Should show: [INFO] registered_container name=svc2 http_hosts=1
```

## Managing Services

**Update services without touching proxy:**
```bash
cd http-services
docker compose pull
docker compose up -d
# Proxy automatically detects changes
```

**Update proxy without touching services:**
```bash
cd proxy
docker compose pull
docker compose up -d
# All service routing continues working
```

**Stop services but leave proxy running:**
```bash
cd http-services
docker compose down
# Services unregister, proxy keeps running
```

## When to Use This Pattern

✅ **Use separate compose files when:**
- You have multiple independent applications
- Different teams manage proxy vs. services
- Services need to be deployed/updated independently
- Running in production or staging environments

❌ **Use single compose file when:**
- Quick local testing
- Demo or development setup
- All services deployed together
- Simple single-application deployment

## Real-World Example

In production, you might have:

```
infrastructure/
├── proxy/
│   └── docker-compose.yml      # Shared proxy + Cloudflare Tunnel
services/
├── api/
│   └── docker-compose.yml      # API service
├── web/
│   └── docker-compose.yml      # Frontend app
└── admin/
    └── docker-compose.yml      # Admin panel
```

All connect to the same `proxy-network`, managed independently.

## Troubleshooting

### Service Not Accessible

**Verify container is on proxy-network:**
```bash
docker ps --format "{{.Names}}\t{{.Networks}}" | grep proxy-network
```

If your service is missing, check your docker-compose.yml has:
```yaml
networks:
  - proxy-network
```

### Network Not Found Error

```
ERROR: Network proxy-network declared as external, but could not be found
```

**Solution:**
```bash
docker network create --driver bridge proxy-network
```

### Services Can't See Each Other

If services need to communicate directly (not through proxy), add them to a shared local network:

```yaml
# In both service compose files
networks:
  proxy-network:
    external: true
  shared-backend:
    name: shared-backend
    driver: bridge
```