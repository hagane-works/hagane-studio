import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { STATE_INDEX_FIXTURE as INDEX } from "@/lib/__fixtures__/vanillaStateIndex";
import { selectVanillaDlcAffectedStates } from "@/lib/vanillaDlcNotice";
import {
  RESTORE_REASON,
  RESTORE_VANILLA_I18N_KEYS,
  RESTORE_VANILLA_NAMESPACE,
  buildRestoreRequest,
  groupResultStateIds,
  hasRestorableStates,
  previewFromIndex,
  restoreErrorMessageKey,
  selectRestorableStates,
  summarizeRestoreResult,
  vanillaSourceErrorCode,
  type RestorableState,
  type RestoreVanillaResult,
} from "@/lib/restoreVanillaContent";

const BEIJING = 608;
const GREAT_WALL = 609;

function state(
  id: number,
  stateId: number,
  rawExtras: Record<string, unknown> | null = {},
): RestorableState {
  return { id, state_id: stateId, raw_extras: rawExtras };
}

describe("selectRestorableStates", () => {
  it("只选『原版有内容 + 本行 raw_extras 为空』的州", () => {
    const states = [
      state(1, BEIJING),
      state(2, GREAT_WALL, { buildings: "9 = {}" }),
      state(3, 999999),
      state(4, GREAT_WALL, null),
    ];
    const sel = selectRestorableStates(states, INDEX);
    expect(sel.stateDbIds).toEqual([1, 4]);
    expect(sel.stateIds).toEqual([GREAT_WALL, BEIJING].sort((a, b) => a - b));
    expect(hasRestorableStates(sel)).toBe(true);
  });

  it("🔴 判据与 β2 的 finding 逐条相同 —— 同一批州,两个模块必须给出同一个集合", () => {
    const states = [
      state(10, BEIJING),
      state(11, GREAT_WALL, { "history.1939.1.1": "…" }),
      state(12, 2),
      state(13, 999999),
      state(14, BEIJING),
    ];
    const fromNotice = selectVanillaDlcAffectedStates(states, INDEX);
    const fromRestore = selectRestorableStates(states, INDEX).stateIds;
    expect(fromRestore).toEqual(fromNotice);
  });

  it("🔴 回归守卫:曾经从索引里消失的 3 个州,现在必须选得中", () => {
    const onceUnreachable = [379, 987, 1057];
    for (const id of onceUnreachable) {
      expect(INDEX.content[id], `state ${id} 必须在索引里`).toBeDefined();
      expect(
        selectRestorableStates([state(1, id)], INDEX).stateDbIds,
        `state ${id} 必须选得中(否则作者既不被告知也点不到)`,
      ).toEqual([1]);
    }

    expect(INDEX.content[379]!.otherContent).toBe(1);
    expect(INDEX.content[987]!.resistance).toBe(3);
  });

  it("空输入 / 脏输入不炸", () => {
    expect(selectRestorableStates(null, INDEX).stateDbIds).toEqual([]);
    expect(selectRestorableStates([], INDEX).stateIds).toEqual([]);
    expect(
      selectRestorableStates(
        [
          { id: "x", state_id: BEIJING },
          null,
          { id: 5, state_id: Number.NaN },
        ] as unknown as Parameters<typeof selectRestorableStates>[0],
        INDEX,
      ).stateDbIds,
    ).toEqual([]);
  });

  it("重复 state_id 的每一行都要发(它们各自都是空的),但展示时去重", () => {
    const sel = selectRestorableStates(
      [state(7, BEIJING), state(8, BEIJING)],
      INDEX,
    );
    expect(sel.stateDbIds).toEqual([7, 8]);
    expect(sel.stateIds).toEqual([BEIJING]);
  });
});

