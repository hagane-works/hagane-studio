import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { afterAll, describe, expect, it } from "vitest";

const tempDir = mkdtempSync(resolve(tmpdir(), "hoi4-eslint-ratchet-"));
const script = resolve(
  process.cwd(),
  "scripts",
  "check-set-state-in-effect-ratchet.mjs",
);

afterAll(() => rmSync(tempDir, { recursive: true, force: true }));

function reportWithRuleErrors(count: number) {
  return [
    {
      filePath: "fixture.tsx",
      messages: Array.from({ length: count }, () => ({
        ruleId: "react-hooks/set-state-in-effect",
        severity: 2,
        message: "fixture violation",
      })),
    },
  ];
}

function runReport(name: string, report: unknown) {
  const reportPath = resolve(tempDir, `${name}.json`);
  writeFileSync(reportPath, JSON.stringify(report), "utf8");
  return spawnSync(process.execPath, [script, "--report", reportPath], {
    cwd: process.cwd(),
    encoding: "utf8",
  });
}

describe("set-state-in-effect build ratchet", () => {
  it("accepts the zero-error baseline", () => {
    const result = runReport("at-limit", reportWithRuleErrors(0));
    expect(result.status).toBe(0);
    expect(result.stdout).toContain("passed (0/0)");
  });

  it("rejects one newly introduced violation", () => {
    const result = runReport("over-limit", reportWithRuleErrors(1));
    expect(result.status).toBe(1);
    expect(result.stderr).toContain("regressed: 1 errors");
  });

  it("fails closed on a fatal ESLint result", () => {
    const result = runReport("fatal", [
      {
        filePath: "broken.tsx",
        messages: [{ fatal: true, severity: 2, message: "parse failed" }],
      },
    ]);
    expect(result.status).toBe(1);
    expect(result.stderr).toContain("fatal ESLint result");
  });
});
