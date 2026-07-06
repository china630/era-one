#!/bin/sh
# Pilot connector: isolate host (replace with SSH/firewall script at customer site).
set -e
NODE="${1:?node_id required}"
LOG="${ERA_SOAR_ISOLATE_LOG:-/var/log/era-isolate.log}"
TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "$TS isolate node=$NODE" >> "$LOG"
echo "OK isolated $NODE via pilot script"
