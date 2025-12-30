package config

import (
	"os"
	"strings"
)

const (
	// DefaultNetworkName is the default Docker network name for proxy communication
	DefaultNetworkName = "proxy-network"
)

// Config holds all proxy configuration
type Config struct {
	// docker
	DockerHost  string
	NetworkName string // docker network name for proxy communication (default: proxy-network)

	// nginx configuration paths
	StreamConfigPath string // path to stream module config (default: /etc/nginx/conf.d/proxy.conf)
	HTTPConfigPath   string // path to HTTP module config (default: /etc/nginx/conf.d/http-proxy.conf)
	NginxReloadCmd   string // nginx reload command (default: nginx -s reload)

	// logging
	LogLevel  string
	LogCaller bool
}

// Load parses environment variables and returns Config
// This is kept for backwards compatibility but config is now mostly handled via CLI flags
func Load() (*Config, error) {
	cfg := &Config{}

	// docker configuration
	cfg.DockerHost = getEnvOrDefault("DOCKER_HOST", "unix:///var/run/docker.sock")
	cfg.NetworkName = getEnvOrDefault("PROXY_NETWORK", DefaultNetworkName)

	// nginx configuration paths
	cfg.StreamConfigPath = getEnvOrDefault("NGINX_STREAM_CONFIG_PATH", "/etc/nginx/conf.d/proxy.conf")
	cfg.HTTPConfigPath = getEnvOrDefault("NGINX_HTTP_CONFIG_PATH", "/etc/nginx/conf.d/http-proxy.conf")
	cfg.NginxReloadCmd = getEnvOrDefault("NGINX_RELOAD_CMD", "nginx -s reload")

	// logging configuration
	cfg.LogLevel = strings.ToUpper(getEnvOrDefault("LOG_LEVEL", "INFO"))
	cfg.LogCaller = getEnvOrDefault("LOG_CALLER", "false") == "true"

	return cfg, nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
