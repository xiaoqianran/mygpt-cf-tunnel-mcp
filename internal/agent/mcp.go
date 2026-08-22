package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/config"
)

type serverView struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Transport   string `json:"transport"`
	Connected   bool   `json:"connected"`
	ToolCount   int    `json:"tool_count"`
	LastError   string `json:"last_error,omitempty"`
}
type searchRequest struct {
	Query   string `json:"query"`
	Server  string `json:"server,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Refresh bool   `json:"refresh,omitempty"`
}
type toolView struct {
	Server      string `json:"server"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}
type callRequest struct {
	Server        string         `json:"server"`
	Tool          string         `json:"tool"`
	Arguments     map[string]any `json:"arguments,omitempty"`
	ArgumentsJSON string         `json:"arguments_json,omitempty"`
}

func (r callRequest) toolArguments() (map[string]any, error) {
	if strings.TrimSpace(r.ArgumentsJSON) == "" {
		return r.Arguments, nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(r.ArgumentsJSON), &args); err != nil {
		return nil, err
	}
	return args, nil
}

func (s *Server) syncRegistry() error {
	reg, err := config.LoadRegistry(s.cfg.RegistryPath)
	if err != nil {
		return err
	}
	s.mcp.Reload(reg)
	return nil
}

func (s *Server) reloadRegistry(w http.ResponseWriter, r *http.Request) {
	reg, err := config.LoadRegistry(s.cfg.RegistryPath)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	s.mcp.Reload(reg)
	writeJSON(w, 200, map[string]any{"ok": true, "servers": len(reg.Servers)})
}

func (s *Server) listServers(w http.ResponseWriter, r *http.Request) {
	if err := s.syncRegistry(); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	statuses := s.mcp.Status()
	out := make([]serverView, 0, len(statuses))
	for _, v := range statuses {
		out = append(out, serverView{Name: v.Name, Description: v.Description, Transport: v.Transport, Connected: v.Connected, ToolCount: v.ToolCount, LastError: v.LastError})
	}
	writeJSON(w, 200, map[string]any{"servers": out})
}
func (s *Server) serverStatus(w http.ResponseWriter, r *http.Request) {
	if err := s.syncRegistry(); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"servers": s.mcp.Status()})
}
func (s *Server) searchTools(w http.ResponseWriter, r *http.Request) {
	if err := s.syncRegistry(); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]any{"error": "invalid JSON"})
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 20
	}
	statuses := s.mcp.Status()
	names := make([]string, 0, len(statuses))
	if req.Server != "" {
		found := false
		for _, v := range statuses {
			if v.Name == req.Server {
				found = true
				break
			}
		}
		if !found {
			writeJSON(w, 404, map[string]any{"error": "unknown server"})
			return
		}
		names = []string{req.Server}
	} else {
		for _, v := range statuses {
			names = append(names, v.Name)
		}
		sort.Strings(names)
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()
	matches := []toolView{}
	q := strings.ToLower(req.Query)
	for _, name := range names {
		tools, e := s.mcp.ListTools(ctx, name, req.Refresh)
		if e != nil {
			s.log.Warn("mcp list failed", "server", name, "error", e)
			continue
		}
		for _, t := range tools {
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
	if err := s.syncRegistry(); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	var req callRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Server == "" || req.Tool == "" {
		writeJSON(w, 400, map[string]any{"error": "server and tool are required"})
		return
	}
	args, err := req.toolArguments()
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": "arguments_json must be a JSON object"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()
	res, err := s.mcp.CallTool(ctx, req.Server, req.Tool, args)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, res)
}

type listResourceRequest struct {
	Server string `json:"server"`
	Cursor string `json:"cursor,omitempty"`
}
type readResourceRequest struct {
	Server string `json:"server"`
	URI    string `json:"uri"`
}
type listPromptRequest struct {
	Server string `json:"server"`
	Cursor string `json:"cursor,omitempty"`
}
type getPromptRequest struct {
	Server    string            `json:"server"`
	Prompt    string            `json:"prompt"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

func (s *Server) listResources(w http.ResponseWriter, r *http.Request) {
	if err := s.syncRegistry(); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	var req listResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Server == "" {
		writeJSON(w, 400, map[string]any{"error": "server is required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()
	res, err := s.mcp.ListResources(ctx, req.Server, req.Cursor)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, res)
}
func (s *Server) readResource(w http.ResponseWriter, r *http.Request) {
	if err := s.syncRegistry(); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	var req readResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Server == "" || req.URI == "" {
		writeJSON(w, 400, map[string]any{"error": "server and uri are required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()
	res, err := s.mcp.ReadResource(ctx, req.Server, req.URI)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, res)
}
func (s *Server) listPrompts(w http.ResponseWriter, r *http.Request) {
	if err := s.syncRegistry(); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	var req listPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Server == "" {
		writeJSON(w, 400, map[string]any{"error": "server is required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()
	res, err := s.mcp.ListPrompts(ctx, req.Server, req.Cursor)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, res)
}
func (s *Server) getPrompt(w http.ResponseWriter, r *http.Request) {
	if err := s.syncRegistry(); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	var req getPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Server == "" || req.Prompt == "" {
		writeJSON(w, 400, map[string]any{"error": "server and prompt are required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()
	res, err := s.mcp.GetPrompt(ctx, req.Server, req.Prompt, req.Arguments)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, res)
}
