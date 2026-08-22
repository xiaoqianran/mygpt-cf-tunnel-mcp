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
    "version":"0.6.0",
    "description":"面向受信任 MyGPT 的精简 MCP Host。保留上游 MCP 的 schema、instructions 与 annotations；统一由一个通用调用入口执行已注册工具。"
  },
  "servers":[{"url":"https://arm-sg-mcp.202820.xyz"}],
  "paths":{
    "/v1/mcp/servers":{
      "get":{
        "operationId":"listMcpServers",
        "summary":"列出 MCP 服务和项目别名",
        "description":"发现已注册 MCP 集成和项目别名。state=idle 表示惰性连接尚未建立，并非故障。",
        "responses":{"200":{"description":"MCP 服务和项目别名","content":{"application/json":{"schema":{"$ref":"#/components/schemas/ServerListResponse"}}}}}
      }
    },
    "/v1/mcp/tools/search":{
      "post":{
        "operationId":"searchMcpTools",
        "summary":"发现 MCP 工具及其使用语义",
        "description":"按能力搜索工具。空 query 可枚举指定 server。返回 input/output schema、annotations、server instructions 和项目要求；annotations 仅用于理解工具语义，不作为权限分流。",
        "requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/SearchToolsRequest"}}}},
        "responses":{"200":{"description":"匹配的 MCP 工具","content":{"application/json":{"schema":{"$ref":"#/components/schemas/ToolSearchResponse"}}}}}
      }
    },
    "/v1/mcp/tools/call":{
      "post":{
        "operationId":"callMcpTool",
        "summary":"调用 MCP 工具",
        "description":"统一调用 searchMcpTools 发现的已注册 MCP 工具。项目型 server 使用 project 别名；不要在 arguments_json 中直接传项目绝对路径。",
        "x-openai-isConsequential":false,
        "requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/CallToolRequest"}}}},
        "responses":{"200":{"description":"MCP 工具结果"},"400":{"description":"参数或项目错误"},"502":{"description":"MCP 连接、策略或工具错误"}}
      }
    }
  },
  "components":{
    "schemas":{
      "ServerListResponse":{
        "type":"object",
        "properties":{
          "servers":{"type":"array","items":{"$ref":"#/components/schemas/McpServer"}},
          "projects":{"type":"array","items":{"type":"string"},"description":"项目型 MCP server 可使用的项目别名。"}
        },
        "required":["servers","projects"]
      },
      "McpServer":{
        "type":"object",
        "properties":{
          "name":{"type":"string"},
          "description":{"type":"string"},
          "transport":{"type":"string","enum":["stdio","streamable_http"]},
          "state":{"type":"string","enum":["idle","connected","error"]},
          "connected":{"type":"boolean"},
          "tool_count":{"type":"integer"},
          "last_error":{"type":"string"},
          "project_argument":{"type":"string","description":"上游工具接收项目路径的参数名；调用时仍应使用 project 别名。"},
          "project_required":{"type":"boolean"}
        },
        "required":["name","transport","state","connected","tool_count"]
      },
      "SearchToolsRequest":{
        "type":"object",
        "properties":{
          "query":{"type":"string","description":"关键词或能力；空字符串表示枚举工具。"},
          "server":{"type":"string","description":"可选的已注册 MCP server 别名。"},
          "limit":{"type":"integer","minimum":1,"maximum":50,"default":20},
          "refresh":{"type":"boolean","default":false,"description":"仅在上游工具列表可能变化时绕过缓存。"}
        },
        "required":["query"]
      },
      "ToolSearchResponse":{
        "type":"object",
        "properties":{
          "tools":{"type":"array","items":{"$ref":"#/components/schemas/McpTool"}},
          "server_instructions":{"type":"array","items":{"$ref":"#/components/schemas/ServerInstructions"}},
          "projects":{"type":"array","items":{"type":"string"},"description":"可用于 project_required=true 工具的项目别名。"}
        },
        "required":["tools","server_instructions","projects"]
      },
      "McpTool":{
        "type":"object",
        "properties":{
          "server":{"type":"string"},
          "name":{"type":"string"},
          "title":{"type":"string"},
          "description":{"type":"string"},
          "input_schema":{"type":"object","additionalProperties":true},
          "output_schema":{"type":"object","additionalProperties":true},
          "annotations":{"$ref":"#/components/schemas/ToolAnnotations"},
          "project_required":{"type":"boolean","description":"为 true 时调用必须传 project 别名。"}
        },
        "required":["server","name"]
      },
      "ToolAnnotations":{
        "type":"object",
        "properties":{
          "title":{"type":"string"},
          "readOnlyHint":{"type":"boolean"},
          "destructiveHint":{"type":"boolean"},
          "idempotentHint":{"type":"boolean"},
          "openWorldHint":{"type":"boolean"}
        }
      },
      "ServerInstructions":{
        "type":"object",
        "properties":{"server":{"type":"string"},"instructions":{"type":"string"}},
        "required":["server","instructions"]
      },
      "CallToolRequest":{
        "type":"object",
        "properties":{
          "server":{"type":"string","description":"已注册 MCP server 别名。"},
          "tool":{"type":"string","description":"searchMcpTools 返回的精确工具名。"},
          "project":{"type":"string","description":"项目别名。project_required=true 时必填；由网关映射到真实路径。"},
          "arguments_json":{"type":"string","default":"{}","description":"按 input_schema 构造并编码成字符串的 JSON 对象。不要直接传 project_argument 对应的原始路径。"}
        },
        "required":["server","tool"]
      }
    },
    "securitySchemes":{"bearerAuth":{"type":"http","scheme":"bearer"}}
  },
  "security":[{"bearerAuth":[]}]
}`
