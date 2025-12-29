#!/bin/sh
set -e

echo "[Entrypoint] Starting Nginx proxy configurator"

# Start Nginx in background
echo "[Entrypoint] Starting Nginx"
nginx

# Generate initial configuration
echo "[Entrypoint] Generating initial Nginx configurations"
/usr/local/bin/proxy-nginx generate || {
    echo "[Entrypoint] WARNING: Initial config generation failed, continuing anyway"
}

# Reload Nginx with initial config (if generation succeeded)
if [ -f /etc/nginx/conf.d/proxy.conf ] || [ -f /etc/nginx/conf.d/http-proxy.conf ]; then
    echo "[Entrypoint] Reloading Nginx with generated configs"
    nginx -s reload || echo "[Entrypoint] WARNING: Nginx reload failed"
fi

# Execute proxy-nginx command (default: watch)
echo "[Entrypoint] Starting proxy-nginx $@"
exec /usr/local/bin/proxy-nginx "$@"
