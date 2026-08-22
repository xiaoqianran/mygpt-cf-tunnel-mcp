# mygpt-cf-tunnel-mcp

A **standalone MCP Action Gateway** for ChatGPT Custom GPTs.

It is deliberately separate from [`mygpt-cf-tunnel`](https://github.com/xiaoqianran/mygpt-cf-tunnel): the command project owns remote shell execution; this project owns MCP discovery and invocation only.

## Architecture

```text
Custom GPT
  ├─ Action A → command.example.com → mygpt-cf-tunnel → /bin/bash -lc
  └─ Action B → mcp.example.com     → this project
                                      ├─ listMcpServers
                                      ├─ searchMcpTools
                                      └─ callMcpTool
                                             ↓
                                   registered remote MCP servers
                                   (Streamable HTTP /mcp)
```

**Trust boundary:** the GPT sends only a registered `server` alias, tool name, and arguments. It cannot supply arbitrary upstream URLs. MCP URLs and upstream credentials live on the VPS.

## Why a separate repo

- independent Action schema and domain
- independent API token and Cloudflare Tunnel ingress
- independent release / rollback / logs
- no shell execution in the MCP gateway
- compromise of an MCP credential does not expose the command gateway token

## Action surface

- `listMcpServers` — list safe server aliases
- `searchMcpTools` — discover tools without dumping every tool schema into the GPT context
- `callMcpTool` — call a discovered tool

`callMcpTool` is marked consequential in the OpenAPI schema because an upstream MCP tool may mutate external state.

## MCP transport

v0.1 targets **remote Streamable HTTP MCP**. It uses the official Go SDK (`github.com/modelcontextprotocol/go-sdk`) and disables the optional standalone SSE stream, which fits request/response gateway traffic and the 2026 stateless direction.

Local stdio MCP is intentionally not part of v0.1. If needed later, add it as a second registry transport without changing the MyGPT Action API.

## Configuration

```bash
cp .env.example /etc/mygpt-mcp/agent.env
cp servers.example.json /etc/mygpt-mcp/servers.json
```

Example registry:

```json
{
  "servers": {
    "my-service": {
      "url": "https://mcp.example.com/mcp",
      "description": "Production tools",
      "token_env": "MY_SERVICE_MCP_TOKEN"
    }
  }
}
```

`token_env` is optional and is resolved only on the gateway host.

## Build

```bash
go build -o mygpt-mcp-agent ./cmd/mygpt-mcp-agent
```

## Cloudflare Tunnel

Use a **different hostname** from the command gateway:

```yaml
ingress:
  - hostname: command.example.com
    service: http://127.0.0.1:8787
  - hostname: mcp.example.com
    service: http://127.0.0.1:8788
  - service: http_status:404
```

Then import:

```text
https://mcp.example.com/openapi.json
```

as a separate Custom GPT Action and configure Bearer authentication with this service's `API_TOKEN`.

## Security model

1. No arbitrary upstream MCP URL in Action requests.
2. Upstream secrets are environment variables, never GPT arguments.
3. Registry is server-controlled and can disable aliases.
4. Remote MCP must use HTTPS, except explicit localhost endpoints.
5. Gateway has no command-execution endpoint.
6. Keep `API_TOKEN` distinct from `mygpt-cf-tunnel`.

## Roadmap

- cached tool index with TTL
- OAuth-backed upstream MCP servers
- per-server/per-tool allow and deny rules
- output-size enforcement and structured truncation
- audit IDs and call logging
- optional stdio transport
- health probes per upstream
