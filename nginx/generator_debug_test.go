package nginx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-pkgz/lgr"
	"github.com/moontechs/proxy/docker"
)

// TestDebugOutput demonstrates debug-level config printing
// Run with: go test -v -run TestDebugOutput
func TestDebugOutput(t *testing.T) {
	// Create logger with DEBUG level enabled
	log := lgr.New(lgr.Debug, lgr.Msec, lgr.LevelBraces)

	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "stream.conf")
	httpPath := filepath.Join(tmpDir, "http.conf")

	gen, err := NewGenerator(streamPath, httpPath, log)
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	t.Run("debug output for stream config (TCP/UDP)", func(t *testing.T) {
		containers := []docker.ContainerInfo{
			{
				Name: "web-server",
				ID:   "abc123def456",
				IP:   "172.17.0.2",
				Mappings: []docker.PortMapping{
					{ProxyPort: 80, ContainerPort: 8080, Protocol: docker.TCP},
					{ProxyPort: 443, ContainerPort: 8443, Protocol: docker.TCP},
					{ProxyPort: 53, ContainerPort: 53, Protocol: docker.UDP},
				},
			},
		}

		t.Log("\n=== Generating Stream Config (DEBUG level) ===")
		changed, err := gen.Generate(containers)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if !changed {
			t.Error("expected config to be generated")
		}

		// Verify file was written
		if _, err := os.Stat(streamPath); err != nil {
			t.Errorf("stream config file not created: %v", err)
		}
	})

	t.Run("debug output for HTTP config (hostname routing)", func(t *testing.T) {
		containers := []docker.ContainerInfo{
			{
				Name: "api-server",
				ID:   "xyz789abc123",
				IP:   "172.17.0.3",
				HTTPMapping: &docker.HTTPMapping{
					Hostnames:     []string{"api.example.com", "api.test.com"},
					ContainerPort: 8080,
					HTTPS:         false,
				},
			},
			{
				Name: "secure-api",
				ID:   "secure456def789",
				IP:   "172.17.0.4",
				HTTPMapping: &docker.HTTPMapping{
					Hostnames:     []string{"secure.example.com"},
					ContainerPort: 8443,
					HTTPS:         true,
				},
			},
		}

		t.Log("\n=== Generating HTTP Config (DEBUG level) ===")
		changed, err := gen.Generate(containers)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if !changed {
			t.Error("expected config to be generated")
		}

		// Verify file was written
		if _, err := os.Stat(httpPath); err != nil {
			t.Errorf("HTTP config file not created: %v", err)
		}
	})

	t.Run("debug output for mixed config (stream + HTTP)", func(t *testing.T) {
		containers := []docker.ContainerInfo{
			{
				Name: "database",
				ID:   "db123456",
				IP:   "172.17.0.5",
				Mappings: []docker.PortMapping{
					{ProxyPort: 5432, ContainerPort: 5432, Protocol: docker.TCP},
				},
			},
			{
				Name: "web-app",
				ID:   "webapp789",
				IP:   "172.17.0.6",
				HTTPMapping: &docker.HTTPMapping{
					Hostnames:     []string{"app.example.com", "www.example.com"},
					ContainerPort: 3000,
					HTTPS:         false,
				},
			},
		}

		t.Log("\n=== Generating Mixed Config (DEBUG level) ===")
		changed, err := gen.Generate(containers)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if !changed {
			t.Error("expected config to be generated")
		}
	})
}
