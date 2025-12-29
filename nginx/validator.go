package nginx

import (
	"fmt"
	"os/exec"

	"github.com/go-pkgz/lgr"
)

// Validator validates Nginx configuration files
type Validator struct {
	log *lgr.Logger
}

// NewValidator creates a new Nginx config validator
func NewValidator(log *lgr.Logger) *Validator {
	return &Validator{log: log}
}

// Validate runs 'nginx -t' to validate the configuration
func (v *Validator) Validate() error {
	v.log.Logf("DEBUG [Validator] running nginx -t")

	//nolint:noctx // validation command, context not needed
	cmd := exec.Command("nginx", "-t")
	output, err := cmd.CombinedOutput()

	if err != nil {
		v.log.Logf("ERROR [Validator] validation failed output=%q", string(output))
		return fmt.Errorf("nginx config invalid: %w\nOutput: %s", err, string(output))
	}

	v.log.Logf("DEBUG [Validator] validation output=%q", string(output))
	v.log.Logf("INFO [Validator] validation successful")

	return nil
}
