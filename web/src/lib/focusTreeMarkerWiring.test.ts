import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

describe("focus tree parent-aware marker wiring", () => {
  it("projects saved completion_reward spans and feeds Monaco markers", () => {
    const source = fs.readFileSync(
      path.resolve("src/app/[locale]/project/[id]/focus-tree/page.tsx"),
      "utf8",
    );

    expect(source).toContain("previewMarkersForLintTarget(");
    expect(source).toContain("entityId: selectedNode.focus_id");
    expect(source).toContain('field: "completion_reward"');
    expect(source).toContain("fieldSource: selectedNode.completion_reward");
    expect(source).toContain("externalMarkers: treeLintMarkers");
    expect(source).toContain("onReport={setTreeLintReport}");
  });
});
