package docker

import (
	"testing"
)

func TestParsePortMappings(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []PortMapping
		wantErr bool
	}{
		{
			name:  "single port with mapping",
			input: "80:8080",
			want: []PortMapping{
				{ProxyPort: 80, ContainerPort: 8080},
			},
			wantErr: false,
		},
		{
			name:  "single port without mapping",
			input: "53",
			want: []PortMapping{
				{ProxyPort: 53, ContainerPort: 53},
			},
			wantErr: false,
		},
		{
			name:  "multiple ports mixed",
			input: "80:8080,443:8443,53,9090",
			want: []PortMapping{
				{ProxyPort: 80, ContainerPort: 8080},
				{ProxyPort: 443, ContainerPort: 8443},
				{ProxyPort: 53, ContainerPort: 53},
				{ProxyPort: 9090, ContainerPort: 9090},
			},
			wantErr: false,
		},
		{
			name:  "ports with spaces",
			input: "80:8080, 443:8443, 53",
			want: []PortMapping{
				{ProxyPort: 80, ContainerPort: 8080},
				{ProxyPort: 443, ContainerPort: 8443},
				{ProxyPort: 53, ContainerPort: 53},
			},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    []PortMapping{},
			wantErr: false,
		},
		{
			name:    "invalid format - too many colons",
			input:   "80:8080:9090",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid proxy port",
			input:   "abc:8080",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid container port",
			input:   "80:abc",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid single port",
			input:   "abc",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "proxy port out of range",
			input:   "70000:8080",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "container port out of range",
			input:   "80:70000",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "port zero",
			input:   "0:8080",
			want:    nil,
			wantErr: true,
		},
		{
			name:  "complex realistic example",
			input: "80:1080,443:1443,53,8080:3000,9000",
			want: []PortMapping{
				{ProxyPort: 80, ContainerPort: 1080},
				{ProxyPort: 443, ContainerPort: 1443},
				{ProxyPort: 53, ContainerPort: 53},
				{ProxyPort: 8080, ContainerPort: 3000},
				{ProxyPort: 9000, ContainerPort: 9000},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePortMappings(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePortMappings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("parsePortMappings() got %d mappings, want %d", len(got), len(tt.want))
					return
				}

				for i, mapping := range got {
					if mapping.ProxyPort != tt.want[i].ProxyPort {
						t.Errorf("mapping[%d].ProxyPort = %d, want %d", i, mapping.ProxyPort, tt.want[i].ProxyPort)
					}
					if mapping.ContainerPort != tt.want[i].ContainerPort {
						t.Errorf("mapping[%d].ContainerPort = %d, want %d", i, mapping.ContainerPort, tt.want[i].ContainerPort)
					}
				}
			}
		})
	}
}

func TestPortMapping(t *testing.T) {
	t.Run("valid TCP port mapping struct", func(t *testing.T) {
		pm := PortMapping{
			ProxyPort:     80,
			ContainerPort: 8080,
			Protocol:      TCP,
		}

		if pm.ProxyPort != 80 {
			t.Errorf("ProxyPort = %d, want 80", pm.ProxyPort)
		}
		if pm.ContainerPort != 8080 {
			t.Errorf("ContainerPort = %d, want 8080", pm.ContainerPort)
		}
		if pm.Protocol != TCP {
			t.Errorf("Protocol = %d, want TCP", pm.Protocol)
		}
	})

	t.Run("valid UDP port mapping struct", func(t *testing.T) {
		pm := PortMapping{
			ProxyPort:     53,
			ContainerPort: 5353,
			Protocol:      UDP,
		}

		if pm.ProxyPort != 53 {
			t.Errorf("ProxyPort = %d, want 53", pm.ProxyPort)
		}
		if pm.ContainerPort != 5353 {
			t.Errorf("ContainerPort = %d, want 5353", pm.ContainerPort)
		}
		if pm.Protocol != UDP {
			t.Errorf("Protocol = %d, want UDP", pm.Protocol)
		}
	})
}

