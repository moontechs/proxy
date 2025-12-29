package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/go-pkgz/lgr"
)

// Client wraps Docker API client
type Client struct {
	cli *client.Client
	log *lgr.Logger
}

// Protocol represents the network protocol type
type Protocol int

const (
	// TCP represents the TCP protocol
	TCP Protocol = iota
	// UDP represents the UDP protocol
	UDP
)

// ContainerInfo holds parsed container information
type ContainerInfo struct {
	Name        string
	ID          string
	IP          string
	Mappings    []PortMapping // TCP/UDP port mappings
	HTTPMapping *HTTPMapping  // HTTP hostname routing (optional)
}

// PortMapping represents a proxy port to container port mapping with protocol
type PortMapping struct {
	ProxyPort     int
	ContainerPort int
	Protocol      Protocol
}

// HTTPMapping represents HTTP hostname-based routing configuration
type HTTPMapping struct {
	Hostnames     []string // list of hostnames for this container
	ContainerPort int      // container HTTP port
	HTTPS         bool     // whether to listen on 443 instead of 80
}

// NewClient creates a new Docker client
func NewClient(host string, log *lgr.Logger) (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(host),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	log.Logf("INFO connecting to Docker socket=%s", host)

	// test connection
	ctx := context.Background()
	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping Docker daemon: %w", err)
	}

	log.Logf("DEBUG docker connection established")

	return &Client{cli: cli, log: log}, nil
}

// ScanContainers finds all running containers with proxy labels
func (c *Client) ScanContainers(ctx context.Context) ([]ContainerInfo, error) {
	c.log.Logf("INFO scanning containers for proxy labels")
	c.log.Logf("DEBUG [Docker] listing_all_containers")

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	c.log.Logf("DEBUG [Docker] found_running_containers count=%d", len(containers))

	var results []ContainerInfo
	for _, ctr := range containers {
		info, err := c.parseContainer(ctx, ctr)
		if err != nil {
			c.log.Logf("WARN [Docker] container=%s parse_error=%q", ctr.Names[0], err)
			continue
		}
		if info != nil {
			results = append(results, *info)
		}
	}

	c.log.Logf("INFO route discovery complete: containers=%d", len(results))
	return results, nil
}

