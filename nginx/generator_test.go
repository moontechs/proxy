package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-pkgz/lgr"
	"github.com/moontechs/proxy/docker"
)

func TestNewGenerator(t *testing.T) {
	log := lgr.New()

	t.Run("creates generator with valid paths", func(t *testing.T) {
		gen, err := NewGenerator("/tmp/stream.conf", "/tmp/http.conf", log)
		if err != nil {
			t.Fatalf("NewGenerator() error = %v", err)
		}
		if gen == nil {
			t.Fatal("expected non-nil generator")
		}
	})
}

func TestValidateConflicts(t *testing.T) {
	log := lgr.New()
	gen, _ := NewGenerator("/tmp/stream.conf", "/tmp/http.conf", log)

	tests := []struct {
		name        string
		containers  []docker.ContainerInfo
		wantErr     bool
		errContains string
	}{
		{
			name: "no conflicts",
			containers: []docker.ContainerInfo{
				{
					Name: "web",
					IP:   "172.17.0.2",
					Mappings: []docker.PortMapping{
						{ProxyPort: 80, ContainerPort: 8080, Protocol: docker.TCP},
					},
				},
				{
					Name: "dns",
					IP:   "172.17.0.3",
					Mappings: []docker.PortMapping{
						{ProxyPort: 53, ContainerPort: 53, Protocol: docker.UDP},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "TCP port conflict",
			containers: []docker.ContainerInfo{
				{
					Name: "web1",
					IP:   "172.17.0.2",
					Mappings: []docker.PortMapping{
						{ProxyPort: 80, ContainerPort: 8080, Protocol: docker.TCP},
					},
				},
				{
					Name: "web2",
					IP:   "172.17.0.3",
					Mappings: []docker.PortMapping{
						{ProxyPort: 80, ContainerPort: 3000, Protocol: docker.TCP},
					},
				},
			},
			wantErr:     true,
			errContains: "TCP port conflict: port 80",
		},
		{
			name: "UDP port conflict",
			containers: []docker.ContainerInfo{
				{
					Name: "dns1",
					IP:   "172.17.0.2",
					Mappings: []docker.PortMapping{
						{ProxyPort: 53, ContainerPort: 53, Protocol: docker.UDP},
					},
				},
				{
					Name: "dns2",
					IP:   "172.17.0.3",
					Mappings: []docker.PortMapping{
						{ProxyPort: 53, ContainerPort: 5353, Protocol: docker.UDP},
					},
				},
			},
			wantErr:     true,
			errContains: "UDP port conflict: port 53",
		},
		{
			name: "HTTP hostname conflict",
			containers: []docker.ContainerInfo{
				{
					Name: "api1",
					IP:   "172.17.0.2",
					HTTPMapping: &docker.HTTPMapping{
						Hostnames:     []string{"api.example.com"},
						ContainerPort: 8080,
						HTTPS:         false,
					},
				},
				{
					Name: "api2",
					IP:   "172.17.0.3",
					HTTPMapping: &docker.HTTPMapping{
						Hostnames:     []string{"api.example.com"},
						ContainerPort: 3000,
						HTTPS:         false,
					},
				},
			},
			wantErr:     true,
			errContains: "HTTP hostname conflict: api.example.com",
		},
		{
			name: "same port TCP and UDP - no conflict",
			containers: []docker.ContainerInfo{
				{
					Name: "dns-tcp",
					IP:   "172.17.0.2",
					Mappings: []docker.PortMapping{
						{ProxyPort: 53, ContainerPort: 53, Protocol: docker.TCP},
					},
				},
				{
					Name: "dns-udp",
					IP:   "172.17.0.3",
					Mappings: []docker.PortMapping{
						{ProxyPort: 53, ContainerPort: 53, Protocol: docker.UDP},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "HTTP and TCP same port - no conflict (different modules)",
			containers: []docker.ContainerInfo{
				{
					Name: "web-tcp",
					IP:   "172.17.0.2",
					Mappings: []docker.PortMapping{
						{ProxyPort: 80, ContainerPort: 8080, Protocol: docker.TCP},
					},
				},
				{
					Name: "web-http",
					IP:   "172.17.0.3",
					HTTPMapping: &docker.HTTPMapping{
						Hostnames:     []string{"web.example.com"},
						ContainerPort: 3000,
						HTTPS:         false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple hostnames same container - no conflict",
			containers: []docker.ContainerInfo{
				{
					Name: "api",
					IP:   "172.17.0.2",
					HTTPMapping: &docker.HTTPMapping{
						Hostnames:     []string{"api.example.com", "api.test.com"},
						ContainerPort: 8080,
						HTTPS:         false,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streamData, httpData := gen.buildTemplateData(tt.containers)
			err := gen.validateConflicts(streamData, httpData)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateConflicts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateConflicts() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestHostnameToUpstream(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{
			name:     "simple domain",
			hostname: "api.example.com",
			want:     "http_api_example_com",
		},
		{
			name:     "subdomain with dashes",
			hostname: "my-api.test-domain.com",
			want:     "http_my_api_test_domain_com",
		},
		{
			name:     "localhost",
			hostname: "localhost",
			want:     "http_localhost",
		},
		{
			name:     "IP address",
			hostname: "192.168.1.1",
			want:     "http_192_168_1_1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hostnameToUpstream(tt.hostname)
			if got != tt.want {
				t.Errorf("hostnameToUpstream(%q) = %q, want %q", tt.hostname, got, tt.want)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	// Create temp directory for test configs
	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "stream.conf")
	httpPath := filepath.Join(tmpDir, "http.conf")

	log := lgr.New()
	gen, err := NewGenerator(streamPath, httpPath, log)
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	t.Run("generates stream config for TCP/UDP", func(t *testing.T) {
		containers := []docker.ContainerInfo{
			{
				Name: "web",
				ID:   "abc123",
				IP:   "172.17.0.2",
				Mappings: []docker.PortMapping{
					{ProxyPort: 80, ContainerPort: 8080, Protocol: docker.TCP},
					{ProxyPort: 53, ContainerPort: 53, Protocol: docker.UDP},
				},
			},
		}

		changed, err := gen.Generate(containers)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if !changed {
			t.Error("expected config to be generated (changed=true)")
		}

		// Verify stream config exists and contains expected content
		streamContent, err := os.ReadFile(streamPath)
		if err != nil {
			t.Fatalf("failed to read stream config: %v", err)
		}

		content := string(streamContent)
		if !strings.Contains(content, "upstream tcp_80") {
			t.Error("stream config should contain TCP upstream")
		}
		if !strings.Contains(content, "upstream udp_53") {
			t.Error("stream config should contain UDP upstream")
		}
		if !strings.Contains(content, "listen 80;") {
			t.Error("stream config should contain TCP listen directive")
		}
		if !strings.Contains(content, "listen 53 udp;") {
			t.Error("stream config should contain UDP listen directive")
		}
	})

	t.Run("generates HTTP config for hostname routing", func(t *testing.T) {
		containers := []docker.ContainerInfo{
			{
				Name: "api",
				ID:   "def456",
				IP:   "172.17.0.3",
				HTTPMapping: &docker.HTTPMapping{
					Hostnames:     []string{"api.example.com", "api.test.com"},
					ContainerPort: 8080,
					HTTPS:         false,
				},
			},
		}

		changed, err := gen.Generate(containers)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if !changed {
			t.Error("expected config to be generated (changed=true)")
		}

		// Verify HTTP config exists and contains expected content
		httpContent, err := os.ReadFile(httpPath)
		if err != nil {
			t.Fatalf("failed to read HTTP config: %v", err)
		}

		content := string(httpContent)
		if !strings.Contains(content, "upstream http_api_example_com") {
			t.Error("HTTP config should contain upstream for api.example.com")
		}
		if !strings.Contains(content, "server_name api.example.com;") {
			t.Error("HTTP config should contain server_name directive")
		}
		if !strings.Contains(content, "listen 80;") {
			t.Error("HTTP config should contain listen 80 for non-HTTPS")
		}
		if !strings.Contains(content, "proxy_pass http://http_api_example_com;") {
			t.Error("HTTP config should contain proxy_pass directive")
		}
	})

	t.Run("generates HTTPS listener when HTTPS=true", func(t *testing.T) {
		containers := []docker.ContainerInfo{
			{
				Name: "secure-api",
				ID:   "ghi789",
				IP:   "172.17.0.4",
				HTTPMapping: &docker.HTTPMapping{
					Hostnames:     []string{"secure.example.com"},
					ContainerPort: 8443,
					HTTPS:         true,
				},
			},
		}

		changed, err := gen.Generate(containers)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if !changed {
			t.Error("expected config to be generated (changed=true)")
		}

		httpContent, err := os.ReadFile(httpPath)
		if err != nil {
			t.Fatalf("failed to read HTTP config: %v", err)
		}

		content := string(httpContent)
		if !strings.Contains(content, "listen 443 ssl;") {
			t.Error("HTTP config should contain listen 443 ssl for HTTPS")
		}
	})

	t.Run("detects no change when regenerating same config", func(t *testing.T) {
		containers := []docker.ContainerInfo{
			{
				Name: "web",
				IP:   "172.17.0.2",
				Mappings: []docker.PortMapping{
					{ProxyPort: 80, ContainerPort: 8080, Protocol: docker.TCP},
				},
			},
		}

		// First generation
		changed1, err := gen.Generate(containers)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if !changed1 {
			t.Error("first generation should detect change")
		}

		// Second generation with same data
		changed2, err := gen.Generate(containers)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if changed2 {
			t.Error("second generation should not detect change (same config)")
		}
	})

	t.Run("returns error on port conflicts", func(t *testing.T) {
		containers := []docker.ContainerInfo{
			{
				Name: "web1",
				IP:   "172.17.0.2",
				Mappings: []docker.PortMapping{
					{ProxyPort: 80, ContainerPort: 8080, Protocol: docker.TCP},
				},
			},
			{
				Name: "web2",
				IP:   "172.17.0.3",
				Mappings: []docker.PortMapping{
					{ProxyPort: 80, ContainerPort: 3000, Protocol: docker.TCP},
				},
			},
		}

		_, err := gen.Generate(containers)
		if err == nil {
			t.Error("expected error on TCP port conflict")
		}
		if !strings.Contains(err.Error(), "TCP port conflict: port 80") {
			t.Errorf("error should mention port conflict, got: %v", err)
		}
	})
}

func TestGenerateEmptyConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "stream.conf")
	httpPath := filepath.Join(tmpDir, "http.conf")

	log := lgr.New()
	gen, _ := NewGenerator(streamPath, httpPath, log)

	t.Run("generates empty configs when no containers", func(t *testing.T) {
		changed, err := gen.Generate([]docker.ContainerInfo{})
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if !changed {
			t.Error("expected config to be generated")
		}

		// Verify configs exist but are minimal/empty
		streamContent, err := os.ReadFile(streamPath)
		if err != nil {
			t.Fatalf("failed to read stream config: %v", err)
		}

		httpContent, err := os.ReadFile(httpPath)
		if err != nil {
			t.Fatalf("failed to read HTTP config: %v", err)
		}

		// Should contain comments but no upstream/server blocks
		if strings.Contains(string(streamContent), "upstream") {
			t.Error("empty stream config should not contain upstream blocks")
		}
		if strings.Contains(string(httpContent), "upstream") {
			t.Error("empty HTTP config should not contain upstream blocks")
		}
	})
}
