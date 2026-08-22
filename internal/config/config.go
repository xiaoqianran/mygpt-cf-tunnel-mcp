package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	ListenAddr       string
	APIToken         string
	RegistryPath     string
	RequestTimeout   time.Duration
	ToolCacheTTL     time.Duration
	MaxResponseBytes int64
}

type Registry struct {
	Servers map[string]Server `json:"servers"`
}

type Server struct {
	Transport      string            `json:"transport,omitempty"`
	URL            string            `json:"url,omitempty"`
	Description    string            `json:"description,omitempty"`
	TokenEnv       string            `json:"token_env,omitempty"`
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	WorkingDir     string            `json:"working_dir,omitempty"`
	Enabled        *bool             `json:"enabled,omitempty"`
	AllowTools     []string          `json:"allow_tools,omitempty"`
	DenyTools      []string          `json:"deny_tools,omitempty"`
	AllowResources []string          `json:"allow_resources,omitempty"`
	DenyResources  []string          `json:"deny_resources,omitempty"`
	AllowPrompts   []string          `json:"allow_prompts,omitempty"`
	DenyPrompts    []string          `json:"deny_prompts,omitempty"`
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
	cfg.ToolCacheTTL, err = time.ParseDuration(env("TOOL_CACHE_TTL", "5m"))
	if err != nil {
		return Config{}, fmt.Errorf("TOOL_CACHE_TTL: %w", err)
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
	if r.Servers == nil {
		r.Servers = map[string]Server{}
	}
	for name, srv := range r.Servers {
		if err := srv.Validate(name); err != nil {
			return Registry{}, err
		}
	}
	return r, nil
}

func (s Server) IsEnabled() bool { return s.Enabled == nil || *s.Enabled }

func (s Server) Validate(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("server name cannot be empty")
	}
	t := s.TransportName()
	switch t {
	case "streamable_http":
		if s.URL == "" {
			return fmt.Errorf("server %q: url required", name)
		}
	case "stdio":
		if s.Command == "" {
			return fmt.Errorf("server %q: command required", name)
		}
	default:
		return fmt.Errorf("server %q: unsupported transport %q", name, s.Transport)
	}
	return nil
}

func (s Server) TransportName() string {
	t := strings.ToLower(strings.TrimSpace(s.Transport))
	if t == "" && s.URL != "" {
		return "streamable_http"
	}
	if t == "http" || t == "streamable-http" {
		return "streamable_http"
	}
	return t
}

func (s Server) ToolAllowed(name string) bool {
	return allowedByPolicy(name, s.AllowTools, s.DenyTools)
}
func (s Server) ResourceAllowed(uri string) bool {
	return allowedByPolicy(uri, s.AllowResources, s.DenyResources)
}
func (s Server) PromptAllowed(name string) bool {
	return allowedByPolicy(name, s.AllowPrompts, s.DenyPrompts)
}

func allowedByPolicy(value string, allow, deny []string) bool {
	for _, p := range deny {
		if matchTool(p, value) {
			return false
		}
	}
	if len(allow) == 0 {
		return true
	}
	for _, p := range allow {
		if matchTool(p, value) {
			return true
		}
	}
	return false
}

func matchTool(pattern, name string) bool {
	if pattern == "*" || pattern == name {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, strings.TrimSuffix(pattern, "*"))
	}
	return false
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
