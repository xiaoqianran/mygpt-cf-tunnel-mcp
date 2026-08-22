package agent

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/config"
	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/mcpclient"
)

type Server struct {
	cfg config.Config
	log *slog.Logger
	mcp *mcpclient.Manager
}

func New(cfg config.Config, log *slog.Logger) (*Server, error) {
	reg, err := config.LoadRegistry(cfg.RegistryPath)
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, log: log, mcp: mcpclient.NewManager(reg, cfg.ToolCacheTTL)}, nil
}
func (s *Server) Close() { s.mcp.Close() }
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /openapi.json", s.openAPI)
	mux.Handle("GET /v1/mcp/servers", s.auth(http.HandlerFunc(s.listServers)))
	mux.Handle("GET /v1/mcp/status", s.auth(http.HandlerFunc(s.serverStatus)))
	mux.Handle("POST /v1/mcp/tools/search", s.auth(http.HandlerFunc(s.searchTools)))
	mux.Handle("POST /v1/mcp/tools/call", s.auth(http.HandlerFunc(s.callTool)))
	mux.Handle("POST /v1/mcp/tools/call-readonly", s.auth(http.HandlerFunc(s.callReadOnlyTool)))
	mux.Handle("POST /v1/mcp/resources/list", s.auth(http.HandlerFunc(s.listResources)))
	mux.Handle("POST /v1/mcp/resources/read", s.auth(http.HandlerFunc(s.readResource)))
	mux.Handle("POST /v1/mcp/prompts/list", s.auth(http.HandlerFunc(s.listPrompts)))
	mux.Handle("POST /v1/mcp/prompts/get", s.auth(http.HandlerFunc(s.getPrompt)))
	mux.Handle("POST /v1/mcp/reload", s.auth(http.HandlerFunc(s.reloadRegistry)))
	return s.logging(mux)
}
func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.cfg.APIToken)) != 1 {
			writeJSON(w, 401, map[string]any{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.log.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"ok": true, "version": "0.5.0"})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
