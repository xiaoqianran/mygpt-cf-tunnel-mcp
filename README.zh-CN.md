# mygpt-cf-tunnel-mcp

这是一个给 ChatGPT Custom GPT 使用的**独立 MCP Action Gateway**。

它与 [`mygpt-cf-tunnel`](https://github.com/xiaoqianran/mygpt-cf-tunnel) 完全分离：原项目只负责远程命令执行；本项目只负责 MCP 工具发现与调用。

## 架构

```text
Custom GPT
  ├─ Action 1 → command.example.com → mygpt-cf-tunnel → /bin/bash -lc
  └─ Action 2 → mcp.example.com     → mygpt-cf-tunnel-mcp
                                      ├─ listMcpServers
                                      ├─ searchMcpTools
                                      └─ callMcpTool
                                             ↓
                                  已注册的远程 MCP Server
                                  Streamable HTTP /mcp
```

核心安全边界：GPT **不能传任意 MCP URL**。它只能传服务端预先注册的 `server` 别名、工具名和参数。真正的 MCP URL 与凭据只保存在 VPS。

## 为什么独立仓库

- 独立 Action Schema
- 独立域名
- 独立 Bearer Token
- 独立部署、升级、回滚与日志
- MCP Gateway 内完全没有 shell 执行入口
- MCP 凭据与 root command gateway 凭据彻底隔离

## MyGPT 暴露的三个操作

- `listMcpServers`：列出允许访问的 MCP 服务别名
- `searchMcpTools`：按能力/关键词寻找工具，避免一次性把大量 schema 塞进上下文
- `callMcpTool`：调用具体 MCP Tool

`callMcpTool` 在 OpenAPI 中标记为 consequential，因为上游 MCP Tool 可能产生写操作。

## v0.1 范围

v0.2 同时支持 **本地 stdio MCP** 与 **远程 Streamable HTTP MCP**，使用官方 `modelcontextprotocol/go-sdk`。stdio 配置会像 CodeBuddy、Grok、Claude Code 一样启动子进程并通过 stdin/stdout 与 MCP Server 通信。

## Registry

`/etc/mygpt-mcp/servers.json`：

```json
{
  "servers": {
    "example": {
      "url": "https://mcp.example.com/mcp",
      "description": "Example MCP",
      "token_env": "EXAMPLE_MCP_TOKEN"
    }
  }
}
```

上游 token 放在环境变量，不会经过 GPT。

## Cloudflare Tunnel

建议和 command gateway 使用不同 hostname：

```yaml
ingress:
  - hostname: command.example.com
    service: http://127.0.0.1:8787
  - hostname: mcp.example.com
    service: http://127.0.0.1:8788
  - service: http_status:404
```

然后把：

```text
https://mcp.example.com/openapi.json
```

作为 MyGPT 的第二个 Action 导入，并单独配置 Bearer `API_TOKEN`。

## 后续重点

- Tool schema TTL 缓存与索引
- OAuth 上游 MCP
- server/tool allowlist 与 denylist
- 输出大小限制和结构化截断
- audit id / 调用日志
- 可选 stdio transport
- 上游健康检查
