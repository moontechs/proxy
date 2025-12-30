package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/moontechs/proxy/docker"
	"github.com/moontechs/proxy/nginx"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch Docker events and regenerate configs on container changes",
	Long: `Watches Docker container events (start/stop/die) and automatically
regenerates Nginx configurations when containers change.

Features:
- 2-second debouncing to batch rapid changes
- Automatic Nginx validation before reload
- Graceful shutdown on SIGINT/SIGTERM
- Keeps old config if new one fails validation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := GetConfig()
		log := GetLogger()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log.Logf("INFO [Watch] starting watch mode")

		// Setup components
		dockerClient, err := docker.NewClient(cfg.DockerHost, log)
		if err != nil {
			return logError("docker connection failed: %w", err)
		}
		defer func() {
			if closeErr := dockerClient.Close(); closeErr != nil {
				log.Logf("WARN [Watch] failed to close docker client: %v", closeErr)
			}
		}()

		// Ensure proxy network exists
		if err := dockerClient.EnsureNetwork(ctx, cfg.NetworkName); err != nil {
			return logError("network setup failed: %w", err)
		}

		generator, err := nginx.NewGenerator(cfg.StreamConfigPath, cfg.HTTPConfigPath, log)
		if err != nil {
			return logError("generator initialization failed: %w", err)
		}

		validator := nginx.NewValidator(log)

		reloader, err := nginx.NewReloader(cfg.NginxReloadCmd, log)
		if err != nil {
			return logError("reloader initialization failed: %w", err)
		}

		// Initial generation
		log.Logf("INFO [Watch] performing initial config generation")
		if err := generateAndReload(ctx, dockerClient, generator, validator, reloader, log); err != nil {
			return logError("initial generation failed: %w", err)
		}

		// Watch events
		eventCh, errCh := dockerClient.WatchEvents(ctx)

		// Setup signal handling
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		log.Logf("INFO [Watch] ready and watching for container events")
		fmt.Println("✓ Watching Docker events (Ctrl+C to stop)")

		// Event loop with debouncing
		var pendingReload bool
		debounceTimer := time.NewTimer(0)
		<-debounceTimer.C // Drain initial timer

		for {
			select {
			case event := <-eventCh:
				log.Logf("INFO [Watch] event received type=%s container=%s", event.Type, event.Name)

				// Mark for reload and start/reset debounce timer
				pendingReload = true
				debounceTimer.Reset(2 * time.Second)

			case <-debounceTimer.C:
				if pendingReload {
					log.Logf("INFO [Watch] triggering config regeneration")

					if err := generateAndReload(ctx, dockerClient, generator, validator, reloader, log); err != nil {
						log.Logf("ERROR [Watch] regeneration failed error=%q", err)
						// Don't exit, continue watching
					}

					pendingReload = false
				}

			case err := <-errCh:
				log.Logf("ERROR [Watch] event stream error=%q", err)
				return logError("event stream error: %w", err)

			case sig := <-sigCh:
				log.Logf("INFO [Watch] shutdown signal=%s", sig)
				fmt.Println("\n✓ Shutting down gracefully...")
				cancel()
				return nil
			}
		}
	},
}

// generateAndReload performs the full workflow: scan → generate → validate → reload
func generateAndReload(ctx context.Context, dockerClient *docker.Client, gen *nginx.Generator,
	val *nginx.Validator, reload *nginx.Reloader, log *lgr.Logger) error {
	// scan containers
	containers, err := dockerClient.ScanContainers(ctx)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	log.Logf("INFO [Watch] scanned containers=%d", len(containers))

	// generate configs
	changed, err := gen.Generate(containers)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	if !changed {
		log.Logf("INFO [Watch] configs unchanged, skipping reload")
		return nil
	}

	// validate
	if err := val.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// reload Nginx
	if err := reload.Reload(); err != nil {
		return fmt.Errorf("reload failed: %w", err)
	}

	log.Logf("INFO [Watch] configs reloaded successfully")
	return nil
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
