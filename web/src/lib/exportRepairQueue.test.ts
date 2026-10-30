import { describe, expect, it } from "vitest";
import type { CtdIssue } from "@/lib/api/export";
import { presentCheckIssues } from "@/lib/exportCheckPresentation";
import {
  buildRepairQueue,
  countGroupsWithEntryPoint,
  groupRepairQueue,
  findRepairIndexByHash,
  nextUnresolvedIndex,
  normalizeRepairHash,
  reconcileRepairQueue,
  repairHash,
  repairItemKey,
  repairProgress,
  repairRoutePath,
} from "@/lib/exportRepairQueue";

function issue(partial: Partial<CtdIssue> & { code: string }): CtdIssue {
  return {
    entity_ref: null,
    severity: "warning",
    message: `backend message for ${partial.code}`,
    fix_hint: "",
    ...partial,
  };
}

const capitalITA = issue({
  code: "country.capital_missing",
  severity: "blocker",
  entity_ref: { type: "country", id: "ITA" },
});
const capitalFRA = issue({
  code: "country.capital_missing",
  severity: "blocker",
  entity_ref: { type: "country", id: "FRA" },
});
const focusName = issue({
  code: "focus.display_name.missing_created",
  entity_ref: { type: "focus", id: 42, tree_id: 7 },
});

describe("buildRepairQueue · same set the export page calls attention", () => {
  it("mirrors presentCheckIssues().attention exactly, in order", () => {
    const issues = [
      focusName,
      issue({ code: "timeline.start_date_ok", severity: "info" }),
      issue({ code: "focus_file_policy_drift", severity: "info" }),
      capitalITA,
      issue({ code: "focus_registry_unavailable", severity: "info" }),
    ];
    const queue = buildRepairQueue(issues);
    expect(queue.map((item) => item.finding.issue.code)).toEqual(
      presentCheckIssues(issues).attention.map((f) => f.issue.code),
    );

    expect(queue.map((item) => item.finding.issue.code)).toEqual([
      "country.capital_missing",
      "focus.display_name.missing_created",
    ]);
  });

  it("keeps unknown non-info codes (verbatim rows are repairable too)", () => {
    const queue = buildRepairQueue([
      issue({ code: "brand_new_backend_code", severity: "blocker" }),
      issue({ code: "brand_new_info_code", severity: "info" }),
    ]);
    expect(queue).toHaveLength(1);
    expect(queue[0].finding.meta).toBeNull();
    expect(queue[0].finding.issue.message).toContain("brand_new_backend_code");
  });

  it("gives sibling findings on one entity distinct keys", () => {
    const detailed = (reference: string) =>
      issue({
        code: "focus.display_name.missing_created",
        entity_ref: { type: "focus", id: 42 },
        detail: { reference },
      });
    const queue = buildRepairQueue([detailed("a"), detailed("b"), focusName]);
    expect(new Set(queue.map((item) => item.key)).size).toBe(queue.length);
  });

  it("still separates two findings whose identity is byte-identical", () => {
    const queue = buildRepairQueue([capitalITA, { ...capitalITA }]);
    expect(queue).toHaveLength(2);
    expect(queue[0].key).not.toBe(queue[1].key);
  });
});

