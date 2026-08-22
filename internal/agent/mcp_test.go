package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCallRequestToolArguments(t *testing.T) {
	t.Run("arguments json", func(t *testing.T) {
		r := callRequest{ArgumentsJSON: `{"query":"manager call path","maxFiles":8}`}
		args, err := r.toolArguments()
		if err != nil {
			t.Fatal(err)
		}
		if args["query"] != "manager call path" || args["maxFiles"] != float64(8) {
			t.Fatalf("unexpected arguments: %#v", args)
		}
	})

	t.Run("legacy object", func(t *testing.T) {
		r := callRequest{Arguments: map[string]any{"query": "legacy"}}
		args, err := r.toolArguments()
		if err != nil || args["query"] != "legacy" {
			t.Fatalf("args=%#v err=%v", args, err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := (callRequest{ArgumentsJSON: `[]`}).toolArguments()
		if err == nil {
			t.Fatal("expected JSON object error")
		}
	})
}

func TestLimitToolResult(t *testing.T) {
	res := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: strings.Repeat("项目源码", 3000)}}}
	got := limitToolResult(res, 4096)
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) > 4096 {
		t.Fatalf("limited result still too large: %d", len(b))
	}
	text := got.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "网关已截断") {
		t.Fatalf("missing truncation marker: %q", text[len(text)-80:])
	}
}

func TestOpenAPISchema(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(openAPISchema), &doc); err != nil {
		t.Fatal(err)
	}
	paths := doc["paths"].(map[string]any)
	if len(paths) != 3 {
		t.Fatalf("expected 3 MyGPT actions, got %d", len(paths))
	}
	if paths["/v1/mcp/tools/call-readonly"] != nil {
		t.Fatal("read-only call path must not be exposed")
	}
	callPath := paths["/v1/mcp/tools/call"].(map[string]any)
	post := callPath["post"].(map[string]any)
	if consequential, ok := post["x-openai-isConsequential"].(bool); !ok || consequential {
		t.Fatal("callMcpTool should use the trusted full-access action model")
	}
	components := doc["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	call := schemas["CallToolRequest"].(map[string]any)
	props := call["properties"].(map[string]any)
	for _, key := range []string{"server", "tool", "project", "arguments_json"} {
		if props[key] == nil {
			t.Fatalf("missing CallToolRequest property %q", key)
		}
	}
}
