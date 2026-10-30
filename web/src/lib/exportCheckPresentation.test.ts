import { describe, expect, it } from "vitest";
import type { CtdIssue } from "@/lib/api/export";
import {
  MAP_INTEGRITY_LEGACY_UNVERIFIED_CODES,
  RAW_ICU_FINDING_KEYS,
  countDistinctEntities,
  formatRawFindingTemplate,
  groupAttentionFindings,
  legacyMapStateCount,
  presentCheckIssues,
  synthesizeClientFindings,
} from "@/lib/exportCheckPresentation";
import { EXPORT_FINDINGS } from "@/lib/exportFindingsMap";
import {
  VANILLA_DLC_NOTICE_CODE,
  VANILLA_DLC_NOTICE_EXPECTED_TIER,
} from "@/lib/vanillaDlcNotice";
import { STATE_INDEX_FIXTURE as INDEX } from "@/lib/__fixtures__/vanillaStateIndex";

function issue(partial: Partial<CtdIssue> & { code: string }): CtdIssue {
  return {
    entity_ref: null,
    severity: "warning",
    message: `backend message for ${partial.code}`,
    fix_hint: "",
    ...partial,
  };
}

describe("presentCheckIssues · four display tiers", () => {
  it("routes known codes to the tier the display map declares", () => {
    const result = presentCheckIssues([
      issue({ code: "country.capital_missing", severity: "blocker" }),
      issue({ code: "focus_file_policy_drift", severity: "info" }),
      issue({ code: "focus_registry_unavailable", severity: "info" }),
      issue({ code: "timeline.start_date_ok", severity: "info" }),
    ]);
    expect(result.attention.map((f) => f.issue.code)).toEqual([
      "country.capital_missing",
    ]);
    expect(result.auto.map((f) => f.issue.code)).toEqual([
      "focus_file_policy_drift",
    ]);
    expect(result.health.map((f) => f.issue.code)).toEqual([
      "focus_registry_unavailable",
    ]);
    expect(result.pass.map((f) => f.issue.code)).toEqual([
      "timeline.start_date_ok",
    ]);
  });

  it("orders the attention card blockers-first", () => {
    const result = presentCheckIssues([
      issue({
        code: "focus.display_name.missing_created",
        severity: "warning",
      }),
      issue({ code: "country.capital_missing", severity: "blocker" }),
    ]);
    expect(result.attention.map((f) => f.issue.severity)).toEqual([
      "blocker",
      "warning",
    ]);
  });

  it("NEVER drops an unknown code — non-info goes to attention with meta null", () => {
    const result = presentCheckIssues([
      issue({
        code: "totally_new_backend_code",
        severity: "blocker",
        message: "raw backend sentence the user must see",
      }),
    ]);
    expect(result.attention).toHaveLength(1);
    expect(result.attention[0].meta).toBeNull();
    expect(result.attention[0].issue.message).toBe(
      "raw backend sentence the user must see",
    );
  });

  it("keeps unknown info-severity codes visible in the dialog's auto list", () => {
    const result = presentCheckIssues([
      issue({ code: "unknown_info_code", severity: "info" }),
    ]);
    expect(result.attention).toHaveLength(0);
    expect(result.auto.map((f) => f.issue.code)).toEqual(["unknown_info_code"]);
    expect(result.auto[0].meta).toBeNull();
  });

  it("G-11-a: aggregates the whole map_integrity legacy family into ONE line", () => {
    const legacy = [...MAP_INTEGRITY_LEGACY_UNVERIFIED_CODES].map((code) =>
      issue({ code, severity: "warning" }),
    );
    const result = presentCheckIssues([
      ...legacy,
      issue({ code: "country.capital_missing", severity: "blocker" }),
    ]);

    for (const tier of [
      result.attention,
      result.auto,
      result.health,
      result.pass,
    ]) {
      for (const f of tier) {
        expect(
          MAP_INTEGRITY_LEGACY_UNVERIFIED_CODES.has(String(f.issue.code)),
        ).toBe(false);
      }
    }
    expect(result.legacyMap).not.toBeNull();
    expect(result.legacyMap?.issues).toHaveLength(legacy.length);

    expect(result.legacyMap?.count).toBe(legacy.length);
  });

  it("legacy aggregate counts DISTINCT states when findings carry state_ids", () => {
    const issues = [
      issue({
        code: "map_integrity.final_membership_not_verified",
        severity: "warning",
        evidence: { state_ids: [10, 11] },
      }),
      issue({
        code: "map_integrity.overlay_not_verified",
        severity: "warning",
        evidence: { state_ids: [11, 12] },
      }),
    ];
    expect(legacyMapStateCount(issues)).toBe(3);
    expect(presentCheckIssues(issues).legacyMap?.count).toBe(3);
  });

  it("legacy family codes exist verbatim in the backend map_integrity module list", () => {
    for (const code of MAP_INTEGRITY_LEGACY_UNVERIFIED_CODES) {
      expect(code.startsWith("map_integrity.")).toBe(true);
      expect(EXPORT_FINDINGS[code]).toBeUndefined();
    }
  });
});