describe("repairItemKey / repairHash / deep links", () => {
  it("keys the same finding identically across two check runs", () => {
    expect(repairItemKey(capitalITA)).toBe(repairItemKey({ ...capitalITA }));
    expect(repairItemKey(capitalITA)).not.toBe(repairItemKey(capitalFRA));
  });

  it("builds `#<code>` without an entity and `#<code>-<id>` with one", () => {
    expect(repairHash(issue({ code: "descriptor.replace_path_empty" }))).toBe(
      "descriptor.replace_path_empty",
    );
    expect(repairHash(capitalITA)).toBe("country.capital_missing-ITA");
    expect(repairHash(focusName)).toBe("focus.display_name.missing_created-42");
  });

  it("percent-encodes ids so a slash in an id cannot forge a second segment", () => {
    const weird = issue({
      code: "state.raw_parse_failed",
      entity_ref: { type: "state", id: "map/states/1.txt" },
    });
    expect(repairHash(weird)).toBe(
      "state.raw_parse_failed-map%2Fstates%2F1.txt",
    );
  });

  it("resolves the entity-qualified hash before the code-only one", () => {
    const queue = buildRepairQueue([capitalITA, capitalFRA]);
    expect(findRepairIndexByHash(queue, "#country.capital_missing-FRA")).toBe(
      1,
    );

    expect(findRepairIndexByHash(queue, "country.capital_missing-FRA")).toBe(1);

    expect(findRepairIndexByHash(queue, "#country.capital_missing")).toBe(0);
  });

  it("returns -1 for an empty or unknown hash instead of defaulting to row 0", () => {
    const queue = buildRepairQueue([capitalITA]);
    expect(findRepairIndexByHash(queue, "")).toBe(-1);
    expect(findRepairIndexByHash(queue, "#")).toBe(-1);
    expect(findRepairIndexByHash(queue, undefined)).toBe(-1);
    expect(findRepairIndexByHash(queue, "#nothing_like_this")).toBe(-1);
  });

  it("normalizes a leading # and surrounding whitespace", () => {
    expect(normalizeRepairHash("#capital ")).toBe("capital");
    expect(normalizeRepairHash(null)).toBe("");
  });
});

describe("repairRoutePath · honest about missing entry points", () => {
  it("maps known entity types to their workbench segment", () => {
    expect(repairRoutePath(capitalITA)).toBe("countries");
    expect(repairRoutePath(focusName)).toBe("focus-tree");
  });

  it("returns null (never a guessed route) with no entity or no mapping", () => {
    expect(
      repairRoutePath(issue({ code: "descriptor.replace_path_empty" })),
    ).toBeNull();
    expect(
      repairRoutePath(
        issue({
          code: "unknown_province",
          entity_ref: { type: "no_such_entity_kind", id: 1 },
        }),
      ),
    ).toBeNull();
  });
});

describe("reconcileRepairQueue · done is earned by a re-check", () => {
  it("resolves only the findings the re-check stopped reporting", () => {
    const initial = buildRepairQueue([capitalITA, capitalFRA, focusName]);
    const result = reconcileRepairQueue(initial, [capitalFRA, focusName]);
    expect(result.items.map((item) => item.resolved)).toEqual([
      true,
      false,
      false,
    ]);
    expect(result.newlyResolved).toEqual([initial[0].key]);
    expect(result.added).toEqual([]);
  });

  it("keeps queue positions stable and refreshes the surviving payload", () => {
    const initial = buildRepairQueue([capitalITA, focusName]);
    const movedAndReworded = {
      ...capitalITA,
      message: "backend reworded this sentence",
    };
    const result = reconcileRepairQueue(initial, [focusName, movedAndReworded]);
    expect(result.items.map((item) => item.key)).toEqual(
      initial.map((item) => item.key),
    );
    expect(result.items[0].finding.issue.message).toBe(
      "backend reworded this sentence",
    );
  });

  it("appends findings the re-check newly reports instead of dropping them", () => {
    const initial = buildRepairQueue([capitalITA]);
    const result = reconcileRepairQueue(initial, [capitalITA, capitalFRA]);
    expect(result.items).toHaveLength(2);
    expect(result.added).toEqual([result.items[1].key]);
    expect(result.items[1].addedByRecheck).toBe(true);
    expect(result.items[1].resolved).toBe(false);
  });

  it("re-opens a resolved item when the finding comes back", () => {
    const initial = buildRepairQueue([capitalITA]);
    const gone = reconcileRepairQueue(initial, []);
    expect(gone.items[0].resolved).toBe(true);
    const back = reconcileRepairQueue(gone.items, [capitalITA]);
    expect(back.items[0].resolved).toBe(false);
    expect(back.added).toEqual([]);
  });

  it("never reports the same key as both resolved and added", () => {
    const initial = buildRepairQueue([capitalITA, focusName]);
    const result = reconcileRepairQueue(initial, [capitalFRA]);
    expect(
      result.added.filter((key) => result.newlyResolved.includes(key)),
    ).toEqual([]);
  });
});

