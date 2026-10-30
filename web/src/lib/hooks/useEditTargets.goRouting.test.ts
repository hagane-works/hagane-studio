import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("edit targets are permanently Go-owned", () => {
  it("calls the Go file-index route with no Python fallback", () => {
    const source = readFileSync(
      resolve(process.cwd(), "src/lib/hooks/useEditTargets.ts"),
      "utf8",
    );
    expect(source).toContain(
      "`${API_BASE}/go/projects/${projectId}/edit-targets`",
    );
    expect(source).not.toContain(
      "`${API_BASE}/projects/${projectId}/edit-targets`",
    );
  });
});
