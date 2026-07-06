#!/bin/bash
# ERA XDR agent Linux install sketch (GA-1 S5-7/S5-24)
set -euo pipefail
BIN="${1:-./target/release/era-agent}"
install -m 0755 "$BIN" /usr/local/bin/era-agent
install -d -m 0750 /etc/era-one
install -m 0640 deploy/agent/agent.env.example /etc/era-one/agent.env
install -m 0644 deploy/agent/era-agent.service /etc/systemd/system/era-agent.service
systemctl daemon-reload
echo "Installed. Edit /etc/era-one/agent.env then: systemctl enable --now era-agent"
