import { describe, expect, it } from "vitest";
import { STATE_INDEX_FIXTURE as INDEX } from "@/lib/__fixtures__/vanillaStateIndex";
import {
  VANILLA_DLC_NOTICE_CODE,
  VANILLA_DLC_NOTICE_EXPECTED_TIER,
  buildVanillaDlcNotice,
  selectVanillaDlcAffectedStates,
  vanillaDlcNamesForStates,
  type DlcNoticeState,
} from "@/lib/vanillaDlcNotice";
import { EXPORT_FINDINGS } from "@/lib/exportFindingsMap";
import type { State } from "@/types/state";
import type { Project } from "@/types/project";

const ALXA = 1040;
const SINGLE_DLC_STATE = 2;

const UNAFFECTED_STATE = 3;

function state(
  partial: Partial<DlcNoticeState> & { state_id: number },
): DlcNoticeState {
  return { raw_extras: {}, ...partial };
}

describe("selectVanillaDlcAffectedStates · the hit predicate", () => {
  it("matches project states whose vanilla counterpart has DLC history", () => {
    expect(
      selectVanillaDlcAffectedStates(
        [
          state({ state_id: ALXA }),
          state({ state_id: UNAFFECTED_STATE }),
          state({ state_id: SINGLE_DLC_STATE }),
        ],
        INDEX,
      ),
    ).toEqual([SINGLE_DLC_STATE, ALXA]);
  });

  it("SKIPS rows that still carry preserved raw script (never overclaims)", () => {
    expect(
      selectVanillaDlcAffectedStates(
        [
          state({
            state_id: ALXA,
            raw_extras: { history: "1936.1.1 = { ... }" },
          }),
        ],
        INDEX,
      ),
    ).toEqual([]);
  });

  it("counts DISTINCT state ids when the project holds duplicate rows", () => {
    expect(
      selectVanillaDlcAffectedStates(
        [state({ state_id: ALXA }), state({ state_id: ALXA })],
        INDEX,
      ),
    ).toEqual([ALXA]);
  });

  it("tolerates empty, null and malformed input without throwing", () => {
    expect(selectVanillaDlcAffectedStates([], INDEX)).toEqual([]);
    expect(selectVanillaDlcAffectedStates(null, INDEX)).toEqual([]);
    expect(selectVanillaDlcAffectedStates(undefined, INDEX)).toEqual([]);
    expect(
      selectVanillaDlcAffectedStates(
        [{ state_id: Number.NaN }, { state_id: ALXA, raw_extras: null }],
        INDEX,
      ),
    ).toEqual([ALXA]);
  });

  it("treats a missing raw_extras bag as 'no preserved script'", () => {
    expect(selectVanillaDlcAffectedStates([{ state_id: ALXA }], INDEX)).toEqual(
      [ALXA],
    );
  });
});

describe("vanillaDlcNamesForStates", () => {
  it("returns the sorted distinct DLC union for the given states", () => {
    expect(vanillaDlcNamesForStates([ALXA, SINGLE_DLC_STATE], INDEX)).toEqual([
      "Gotterdammerung",
      "No Compromise, No Surrender",
      "Waking the Tiger",
    ]);
  });

  it("ignores ids that are not in the dataset", () => {
    expect(vanillaDlcNamesForStates([UNAFFECTED_STATE], INDEX)).toEqual([]);
  });
});

