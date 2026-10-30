import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const frontendRoot = join(here, "..", "..", "..");
const read = (path: string) => readFileSync(join(frontendRoot, path), "utf8");
const stepper = read("src/components/export/ExportStepper.tsx");
const page = read("src/app/[locale]/project/[id]/export/page.tsx");

describe("A1 · the manual re-check control", () => {
  it("the stepper renders a re-check button when a handler is supplied", () => {
    expect(stepper).toContain("onRerunCheck");
    expect(stepper).toContain('data-testid="export-rerun-check"');

    expect(stepper).toContain('t("ui.repair.recheck")');
    expect(stepper).toContain('t("ui.repair.recheckRunning")');
  });

  it("is the page's only accent BEFORE the first check, and steps back after", () => {
    const button = stepper.slice(
      stepper.indexOf('data-testid="export-rerun-check"'),
    );
    const className = button.slice(0, button.indexOf("</button>"));
    expect(className).toContain('phase === "idle"');
    expect(className).toContain("bg-accent");
    expect(className).toContain("border-border");
  });

  it("it is inert while a check is already running", () => {
    expect(stepper).toContain('disabled={phase === "running"}');
  });

  it("says what it will do — 'start' before, 're-check' after", () => {
    expect(stepper).toContain('t("step1ButtonIdle")');
    expect(stepper).toContain('t("ui.repair.recheck")');
  });

  it("the page starts a check ONLY from this control", () => {
    expect(page).toContain("const runCheck = useCallback(");
    expect(page).toContain("onRerunCheck={runCheck}");
    expect(page).not.toMatch(/useEffect\([^)]*runCheck/);
    expect(page).not.toMatch(/\brunCheck\b[^;]*\],\s*\)/);
  });

  it("the kept result carries when it was produced", () => {
    expect(page).toContain("ranAt: Date.now()");
    expect(page).toContain("checkStaleReason(");
  });
});
