#!/usr/bin/env bash
set -euo pipefail
install -d -m 700 /etc/mygpt-mcp
install -m 755 ./mygpt-mcp-agent /usr/local/bin/mygpt-mcp-agent
install -m 644 deploy/mygpt-mcp-agent.service /etc/systemd/system/mygpt-mcp-agent.service
[ -f /etc/mygpt-mcp/servers.json ] || install -m 600 servers.example.json /etc/mygpt-mcp/servers.json
systemctl daemon-reload
printf 'Installed. Create /etc/mygpt-mcp/agent.env, then run: systemctl enable --now mygpt-mcp-agent\n'