describe("buildRestoreRequest", () => {
  it("发显式 id 列表,不发 all", () => {
    const sel = selectRestorableStates(
      [state(3, BEIJING), state(1, GREAT_WALL)],
      INDEX,
    );
    expect(buildRestoreRequest(sel)).toEqual({ state_db_ids: [1, 3] });
  });

  it("dry_run 只在要预览时出现(缺省不带,免得后端把 false 也当一次预览分支)", () => {
    const sel = selectRestorableStates([state(1, BEIJING)], INDEX);
    expect(buildRestoreRequest(sel)).not.toHaveProperty("dry_run");
    expect(buildRestoreRequest(sel, { dryRun: true })).toEqual({
      state_db_ids: [1],
      dry_run: true,
    });
  });

  it("🔴 永远不发 all:true —— 弹窗说的『这 N 个州』必须与真正被处理的那批同源", () => {
    const sel = selectRestorableStates([state(1, BEIJING)], INDEX);
    expect(buildRestoreRequest(sel, { dryRun: true })).not.toHaveProperty(
      "all",
    );
  });
});

describe("previewFromIndex", () => {
  it("数字取自 β2 索引里那个州的内容", () => {
    const beijing = INDEX.content[BEIJING];
    expect(beijing).toBeDefined();
    const loss = previewFromIndex([state(1, BEIJING)], INDEX);
    expect(loss.states).toBe(1);
    expect(loss.navalBases).toBe(beijing!.navalBases ?? 0);
    expect(loss.landmarks).toBe((beijing!.landmarks ?? []).length);
    expect(loss.landmarkKeys).toEqual(["landmark_forbidden_city"]);
    expect(loss.datedBlocks).toBe(beijing!.datedBlocks ?? 0);
    expect(loss.hasDlc).toBe(beijing!.hasDlc ?? 0);
  });

  it("已有内容的州不计入预览", () => {
    expect(
      previewFromIndex([state(1, BEIJING, { buildings: "x" })], INDEX).states,
    ).toBe(0);
  });
});

describe("summarizeRestoreResult", () => {
  const result: RestoreVanillaResult = {
    dry_run: true,
    restored: 2,
    skipped: 1,
    missing: 3,
    restored_state_ids: [BEIJING, GREAT_WALL],
    skipped_state_ids: [2],
    missing_state_ids: [90001, 90003, 90002],
    totals: {
      naval_bases: 2,
      bunkers: 4,
      coastal_bunkers: 3,
      dams: 1,
      landmarks: [
        "landmark_forbidden_city",
        "landmark_great_wall_section",
        "landmark_great_wall_section",
      ],
      dated_blocks: 3,
      has_dlc: 4,
      force_link: true,
      resistance: 2,
      other_content: 5,
    },
    details: [],
    vanilla_files: 1081,
    vanilla_states: 1081,
  };

  it("念的是服务端给的数字", () => {
    const s = summarizeRestoreResult(result);
    expect(s.dryRun).toBe(true);
    expect(s.restored).toBe(2);
    expect(s.skipped).toBe(1);
    expect(s.missing).toBe(3);
    expect(s.navalBases).toBe(2);

    expect(s.bunkers).toBe(7);
    expect(s.dams).toBe(1);
    expect(s.datedBlocks).toBe(3);
    expect(s.hasDlc).toBe(4);
    expect(s.forceLink).toBe(true);
    expect(s.hasContent).toBe(true);
  });

  it("🔴 地标按**个数**算,不按名字去重 —— 长城是多段,去重会低报作者的损失", () => {
    const s = summarizeRestoreResult(result);
    expect(s.landmarks).toBe(3);
    expect(s.landmarkKeys).toEqual([
      "landmark_forbidden_city",
      "landmark_great_wall_section",
    ]);
  });

  it("全零时 hasContent=false(弹窗不该列一张空清单)", () => {
    const s = summarizeRestoreResult({
      ...result,
      totals: {
        naval_bases: 0,
        bunkers: 0,
        coastal_bunkers: 0,
        dams: 0,
        landmarks: [],
        dated_blocks: 0,
        has_dlc: 0,
        force_link: false,
        resistance: 0,
        other_content: 0,
      },
    });
    expect(s.hasContent).toBe(false);
  });

  it("缺字段 / null 不炸(后端加字段时前端不该白屏)", () => {
    expect(summarizeRestoreResult(null).restored).toBe(0);
    expect(
      summarizeRestoreResult({ restored: 1 } as unknown as RestoreVanillaResult)
        .hasContent,
    ).toBe(false);
  });

  it("state id 分组去重升序", () => {
    expect(groupResultStateIds(result)).toEqual({
      restored: [BEIJING, GREAT_WALL],
      skipped: [2],
      missing: [90001, 90002, 90003],
    });
  });
});

