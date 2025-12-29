package nginx

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/moontechs/proxy/docker"
)

// Generator generates Nginx configuration files from container info
type Generator struct {
	streamConfigPath string
	httpConfigPath   string
	streamTemplate   *template.Template
	httpTemplate     *template.Template
	log              *lgr.Logger
}

// StreamData holds data for stream config template
type StreamData struct {
	Timestamp  string
	Containers []StreamContainer
}

// StreamContainer represents a container's stream proxy configuration
type StreamContainer struct {
	Name        string
	ID          string
	TCPMappings []StreamMapping
	UDPMappings []StreamMapping
}

// StreamMapping represents a single port mapping for stream module
type StreamMapping struct {
	ProxyPort     int
	ContainerPort int
	ContainerIP   string
}

// HTTPData holds data for HTTP config template
type HTTPData struct {
	Timestamp   string
	HTTPServers []HTTPServer
}

// HTTPServer represents an HTTP server block configuration
type HTTPServer struct {
	ContainerName string
	ContainerID   string
	UpstreamName  string
	Hostname      string
	ContainerIP   string
	ContainerPort int
	HTTPS         bool
}

// NewGenerator creates a new Nginx config generator
func NewGenerator(streamConfigPath, httpConfigPath string, log *lgr.Logger) (*Generator, error) {
	streamTmpl, err := template.New("stream").Parse(StreamTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse stream template: %w", err)
	}

	httpTmpl, err := template.New("http").Parse(HTTPTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTTP template: %w", err)
	}

	return &Generator{
		streamConfigPath: streamConfigPath,
		httpConfigPath:   httpConfigPath,
		streamTemplate:   streamTmpl,
		httpTemplate:     httpTmpl,
		log:              log,
	}, nil
}

// Generate generates both stream and HTTP configs from container info
// Returns true if any config changed, false if unchanged
func (g *Generator) Generate(containers []docker.ContainerInfo) (bool, error) {
	g.log.Logf("DEBUG [Generator] processing containers=%d", len(containers))

	// build template data
	streamData, httpData := g.buildTemplateData(containers)

	// validate for conflicts
	if err := g.validateConflicts(streamData, httpData); err != nil {
		return false, err
	}

	// generate and write stream config
	streamChanged, err := g.generateStreamConfig(streamData)
	if err != nil {
		return false, fmt.Errorf("stream config generation failed: %w", err)
	}

	// generate and write HTTP config
	httpChanged, err := g.generateHTTPConfig(httpData)
	if err != nil {
		return false, fmt.Errorf("HTTP config generation failed: %w", err)
	}

	changed := streamChanged || httpChanged
	g.log.Logf("INFO [Generator] generation complete stream_changed=%t http_changed=%t", streamChanged, httpChanged)

	return changed, nil
}

// buildTemplateData transforms container info into template data structures
func (g *Generator) buildTemplateData(containers []docker.ContainerInfo) (StreamData, HTTPData) {
	streamData := StreamData{
		Timestamp:  time.Now().Format(time.RFC3339),
		Containers: make([]StreamContainer, 0, len(containers)),
	}

	httpData := HTTPData{
		Timestamp:   time.Now().Format(time.RFC3339),
		HTTPServers: make([]HTTPServer, 0),
	}

	for _, container := range containers {
		// process stream mappings (TCP/UDP)
		if len(container.Mappings) > 0 {
			streamContainer := StreamContainer{
				Name:        container.Name,
				ID:          container.ID,
				TCPMappings: make([]StreamMapping, 0),
				UDPMappings: make([]StreamMapping, 0),
			}

			for _, mapping := range container.Mappings {
				streamMapping := StreamMapping{
					ProxyPort:     mapping.ProxyPort,
					ContainerPort: mapping.ContainerPort,
					ContainerIP:   container.IP,
				}

				if mapping.Protocol == docker.TCP {
					streamContainer.TCPMappings = append(streamContainer.TCPMappings, streamMapping)
				} else {
					streamContainer.UDPMappings = append(streamContainer.UDPMappings, streamMapping)
				}
			}

			streamData.Containers = append(streamData.Containers, streamContainer)
		}

		// process HTTP mappings
		if container.HTTPMapping != nil {
			for _, hostname := range container.HTTPMapping.Hostnames {
				httpServer := HTTPServer{
					ContainerName: container.Name,
					ContainerID:   container.ID,
					UpstreamName:  hostnameToUpstream(hostname),
					Hostname:      hostname,
					ContainerIP:   container.IP,
					ContainerPort: container.HTTPMapping.ContainerPort,
					HTTPS:         container.HTTPMapping.HTTPS,
				}
				httpData.HTTPServers = append(httpData.HTTPServers, httpServer)
			}
		}
	}

	return streamData, httpData
}

