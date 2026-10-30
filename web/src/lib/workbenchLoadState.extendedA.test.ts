import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const ROUTE_ROOT = path.resolve(process.cwd(), "src/app/[locale]/project/[id]");

const ROUTES = {
  "autonomy-states": "list.length === 0",
  deployments: "list.length === 0",
  "division-templates": "list.length === 0",
  "dynamic-modifiers": "list.length === 0",
  equipment: "list.length === 0",
  intelligence: "list.length === 0",
  "special-projects": "specs.length === 0 && projects.length === 0",
  "sub-units": "rows.length === 0",
  "ui-panels": "list.length === 0",
} as const;

const ADDITIONAL_IMPLEMENTATIONS = [
  {
    name: "modifier-definitions",
    file: path.join(ROUTE_ROOT, "modifier-definitions", "page.tsx"),
    emptyGuard: "listLoadError && rows.length === 0",
  },
  {
    name: "ai planning routes",
    file: path.resolve(
      process.cwd(),
      "src/components/workbench/ai-planning/AiPlanningWorkbench.tsx",
    ),
    emptyGuard: "listLoadError && rows.length === 0",
  },
  {
    name: "oob-units",
    file: path.resolve(
      process.cwd(),
      "src/components/workbench/oob-units/OobUnitsEditor.tsx",
    ),
    emptyGuard: "showSnapshotLoadError && !hasSnapshotData",
    manualRetry: "onRetry={() => loadSnapshot(selectedOob)}",
  },
] as const;

describe("extended A workbench list load state", () => {
  for (const [route, emptyGuard] of Object.entries(ROUTES)) {
    it(`${route} surfaces an initial load failure and wires retry`, () => {
      const source = readFileSync(
        path.join(ROUTE_ROOT, route, "page.tsx"),
        "utf8",
      );

      expect(source).toContain(
        'import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";',
      );
      expect(source).toMatch(/error:\s*listLoadError/);
      expect(source).toMatch(/mutate:\s*retryList/);
      expect(source).toContain(`listLoadError && ${emptyGuard}`);
      expect(source).toContain(
        "<WorkbenchLoadError onRetry={() => retryList(undefined, { throwOnError: false })} />",
      );
      expect(source).not.toMatch(
        /<WorkbenchLoadError[^>]*\b(?:error|message|detail)=/,
      );
    });
  }

  for (const implementation of ADDITIONAL_IMPLEMENTATIONS) {
    it(`${implementation.name} gives list failure priority over empty state`, () => {
      const source = readFileSync(implementation.file, "utf8");

      expect(source).toContain(
        'import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";',
      );
      expect(source).toContain(implementation.emptyGuard);
      expect(source).toContain("<WorkbenchLoadError");
      if ("manualRetry" in implementation) {
        expect(source).toContain("snapshotLoadFailed");
        expect(source).toContain(implementation.manualRetry);
      } else {
        expect(source).toMatch(/error:\s*listLoadError/);
        expect(source).toMatch(/mutate:\s*retryList/);
        expect(source).toContain(
          "onRetry={() => retryList(undefined, { throwOnError: false })}",
        );
        expect(source).not.toMatch(/\{listLoadError\}/);
      }
    });
  }

  it("the thin AI planning and OOB routes remain wired to the guarded implementations", () => {
    const strategyPage = readFileSync(
      path.join(ROUTE_ROOT, "ai-strategy-plans", "page.tsx"),
      "utf8",
    );
    const templatePage = readFileSync(
      path.join(ROUTE_ROOT, "ai-templates", "page.tsx"),
      "utf8",
    );
    const oobPage = readFileSync(
      path.join(ROUTE_ROOT, "oob-units", "page.tsx"),
      "utf8",
    );

    expect(strategyPage).toContain("<AiStrategyPlansWorkbench />");
    expect(templatePage).toContain("<AiTemplatesWorkbench />");
    expect(oobPage).toContain("<OobUnitsEditor projectId={Number(id)} />");
  });
});
