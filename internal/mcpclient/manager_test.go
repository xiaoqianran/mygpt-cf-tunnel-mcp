package mcpclient

import (
	"testing"
	"time"

	"github.com/xiaoqianran/mygpt-cf-tunnel-mcp/internal/config"
)

func projectManager() *Manager {
	return NewManager(config.Registry{
		Projects: map[string]string{"embodiedgen": "/srv/EmbodiedGen"},
		Servers: map[string]config.Server{
			"codegraph": {Transport: "stdio", Command: "codegraph", ProjectArgument: "projectPath", RequireProject: true},
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

func TestServerInstructions(t *testing.T) {
	m := projectManager()
	ms, _ := m.get("docs")
	ms.instructions = "优先查询文档，再调用工具。"
	if got := m.ServerInstructions("docs"); got != ms.instructions {
		t.Fatalf("got %q", got)
	}
}
