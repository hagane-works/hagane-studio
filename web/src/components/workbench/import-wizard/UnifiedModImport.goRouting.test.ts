import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const source = fs.readFileSync(
  path.resolve(
    process.cwd(),
    "src/components/workbench/import-wizard/UnifiedModImport.tsx",
  ),
  "utf8",
);

describe("UnifiedModImport Go routing", () => {
  it("creates detected and suggested countries through projectsvc", () => {
    expect(source).toContain(
      "`${API_BASE}/go/projects/${projectId}/countries`",
    );
    expect(source).not.toContain(
      "`${API_BASE}/projects/${projectId}/countries`",
    );
  });
});
