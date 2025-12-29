package cmd

import (
	"fmt"

	"github.com/moontechs/proxy/nginx"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate Nginx configuration syntax",
	Long: `Validates Nginx configuration files using 'nginx -t'.

If no config file is specified, validates the main nginx.conf.
Returns exit code 0 if valid, non-zero if invalid.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := GetLogger()

		// Determine which config to validate
		configPath := "/etc/nginx/nginx.conf"
		if len(args) > 0 {
			configPath = args[0]
		}

		log.Logf("INFO [Validate] validating config path=%s", configPath)

		// Create validator
		validator := nginx.NewValidator(log)

		// Validate
		if err := validator.Validate(); err != nil {
			fmt.Printf("✗ Nginx configuration is invalid\n")
			return logError("validation failed: %w", err)
		}

		log.Logf("INFO [Validate] validation successful path=%s", configPath)
		fmt.Printf("✓ Nginx configuration is valid\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
