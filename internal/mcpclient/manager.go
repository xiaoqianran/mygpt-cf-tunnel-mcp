package mcpclient

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/config"
)

type ManagedServer struct {
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	Transport       string    `json:"transport"`
	State           string    `json:"state"`
	Connected       bool      `json:"connected"`
	LastError       string    `json:"last_error,omitempty"`
	ToolCount       int       `json:"tool_count"`
	CachedAt        time.Time `json:"cached_at,omitempty"`
	ProjectArgument string    `json:"project_argument,omitempty"`
	ProjectRequired bool      `json:"project_required,omitempty"`
}

type cachedTools struct {
	tools []*mcp.Tool
	at    time.Time
}
type managedSession struct {
	mu           sync.Mutex
	session      *mcp.ClientSession
	cfg          config.Server
	tools        cachedTools
	instructions string
	lastErr      string
}

type Manager struct {
	mu       sync.RWMutex
	servers  map[string]*managedSession
	projects map[string]string
	cacheTTL time.Duration
}

func NewManager(reg config.Registry, ttl time.Duration) *Manager {
	m := &Manager{servers: map[string]*managedSession{}, projects: map[string]string{}, cacheTTL: ttl}
	m.Reload(reg)
	return m
}

func (m *Manager) Reload(reg config.Registry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, cur := range m.servers {
		if next, ok := reg.Servers[name]; !ok || !reflect.DeepEqual(next, cur.cfg) {
			cur.mu.Lock()
			if cur.session != nil {
				_ = cur.session.Close()
			}
			cur.session = nil
			cur.mu.Unlock()
			delete(m.servers, name)
		}
	}
	for name, srv := range reg.Servers {
		if _, ok := m.servers[name]; !ok {
			m.servers[name] = &managedSession{cfg: srv}
		}
	}
	m.projects = make(map[string]string, len(reg.Projects))
	for name, path := range reg.Projects {
		m.projects[name] = path
	}
}

func (m *Manager) get(name string) (*managedSession, error) {
	m.mu.RLock()
	s := m.servers[name]
	m.mu.RUnlock()
	if s == nil {
		return nil, fmt.Errorf("unknown server %q", name)
	}
	return s, nil
}

func (m *Manager) session(ctx context.Context, name string, ms *managedSession) (*mcp.ClientSession, error) {
	if ms.session != nil {
		return ms.session, nil
	}
	s, err := Connect(ctx, name, ms.cfg)
	if err != nil {
		ms.lastErr = err.Error()
		return nil, err
	}
	ms.session = s
	if init := s.InitializeResult(); init != nil {
		ms.instructions = init.Instructions
	}
	ms.lastErr = ""
	return s, nil
}

func (m *Manager) ListTools(ctx context.Context, name string, force bool) ([]*mcp.Tool, error) {
	ms, err := m.get(name)
	if err != nil {
		return nil, err
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if !force && len(ms.tools.tools) > 0 && time.Since(ms.tools.at) < m.cacheTTL {
		return append([]*mcp.Tool(nil), ms.tools.tools...), nil
	}
	sess, err := m.session(ctx, name, ms)
	if err != nil {
		return nil, err
	}
	res, err := sess.ListTools(ctx, nil)
	if err != nil {
		ms.lastErr = err.Error()
		_ = sess.Close()
		ms.session = nil
		sess, err = m.session(ctx, name, ms)
		if err != nil {
			return nil, err
		}
		res, err = sess.ListTools(ctx, nil)
	}
	if err != nil {
		ms.lastErr = err.Error()
		return nil, err
	}
	filtered := make([]*mcp.Tool, 0, len(res.Tools))
	for _, t := range res.Tools {
		if ms.cfg.ToolAllowed(t.Name) {
			filtered = append(filtered, t)
		}
	}
	ms.tools = cachedTools{tools: filtered, at: time.Now()}
	ms.lastErr = ""
	return append([]*mcp.Tool(nil), filtered...), nil
}

func (m *Manager) ProjectNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.projects))
	for name := range m.projects {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m *Manager) ServerInstructions(name string) string {
	ms, err := m.get(name)
	if err != nil {
		return ""
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.instructions
}

func (m *Manager) PrepareToolArguments(server, project string, args map[string]any) (map[string]any, error) {
	ms, err := m.get(server)
	if err != nil {
		return nil, err
	}
	ms.mu.Lock()
	cfg := ms.cfg
	ms.mu.Unlock()
	out := make(map[string]any, len(args)+1)
	for k, v := range args {
		out[k] = v
	}
	if cfg.ProjectArgument == "" {
		if project != "" {
			return nil, fmt.Errorf("server %q does not support project selection", server)
		}
		return out, nil
	}
	if _, ok := out[cfg.ProjectArgument]; ok {
		return nil, fmt.Errorf("use project alias instead of raw %s", cfg.ProjectArgument)
	}
	if project == "" {
		if cfg.RequireProject {
			return nil, fmt.Errorf("project is required for server %q", server)
		}
		return out, nil
	}
	m.mu.RLock()
	path, ok := m.projects[project]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown project %q", project)
	}
	out[cfg.ProjectArgument] = path
	return out, nil
}

