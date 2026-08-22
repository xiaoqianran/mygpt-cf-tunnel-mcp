package mcpclient

import (
	"context"
	"fmt"
	"net/http"
	"os"
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
	transport := &mcp.StreamableClientTransport{
		Endpoint:             srv.URL,
		HTTPClient:           hc,
		DisableStandaloneSSE: true,
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "mygpt-cf-tunnel-mcp", Version: "0.1.0"}, nil)
	sess, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", alias, err)
	}
	return sess, nil
}