describe("buildVanillaDlcNotice · the synthesized ℹ️ finding", () => {
  it("produces one info finding with the affected-state count as {count}", () => {
    const notice = buildVanillaDlcNotice(
      [
        state({ state_id: ALXA }),
        state({ state_id: SINGLE_DLC_STATE }),
        state({ state_id: UNAFFECTED_STATE }),
      ],
      { vanillaStates: INDEX },
    );
    expect(notice).not.toBeNull();
    expect(notice?.code).toBe(VANILLA_DLC_NOTICE_CODE);
    expect(notice?.severity).toBe("info");

    expect(notice?.detail?.count).toBe(2);
    expect(notice?.evidence?.state_ids).toEqual([SINGLE_DLC_STATE, ALXA]);
    expect(notice?.evidence?.dlc).toEqual([
      "Gotterdammerung",
      "No Compromise, No Surrender",
      "Waking the Tiger",
    ]);
  });

  it("knows nothing about the base game without the state index", () => {
    expect(buildVanillaDlcNotice([state({ state_id: ALXA })])).toBeNull();
    expect(
      buildVanillaDlcNotice([state({ state_id: ALXA })], {
        loadedExistingMod: false,
      }),
    ).toBeNull();
  });

  it("is NOT unconditionally shown — no affected state ⇒ no finding", () => {
    expect(
      buildVanillaDlcNotice([state({ state_id: UNAFFECTED_STATE })], {
        vanillaStates: INDEX,
      }),
    ).toBeNull();
    expect(buildVanillaDlcNotice([], { vanillaStates: INDEX })).toBeNull();
    expect(buildVanillaDlcNotice(null, { vanillaStates: INDEX })).toBeNull();
    expect(
      buildVanillaDlcNotice(undefined, { vanillaStates: INDEX }),
    ).toBeNull();
  });

  it("β3 已落地:fix_hint 指向「从原版补齐」,且形状能当按钮文字", () => {
    const notice = buildVanillaDlcNotice([state({ state_id: ALXA })], {
      vanillaStates: INDEX,
    });
    const hint = notice?.fix_hint ?? "";
    expect(hint).not.toBe("");

    expect(hint).toMatch(/\s/);
    expect(hint).not.toMatch(/[.!?。！？；;]/);
    expect(hint.length).toBeLessThanOrEqual(68);
    expect(notice?.entity_ref).toBeNull();
  });

  it("β2.1 · detail 带上新的具名类别与兜底桶", () => {
    const notice = buildVanillaDlcNotice(
      [state({ state_id: 987 }), state({ state_id: 379 })],
      { vanillaStates: INDEX },
    );
    expect(notice?.detail?.count).toBe(2);
    expect(notice?.detail?.resistance).toBe(3);
    expect(notice?.detail?.otherContent).toBe(1);
  });

  it("suppresses on mod-loaded projects, emits when seeded or unknown", () => {
    const states = [state({ state_id: ALXA })];

    expect(
      buildVanillaDlcNotice(states, {
        loadedExistingMod: true,
        vanillaStates: INDEX,
      }),
    ).toBeNull();
    expect(
      buildVanillaDlcNotice(states, {
        loadedExistingMod: false,
        vanillaStates: INDEX,
      }),
    ).not.toBeNull();
    expect(
      buildVanillaDlcNotice(states, {
        loadedExistingMod: undefined,
        vanillaStates: INDEX,
      }),
    ).not.toBeNull();
    expect(
      buildVanillaDlcNotice(states, { vanillaStates: INDEX }),
    ).not.toBeNull();
  });

  it("SEAM · accepts the store's State[] and the project flag without a cast", () => {
    const fromStore: readonly DlcNoticeState[] = [] as State[];
    const flag: boolean | null | undefined = ({} as Project)
      .is_loaded_existing_mod;
    expect(
      buildVanillaDlcNotice(fromStore, {
        loadedExistingMod: flag,
        vanillaStates: INDEX,
      }),
    ).toBeNull();
  });

  it("SEAM · 注册表 tier 应随 β3 升成可行动的那一档(未落地前此条为红)", () => {
    const meta = EXPORT_FINDINGS[VANILLA_DLC_NOTICE_CODE];
    expect(meta).toBeDefined();
    expect(meta.surface).toBe("export");
    expect(meta.i18nKey).toBe(VANILLA_DLC_NOTICE_CODE);
    expect(meta.tier).toBe(VANILLA_DLC_NOTICE_EXPECTED_TIER);
  });
});
