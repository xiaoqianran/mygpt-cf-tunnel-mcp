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
    "version":"0.5.0",
    "description":"面向 MyGPT 的精简 MCP Host。保留上游 MCP 的工具 schema、instructions 与 annotations，并由网关统一执行项目路径和只读策略。"
  },
  "servers":[{"url":"https://arm-sg-mcp.202820.xyz"}],
  "paths":{
    "/v1/mcp/servers":{
      "get":{
        "operationId":"listMcpServers",
        "summary":"列出 MCP 服务和项目别名",
        "description":"需要了解可用 MCP 集成或项目别名时调用。state=idle 表示已配置但尚未建立惰性连接，并不代表故障。",
        "responses":{"200":{"description":"MCP 服务和项目别名","content":{"application/json":{"schema":{"$ref":"#/components/schemas/ServerListResponse"}}}}}
      }
    },
    "/v1/mcp/tools/search":{
      "post":{
        "operationId":"searchMcpTools",
        "summary":"发现 MCP 工具及其使用语义",
        "description":"按能力搜索 MCP 工具。空 query 可枚举指定 server 的工具。返回 input_schema、annotations 和 server_instructions；调用前应遵循这些信息。",
        "requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/SearchToolsRequest"}}}},
        "responses":{"200":{"description":"匹配的 MCP 工具","content":{"application/json":{"schema":{"$ref":"#/components/schemas/ToolSearchResponse"}}}}}
      }
    },
    "/v1/mcp/tools/call-readonly":{
      "post":{
        "operationId":"callReadOnlyMcpTool",
        "summary":"调用只读 MCP 工具",
        "description":"仅用于 searchMcpTools 返回 annotations.readOnlyHint=true 且 annotations_trusted=true 的工具。网关会再次刷新并校验；其他工具会被拒绝。项目型 server 应传 project 别名，不要在 arguments_json 中直接传项目路径。",
        "x-openai-isConsequential":false,
        "requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/CallToolRequest"}}}},
        "responses":{"200":{"description":"MCP 工具结果"},"400":{"description":"参数或项目错误"},"502":{"description":"MCP 连接、策略或工具错误"}}
      }
    },
    "/v1/mcp/tools/call":{
      "post":{
        "operationId":"callMcpTool",
        "summary":"调用可能产生副作用的 MCP 工具",
        "description":"调用已发现的 MCP 工具。只读工具优先使用 callReadOnlyMcpTool；此入口保留给未声明只读或可能修改外部状态的工具。",
        "x-openai-isConsequential":true,
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
          "projects":{"type":"array","items":{"type":"string"},"description":"允许用于项目型 MCP server 的项目别名；不暴露服务器绝对路径。"}
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
          "project_argument":{"type":"string","description":"上游工具用于接收项目路径的参数名；仅用于说明，调用时请传 project 别名。"},
          "project_required":{"type":"boolean"},
          "annotations_trusted":{"type":"boolean","description":"管理员是否允许此 server 的 annotations 驱动只读免确认策略。"}
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
          "project_required":{"type":"boolean","description":"为 true 时调用必须传 project 别名。"},
          "annotations_trusted":{"type":"boolean","description":"为 true 且 readOnlyHint=true 时才可使用 callReadOnlyMcpTool。"}
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
          "project":{"type":"string","description":"项目别名。project_required=true 时必填；由网关安全映射到真实路径。"},
          "arguments_json":{"type":"string","default":"{}","description":"按 input_schema 构造并编码成字符串的 JSON 对象。不要直接传 project_argument 对应的原始路径。"}
        },
        "required":["server","tool"]
      }
    },
    "securitySchemes":{"bearerAuth":{"type":"http","scheme":"bearer"}}
  },
  "security":[{"bearerAuth":[]}]
}`
