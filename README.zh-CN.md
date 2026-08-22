# mygpt-cf-tunnel-mcp

给 ChatGPT Custom GPT / MyGPT 使用的**精简 MCP Host**。

它与 [`mygpt-cf-tunnel`](https://github.com/xiaoqianran/mygpt-cf-tunnel) 分工明确：

- `mygpt-cf-tunnel`：负责 VPS Shell、文件、Git、安装、构建、部署和调试。
- `mygpt-cf-tunnel-mcp`：负责 MCP 注册、连接、发现、语义保真和调用策略。

## 架构

```text
MyGPT
  ├─ Command Action ──→ mygpt-cf-tunnel ──→ Linux / Git / Build / Service
  │
  └─ MCP Action ──────→ mygpt-cf-tunnel-mcp
                           │
                           ├─ Registry / Project identity
                           ├─ Session / reconnect / cache
                           ├─ Instructions / annotations
                           ├─ Policy / response boundary
                           │
                           ├─ CodeGraph
                           ├─ Context7
                           ├─ DeepWiki
                           └─ 其他 MCP server
```

Gateway 保持很薄：它不理解 AST、类、调用图，也不替上游 MCP 做领域规划；它只保留 MCP 原本提供给 Agent 的语义，并执行确定性的安全策略。

## v0.5 核心能力

### MyGPT 只暴露四个 Action

- `listMcpServers`：发现已注册 server 和允许使用的项目别名。
- `searchMcpTools`：发现工具，并返回 `input_schema`、`output_schema`、`annotations` 和 `server_instructions`。
- `callReadOnlyMcpTool`：只调用受信任 server 且明确声明 `readOnlyHint=true` 的工具，网关会再次刷新并校验。
- `callMcpTool`：调用未声明只读或可能产生副作用的工具，保留确认边界。

Resources、Prompts、Status、Reload 等 MCP 能力仍保留在后端 HTTP API，不额外占用 MyGPT 的 Action 工具面。

### 动态参数

MyGPT 不直接发送自由形态 `arguments` 对象，而是按照 `searchMcpTools` 返回的 `input_schema` 构造 JSON，并编码到 `arguments_json`：

```json
{
  "server": "context7",
  "tool": "query-docs",
  "arguments_json": "{\"libraryId\":\"/golang/go\",\"query\":\"context cancellation\"}"
}
```

HTTP 层仍兼容旧的 `arguments` 对象字段。

### MCP 语义保真

上游 MCP 建立连接后，Gateway 保存 server 的 `instructions`。工具发现同时保留：

- `title`
- `input_schema`
- `output_schema`
- `annotations.readOnlyHint`
- `annotations.destructiveHint`
- `annotations.idempotentHint`
- `annotations.openWorldHint`

这些信息用于帮助 MyGPT 正确选择和调用工具。Annotations 是上游提示，不替代 Registry、allow/deny 和服务器端校验。

### 项目身份

对于 CodeGraph 这类“同一个 MCP server 可以查看多个项目”的工具，不让模型自由传 VPS 绝对路径。Registry 维护项目别名：

```json
{
  "projects": {
    "mygpt": "/opt/mygpt-cf-tunnel-mcp",
    "embodiedgen": "/root/agentscape-research/EmbodiedGen"
  },
  "servers": {
    "codegraph-mcp-gateway": {
      "transport": "stdio",
      "command": "/usr/local/bin/codegraph",
      "args": ["serve", "--mcp", "--path", "/opt/mygpt-cf-tunnel-mcp"],
      "working_dir": "/opt/mygpt-cf-tunnel-mcp",
      "project_argument": "projectPath",
      "require_project": true,
      "trust_annotations": true
    }
  }
}
```

调用时只传别名：

```json
{
  "server": "codegraph-mcp-gateway",
  "tool": "codegraph_explore",
  "project": "embodiedgen",
  "arguments_json": "{\"query\":\"从文本到 3D 的主要入口和跨文件调用链\",\"maxFiles\":12}"
}
```

Gateway 会把 `embodiedgen` 安全映射成真实路径，再注入上游的 `projectPath`。如果 `require_project=true` 却没有传 `project`，调用会失败；如果在 `arguments_json` 中直接塞原始 `projectPath`，也会被拒绝。这样可以避免模型忘记项目上下文后静默查询错误仓库。

### CodeGraph 推荐工作流

对陌生仓库不要只发一句宽泛的“解释整个项目”。更稳定的闭环是：

```text
项目定位
  ↓
快速确认 README / 顶层目录 / 业务入口
  ↓
CodeGraph 沿入口、符号、caller、依赖深入
  ↓
Shell 精确修改
  ↓
必要时同步索引
  ↓
CodeGraph 验证 caller / blast radius
  ↓
测试 / 构建 / 服务验证
```

CodeGraph 负责语义关系和影响分析，Shell 负责磁盘事实、修改和运行验证；二者互补，不让任何一个工具承担全部项目理解。

### Session、缓存与重连

- stdio MCP 子进程和 Streamable HTTP session 持久复用。
- `tools/list` 默认缓存 `5m`，`searchMcpTools(refresh=true)` 可主动刷新。
- Tool / Resource / Prompt 调用出现连接错误后，会关闭失效 session、重新连接并重试一次。
- `listMcpServers.state=idle` 只表示惰性连接尚未建立，不代表 MCP 故障。

环境变量：

```env
REQUEST_TIMEOUT=90s
TOOL_CACHE_TTL=5m
MAX_RESPONSE_BYTES=1048576
```

超过 `MAX_RESPONSE_BYTES` 的 MCP 文本结果会被明确截断并提示缩小查询范围，避免一次工具返回污染过多对话上下文。

## Registry 策略

每个 MCP server 可以设置：

```json
{
  "allow_tools": ["read_*", "query-docs"],
  "deny_tools": ["delete_*"],
  "allow_resources": ["docs://*"],
  "deny_resources": ["secret://*"],
  "allow_prompts": ["review-*"],
  "deny_prompts": ["admin-*"],
  "project_argument": "projectPath",
  "require_project": true,
  "trust_annotations": true
}
```

规则：deny 优先；allow 为空时默认允许；支持精确名称、`*` 和前缀通配符 `xxx*`。

GPT 不能传任意上游 MCP URL，只能使用 Registry 中已登记的 server alias。项目型 server 只能使用 Registry 中已登记的 project alias。

完整示例见 `servers.example.json`。

## 管理 CLI

```bash
mygpt-mcpctl list
mygpt-mcpctl validate
mygpt-mcpctl add-http context7 https://mcp.context7.com/mcp
mygpt-mcpctl add-stdio codegraph /usr/local/bin/codegraph serve --mcp
mygpt-mcpctl disable context7
mygpt-mcpctl enable context7
mygpt-mcpctl remove context7
```

Registry 修改会在下一次 MCP Action 请求时自动同步。Server 配置变化会关闭旧 session 并按新配置重新连接，不需要重启整个 Gateway。

## 安全边界

- MCP Gateway 没有 Shell / `runCommand` API。
- Command Action 和 MCP Action 使用独立服务、端口和认证。
- GPT 不能动态指定 MCP URL，避免 Gateway 变成 SSRF Proxy。
- 上游 token 通过 `token_env` 从 VPS 环境变量读取，不进入 GPT 参数。
- Tool / Resource / Prompt 由 allow/deny policy 约束。
- `callReadOnlyMcpTool` 只接受管理员显式设置 `trust_annotations=true`，且当前工具元数据中 `readOnlyHint=true` 的工具。
- 项目绝对路径由服务端 Registry 持有，不暴露为 MyGPT 自由输入。
- systemd 使用 `NoNewPrivileges=true`、`ProtectSystem=full`、`PrivateTmp=true`。

## 当前部署

```text
Listen:   127.0.0.1:8788
Public:   https://arm-sg-mcp.202820.xyz
OpenAPI:  https://arm-sg-mcp.202820.xyz/openapi.json
```

当前正式 Registry 已联调：CodeGraph、Context7、DeepWiki。

## 构建

```bash
go test ./...
go test -race ./...
go vet ./...
go build -o mygpt-mcp-agent ./cmd/mygpt-mcp-agent
go build -o mygpt-mcpctl ./cmd/mygpt-mcpctl
```

安装：

```bash
sudo ./scripts/install.sh
```
