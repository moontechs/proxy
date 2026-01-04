#!/bin/sh
set -e

echo "[Entrypoint] Starting Nginx proxy configurator"

# Set default worker_connections if not provided
export NGINX_WORKER_CONNECTIONS=${NGINX_WORKER_CONNECTIONS:-1000}

# Process nginx.conf template with environment variables
echo "[Entrypoint] Configuring Nginx with NGINX_WORKER_CONNECTIONS=$NGINX_WORKER_CONNECTIONS"
envsubst '${NGINX_WORKER_CONNECTIONS}' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

# Generate initial configuration
echo "[Entrypoint] Generating initial Nginx configurations"
/usr/local/bin/proxy generate || {
    echo "[Entrypoint] WARNING: Initial config generation failed, continuing anyway"
}

# Validate Nginx configuration
echo "[Entrypoint] Validating Nginx configuration"
nginx -t

# Delegate to nginx's original entrypoint
echo "[Entrypoint] Delegating to nginx entrypoint"
exec /nginx-entrypoint.sh "$@"
