#!/usr/bin/env node
/**
 * Cursor stop hook — nudge acceptance / quality closeout once per agent turn.
 * Emits followup_message only when the session looks like a feature closeout
 * (heuristic: status/status text mentions Matrix or Scaffold, or many edits).
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
const status = String(input.status || input.completion_status || "").toLowerCase();
const summary = String(input.summary || input.last_assistant_message || input.message || "");
const loopCount = Number(input.loop_count || input.followup_count || 0);

// Avoid infinite follow-up loops.
if (loopCount > 0) {
  process.stdout.write("{}");
  process.exit(0);
}

const looksProductive =
  /matrix|scaffold|pilot-ready|implementation-matrix|acceptance|gate\[x\]|editions-/i.test(
    summary
  ) ||
  status === "completed" ||
  status === "success";

if (!looksProductive) {
  process.stdout.write("{}");
  process.exit(0);
}

const followup = [
  "Closeout checklist (Acceptance Standard v1.2):",
  "1) Run: pwsh -File scripts/check-acceptance-consistency.ps1",
  "2) Matrix SSOT: Scaffold ✅ only with PRD wording + negative path; else 🟡",
  "3) Sync Index / Gap / MVP rollup to Matrix (no false all-✅ prose)",
  "4) If code changed AuthZ/license/enforce: run targeted tests + quality gates skill",
  "5) Do not mark Pilot-ready / editions ga without field proof",
].join("\n");

process.stdout.write(
  JSON.stringify({
    followup_message: followup,
  })
);
process.exit(0);
