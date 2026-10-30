import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { buildRepairQueue, groupRepairQueue } from "@/lib/exportRepairQueue";
import type { CtdIssue } from "@/lib/api/export";

const here = dirname(fileURLToPath(import.meta.url));
const list = readFileSync(join(here, "RepairQueueList.tsx"), "utf8");

function issue(code: string, id: string): CtdIssue {
  return {
    code,
    severity: "warning",
    message: `backend message for ${code}`,
    fix_hint: "",
    entity_ref: { type: "focus", id },
  } as CtdIssue;
}

describe("A2 · the repair queue list is grouped", () => {
  it("iterates groups, not the flat item array", () => {
    expect(list).toContain("groupRepairQueue");
    expect(list).toContain("groups.map(");

    expect(list).toContain("<li key={group.code}>");
    expect(list).not.toContain("<li key={item.key}>");
  });

  it("selecting a group lands on the work left in it, by flat index", () => {
    expect(list).toContain("onSelect(group.targetIndex)");
    expect(list).toContain("indexByKey");
  });

  it("a group of several says how many are affected", () => {
    expect(list).toContain('data-testid="repair-queue-group-count"');
    expect(list).toContain("group.count > 1");

    expect(list).toContain('t("ui.attention.groupAffected"');
  });

  it("a re-check badge survives grouping instead of being hidden by it", () => {
    expect(list).toContain("group.rows.some(");
    expect(list).toContain("row.addedByRecheck && !row.resolved");
  });

  it("collapses a real many-to-one distribution", () => {
    const issues = [
      ...Array.from({ length: 8 }, (_, i) =>
        issue("focus_layout_same_cell", `f${i}`),
      ),
      ...Array.from({ length: 3 }, (_, i) =>
        issue("focus_tree_no_nodes", `t${i}`),
      ),
      issue("idea_no_modifier", "i0"),
    ];
    const items = buildRepairQueue(issues);
    const groups = groupRepairQueue(items);
    expect(items).toHaveLength(12);
    expect(groups).toHaveLength(3);
    expect(groups.map((g) => g.count)).toEqual([8, 3, 1]);

    expect(groups.reduce((n, g) => n + g.rows.length, 0)).toBe(items.length);
  });
});
