import fs from "node:fs";
import path from "node:path";

import { describe, expect, it } from "vitest";

const root = process.cwd();

describe("country-tag import preflight Go ownership", () => {
  it("hard-wires both UI flows to /api/go without a Python fallback", () => {
    const dialog = fs.readFileSync(
      path.join(root, "src/components/workbench/ImportAuditDialog.tsx"),
      "utf8",
    );
    const wizard = fs.readFileSync(
      path.join(
        root,
        "src/components/workbench/import-wizard/UnifiedModImport.tsx",
      ),
      "utf8",
    );
    expect(dialog).toContain(
      "`/api/go/projects/${projectId}/import/audit-tags`",
    );
    expect(wizard).toContain(
      "`/api/go/projects/${projectId}/import/audit-tags`",
    );
    expect(dialog).not.toContain(
      "`${API_BASE}/projects/${projectId}/import/audit-tags`",
    );
    expect(wizard).not.toContain(
      "`${API_BASE}/projects/${projectId}/import/audit-tags`",
    );
  });
});
