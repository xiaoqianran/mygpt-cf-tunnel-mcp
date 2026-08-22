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
    "version":"0.4.1",
    "description":"Minimal MyGPT Action surface for a persistent MCP Host. The backend supports registered stdio and Streamable HTTP MCP servers, persistent sessions, tool discovery cache, policy filtering, resources and prompts, while MyGPT only receives the three core MCP actions it needs."
  },
  "servers":[{"url":"https://arm-sg-mcp.202820.xyz"}],
  "paths":{
    "/v1/mcp/servers":{
      "get":{
        "operationId":"listMcpServers",
        "summary":"List configured MCP servers",
        "description":"Use this when you need to know which MCP integrations are currently available.",
        "responses":{
          "200":{
            "description":"Configured MCP servers",
            "content":{"application/json":{"schema":{"$ref":"#/components/schemas/ServerListResponse"}}}
          }
        }
      }
    },
    "/v1/mcp/tools/search":{
      "post":{
        "operationId":"searchMcpTools",
        "summary":"Discover MCP tools",
        "description":"Search registered MCP servers for a capability. Use an empty query to enumerate tools for one server. Set refresh=true only when the upstream tool list may have changed.",
        "requestBody":{
          "required":true,
          "content":{"application/json":{"schema":{"$ref":"#/components/schemas/SearchToolsRequest"}}}
        },
        "responses":{
          "200":{
            "description":"Matching MCP tools",
            "content":{"application/json":{"schema":{"$ref":"#/components/schemas/ToolSearchResponse"}}}
          }
        }
      }
    },
    "/v1/mcp/tools/call":{
      "post":{
        "operationId":"callMcpTool",
        "summary":"Call one MCP tool",
        "description":"Call a previously discovered tool on a registered MCP server alias. Arbitrary upstream MCP URLs are not accepted.",
        "x-openai-isConsequential":true,
        "requestBody":{
          "required":true,
          "content":{"application/json":{"schema":{"$ref":"#/components/schemas/CallToolRequest"}}}
        },
        "responses":{
          "200":{"description":"MCP tool result"},
          "502":{"description":"MCP connection, policy or tool error"}
        }
      }
    }
  },
  "components":{
    "schemas":{
      "ServerListResponse":{
        "type":"object",
        "properties":{
          "servers":{
            "type":"array",
            "items":{
              "type":"object",
              "properties":{
                "name":{"type":"string"},
                "description":{"type":"string"},
                "transport":{"type":"string","enum":["stdio","streamable_http"]},
                "connected":{"type":"boolean"},
                "tool_count":{"type":"integer"},
                "last_error":{"type":"string"}
              },
              "required":["name","transport","connected","tool_count"]
            }
          }
        },
        "required":["servers"]
      },
      "SearchToolsRequest":{
        "type":"object",
        "properties":{
          "query":{"type":"string","description":"Keyword or capability to search for. Empty string enumerates tools."},
          "server":{"type":"string","description":"Optional registered MCP server alias"},
          "limit":{"type":"integer","minimum":1,"maximum":50,"default":20},
          "refresh":{"type":"boolean","default":false,"description":"Bypass the gateway tool cache"}
        },
        "required":["query"]
      },
      "ToolSearchResponse":{
        "type":"object",
        "properties":{
          "tools":{
            "type":"array",
            "items":{
              "type":"object",
              "properties":{
                "server":{"type":"string"},
                "name":{"type":"string"},
                "description":{"type":"string"},
                "input_schema":{"type":"object","additionalProperties":true}
              },
              "required":["server","name"]
            }
          }
        },
        "required":["tools"]
      },
      "CallToolRequest":{
        "type":"object",
        "properties":{
          "server":{"type":"string","description":"Registered MCP server alias"},
          "tool":{"type":"string","description":"Exact tool name returned by searchMcpTools"},
          "arguments":{"type":"object","additionalProperties":true,"description":"Arguments matching the tool input_schema"}
        },
        "required":["server","tool"]
      }
    },
    "securitySchemes":{
      "bearerAuth":{"type":"http","scheme":"bearer"}
    }
  },
  "security":[{"bearerAuth":[]}]
}`
