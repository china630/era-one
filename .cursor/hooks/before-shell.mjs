#!/usr/bin/env node
/**
 * Cursor beforeShellExecution — block destructive / prod-risk commands.
 * Reads hook JSON from stdin; writes permission JSON to stdout.
 */
import { readFileSync } from "node:fs";

function readInput() {
  try {
    const raw = readFileSync(0, "utf8");
    if (!raw.trim()) return {};
    return JSON.parse(raw);
  } catch {
    return {};
  }
}

const input = readInput();
const command = String(input.command || input.tool_input?.command || "");
const lower = command.toLowerCase();

/** @type {{ re: RegExp, permission: "deny" | "ask", reason: string }[]} */
const rules = [
  {
    re: /\bgit\s+push\b.*(--force|--force-with-lease|\s-f\b)/i,
    permission: "deny",
    reason: "Force-push blocked by project hooks. Ask the user explicitly if truly required.",
  },
  {
    re: /\bgit\s+reset\s+--hard\b/i,
    permission: "deny",
    reason: "git reset --hard is blocked (destroys working tree).",
  },
  {
    re: /\bgit\s+clean\s+(-[a-z]*f|[a-z]*fd)/i,
    permission: "deny",
    reason: "git clean -f is blocked without explicit user request.",
  },
  {
    re: /\bgit\s+stash\b/i,
    permission: "ask",
    reason: "git stash can hide unrelated work; site-commit-isolation forbids stash without explicit ask.",
  },
  {
    re: /\b(rm\s+-rf|remove-item\s+-recurse)\b.*(\.git\b|deploy[/\\].*prod|editions-.*\.yaml|\.env\b)/i,
    permission: "deny",
    reason: "Recursive delete of critical paths is blocked.",
  },
  {
    re: /\b(drop\s+(database|table)|truncate\s+table)\b/i,
    permission: "deny",
    reason: "Destructive SQL is blocked. Use SELECT-only lab credentials.",
  },
  {
    re: /docker\s+compose\b.*docker-compose\.(prod|comms\.prod|office\.prod).*(\bdown\b|\brm\b).*(-v|--volumes)/i,
    permission: "ask",
    reason: "Prod compose with volume wipe — confirm with user.",
  },
  {
    re: /\b(curl|wget|Invoke-WebRequest)\b.+\b(raw\.githubusercontent|api\.openai|api\.anthropic|telemetry)\b/i,
    permission: "ask",
    reason: "Outbound fetch may violate air-gap policy; confirm intent.",
  },
];

for (const rule of rules) {
  if (rule.re.test(command) || rule.re.test(lower)) {
    process.stdout.write(
      JSON.stringify({
        permission: rule.permission,
        user_message: rule.reason,
        agent_message: rule.reason,
      })
    );
    process.exit(0);
  }
}

process.stdout.write(JSON.stringify({ permission: "allow" }));
process.exit(0);