//nolint:gocognit,gocyclo // complex parsing logic is unavoidable
func (c *Client) parseContainer(ctx context.Context, ctr types.Container) (*ContainerInfo, error) {
	name := strings.TrimPrefix(ctr.Names[0], "/")
	id := ctr.ID[:12]

	// get container IP
	inspect, err := c.cli.ContainerInspect(ctx, ctr.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	ip := inspect.NetworkSettings.IPAddress
	if ip == "" {
		// try default bridge network
		for _, network := range inspect.NetworkSettings.Networks {
			if network.IPAddress != "" {
				ip = network.IPAddress
				break
			}
		}
	}

	if ip == "" {
		c.log.Logf("WARN [Docker] container=%s no_ip_address skipping", name)
		return nil, nil
	}

	c.log.Logf("DEBUG [Docker] processing_container name=%s id=%s ip=%s", name, id, ip)

	// read labels
	c.log.Logf("DEBUG [Docker] reading_labels container=%s", name)

	tcpPortsStr := ctr.Labels["proxy.tcp.ports"]
	udpPortsStr := ctr.Labels["proxy.udp.ports"]
	httpHostStr := ctr.Labels["proxy.http.host"]
	httpPortStr := ctr.Labels["proxy.http.port"]
	httpHTTPSStr := ctr.Labels["proxy.http.https"]

	c.log.Logf("DEBUG [Docker] container=%s proxy.tcp.ports=%q", name, tcpPortsStr)
	c.log.Logf("DEBUG [Docker] container=%s proxy.udp.ports=%q", name, udpPortsStr)
	c.log.Logf("DEBUG [Docker] container=%s proxy.http.host=%q", name, httpHostStr)

	// skip if all labels are empty
	if tcpPortsStr == "" && udpPortsStr == "" && httpHostStr == "" {
		c.log.Logf("WARN [Docker] container=%s no proxy labels, skipping", name)
		return nil, nil
	}

	var mappings []PortMapping
	tcpCount := 0
	udpCount := 0

	// parse TCP port mappings
	if tcpPortsStr != "" {
		c.log.Logf("DEBUG [Docker] parsing_tcp_port_mappings container=%s input=%q", name, tcpPortsStr)
		tcpMappings, err := parsePortMappings(tcpPortsStr)
		if err != nil {
			c.log.Logf("ERROR [Docker] container=%s invalid_tcp_port_mapping format=%q", name, tcpPortsStr)
			c.log.Logf("WARN [Docker] skipping_container name=%s reason=invalid_configuration", name)
			return nil, fmt.Errorf("invalid TCP port mappings: %w", err)
		}
		// tag with TCP protocol
		for i := range tcpMappings {
			tcpMappings[i].Protocol = TCP
			mappings = append(mappings, tcpMappings[i])
			c.log.Logf("DEBUG [Docker] container=%s parsed protocol=TCP proxy_port=%d container_port=%d",
				name, tcpMappings[i].ProxyPort, tcpMappings[i].ContainerPort)
		}
		tcpCount = len(tcpMappings)
	}

	// parse UDP port mappings
	if udpPortsStr != "" {
		c.log.Logf("DEBUG [Docker] parsing_udp_port_mappings container=%s input=%q", name, udpPortsStr)
		udpMappings, err := parsePortMappings(udpPortsStr)
		if err != nil {
			c.log.Logf("ERROR [Docker] container=%s invalid_udp_port_mapping format=%q", name, udpPortsStr)
			c.log.Logf("WARN [Docker] skipping_container name=%s reason=invalid_configuration", name)
			return nil, fmt.Errorf("invalid UDP port mappings: %w", err)
		}
		// tag with UDP protocol
		for i := range udpMappings {
			udpMappings[i].Protocol = UDP
			mappings = append(mappings, udpMappings[i])
			c.log.Logf("DEBUG [Docker] container=%s parsed protocol=UDP proxy_port=%d container_port=%d",
				name, udpMappings[i].ProxyPort, udpMappings[i].ContainerPort)
		}
		udpCount = len(udpMappings)
	}

	// parse HTTP hostname mapping
	var httpMapping *HTTPMapping
	if httpHostStr != "" {
		c.log.Logf("DEBUG [Docker] parsing_http_host container=%s input=%q", name, httpHostStr)

		// parse hostnames (comma-separated)
		hostnames := strings.Split(httpHostStr, ",")
		for i := range hostnames {
			hostnames[i] = strings.TrimSpace(hostnames[i])
		}

		// parse HTTP port (default: 80)
		httpPort := 80
		if httpPortStr != "" {
			var err error
			httpPort, err = strconv.Atoi(strings.TrimSpace(httpPortStr))
			if err != nil {
				c.log.Logf("ERROR [Docker] container=%s invalid_http_port format=%q", name, httpPortStr)
				return nil, fmt.Errorf("invalid HTTP port: %w", err)
			}
			if httpPort < 1 || httpPort > 65535 {
				return nil, fmt.Errorf("HTTP port %d out of range", httpPort)
			}
		}

		// parse HTTPS flag (default: false)
		https := false
		if httpHTTPSStr != "" {
			https = strings.ToLower(strings.TrimSpace(httpHTTPSStr)) == "true"
		}

		httpMapping = &HTTPMapping{
			Hostnames:     hostnames,
			ContainerPort: httpPort,
			HTTPS:         https,
		}

		c.log.Logf("INFO [Docker] container=%s http_mapping hostnames=%d port=%d https=%t",
			name, len(hostnames), httpPort, https)
	}

	c.log.Logf("DEBUG [Docker] container=%s port_mappings_count=%d", name, len(mappings))
	c.log.Logf("INFO [Docker] registered_container name=%s tcp_ports=%d udp_ports=%d http_hosts=%d",
		name, tcpCount, udpCount, func() int {
			if httpMapping != nil {
				return len(httpMapping.Hostnames)
			}
			return 0
		}())

	return &ContainerInfo{
		Name:        name,
		ID:          id,
		IP:          ip,
		Mappings:    mappings,
		HTTPMapping: httpMapping,
	}, nil
}

// parsePortMappings parses the proxy.ports label
// Format: "80:1080,443:1443,53,8080"
//
//nolint:gocognit // complex parsing logic is unavoidable
func parsePortMappings(s string) ([]PortMapping, error) {
	parts := strings.Split(s, ",")
	mappings := make([]PortMapping, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var proxyPort, containerPort int

		if strings.Contains(part, ":") {
			// format: proxy_port:container_port
			splits := strings.Split(part, ":")
			if len(splits) != 2 {
				return nil, fmt.Errorf("invalid port mapping format: %q", part)
			}

			var err error
			proxyPort, err = strconv.Atoi(strings.TrimSpace(splits[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid proxy port %q: %w", splits[0], err)
			}

			containerPort, err = strconv.Atoi(strings.TrimSpace(splits[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid container port %q: %w", splits[1], err)
			}
		} else {
			// format: port (same for both)
			var err error
			proxyPort, err = strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid port %q: %w", part, err)
			}
			containerPort = proxyPort
		}

		if proxyPort < 1 || proxyPort > 65535 {
			return nil, fmt.Errorf("proxy port %d out of range", proxyPort)
		}
		if containerPort < 1 || containerPort > 65535 {
			return nil, fmt.Errorf("container port %d out of range", containerPort)
		}

		mappings = append(mappings, PortMapping{
			ProxyPort:     proxyPort,
			ContainerPort: containerPort,
		})
	}

	return mappings, nil
}

// EventType represents container lifecycle events
type EventType string

const (
	// EventStart represents a container start event
	EventStart EventType = "start"
	// EventStop represents a container stop event
	EventStop EventType = "stop"
	// EventDie represents a container die event
	EventDie EventType = "die"
)

// ContainerEvent represents a Docker container event
type ContainerEvent struct {
	Type        EventType
	ContainerID string
	Name        string
	Timestamp   time.Time
}

// WatchEvents watches Docker events and returns channels for events and errors
func (c *Client) WatchEvents(ctx context.Context) (<-chan ContainerEvent, <-chan error) {
	eventCh := make(chan ContainerEvent, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		// filter for container events only
		filters := filters.NewArgs()
		filters.Add("type", "container")
		filters.Add("event", "start")
		filters.Add("event", "stop")
		filters.Add("event", "die")

		eventStream, eventErrCh := c.cli.Events(ctx, types.EventsOptions{
			Filters: filters,
		})

		c.log.Logf("INFO [Docker] watching events")

		for {
			select {
			case event := <-eventStream:
				containerEvent := ContainerEvent{
					Type:        EventType(event.Action),
					ContainerID: event.Actor.ID[:12],
					Name:        strings.TrimPrefix(event.Actor.Attributes["name"], "/"),
					Timestamp:   time.Unix(event.Time, 0),
				}

				c.log.Logf("INFO [Docker] event type=%s container=%s id=%s",
					containerEvent.Type, containerEvent.Name, containerEvent.ContainerID)

				eventCh <- containerEvent

			case err := <-eventErrCh:
				if err != nil {
					c.log.Logf("ERROR [Docker] event_stream_error error=%q", err)
					errCh <- err
					return
				}

			case <-ctx.Done():
				c.log.Logf("INFO [Docker] event_stream_closed")
				return
			}
		}
	}()

	return eventCh, errCh
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	c.log.Logf("INFO closing_docker_client")
	return c.cli.Close()
}
