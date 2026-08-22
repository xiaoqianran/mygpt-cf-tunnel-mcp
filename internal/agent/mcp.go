package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/config"
	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/mcpclient"
)

type serverView struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
type searchRequest struct {
	Query  string `json:"query"`
	Server string `json:"server,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
type toolView struct {
	Server      string `json:"server"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}
type callRequest struct {
	Server    string         `json:"server"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

func (s *Server) registry() (config.Registry, error) { return config.LoadRegistry(s.cfg.RegistryPath) }
func (s *Server) listServers(w http.ResponseWriter, r *http.Request) {
	reg, err := s.registry()
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	out := make([]serverView, 0, len(reg.Servers))
	for name, v := range reg.Servers {
		if v.IsEnabled() {
			out = append(out, serverView{Name: name, Description: v.Description})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, 200, map[string]any{"servers": out})
}
func (s *Server) searchTools(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]any{"error": "invalid JSON"})
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 20
	}
	reg, err := s.registry()
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	names := make([]string, 0, len(reg.Servers))
	if req.Server != "" {
		if _, ok := reg.Servers[req.Server]; !ok {
			writeJSON(w, 404, map[string]any{"error": "unknown server"})
			return
		}
		names = []string{req.Server}
	} else {
		for n, v := range reg.Servers {
			if v.IsEnabled() {
				names = append(names, n)
			}
		}
		sort.Strings(names)
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()
	matches := []toolView{}
	q := strings.ToLower(req.Query)
	for _, name := range names {
		sess, e := mcpclient.Connect(ctx, name, reg.Servers[name])
		if e != nil {
			s.log.Warn("mcp connect failed", "server", name, "error", e)
			continue
		}
		res, e := sess.ListTools(ctx, nil)
		_ = sess.Close()
		if e != nil {
			s.log.Warn("mcp list failed", "server", name, "error", e)
			continue
		}
		for _, t := range res.Tools {
			hay := strings.ToLower(t.Name + " " + t.Description)
			if q == "" || strings.Contains(hay, q) {
				matches = append(matches, toolView{Server: name, Name: t.Name, Description: t.Description, InputSchema: t.InputSchema})
				if len(matches) >= req.Limit {
					writeJSON(w, 200, map[string]any{"tools": matches})
					return
				}
			}
		}
	}
	writeJSON(w, 200, map[string]any{"tools": matches})
}
func (s *Server) callTool(w http.ResponseWriter, r *http.Request) {
	var req callRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Server == "" || req.Tool == "" {
		writeJSON(w, 400, map[string]any{"error": "server and tool are required"})
		return
	}
	reg, err := s.registry()
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	srv, ok := reg.Servers[req.Server]
	if !ok {
		writeJSON(w, 404, map[string]any{"error": "unknown server"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()
	sess, err := mcpclient.Connect(ctx, req.Server, srv)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": err.Error()})
		return
	}
	defer sess.Close()
	res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: req.Tool, Arguments: req.Arguments})
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, res)
}