func TestContainerInfo(t *testing.T) {
	t.Run("container info struct with TCP/UDP mappings", func(t *testing.T) {
		info := ContainerInfo{
			Name: "test-container",
			ID:   "abc123",
			IP:   "172.17.0.2",
			Mappings: []PortMapping{
				{ProxyPort: 80, ContainerPort: 8080, Protocol: TCP},
				{ProxyPort: 53, ContainerPort: 5353, Protocol: UDP},
			},
		}

		if info.Name != "test-container" {
			t.Errorf("Name = %s, want test-container", info.Name)
		}
		if info.ID != "abc123" {
			t.Errorf("ID = %s, want abc123", info.ID)
		}
		if info.IP != "172.17.0.2" {
			t.Errorf("IP = %s, want 172.17.0.2", info.IP)
		}
		if len(info.Mappings) != 2 {
			t.Errorf("got %d mappings, want 2", len(info.Mappings))
		}
		if info.Mappings[0].Protocol != TCP {
			t.Errorf("Mappings[0].Protocol = %d, want TCP", info.Mappings[0].Protocol)
		}
		if info.Mappings[1].Protocol != UDP {
			t.Errorf("Mappings[1].Protocol = %d, want UDP", info.Mappings[1].Protocol)
		}
	})

	t.Run("container info struct with HTTP mapping", func(t *testing.T) {
		info := ContainerInfo{
			Name: "api-container",
			ID:   "def456",
			IP:   "172.17.0.3",
			HTTPMapping: &HTTPMapping{
				Hostnames:     []string{"api.example.com", "api.test.com"},
				ContainerPort: 8080,
				HTTPS:         false,
			},
		}

		if info.Name != "api-container" {
			t.Errorf("Name = %s, want api-container", info.Name)
		}
		if info.HTTPMapping == nil {
			t.Fatal("HTTPMapping should not be nil")
		}
		if len(info.HTTPMapping.Hostnames) != 2 {
			t.Errorf("got %d hostnames, want 2", len(info.HTTPMapping.Hostnames))
		}
		if info.HTTPMapping.Hostnames[0] != "api.example.com" {
			t.Errorf("Hostnames[0] = %s, want api.example.com", info.HTTPMapping.Hostnames[0])
		}
		if info.HTTPMapping.ContainerPort != 8080 {
			t.Errorf("ContainerPort = %d, want 8080", info.HTTPMapping.ContainerPort)
		}
		if info.HTTPMapping.HTTPS {
			t.Error("HTTPS should be false")
		}
	})

	t.Run("container info struct with both TCP and HTTP mapping", func(t *testing.T) {
		info := ContainerInfo{
			Name: "hybrid-container",
			IP:   "172.17.0.4",
			Mappings: []PortMapping{
				{ProxyPort: 22, ContainerPort: 22, Protocol: TCP},
			},
			HTTPMapping: &HTTPMapping{
				Hostnames:     []string{"ssh.example.com"},
				ContainerPort: 2222,
				HTTPS:         true,
			},
		}

		if len(info.Mappings) != 1 {
			t.Errorf("got %d port mappings, want 1", len(info.Mappings))
		}
		if info.HTTPMapping == nil {
			t.Fatal("HTTPMapping should not be nil")
		}
		if info.HTTPMapping.HTTPS != true {
			t.Error("HTTPS should be true")
		}
	})
}

func TestHTTPMapping(t *testing.T) {
	t.Run("HTTP mapping with single hostname", func(t *testing.T) {
		mapping := HTTPMapping{
			Hostnames:     []string{"api.example.com"},
			ContainerPort: 8080,
			HTTPS:         false,
		}

		if len(mapping.Hostnames) != 1 {
			t.Errorf("got %d hostnames, want 1", len(mapping.Hostnames))
		}
		if mapping.Hostnames[0] != "api.example.com" {
			t.Errorf("Hostnames[0] = %s, want api.example.com", mapping.Hostnames[0])
		}
		if mapping.ContainerPort != 8080 {
			t.Errorf("ContainerPort = %d, want 8080", mapping.ContainerPort)
		}
		if mapping.HTTPS {
			t.Error("HTTPS should be false")
		}
	})

	t.Run("HTTP mapping with multiple hostnames", func(t *testing.T) {
		mapping := HTTPMapping{
			Hostnames:     []string{"web.example.com", "app.example.com", "www.example.com"},
			ContainerPort: 3000,
			HTTPS:         true,
		}

		if len(mapping.Hostnames) != 3 {
			t.Errorf("got %d hostnames, want 3", len(mapping.Hostnames))
		}
		if mapping.ContainerPort != 3000 {
			t.Errorf("ContainerPort = %d, want 3000", mapping.ContainerPort)
		}
		if !mapping.HTTPS {
			t.Error("HTTPS should be true")
		}
	})

	t.Run("HTTPS mapping on port 443", func(t *testing.T) {
		mapping := HTTPMapping{
			Hostnames:     []string{"secure.example.com"},
			ContainerPort: 8443,
			HTTPS:         true,
		}

		if mapping.ContainerPort != 8443 {
			t.Errorf("ContainerPort = %d, want 8443", mapping.ContainerPort)
		}
		if !mapping.HTTPS {
			t.Error("HTTPS should be true for secure mapping")
		}
	})
}
