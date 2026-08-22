package mcpclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/config"
)

func projectManager() *Manager {
	return NewManager(config.Registry{
		Projects: map[string]string{"embodiedgen": "/srv/EmbodiedGen"},
		Servers: map[string]config.Server{
			"codegraph": {Transport: "stdio", Command: "codegraph", ProjectArgument: "projectPath", RequireProject: true, TrustAnnotations: true},
			"docs":      {Transport: "stdio", Command: "docs"},
		},
	}, time.Minute)
}

func TestPrepareToolArguments(t *testing.T) {
	m := projectManager()
	args, err := m.PrepareToolArguments("codegraph", "embodiedgen", map[string]any{"query": "entrypoint"})
	if err != nil {
		t.Fatal(err)
	}
	if args["projectPath"] != "/srv/EmbodiedGen" || args["query"] != "entrypoint" {
		t.Fatalf("unexpected args: %#v", args)
	}
	if _, err := m.PrepareToolArguments("codegraph", "", map[string]any{"query": "x"}); err == nil {
		t.Fatal("expected missing project error")
	}
	if _, err := m.PrepareToolArguments("codegraph", "embodiedgen", map[string]any{"projectPath": "/tmp/escape"}); err == nil {
		t.Fatal("expected raw project path rejection")
	}
	if _, err := m.PrepareToolArguments("codegraph", "missing", nil); err == nil {
		t.Fatal("expected unknown project error")
	}
	if _, err := m.PrepareToolArguments("docs", "embodiedgen", nil); err == nil {
		t.Fatal("expected unsupported project error")
	}
}

func TestToolReadOnly(t *testing.T) {
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true}
	tools := []*mcp.Tool{{Name: "read", Annotations: readOnly}, {Name: "write"}}
	if found, ok := toolReadOnly(tools, "read"); !found || !ok {
		t.Fatal("read tool should be recognized as read-only")
	}
	if found, ok := toolReadOnly(tools, "write"); !found || ok {
		t.Fatal("unannotated tool must not be treated as read-only")
	}
	if found, _ := toolReadOnly(tools, "missing"); found {
		t.Fatal("missing tool reported as found")
	}
}

func TestReadOnlyToolRequiresTrustedAnnotations(t *testing.T) {
	m := projectManager()
	_, err := m.CallReadOnlyTool(context.Background(), "docs", "read", nil)
	if err == nil || !strings.Contains(err.Error(), "not trusted") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServerInstructions(t *testing.T) {
	m := projectManager()
	ms, _ := m.get("docs")
	ms.instructions = "优先查询文档，再调用工具。"
	if got := m.ServerInstructions("docs"); got != ms.instructions {
		t.Fatalf("got %q", got)
	}
}
