package agent

import "net/http"

func (s *Server) openAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(openAPISchema))
}

const openAPISchema = `{
  "openapi":"3.1.0",
  "info":{"title":"MyGPT MCP Gateway","version":"0.1.0","description":"Connect a Custom GPT Action to registered remote MCP servers."},
  "servers":[{"url":"https://mcp-agent.example.com"}],
  "paths":{
    "/v1/mcp/servers":{"get":{"operationId":"listMcpServers","summary":"List configured MCP servers","responses":{"200":{"description":"Configured server aliases"}}}},
    "/v1/mcp/tools/search":{"post":{"operationId":"searchMcpTools","summary":"Discover MCP tools by keyword","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{"query":{"type":"string","description":"Keyword or capability to search for"},"server":{"type":"string","description":"Optional registered MCP server alias"},"limit":{"type":"integer","minimum":1,"maximum":50}},"required":["query"]}}}},"responses":{"200":{"description":"Matching MCP tools"}}}},
    "/v1/mcp/tools/call":{"post":{"operationId":"callMcpTool","summary":"Call one tool on a registered MCP server","x-openai-isConsequential":true,"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{"server":{"type":"string"},"tool":{"type":"string"},"arguments":{"type":"object","additionalProperties":true}},"required":["server","tool"]}}}},"responses":{"200":{"description":"MCP tool result"}}}}
  },
  "components":{"securitySchemes":{"bearerAuth":{"type":"http","scheme":"bearer"}}},
  "security":[{"bearerAuth":[]}]
}`
