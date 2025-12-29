package cmd

import (
	"context"
	"fmt"

	"github.com/moontechs/proxy/docker"
	"github.com/moontechs/proxy/nginx"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Nginx configs from current Docker containers (one-shot)",
	Long: `Scans all running Docker containers with proxy labels and generates
Nginx stream and HTTP configurations.

Reads container labels:
  proxy.tcp.ports: "80:8080,443:8443"
  proxy.udp.ports: "53:53"
  proxy.http.host: "api.example.com"
  proxy.http.port: "80"

Generates two config files:
  - Stream config (TCP/UDP proxying)
  - HTTP config (hostname-based routing)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := GetConfig()
		log := GetLogger()

		log.Logf("INFO [Generate] starting config generation")

		// Connect to Docker
		dockerClient, err := docker.NewClient(cfg.DockerHost, log)
		if err != nil {
			return logError("docker connection failed: %w", err)
		}
		defer func() {
			if closeErr := dockerClient.Close(); closeErr != nil {
				log.Logf("WARN [Generate] failed to close docker client: %v", closeErr)
			}
		}()

		// Scan containers
		ctx := context.Background()
		containers, err := dockerClient.ScanContainers(ctx)
		if err != nil {
			return logError("container scan failed: %w", err)
		}

		log.Logf("INFO [Generate] discovered containers=%d", len(containers))

		// Generate configs
		generator, err := nginx.NewGenerator(cfg.StreamConfigPath, cfg.HTTPConfigPath, log)
		if err != nil {
			return logError("generator initialization failed: %w", err)
		}

		changed, err := generator.Generate(containers)
		if err != nil {
			return logError("config generation failed: %w", err)
		}

		if !changed {
			log.Logf("INFO [Generate] configs unchanged, no action needed")
			return nil
		}

		log.Logf("INFO [Generate] configs written successfully")
		fmt.Println("✓ Nginx configurations generated successfully")
		fmt.Printf("  Stream config: %s\n", cfg.StreamConfigPath)
		fmt.Printf("  HTTP config: %s\n", cfg.HTTPConfigPath)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
