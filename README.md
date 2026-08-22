# mygpt-cf-tunnel-mcp

A standalone **MCP Host for ChatGPT Custom GPT / MyGPT Actions**.

It complements [`mygpt-cf-tunnel`](https://github.com/xiaoqianran/mygpt-cf-tunnel): the command project owns remote shell execution; this project owns MCP integration only.

## Architecture

```text
Custom GPT
  ├─ Command Action → mygpt-cf-tunnel → /bin/bash -lc
  └─ MCP Action     → mygpt-cf-tunnel-mcp :8788
                         ├─ stdio MCP subprocesses
                         └─ remote Streamable HTTP MCP servers
```

The project provides the MCP Host layer normally built into coding agents: registry management, stdio process launch, remote connections, session reuse, tool discovery and MCP invocation.

## v0.4

- stdio + Streamable HTTP transports
- persistent MCP sessions and automatic reconnect
- TTL-cached tool discovery
- per-server allow/deny policies for tools, resources and prompts
- Minimal MyGPT Action surface: `listMcpServers`, `searchMcpTools`, `callMcpTool`
- Backend still supports Tools, Resources, Prompts and runtime status internally
- automatic registry reload
- `mygpt-mcpctl` CLI for add/list/remove/enable/disable/validate
- no arbitrary upstream URLs in Action requests

## CLI

```bash
mygpt-mcpctl list
mygpt-mcpctl validate
mygpt-mcpctl add-http context7 https://mcp.context7.com/mcp
mygpt-mcpctl add-stdio codegraph /usr/local/bin/codegraph serve --mcp
mygpt-mcpctl disable context7
mygpt-mcpctl enable context7
mygpt-mcpctl remove context7
```

## Registry

```json
{
  "servers": {
    "codegraph": {
      "transport": "stdio",
      "command": "/usr/local/bin/codegraph",
      "args": ["serve", "--mcp"],
      "working_dir": "/srv/project",
      "allow_tools": ["codegraph_explore"]
    },
    "context7": {
      "transport": "streamable_http",
      "url": "https://mcp.context7.com/mcp",
      "allow_tools": ["resolve-library-id", "query-docs"]
    }
  }
}
```

For runtimes installed through fnm/nvm/uv and not visible in the systemd PATH, use an absolute command path or set a per-server `env.PATH`.

## Deployment

```text
Listen:  127.0.0.1:8788
Public:  https://arm-sg-mcp.202820.xyz
Schema:  https://arm-sg-mcp.202820.xyz/openapi.json
```

Authentication is Bearer-based and configured with `API_TOKEN` in `/etc/mygpt-mcp/agent.env`.

## Build

```bash
go test ./...
go vet ./...
go build -o mygpt-mcp-agent ./cmd/mygpt-mcp-agent
go build -o mygpt-mcpctl ./cmd/mygpt-mcpctl
```