describe("groupRepairQueue · the same merge the export card does", () => {
  const issues = [capitalITA, capitalFRA, focusName];

  it("produces exactly the export page's attention rows, in the same order", () => {
    const queueGroups = groupRepairQueue(buildRepairQueue(issues));
    const cardGroups = presentCheckIssues(issues).attentionGroups;
    expect(queueGroups.map((group) => group.code)).toEqual(
      cardGroups.map((group) => group.code),
    );
    expect(queueGroups.map((group) => group.count)).toEqual(
      cardGroups.map((group) => group.count),
    );

    expect(queueGroups[0].code).toBe("country.capital_missing");
    expect(queueGroups[0].count).toBe(2);
    expect(queueGroups[0].rows).toHaveLength(2);
  });

  it("keeps every queue item reachable through its group", () => {
    const items = buildRepairQueue(issues);
    const grouped = groupRepairQueue(items).flatMap((group) => group.rows);
    expect(grouped.map((item) => item.key)).toEqual(items.map((i) => i.key));
  });

  it("calls a group resolved only when every member passed a re-check", () => {
    const initial = buildRepairQueue(issues);
    const half = reconcileRepairQueue(initial, [capitalFRA, focusName]).items;
    const [capitals] = groupRepairQueue(half);
    expect(capitals.resolved).toBe(false);
    expect(capitals.openCount).toBe(1);

    const all = reconcileRepairQueue(initial, [focusName]).items;
    const [capitalsDone] = groupRepairQueue(all);
    expect(capitalsDone.resolved).toBe(true);
    expect(capitalsDone.openCount).toBe(0);
  });
});

describe("countGroupsWithEntryPoint · the A9 data-driven claim", () => {
  it("counts only the rows a workbench can actually be opened for", () => {
    const groups = presentCheckIssues([
      capitalITA,
      focusName,

      issue({ code: "descriptor.replace_path_empty", severity: "blocker" }),
    ]).attentionGroups;
    expect(groups).toHaveLength(3);
    expect(countGroupsWithEntryPoint(groups)).toBe(2);
  });

  it("is 0, not a guess, when nothing is deep-linkable", () => {
    const groups = presentCheckIssues([
      issue({ code: "descriptor.replace_path_empty", severity: "blocker" }),
    ]).attentionGroups;
    expect(countGroupsWithEntryPoint(groups)).toBe(0);
  });
});

describe("repairProgress / nextUnresolvedIndex", () => {
  it("counts done vs remaining and never divides by zero", () => {
    expect(repairProgress([])).toEqual({
      total: 0,
      done: 0,
      remaining: 0,
      percent: 0,
    });
    const initial = buildRepairQueue([capitalITA, capitalFRA, focusName]);
    const after = reconcileRepairQueue(initial, [focusName]).items;
    expect(repairProgress(after)).toEqual({
      total: 3,
      done: 2,
      remaining: 1,
      percent: 67,
    });
  });

  it("wraps to the first open item and reports -1 when all are resolved", () => {
    const initial = buildRepairQueue([capitalITA, capitalFRA, focusName]);
    const after = reconcileRepairQueue(initial, [capitalITA]).items;
    expect(nextUnresolvedIndex(after, 1)).toBe(0);
    expect(nextUnresolvedIndex(reconcileRepairQueue(initial, []).items)).toBe(
      -1,
    );
    expect(nextUnresolvedIndex([], 0)).toBe(-1);
  });
});