// validateConflicts checks for port and hostname conflicts
func (g *Generator) validateConflicts(streamData StreamData, httpData HTTPData) error {
	// check TCP port conflicts
	tcpPorts := make(map[int]string)
	for _, container := range streamData.Containers {
		for _, mapping := range container.TCPMappings {
			if existing, exists := tcpPorts[mapping.ProxyPort]; exists {
				return fmt.Errorf("TCP port conflict: port %d claimed by both %s and %s",
					mapping.ProxyPort, existing, container.Name)
			}
			tcpPorts[mapping.ProxyPort] = container.Name
		}
	}

	// check UDP port conflicts
	udpPorts := make(map[int]string)
	for _, container := range streamData.Containers {
		for _, mapping := range container.UDPMappings {
			if existing, exists := udpPorts[mapping.ProxyPort]; exists {
				return fmt.Errorf("UDP port conflict: port %d claimed by both %s and %s",
					mapping.ProxyPort, existing, container.Name)
			}
			udpPorts[mapping.ProxyPort] = container.Name
		}
	}

	// check HTTP hostname conflicts
	hostnames := make(map[string]string)
	for _, server := range httpData.HTTPServers {
		if existing, exists := hostnames[server.Hostname]; exists {
			return fmt.Errorf("HTTP hostname conflict: %s claimed by both %s and %s",
				server.Hostname, existing, server.ContainerName)
		}
		hostnames[server.Hostname] = server.ContainerName
	}

	g.log.Logf("DEBUG [Generator] validation passed tcp_ports=%d udp_ports=%d http_hosts=%d",
		len(tcpPorts), len(udpPorts), len(hostnames))

	return nil
}

// generateStreamConfig generates and writes stream config if changed
func (g *Generator) generateStreamConfig(data StreamData) (bool, error) {
	var buf bytes.Buffer
	if err := g.streamTemplate.Execute(&buf, data); err != nil {
		return false, fmt.Errorf("template execution failed: %w", err)
	}

	content := buf.Bytes()

	// debug: print generated config
	g.log.Logf("DEBUG [Generator] stream config generated:\n%s", string(content))

	return g.writeIfChanged(g.streamConfigPath, content)
}

// generateHTTPConfig generates and writes HTTP config if changed
func (g *Generator) generateHTTPConfig(data HTTPData) (bool, error) {
	var buf bytes.Buffer
	if err := g.httpTemplate.Execute(&buf, data); err != nil {
		return false, fmt.Errorf("template execution failed: %w", err)
	}

	content := buf.Bytes()

	// debug: print generated config
	g.log.Logf("DEBUG [Generator] HTTP config generated:\n%s", string(content))

	return g.writeIfChanged(g.httpConfigPath, content)
}

// writeIfChanged writes config to file only if content changed
func (g *Generator) writeIfChanged(path string, content []byte) (bool, error) {
	newChecksum := checksum(content)

	// read existing file checksum
	// #nosec G304 -- path is from trusted configuration, not user input
	oldContent, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to read existing config: %w", err)
	}

	oldChecksum := checksum(oldContent)

	if newChecksum == oldChecksum {
		g.log.Logf("DEBUG [Generator] config unchanged path=%s checksum=%s", path, newChecksum[:8])
		return false, nil
	}

	// write atomically (tmp file + rename)
	if err := atomicWrite(path, content); err != nil {
		return false, err
	}

	g.log.Logf("INFO [Generator] config written path=%s checksum=%s size=%d", path, newChecksum[:8], len(content))
	return true, nil
}

// checksum computes SHA256 checksum of data
func checksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// atomicWrite writes data to file atomically using tmp file + rename
func atomicWrite(path string, data []byte) error {
	tmpFile := path + ".tmp"

	// write to temp file
	// #nosec G306 -- nginx config files need 0644 to be readable by nginx process
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// atomic rename
	if err := os.Rename(tmpFile, path); err != nil {
		// cleanup on failure
		if removeErr := os.Remove(tmpFile); removeErr != nil {
			// return both errors - can't use %w for second error
			return fmt.Errorf("failed to rename temp file: %w (and failed cleanup: %v)", err, removeErr) //nolint:errorlint // secondary error context only
		}
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// hostnameToUpstream converts a hostname to a valid upstream name
// Example: api.example.com -> http_api_example_com
func hostnameToUpstream(hostname string) string {
	// replace dots and hyphens with underscores
	upstream := regexp.MustCompile(`[.-]`).ReplaceAllString(hostname, "_")
	// prefix with http_
	return "http_" + strings.ToLower(upstream)
}