describe("SPEC-8 · client-synthesized vanilla-DLC notice", () => {
  const AFFECTED = { state_id: 1040, raw_extras: {} };

  const UNAFFECTED = { state_id: 3, raw_extras: {} };

  it("落在 ⚠️ attention 卡片,不再是 ℹ️ auto 列表", () => {
    const result = presentCheckIssues([], {
      states: [AFFECTED],
      vanillaStates: INDEX,
    });
    expect(result.auto).toHaveLength(0);
    expect(result.attention.map((f) => f.issue.code)).toEqual([
      VANILLA_DLC_NOTICE_CODE,
    ]);

    expect(result.attention[0].meta?.tier).toBe(
      VANILLA_DLC_NOTICE_EXPECTED_TIER,
    );
    expect(result.attention[0].issue.detail?.count).toBe(1);
  });

  it("counts only affected states, deduplicated", () => {
    const result = presentCheckIssues([], {
      states: [AFFECTED, AFFECTED, UNAFFECTED, { state_id: 2, raw_extras: {} }],
      vanillaStates: INDEX,
    });
    expect(result.attention[0].issue.detail?.count).toBe(2);
  });

  it("is not produced when no state is affected", () => {
    const result = presentCheckIssues([], {
      states: [UNAFFECTED],
      vanillaStates: INDEX,
    });
    expect(result.auto).toHaveLength(0);
  });

  it("is not produced without options — existing call sites are unchanged", () => {
    expect(presentCheckIssues([]).auto).toHaveLength(0);
    expect(synthesizeClientFindings([], undefined)).toEqual([]);

    expect(presentCheckIssues([], {}).auto).toHaveLength(0);

    expect(
      presentCheckIssues([], { states: [AFFECTED] }).attention,
    ).toHaveLength(0);
  });

  it("yields to a backend producer instead of duplicating the row", () => {
    const fromBackend = issue({
      code: VANILLA_DLC_NOTICE_CODE,
      severity: "info",
      detail: { count: 99 },
    });
    const result = presentCheckIssues([fromBackend], {
      states: [AFFECTED],
      vanillaStates: INDEX,
    });
    expect(result.attention).toHaveLength(1);
    expect(result.attention[0].issue.detail?.count).toBe(99);
  });

  it("respects the mod-loaded suppression flag", () => {
    expect(
      presentCheckIssues([], {
        states: [AFFECTED],
        loadedExistingMod: true,
        vanillaStates: INDEX,
      }).attention,
    ).toHaveLength(0);
    expect(
      presentCheckIssues([], {
        states: [AFFECTED],
        loadedExistingMod: false,
        vanillaStates: INDEX,
      }).attention,
    ).toHaveLength(1);
  });

  it("REGRESSION · every mapped finding keeps its declared tier, with and without the notice", () => {
    const CLIENT_SYNTHESIZED = new Set([
      VANILLA_DLC_NOTICE_CODE,
      "impassable_category_mismatch",

      "impassable_without_force_link",
    ]);
    const backendCodes = Object.keys(EXPORT_FINDINGS).filter(
      (code) => !CLIENT_SYNTHESIZED.has(code),
    );
    expect(backendCodes.length).toBe(
      Object.keys(EXPORT_FINDINGS).length - CLIENT_SYNTHESIZED.size,
    );

    expect(backendCodes.length).toBe(121);

    const all = backendCodes.map((code) => issue({ code, severity: "info" }));
    const tiersOf = (result: ReturnType<typeof presentCheckIssues>) => ({
      attention: result.attention.map((f) => f.issue.code).sort(),
      auto: result.auto.map((f) => f.issue.code).sort(),
      health: result.health.map((f) => f.issue.code).sort(),
      pass: result.pass.map((f) => f.issue.code).sort(),
    });

    const baseline = tiersOf(presentCheckIssues(all));

    for (const code of backendCodes) {
      const tier = EXPORT_FINDINGS[code].tier;
      const bucket = tier === "attn" ? baseline.attention : baseline[tier];
      expect(bucket).toContain(code);
    }

    expect(
      baseline.attention.length +
        baseline.auto.length +
        baseline.health.length +
        baseline.pass.length,
    ).toBe(backendCodes.length);

    const withNotice = tiersOf(
      presentCheckIssues(all, { states: [AFFECTED], vanillaStates: INDEX }),
    );
    expect(withNotice.auto).toEqual(baseline.auto);
    expect(withNotice.health).toEqual(baseline.health);
    expect(withNotice.pass).toEqual(baseline.pass);

    expect(withNotice.attention).toEqual(
      [...baseline.attention, VANILLA_DLC_NOTICE_CODE].sort(),
    );
  });
});

