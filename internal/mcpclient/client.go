package mcpclient

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/config"
)

type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c := req.Clone(req.Context())
	c.Header = req.Header.Clone()
	c.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(c)
}

func Connect(ctx context.Context, alias string, srv config.Server) (*mcp.ClientSession, error) {
	if !srv.IsEnabled() {
		return nil, fmt.Errorf("MCP server %q is disabled", alias)
	}

	var transport mcp.Transport
	switch normalizedTransport(srv) {
	case "streamable_http":
		t, err := httpTransport(alias, srv)
		if err != nil {
			return nil, err
		}
		transport = t
	case "stdio":
		t, err := stdioTransport(alias, srv)
		if err != nil {
			return nil, err
		}
		transport = t
	default:
		return nil, fmt.Errorf("MCP server %q has unsupported transport %q", alias, srv.Transport)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "mygpt-cf-tunnel-mcp", Version: "0.2.0"}, nil)
	sess, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", alias, err)
	}
	return sess, nil
}

func normalizedTransport(srv config.Server) string {
	t := strings.ToLower(strings.TrimSpace(srv.Transport))
	if t == "" && srv.URL != "" {
		return "streamable_http"
	}
	if t == "http" || t == "streamable-http" {
		return "streamable_http"
	}
	return t
}

func httpTransport(alias string, srv config.Server) (mcp.Transport, error) {
	if srv.URL == "" {
		return nil, fmt.Errorf("MCP server %q requires url", alias)
	}
	if !strings.HasPrefix(srv.URL, "https://") && !strings.HasPrefix(srv.URL, "http://127.0.0.1") && !strings.HasPrefix(srv.URL, "http://localhost") {
		return nil, fmt.Errorf("MCP server %q URL must use https (localhost is allowed)", alias)
	}
	hc := &http.Client{}
	if srv.TokenEnv != "" {
		token := os.Getenv(srv.TokenEnv)
		if token == "" {
			return nil, fmt.Errorf("environment variable %s is empty", srv.TokenEnv)
		}
		hc.Transport = bearerTransport{token: token, base: http.DefaultTransport}
	}
	return &mcp.StreamableClientTransport{
		Endpoint:             srv.URL,
		HTTPClient:           hc,
		DisableStandaloneSSE: true,
	}, nil
}

func stdioTransport(alias string, srv config.Server) (mcp.Transport, error) {
	if srv.Command == "" {
		return nil, fmt.Errorf("MCP server %q requires command", alias)
	}
	cmd := exec.Command(srv.Command, srv.Args...)
	if srv.WorkingDir != "" {
		cmd.Dir = srv.WorkingDir
	}
	cmd.Env = os.Environ()
	for k, v := range srv.Env {
		cmd.Env = append(cmd.Env, k+"="+os.ExpandEnv(v))
	}
	// stderr is intentionally inherited by the service log. stdout belongs to MCP JSON-RPC.
	cmd.Stderr = os.Stderr
	return &mcp.CommandTransport{Command: cmd}, nil
}
