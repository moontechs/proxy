package nginx

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/go-pkgz/lgr"
)

// Reloader handles Nginx reload operations
type Reloader struct {
	reloadCmd  string
	log        *lgr.Logger
	lastReload time.Time
}

// NewReloader creates a new Nginx reloader
func NewReloader(reloadCmd string, log *lgr.Logger) (*Reloader, error) {
	return &Reloader{
		reloadCmd: reloadCmd,
		log:       log,
	}, nil
}

// Reload reloads Nginx configuration
// Implements reload throttling (minimum 1 second between reloads)
func (r *Reloader) Reload() error {
	// Prevent reload storms
	if time.Since(r.lastReload) < 1*time.Second {
		r.log.Logf("WARN [Reloader] throttling reload, too_soon_after_last")
		time.Sleep(1 * time.Second)
	}

	r.log.Logf("INFO [Reloader] executing reload_cmd=%s", r.reloadCmd)

	cmd := exec.Command("sh", "-c", r.reloadCmd)
	output, err := cmd.CombinedOutput()

	if err != nil {
		r.log.Logf("ERROR [Reloader] reload failed output=%q error=%q", string(output), err)
		return fmt.Errorf("nginx reload failed: %w\nOutput: %s", err, string(output))
	}

	r.lastReload = time.Now()
	r.log.Logf("INFO [Reloader] reload successful output=%q", string(output))

	return nil
}
