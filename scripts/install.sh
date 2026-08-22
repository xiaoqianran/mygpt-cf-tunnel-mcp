#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
install -d -m 700 /etc/mygpt-mcp

go build -trimpath -ldflags='-s -w' -o /tmp/mygpt-mcp-agent ./cmd/mygpt-mcp-agent
go build -trimpath -ldflags='-s -w' -o /tmp/mygpt-mcpctl ./cmd/mygpt-mcpctl
install -m 755 /tmp/mygpt-mcp-agent /usr/local/bin/mygpt-mcp-agent
install -m 755 /tmp/mygpt-mcpctl /usr/local/bin/mygpt-mcpctl
install -m 644 deploy/mygpt-mcp-agent.service /etc/systemd/system/mygpt-mcp-agent.service
[ -f /etc/mygpt-mcp/servers.json ] || install -m 600 servers.example.json /etc/mygpt-mcp/servers.json
if [ ! -f /etc/mygpt-mcp/agent.env ]; then
  install -m 600 .env.example /etc/mygpt-mcp/agent.env
fi
systemctl daemon-reload
printf 'Installed. Review /etc/mygpt-mcp/agent.env and /etc/mygpt-mcp/servers.json, then run:\n  systemctl enable --now mygpt-mcp-agent\n'
