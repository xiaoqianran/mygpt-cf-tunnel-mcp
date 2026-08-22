package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	ListenAddr       string
	APIToken         string
	RegistryPath     string
	RequestTimeout   time.Duration
	MaxResponseBytes int64
}

type Registry struct {
	Servers map[string]Server `json:"servers"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	TokenEnv    string `json:"token_env,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddr:       env("LISTEN_ADDR", "127.0.0.1:8788"),
		APIToken:         os.Getenv("API_TOKEN"),
		RegistryPath:     env("MCP_REGISTRY", "/etc/mygpt-mcp/servers.json"),
		MaxResponseBytes: 1 << 20,
	}
	if cfg.APIToken == "" {
		return Config{}, errors.New("API_TOKEN is required")
	}
	var err error
	cfg.RequestTimeout, err = time.ParseDuration(env("REQUEST_TIMEOUT", "90s"))
	if err != nil {
		return Config{}, fmt.Errorf("REQUEST_TIMEOUT: %w", err)
	}
	return cfg, nil
}

func LoadRegistry(path string) (Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Registry{}, err
	}
	var r Registry
	if err := json.Unmarshal(b, &r); err != nil {
		return Registry{}, err
	}
	if len(r.Servers) == 0 {
		return Registry{}, errors.New("registry has no servers")
	}
	return r, nil
}

func (s Server) IsEnabled() bool { return s.Enabled == nil || *s.Enabled }
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
