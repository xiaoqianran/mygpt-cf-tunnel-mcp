package agent

import "net/http"

func (s *Server) openAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(openAPISchema))
}

const openAPISchema = `{
  "openapi":"3.1.0",
  "info":{
    "title":"MyGPT MCP Host",
    "version":"0.4.0",
    "description":"A standalone MCP Host for ChatGPT Custom GPT Actions. Connects registered local stdio and remote Streamable HTTP MCP servers, reuses sessions, caches tool discovery, enforces allow/deny policy, and exposes MCP tools, resources and prompts."
  },
  "servers":[{"url":"https://arm-sg-mcp.202820.xyz"}],
  "paths":{
    "/v1/mcp/servers":{"get":{"operationId":"listMcpServers","summary":"List configured MCP servers","description":"Use this first when you do not know which MCP integrations are available.","responses":{"200":{"description":"Configured MCP servers","content":{"application/json":{"schema":{"$ref":"#/components/schemas/ServerListResponse"}}}}}}},
    "/v1/mcp/status":{"get":{"operationId":"getMcpStatus","summary":"Inspect MCP runtime status","description":"Shows transport, persistent connection state, cached tool count and recent errors.","responses":{"200":{"description":"MCP runtime status"}}}},
    "/v1/mcp/tools/search":{"post":{"operationId":"searchMcpTools","summary":"Discover MCP tools","description":"Search tool names and descriptions across registered MCP servers. Set refresh=true only when an upstream server's tool list may have changed.","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/SearchToolsRequest"}}}},"responses":{"200":{"description":"Matching MCP tools","content":{"application/json":{"schema":{"$ref":"#/components/schemas/ToolSearchResponse"}}}}}}},
    "/v1/mcp/tools/call":{"post":{"operationId":"callMcpTool","summary":"Call an MCP tool","description":"Call a previously discovered tool on a registered server alias. The gateway does not accept arbitrary upstream URLs.","x-openai-isConsequential":true,"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/CallToolRequest"}}}},"responses":{"200":{"description":"MCP tool result"},"502":{"description":"MCP connection, policy or tool error"}}}},
    "/v1/mcp/resources/list":{"post":{"operationId":"listMcpResources","summary":"List MCP resources","description":"List resources exposed by a registered MCP server.","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/ListResourcesRequest"}}}},"responses":{"200":{"description":"MCP resources"}}}},
    "/v1/mcp/resources/read":{"post":{"operationId":"readMcpResource","summary":"Read an MCP resource","description":"Read a resource URI previously returned by listMcpResources.","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/ReadResourceRequest"}}}},"responses":{"200":{"description":"MCP resource contents"}}}},
    "/v1/mcp/prompts/list":{"post":{"operationId":"listMcpPrompts","summary":"List MCP prompts","description":"List prompt templates exposed by a registered MCP server.","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/ListPromptsRequest"}}}},"responses":{"200":{"description":"MCP prompts"}}}},
    "/v1/mcp/prompts/get":{"post":{"operationId":"getMcpPrompt","summary":"Get an MCP prompt","description":"Render a prompt template with string arguments.","requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/GetPromptRequest"}}}},"responses":{"200":{"description":"Rendered MCP prompt"}}}}
  },
  "components":{
    "schemas":{
      "ServerListResponse":{"type":"object","properties":{"servers":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"description":{"type":"string"},"transport":{"type":"string","enum":["stdio","streamable_http"]},"connected":{"type":"boolean"},"tool_count":{"type":"integer"},"last_error":{"type":"string"}},"required":["name","transport","connected","tool_count"]}}},"required":["servers"]},
      "SearchToolsRequest":{"type":"object","properties":{"query":{"type":"string","description":"Keyword/capability. Empty string enumerates tools."},"server":{"type":"string","description":"Optional registered server alias"},"limit":{"type":"integer","minimum":1,"maximum":50,"default":20},"refresh":{"type":"boolean","default":false,"description":"Bypass gateway tool cache"}},"required":["query"]},
      "ToolSearchResponse":{"type":"object","properties":{"tools":{"type":"array","items":{"type":"object","properties":{"server":{"type":"string"},"name":{"type":"string"},"description":{"type":"string"},"input_schema":{"type":"object","additionalProperties":true}},"required":["server","name"]}}},"required":["tools"]},
      "CallToolRequest":{"type":"object","properties":{"server":{"type":"string","description":"Registered server alias"},"tool":{"type":"string","description":"Exact discovered tool name"},"arguments":{"type":"object","additionalProperties":true}},"required":["server","tool"]},
      "ListResourcesRequest":{"type":"object","properties":{"server":{"type":"string"},"cursor":{"type":"string"}},"required":["server"]},
      "ReadResourceRequest":{"type":"object","properties":{"server":{"type":"string"},"uri":{"type":"string"}},"required":["server","uri"]},
      "ListPromptsRequest":{"type":"object","properties":{"server":{"type":"string"},"cursor":{"type":"string"}},"required":["server"]},
      "GetPromptRequest":{"type":"object","properties":{"server":{"type":"string"},"prompt":{"type":"string"},"arguments":{"type":"object","additionalProperties":{"type":"string"}}},"required":["server","prompt"]}
    },
    "securitySchemes":{"bearerAuth":{"type":"http","scheme":"bearer"}}
  },
  "security":[{"bearerAuth":[]}]
}`
