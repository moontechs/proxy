# HTTP Services Example

This example demonstrates HTTP hostname-based routing using the proxy-nginx system. Multiple services are accessible via different hostnames on the same port (80).

## Architecture

```
                    ┌─────────────────┐
curl → svc1.local →│   Nginx Proxy   │→ svc1:8080
                    │   (Port 80)     │
curl → svc2.local →│                 │→ svc2:8080
                    └─────────────────┘
```

## Label Schema

For HTTP hostname-based routing, use these Docker labels:

```yaml
labels:
  proxy.http.host: "hostname.local"    # Required: hostname(s) for routing
  proxy.http.port: "8080"              # Optional: container port (default: 80)
  proxy.http.https: "false"            # Optional: use HTTPS listener (default: false)
```

### Multiple Hostnames

You can specify multiple hostnames for the same service:

```yaml
labels:
  proxy.http.host: "app.local,www.local,api.local"
  proxy.http.port: "3000"
```

## Usage

### 1. Add Hostnames to /etc/hosts

```bash
sudo bash -c 'cat >> /etc/hosts <<EOF
127.0.0.1 svc1.local
127.0.0.1 svc2.local
EOF'
```

### 2. Start Services

```bash
cd examples/http-services
docker compose up --build
```

### 3. Test HTTP Routing

```bash
# Using curl
curl http://svc1.local
# Output: hello world from svc1

curl http://svc2.local
# Output: hello world from svc2
```

### 4. Verify Generated Configs

With `LOG_LEVEL=DEBUG`, you'll see the generated nginx configs:

```nginx
# HTTP config (/etc/nginx/conf.d/http-proxy.conf)
upstream http_svc1_local {
    server 172.18.0.2:8080;
}

server {
    listen 80;
    server_name svc1.local;

    location / {
        proxy_pass http://http_svc1_local;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        # ... additional headers
    }
}
```

### 5. Watch Container Events

The proxy automatically detects container start/stop events:

```bash
# In another terminal, restart a service
docker compose restart svc1

# Proxy will automatically regenerate configs and reload nginx
```

## Troubleshooting

### Request Not Reaching Service

**Check hostname resolution:**
```bash
nslookup svc1.local
# Should return 127.0.0.1
```

**Verify containers are on same network:**
```bash
docker network inspect http-services_proxy-network
# Should show proxy, svc1, and svc2
```

**Check generated nginx config:**
```bash
docker exec proxy-http-services cat /etc/nginx/conf.d/http-proxy.conf
```

### Hostname Conflict Error

If two containers claim the same hostname:
```
ERROR: HTTP hostname conflict: svc1.local claimed by both svc1 and svc1-backup
```

Solution: Use unique hostnames or remove one container.

### Container Not Detected

**Verify labels are correct:**
```bash
docker inspect svc1 | jq '.[0].Config.Labels'
```

**Check proxy logs:**
```bash
docker logs proxy-http-services
```

Expected output:
```
[INFO] [Docker] registered_container name=svc1 http_hosts=1
[INFO] [Generator] generation complete stream_changed=false http_changed=true
[INFO] [Reloader] reload successful
```

## Advanced Examples

### HTTPS Service

```yaml
services:
  secure-api:
    image: myapp:latest
    labels:
      proxy.http.host: "secure.local"
      proxy.http.port: "8443"
      proxy.http.https: "true"  # Listen on port 443
```

### Mixed Routing (TCP + HTTP)

The same container can have both stream (TCP/UDP) and HTTP routing:

```yaml
services:
  ssh-web:
    image: myapp:latest
    labels:
      proxy.tcp.ports: "22:22"           # Stream module (raw TCP)
      proxy.http.host: "admin.local"     # HTTP module (hostname routing)
      proxy.http.port: "8080"
```

This creates:
- TCP proxying on port 22 (stream module)
- HTTP hostname routing on port 80 (HTTP module)

No conflict because they use different nginx modules.