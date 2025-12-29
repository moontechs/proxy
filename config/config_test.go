package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		check   func(*testing.T, *Config)
	}{
		{
			name:    "default configuration",
			envVars: map[string]string{},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.DockerHost != "unix:///var/run/docker.sock" {
					t.Errorf("expected default docker host, got %s", cfg.DockerHost)
				}
				if cfg.StreamConfigPath != "/etc/nginx/conf.d/proxy.conf" {
					t.Errorf("expected default stream config path, got %s", cfg.StreamConfigPath)
				}
				if cfg.HTTPConfigPath != "/etc/nginx/conf.d/http-proxy.conf" {
					t.Errorf("expected default HTTP config path, got %s", cfg.HTTPConfigPath)
				}
				if cfg.NginxReloadCmd != "nginx -s reload" {
					t.Errorf("expected default reload cmd, got %s", cfg.NginxReloadCmd)
				}
				if cfg.LogLevel != "INFO" {
					t.Errorf("expected default log level INFO, got %s", cfg.LogLevel)
				}
				if cfg.LogCaller {
					t.Error("expected LogCaller=false by default")
				}
			},
		},
		{
			name: "custom configuration via environment",
			envVars: map[string]string{
				"DOCKER_HOST":              "tcp://localhost:2375",
				"NGINX_STREAM_CONFIG_PATH": "/custom/stream.conf",
				"NGINX_HTTP_CONFIG_PATH":   "/custom/http.conf",
				"NGINX_RELOAD_CMD":         "systemctl reload nginx",
				"LOG_LEVEL":                "DEBUG",
				"LOG_CALLER":               "true",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.DockerHost != "tcp://localhost:2375" {
					t.Errorf("expected custom docker host, got %s", cfg.DockerHost)
				}
				if cfg.StreamConfigPath != "/custom/stream.conf" {
					t.Errorf("expected custom stream config path, got %s", cfg.StreamConfigPath)
				}
				if cfg.HTTPConfigPath != "/custom/http.conf" {
					t.Errorf("expected custom HTTP config path, got %s", cfg.HTTPConfigPath)
				}
				if cfg.NginxReloadCmd != "systemctl reload nginx" {
					t.Errorf("expected custom reload cmd, got %s", cfg.NginxReloadCmd)
				}
				if cfg.LogLevel != "DEBUG" {
					t.Errorf("expected DEBUG log level, got %s", cfg.LogLevel)
				}
				if !cfg.LogCaller {
					t.Error("expected LogCaller=true")
				}
			},
		},
		{
			name: "log level normalization",
			envVars: map[string]string{
				"LOG_LEVEL": "debug",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.LogLevel != "DEBUG" {
					t.Errorf("expected log level to be uppercased to DEBUG, got %s", cfg.LogLevel)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal string
		envVal     string
		want       string
	}{
		{
			name:       "env var set",
			key:        "TEST_KEY",
			defaultVal: "default",
			envVal:     "custom",
			want:       "custom",
		},
		{
			name:       "env var not set",
			key:        "TEST_KEY_MISSING",
			defaultVal: "default",
			envVal:     "",
			want:       "default",
		},
		{
			name:       "empty env var uses default",
			key:        "TEST_KEY_EMPTY",
			defaultVal: "default",
			envVal:     "",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envVal != "" {
				os.Setenv(tt.key, tt.envVal)
			}

			got := getEnvOrDefault(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnvOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}