func (m *Manager) CallTool(ctx context.Context, server, tool string, args map[string]any) (*mcp.CallToolResult, error) {
	ms, err := m.get(server)
	if err != nil {
		return nil, err
	}
	if !ms.cfg.ToolAllowed(tool) {
		return nil, fmt.Errorf("tool %q is not allowed on server %q", tool, server)
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	sess, err := m.session(ctx, server, ms)
	if err != nil {
		return nil, err
	}
	res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err == nil {
		ms.lastErr = ""
		return res, nil
	}
	ms.lastErr = err.Error()
	_ = sess.Close()
	ms.session = nil
	sess, connErr := m.session(ctx, server, ms)
	if connErr != nil {
		return nil, connErr
	}
	res, err = sess.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		ms.lastErr = err.Error()
		return nil, err
	}
	ms.lastErr = ""
	return res, nil
}

func (m *Manager) Status() []ManagedServer {
	m.mu.RLock()
	names := make([]string, 0, len(m.servers))
	for n := range m.servers {
		names = append(names, n)
	}
	sort.Strings(names)
	m.mu.RUnlock()
	out := make([]ManagedServer, 0, len(names))
	for _, n := range names {
		ms, _ := m.get(n)
		ms.mu.Lock()
		state := "idle"
		if ms.session != nil {
			state = "connected"
		} else if ms.lastErr != "" {
			state = "error"
		}
		v := ManagedServer{Name: n, Description: ms.cfg.Description, Transport: ms.cfg.TransportName(), State: state, Connected: ms.session != nil, LastError: ms.lastErr, ToolCount: len(ms.tools.tools), CachedAt: ms.tools.at, ProjectArgument: ms.cfg.ProjectArgument, ProjectRequired: ms.cfg.RequireProject}
		ms.mu.Unlock()
		out = append(out, v)
	}
	return out
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ms := range m.servers {
		ms.mu.Lock()
		if ms.session != nil {
			_ = ms.session.Close()
		}
		ms.session = nil
		ms.mu.Unlock()
	}
}

func (m *Manager) ListResources(ctx context.Context, server, cursor string) (*mcp.ListResourcesResult, error) {
	ms, err := m.get(server)
	if err != nil {
		return nil, err
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	sess, err := m.session(ctx, server, ms)
	if err != nil {
		return nil, err
	}
	res, err := sess.ListResources(ctx, &mcp.ListResourcesParams{Cursor: cursor})
	if err != nil {
		_ = sess.Close()
		ms.session = nil
		sess, err = m.session(ctx, server, ms)
		if err == nil {
			res, err = sess.ListResources(ctx, &mcp.ListResourcesParams{Cursor: cursor})
		}
	}
	if err != nil {
		ms.lastErr = err.Error()
		return nil, err
	}
	filtered := res.Resources[:0]
	for _, r := range res.Resources {
		if ms.cfg.ResourceAllowed(r.URI) {
			filtered = append(filtered, r)
		}
	}
	res.Resources = filtered
	ms.lastErr = ""
	return res, nil
}

func (m *Manager) ReadResource(ctx context.Context, server, uri string) (*mcp.ReadResourceResult, error) {
	ms, err := m.get(server)
	if err != nil {
		return nil, err
	}
	if !ms.cfg.ResourceAllowed(uri) {
		return nil, fmt.Errorf("resource %q is not allowed on server %q", uri, server)
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	sess, err := m.session(ctx, server, ms)
	if err != nil {
		return nil, err
	}
	res, err := sess.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		_ = sess.Close()
		ms.session = nil
		sess, err = m.session(ctx, server, ms)
		if err == nil {
			res, err = sess.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
		}
	}
	if err != nil {
		ms.lastErr = err.Error()
		return nil, err
	}
	ms.lastErr = ""
	return res, nil
}

func (m *Manager) ListPrompts(ctx context.Context, server, cursor string) (*mcp.ListPromptsResult, error) {
	ms, err := m.get(server)
	if err != nil {
		return nil, err
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	sess, err := m.session(ctx, server, ms)
	if err != nil {
		return nil, err
	}
	res, err := sess.ListPrompts(ctx, &mcp.ListPromptsParams{Cursor: cursor})
	if err != nil {
		_ = sess.Close()
		ms.session = nil
		sess, err = m.session(ctx, server, ms)
		if err == nil {
			res, err = sess.ListPrompts(ctx, &mcp.ListPromptsParams{Cursor: cursor})
		}
	}
	if err != nil {
		ms.lastErr = err.Error()
		return nil, err
	}
	filtered := res.Prompts[:0]
	for _, p := range res.Prompts {
		if ms.cfg.PromptAllowed(p.Name) {
			filtered = append(filtered, p)
		}
	}
	res.Prompts = filtered
	ms.lastErr = ""
	return res, nil
}

func (m *Manager) GetPrompt(ctx context.Context, server, name string, args map[string]string) (*mcp.GetPromptResult, error) {
	ms, err := m.get(server)
	if err != nil {
		return nil, err
	}
	if !ms.cfg.PromptAllowed(name) {
		return nil, fmt.Errorf("prompt %q is not allowed on server %q", name, server)
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	sess, err := m.session(ctx, server, ms)
	if err != nil {
		return nil, err
	}
	res, err := sess.GetPrompt(ctx, &mcp.GetPromptParams{Name: name, Arguments: args})
	if err != nil {
		_ = sess.Close()
		ms.session = nil
		sess, err = m.session(ctx, server, ms)
		if err == nil {
			res, err = sess.GetPrompt(ctx, &mcp.GetPromptParams{Name: name, Arguments: args})
		}
	}
	if err != nil {
		ms.lastErr = err.Error()
		return nil, err
	}
	ms.lastErr = ""
	return res, nil
}
