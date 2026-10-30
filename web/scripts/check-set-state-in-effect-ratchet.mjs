import { readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const RULE_ID = "react-hooks/set-state-in-effect";
const MAX_ALLOWED = 0;
const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url));
const FRONTEND_DIR = resolve(SCRIPT_DIR, "..");

function parseReport(raw) {
  const report = JSON.parse(raw);
  if (!Array.isArray(report)) {
    throw new Error("ESLint JSON report must be an array");
  }
  return report;
}

function runEslint() {
  const eslintPath = resolve(
    FRONTEND_DIR,
    "node_modules",
    "eslint",
    "bin",
    "eslint.js",
  );
  const result = spawnSync(
    process.execPath,
    [eslintPath, ".", "--format", "json"],
    {
      cwd: FRONTEND_DIR,
      encoding: "utf8",
      maxBuffer: 64 * 1024 * 1024,
    },
  );
  if (result.error) throw result.error;
  if (result.status != null && result.status > 1) {
    throw new Error(result.stderr.trim() || `ESLint exited ${result.status}`);
  }
  if (!result.stdout.trim()) {
    throw new Error(result.stderr.trim() || "ESLint produced no JSON report");
  }
  return parseReport(result.stdout);
}

function loadReport(args) {
  if (args.length === 0) return runEslint();
  if (args.length === 2 && args[0] === "--report") {
    return parseReport(readFileSync(resolve(args[1]), "utf8"));
  }
  throw new Error("usage: check-set-state-in-effect-ratchet.mjs [--report path]");
}

function countRuleErrors(report) {
  let count = 0;
  for (const result of report) {
    for (const message of result.messages ?? []) {
      if (message.fatal) {
        throw new Error(
          `fatal ESLint result in ${result.filePath ?? "unknown file"}: ${message.message}`,
        );
      }
      if (message.severity === 2 && message.ruleId === RULE_ID) count += 1;
    }
  }
  return count;
}

function main() {
  try {
    const count = countRuleErrors(loadReport(process.argv.slice(2)));
    if (count > MAX_ALLOWED) {
      console.error(
        `[FAIL] ${RULE_ID} ratchet regressed: ${count} errors (maximum ${MAX_ALLOWED})`,
      );
      process.exitCode = 1;
      return;
    }
    console.log(
      `[OK] ${RULE_ID} ratchet passed (${count}/${MAX_ALLOWED})`,
    );
  } catch (error) {
    console.error(`[FAIL] ${error instanceof Error ? error.message : String(error)}`);
    process.exitCode = 1;
  }
}

main();
