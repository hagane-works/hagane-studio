import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const routeRoot = path.resolve(process.cwd(), "src/app/[locale]/project/[id]");

const routes = {
  characters: "error && characters.length === 0",
  "focus-tree": "error && errorProjectId === projectId && trees.length === 0",
  ideas: "error && ideas.length === 0",
  states: "error && states.length === 0",
  technology: "error && technologies.length === 0",
} as const;

describe("core workbench list load state", () => {
  for (const [route, emptyGuard] of Object.entries(routes)) {
    it(`${route} treats an initial request failure as unknown, not empty`, () => {
      const source = readFileSync(
        path.join(routeRoot, route, "page.tsx"),
        "utf8",
      );

      expect(source).toContain(
        'import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";',
      );
      expect(source).toContain(emptyGuard);
      expect(source).toContain("<WorkbenchLoadError");
      expect(source).not.toMatch(
        /<WorkbenchLoadError[^>]*\b(?:error|message|detail)=/,
      );
    });
  }

  for (const route of ["decisions", "events"] as const) {
    it(`${route} protects both parent and child empty states`, () => {
      const source = readFileSync(
        path.join(routeRoot, route, "page.tsx"),
        "utf8",
      );
      const parent = route === "decisions" ? "categories" : "namespaces";
      const child = route === "decisions" ? "decisions" : "events";

      expect(source).toContain(`error && ${parent}.length === 0`);
      expect(source).toContain(`error && ${child}.length === 0`);
      expect(
        source.match(/<WorkbenchLoadError/g)?.length,
      ).toBeGreaterThanOrEqual(2);
    });
  }

  it("parent list stores preserve failures for the retry surface", () => {
    for (const store of ["decisionStore.ts", "eventStore.ts"]) {
      const source = readFileSync(
        path.resolve(process.cwd(), "src/lib/store", store),
        "utf8",
      );
      expect(source).toContain("error: null");
      expect(source).toMatch(/catch \(err\)[\s\S]*?set\(\{ error:/);
    }
  });
});