describe("formatRawFindingTemplate · literal-brace HOI4 copy", () => {
  it("interpolates {args} while leaving literal quoted braces intact", () => {
    const out = formatRawFindingTemplate(
      "{file} 第 {line} 的 '{' 缺少对应的 '}'",
      { file: "common/ideas/x.txt", line: 42 },
    );
    expect(out).toBe("common/ideas/x.txt 第 42 的 '{' 缺少对应的 '}'");
  });

  it("leaves placeholders without a value untouched (no undefined leakage)", () => {
    expect(
      formatRawFindingTemplate("{file} at {line}", { file: "a.txt" }),
    ).toBe("a.txt at {line}");
  });

  it("covers exactly the two clausewitz literal-brace corpus keys", () => {
    expect([...RAW_ICU_FINDING_KEYS].sort()).toEqual([
      "clausewitz_stray_closing_brace",
      "clausewitz_unclosed_block",
    ]);
    for (const key of RAW_ICU_FINDING_KEYS) {
      const hasMapEntry = Object.values(EXPORT_FINDINGS).some(
        (meta) => meta.i18nKey === key,
      );
      expect(hasMapEntry).toBe(true);
    }
  });
});

describe("A3-b · 客户端合成的「不可通行分类不匹配」提醒", () => {
  const MISMATCH = {
    state_id: 9001,
    impassable: true,
    state_category: "rural",
  };
  const OK = { state_id: 9002, impassable: true, state_category: "wasteland" };

  it("🔴 真的接上了 —— 注册进 EXPORT_FINDINGS 不等于会被合成出来", () => {
    const out = synthesizeClientFindings([], { states: [MISMATCH] });
    expect(out.map((i) => i.code)).toContain("impassable_category_mismatch");
  });

  it("没有命中就不出现", () => {
    const out = synthesizeClientFindings([], { states: [OK] });
    expect(out.map((i) => i.code)).not.toContain(
      "impassable_category_mismatch",
    );
  });

  it("后端已经报了同一个 code 就不重复合成", () => {
    const out = synthesizeClientFindings(
      [{ code: "impassable_category_mismatch" } as never],
      { states: [MISMATCH] },
    );
    expect(out.map((i) => i.code)).not.toContain(
      "impassable_category_mismatch",
    );
  });

  it("走完整 presentCheckIssues 后能被渲染层看到,且落在非阻断档", () => {
    const result = presentCheckIssues([], { states: [MISMATCH] });
    const codesIn = (rows: typeof result.auto) => rows.map((f) => f.issue.code);

    expect(codesIn(result.auto)).toContain("impassable_category_mismatch");
    expect(codesIn(result.attention)).not.toContain(
      "impassable_category_mismatch",
    );
  });
});