describe("错误分诊", () => {
  const apiError = (status: number, payload: unknown) => ({ status, payload });

  it("认出三种『游戏目录里的原版州文件读不到』", () => {
    for (const code of [
      "vanilla_root_unconfigured",
      "vanilla_states_dir_missing",
      "vanilla_states_empty",
    ]) {
      expect(vanillaSourceErrorCode(apiError(503, { code }))).toBe(code);
    }
  });

  it("不再认 vanilla_version_mismatch", () => {
    expect(
      vanillaSourceErrorCode(
        apiError(503, { code: "vanilla_version_mismatch" }),
      ),
    ).toBeNull();
  });

  it("别的 503 不算(它可能是真的服务挂了,文案不能说成缺文件)", () => {
    expect(
      vanillaSourceErrorCode(apiError(503, { code: "db_down" })),
    ).toBeNull();
    expect(
      vanillaSourceErrorCode(apiError(503, "state store unavailable")),
    ).toBeNull();
    expect(
      vanillaSourceErrorCode(
        apiError(500, { code: "vanilla_root_unconfigured" }),
      ),
    ).toBeNull();
    expect(vanillaSourceErrorCode(null)).toBeNull();
    expect(vanillaSourceErrorCode(new Error("boom"))).toBeNull();
  });

  it("每种 code 都映射到一条 key,未知错误落到通用文案", () => {
    expect(
      restoreErrorMessageKey(
        apiError(503, { code: "vanilla_root_unconfigured" }),
      ),
    ).toBe("errorVanillaRootUnconfigured");
    expect(
      restoreErrorMessageKey(
        apiError(503, { code: "vanilla_states_dir_missing" }),
      ),
    ).toBe("errorVanillaStatesDirMissing");
    expect(
      restoreErrorMessageKey(apiError(503, { code: "vanilla_states_empty" })),
    ).toBe("errorVanillaStatesEmpty");
    expect(restoreErrorMessageKey(new Error("boom"))).toBe("errorGeneric");
  });
});

describe("契约常量", () => {
  it("reason 取值与 Go 侧常量一致", () => {
    expect(Object.values(RESTORE_REASON)).toEqual([
      "already_has_content",
      "not_in_vanilla",
      "vanilla_has_no_extras",
    ]);
  });
});

describe(`i18n 语料 · ${RESTORE_VANILLA_NAMESPACE}`, () => {
  const LOCALES = ["en", "zh", "ja", "ko", "de", "fr", "es", "ru"] as const;
  const root = path.resolve(process.cwd(), "messages");

  it.each(LOCALES)("%s 有全部 key", (locale) => {
    const file = path.join(root, locale, `${RESTORE_VANILLA_NAMESPACE}.json`);
    let parsed: Record<string, unknown>;
    try {
      parsed = JSON.parse(readFileSync(file, "utf8"));
    } catch {
      throw new Error(
        `缺语料文件 ${file} —— β3 的 seam:中央需要新建 ` +
          `messages/<locale>/${RESTORE_VANILLA_NAMESPACE}.json,` +
          `key 清单见 RESTORE_VANILLA_I18N_KEYS(${RESTORE_VANILLA_I18N_KEYS.length} 条)。`,
      );
    }
    const missing = RESTORE_VANILLA_I18N_KEYS.filter(
      (key) => typeof parsed[key] !== "string" || parsed[key] === "",
    );
    expect(missing, `${locale} 缺 key`).toEqual([]);
  });
});
