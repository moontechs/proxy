package config

import (
	"os"
	"strings"
)

// Config holds all proxy configuration
type Config struct {
	// Docker
	DockerHost string

	// Nginx configuration paths
	StreamConfigPath string // Path to stream module config (default: /etc/nginx/conf.d/proxy.conf)
	HTTPConfigPath   string // Path to HTTP module config (default: /etc/nginx/conf.d/http-proxy.conf)
	NginxReloadCmd   string // Nginx reload command (default: nginx -s reload)

	// Logging
	LogLevel  string
	LogCaller bool
}

// Load parses environment variables and returns Config
// This is kept for backwards compatibility but config is now mostly handled via CLI flags
func Load() (*Config, error) {
	cfg := &Config{}

	// Docker configuration
	cfg.DockerHost = getEnvOrDefault("DOCKER_HOST", "unix:///var/run/docker.sock")

	// Nginx configuration paths
	cfg.StreamConfigPath = getEnvOrDefault("NGINX_STREAM_CONFIG_PATH", "/etc/nginx/conf.d/proxy.conf")
	cfg.HTTPConfigPath = getEnvOrDefault("NGINX_HTTP_CONFIG_PATH", "/etc/nginx/conf.d/http-proxy.conf")
	cfg.NginxReloadCmd = getEnvOrDefault("NGINX_RELOAD_CMD", "nginx -s reload")

	// Logging configuration
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
