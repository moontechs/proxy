package cmd

import (
	"fmt"
	"os"

	"github.com/go-pkgz/lgr"
	"github.com/moontechs/proxy/config"
	"github.com/spf13/cobra"
)

var (
	cfg *config.Config
	log *lgr.Logger
)

var rootCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Docker-aware Nginx stream and HTTP proxy configurator",
	Long: `Auto-generates Nginx configurations from Docker container labels.

Supports:
- Stream module (Layer 4): Raw TCP/UDP proxying by port
- HTTP module (Layer 7): Hostname-based routing for Cloudflared setups

Container labels:
  proxy.tcp.ports: "80:8080,443:8443"    # TCP proxying
  proxy.udp.ports: "53:53"               # UDP proxying
  proxy.http.host: "api.example.com"    # HTTP hostname routing
  proxy.http.port: "80"                  # Container HTTP port (default: 80)
  proxy.http.https: "true"               # Listen on 443 (default: false)`,
	Version: "2.0.0",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize configuration from flags and environment
		cfg = getConfig(cmd)

		// Initialize logger
		log = setupLogger(cmd)
		return nil
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// persistent flags available to all subcommands
	rootCmd.PersistentFlags().String("log-level", "INFO", "Log level (DEBUG, INFO, TRACE)")
	rootCmd.PersistentFlags().String("docker-host", "unix:///var/run/docker.sock", "Docker socket path")
	rootCmd.PersistentFlags().String("stream-config-path", "/etc/nginx/conf.d/proxy.conf", "Nginx stream config output path")
	rootCmd.PersistentFlags().String("http-config-path", "/etc/nginx/conf.d/http-proxy.conf", "Nginx HTTP config output path")
	rootCmd.PersistentFlags().String("reload-cmd", "nginx -s reload", "Nginx reload command")
}

// getConfig builds config from flags and environment variables
func getConfig(cmd *cobra.Command) *config.Config {
	// these flags are defined in init(), so GetString should never error
	logLevel, _ := cmd.Flags().GetString("log-level")                  //nolint:errcheck // flags are predefined
	dockerHost, _ := cmd.Flags().GetString("docker-host")              //nolint:errcheck // flags are predefined
	streamConfigPath, _ := cmd.Flags().GetString("stream-config-path") //nolint:errcheck // flags are predefined
	httpConfigPath, _ := cmd.Flags().GetString("http-config-path")     //nolint:errcheck // flags are predefined
	reloadCmd, _ := cmd.Flags().GetString("reload-cmd")                //nolint:errcheck // flags are predefined

	// override with environment variables if set
	if val := os.Getenv("LOG_LEVEL"); val != "" {
		logLevel = val
	}
	if val := os.Getenv("DOCKER_HOST"); val != "" {
		dockerHost = val
	}
	if val := os.Getenv("NGINX_STREAM_CONFIG_PATH"); val != "" {
		streamConfigPath = val
	}
	if val := os.Getenv("NGINX_HTTP_CONFIG_PATH"); val != "" {
		httpConfigPath = val
	}
	if val := os.Getenv("NGINX_RELOAD_CMD"); val != "" {
		reloadCmd = val
	}

	// get network name from environment or use default
	networkName := config.DefaultNetworkName
	if val := os.Getenv("PROXY_NETWORK"); val != "" {
		networkName = val
	}

	return &config.Config{
		LogLevel:         logLevel,
		LogCaller:        false,
		DockerHost:       dockerHost,
		NetworkName:      networkName,
		StreamConfigPath: streamConfigPath,
		HTTPConfigPath:   httpConfigPath,
		NginxReloadCmd:   reloadCmd,
	}
}

// setupLogger initializes the logger based on configuration
func setupLogger(cmd *cobra.Command) *lgr.Logger {
	logLevel, _ := cmd.Flags().GetString("log-level") //nolint:errcheck // flag is predefined

	// override with environment if set
	if val := os.Getenv("LOG_LEVEL"); val != "" {
		logLevel = val
	}

	opts := []lgr.Option{
		lgr.Msec,        // add millisecond precision
		lgr.LevelBraces, // use [INFO] format
	}

	// set caller info if requested
	if os.Getenv("LOG_CALLER") == "true" {
		opts = append(opts, lgr.CallerFile, lgr.CallerFunc)
	}

	// set log level - lgr only supports Debug and Trace filtering
	switch logLevel {
	case "DEBUG":
		opts = append(opts, lgr.Debug)
	case "TRACE":
		opts = append(opts, lgr.Trace)
	}
	// INFO, WARN, ERROR are default - no option needed

	return lgr.New(opts...)
}

// GetConfig returns the current configuration (used by subcommands)
func GetConfig() *config.Config {
	return cfg
}

// GetLogger returns the current logger (used by subcommands)
func GetLogger() *lgr.Logger {
	return log
}

// logError logs an error and returns it for command return
func logError(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	if log != nil {
		log.Logf("ERROR %v", err)
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	}
	return err
}