describe("A3-a · 客户端合成的「不可通行州没指定归属邻州」告知", () => {
  const MISSING = {
    state_id: 8001,
    impassable: true,
    force_link_ownership_to: null,
  };
  const OK = {
    state_id: 8002,
    impassable: true,
    force_link_ownership_to: 8001,
  };

  it("🔴 真的接上了 —— 注册进 EXPORT_FINDINGS 不等于会被合成出来", () => {
    const out = synthesizeClientFindings([], { states: [MISSING] });
    expect(out.map((i) => i.code)).toContain("impassable_without_force_link");
  });

  it("没有命中就不出现", () => {
    const out = synthesizeClientFindings([], { states: [OK] });
    expect(out.map((i) => i.code)).not.toContain(
      "impassable_without_force_link",
    );
  });

  it("后端已经报了同一个 code 就不重复合成", () => {
    const out = synthesizeClientFindings(
      [{ code: "impassable_without_force_link" } as never],
      { states: [MISSING] },
    );
    expect(out.map((i) => i.code)).not.toContain(
      "impassable_without_force_link",
    );
  });

  it("走完整 presentCheckIssues 后能被渲染层看到,且落在非阻断档", () => {
    const result = presentCheckIssues([], { states: [MISSING] });
    const codesIn = (rows: typeof result.auto) => rows.map((f) => f.issue.code);

    expect(codesIn(result.auto)).toContain("impassable_without_force_link");
    expect(codesIn(result.attention)).not.toContain(
      "impassable_without_force_link",
    );
  });
});

describe("countDistinctEntities · deduped count, never a fake zero", () => {
  const on = (type: string, id: string | number) =>
    issue({ code: "x", entity_ref: { type, id } });

  it("counts one entity once no matter how many findings name it", () => {
    expect(
      countDistinctEntities([on("country", "ITA"), on("country", "ITA")]),
    ).toBe(1);
  });

  it("separates the same id under different entity types", () => {
    expect(countDistinctEntities([on("country", 1), on("focus", 1)])).toBe(2);
  });

  it("counts entity-less findings individually instead of collapsing them", () => {
    expect(
      countDistinctEntities([
        issue({ code: "descriptor.replace_path_empty" }),
        issue({ code: "descriptor.replace_path_empty" }),
        issue({ code: "descriptor.replace_path_empty" }),
      ]),
    ).toBe(3);
    expect(countDistinctEntities([])).toBe(0);
  });

  it("adds attributed and unattributed findings instead of dropping either", () => {
    expect(
      countDistinctEntities([
        on("country", "ITA"),
        on("country", "ITA"),
        issue({ code: "x" }),
      ]),
    ).toBe(2);
  });
});

