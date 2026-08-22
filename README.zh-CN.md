# mygpt-cf-tunnel-mcp

给 ChatGPT Custom GPT / MyGPT 使用的**独立 MCP Host**。

它与 [`mygpt-cf-tunnel`](https://github.com/xiaoqianran/mygpt-cf-tunnel) 分工明确：

- `mygpt-cf-tunnel`：Command Action，负责 VPS Shell / 系统执行。
- `mygpt-cf-tunnel-mcp`：MCP Action，负责托管、连接、发现和调用 MCP。

## 架构

```text
Custom GPT
  ├─ Command Action → mygpt-cf-tunnel → /bin/bash -lc
  └─ MCP Action     → mygpt-cf-tunnel-mcp :8788
                         │
                         ├─ stdio MCP
                         │    └─ codegraph serve --mcp
                         │
                         └─ Streamable HTTP MCP
                              ├─ Context7
                              ├─ DeepWiki
                              └─ 其他远程 MCP
```

本项目相当于给 MyGPT 补上 CodeBuddy / Grok / Claude Code 中的 MCP Host 层：保存 MCP 配置、启动 stdio MCP 子进程、建立远程 MCP 连接，并把 MCP 能力转换成 MyGPT 可调用的 Action。

## v0.4 能力

### Transport

- stdio MCP：`command + args + env + working_dir`
- Streamable HTTP MCP：`url + Bearer token env`
- stdio session 持久复用，不再每次 Action 都重新 spawn 子进程
- HTTP MCP session 复用，失败自动重连

### MCP primitives

Tools：

- `listMcpServers`
- `getMcpStatus`
- `searchMcpTools`
- `callMcpTool`

Resources：

- `listMcpResources`
- `readMcpResource`

Prompts：

- `listMcpPrompts`
- `getMcpPrompt`

### Tool cache

`tools/list` 结果由 Gateway 缓存，默认 TTL：

```env
TOOL_CACHE_TTL=5m
```

`searchMcpTools` 支持 `refresh=true` 主动绕过 Gateway cache。

### 自动重连

Tool / Resource / Prompt 调用出现连接错误时，Manager 会关闭失效 session、重新 initialize，再重试一次。

### Policy

每个 MCP Server 可以设置：

```json
{
  "allow_tools": ["read_*", "query-docs"],
  "deny_tools": ["delete_*"],
  "allow_resources": ["docs://*"],
  "deny_resources": ["secret://*"],
  "allow_prompts": ["review-*"],
  "deny_prompts": ["admin-*"]
}
```

规则：deny 优先；allow 为空时默认允许；支持精确名称、`*` 和前缀通配符 `xxx*`。

GPT 不能传任意上游 MCP URL，只能使用 Registry 中已登记的 server alias。

## Registry

默认：

```text
/etc/mygpt-mcp/servers.json
```

远程 MCP：

```json
{
  "servers": {
    "context7": {
      "transport": "streamable_http",
      "url": "https://mcp.context7.com/mcp",
      "description": "Context7"
    }
  }
}
```

stdio MCP：

```json
{
  "servers": {
    "codegraph": {
      "transport": "stdio",
      "command": "/usr/local/bin/codegraph",
      "args": ["serve", "--mcp"],
      "working_dir": "/srv/project",
      "env": {
        "EXAMPLE": "value"
      }
    }
  }
}
```

对于通过 fnm/nvm/uv 等安装、systemd PATH 中不可见的命令，推荐使用绝对路径，或者在该 MCP 的 `env.PATH` 中显式加入运行时目录。

## `mygpt-mcpctl`

v0.4 增加 MCP 管理 CLI，不再必须手改 JSON。

```bash
mygpt-mcpctl list
mygpt-mcpctl validate

mygpt-mcpctl add-http context7 https://mcp.context7.com/mcp

mygpt-mcpctl add-stdio codegraph \
  /usr/local/bin/codegraph serve --mcp

mygpt-mcpctl disable context7
mygpt-mcpctl enable context7
mygpt-mcpctl remove context7
```

Registry 修改会在下一次 MCP Action 请求时自动同步。配置变化的 server 会关闭旧 session 并按新配置重新连接，不需要重启整个 Gateway。

## 当前部署

```text
Listen:   127.0.0.1:8788
Public:   https://arm-sg-mcp.202820.xyz
OpenAPI:  https://arm-sg-mcp.202820.xyz/openapi.json
```

Cloudflare Tunnel 只负责将公网 HTTPS 转发到本机 8788。

MyGPT Action 使用 Bearer Authentication，`API_TOKEN` 从 `/etc/mygpt-mcp/agent.env` 读取。

## 安全边界

- MCP Gateway 没有 Shell / `runCommand` API。
- Command Action 和 MCP Action 使用独立服务、独立端口、独立认证。
- GPT 不能动态指定 MCP URL，避免将 Gateway 变成 SSRF Proxy。
- 上游 MCP token 通过 `token_env` 从 VPS 环境变量读取，不进入 GPT 参数。
- MCP Tool / Resource / Prompt 可设置 allow / deny policy。
- systemd 使用 `NoNewPrivileges=true`、`ProtectSystem=full`、`PrivateTmp=true`。

## 已联调

当前正式 Registry：

- CodeGraph：stdio MCP
- Context7：Streamable HTTP MCP
- DeepWiki：Streamable HTTP MCP

另外使用官方 `@modelcontextprotocol/server-everything` 完成了 MCP Tools、Resources、Prompts 的端到端协议测试。

## 构建

```bash
go test ./...
go vet ./...
go build -o mygpt-mcp-agent ./cmd/mygpt-mcp-agent
go build -o mygpt-mcpctl ./cmd/mygpt-mcpctl
```

安装：

```bash
sudo ./scripts/install.sh
```