describe("attentionGroups · the attention card's rows", () => {
  it("collapses one code to one row and keeps blockers first", () => {
    const result = presentCheckIssues([
      issue({
        code: "focus.display_name.missing_created",
        entity_ref: { type: "focus", id: 1 },
      }),
      issue({
        code: "focus.display_name.missing_created",
        entity_ref: { type: "focus", id: 2 },
      }),
      issue({
        code: "country.capital_missing",
        severity: "blocker",
        entity_ref: { type: "country", id: "ITA" },
      }),
    ]);
    expect(result.attention).toHaveLength(3);
    expect(result.attentionGroups.map((g) => g.code)).toEqual([
      "country.capital_missing",
      "focus.display_name.missing_created",
    ]);
    expect(result.attentionGroups.map((g) => g.count)).toEqual([1, 2]);
    expect(result.attentionGroups[0].hasBlocker).toBe(true);
    expect(result.attentionGroups[1].hasBlocker).toBe(false);
  });

  it("uses the first finding of the group as the row's representative", () => {
    const groups = groupAttentionFindings(
      presentCheckIssues([
        issue({
          code: "idea_no_modifier",
          entity_ref: { type: "idea", id: "aaa" },
        }),
        issue({
          code: "idea_no_modifier",
          entity_ref: { type: "idea", id: "bbb" },
        }),
      ]).attention,
    );
    expect(groups).toHaveLength(1);
    expect(groups[0].representative.issue.entity_ref?.id).toBe("aaa");
    expect(groups[0].rows).toHaveLength(2);
  });

  it("marks a mixed-severity group as a blocker row", () => {
    const groups = groupAttentionFindings(
      presentCheckIssues([
        issue({
          code: "province_double_mapping",
          entity_ref: { type: "state", id: 1 },
        }),
        issue({
          code: "province_double_mapping",
          severity: "blocker",
          entity_ref: { type: "state", id: 2 },
        }),
      ]).attention,
    );
    expect(groups).toHaveLength(1);
    expect(groups[0].hasBlocker).toBe(true);
  });

  it("never merges two different codes, however similar", () => {
    const groups = presentCheckIssues([
      issue({
        code: "focus_layout_same_cell",
        entity_ref: { type: "focus", id: 1 },
      }),
      issue({
        code: "focus_layout_adjacent_same_row",
        entity_ref: { type: "focus", id: 1 },
      }),
    ]).attentionGroups;
    expect(groups).toHaveLength(2);
  });
});

const MEASURED_CODE_FREQUENCIES: ReadonlyArray<readonly [string, number]> = [
  ["focus.display_name.unproven_fallback", 176],
  ["idea_no_modifier", 55],
  ["focus_layout_same_cell", 48],
  ["ideology_undefined_standalone", 46],
  ["character_no_military_traits", 39],
  ["bookmark.selectable_ideology_missing", 35],
  ["focus_tree_no_nodes", 24],
  ["tech_folder_unknown", 17],
  ["localisation_reference_not_in_project", 8],
  ["focus_layout_adjacent_same_row", 6],
  ["ideology_registry_parent_missing", 5],
  ["tech_circular_prereq", 4],
  ["event_namespace_mismatch", 3],
  ["mod_folder_name", 2],
  ["province_double_mapping", 1],
  ["focus_created_primary_id_collision", 1],
  ["focus_shared_pool_scope_ignored", 1],
  ["doctrine_track_unresolved", 1],
  ["event_no_options", 1],
  ["focus_effective_default_multiple", 1],
];

describe("A2 compression on the measured findings corpus", () => {
  const corpus: CtdIssue[] = MEASURED_CODE_FREQUENCIES.flatMap(
    ([code, times]) =>
      Array.from({ length: times }, (_unused, i) =>
        issue({ code, entity_ref: { type: "focus", id: `${code}#${i}` } }),
      ),
  );

  it("turns 472 attention findings into 19 rows", () => {
    expect(corpus).toHaveLength(474);
    const result = presentCheckIssues(corpus);

    expect(result.attention).toHaveLength(472);
    expect(result.attentionGroups).toHaveLength(19);

    expect(
      result.attentionGroups.reduce((sum, group) => sum + group.rows.length, 0),
    ).toBe(472);
    expect(
      result.attentionGroups.reduce((sum, group) => sum + group.count, 0),
    ).toBe(472);
  });

  it("collapses further when one entity is reported repeatedly", () => {
    const sameEntity = MEASURED_CODE_FREQUENCIES.flatMap(([code, times]) =>
      Array.from({ length: times }, () =>
        issue({ code, entity_ref: { type: "focus", id: code } }),
      ),
    );
    const groups = presentCheckIssues(sameEntity).attentionGroups;
    expect(groups).toHaveLength(19);
    expect(groups.every((group) => group.count === 1)).toBe(true);
  });
});
