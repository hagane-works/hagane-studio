import { describe, expect, it } from "vitest";

import {
  nextDraftOnEntityChange,
  nextDraftOnEntityRefetch,
  stableFingerprint,
} from "./draftReseed";

type CharDraft = {
  id: number;
  name_en: string;
  role_type: string;
  large_portrait: string;
  small_portrait: string;
  is_portrait_modified: boolean;
};

const PORTRAIT_FIELDS = [
  "large_portrait",
  "small_portrait",
  "is_portrait_modified",
] as const;

describe("nextDraftOnEntityRefetch — portrait upload must not reset other settings", () => {
  it("same-id refetch (portrait upload) KEEPS unsaved edits + syncs only portrait fields", () => {
    const draftWithUnsavedEdits: CharDraft = {
      id: 7,
      name_en: "Edited Name",
      role_type: "general",
      large_portrait: "",
      small_portrait: "",
      is_portrait_modified: false,
    };

    const refetched: CharDraft = {
      id: 7,
      name_en: "Old Server Name",
      role_type: "advisor",
      large_portrait: "gfx/leaders/USA/portrait_7.png",
      small_portrait: "gfx/advisors/USA/advisor_7.png",
      is_portrait_modified: true,
    };

    const { draft, seededId } = nextDraftOnEntityRefetch(
      7,
      refetched.id,
      draftWithUnsavedEdits,
      refetched,
      PORTRAIT_FIELDS,
    );

    expect(draft.name_en).toBe("Edited Name");
    expect(draft.role_type).toBe("general");

    expect(draft.large_portrait).toBe("gfx/leaders/USA/portrait_7.png");
    expect(draft.small_portrait).toBe("gfx/advisors/USA/advisor_7.png");
    expect(draft.is_portrait_modified).toBe(true);
    expect(seededId).toBe(7);

    expect(draft).not.toEqual(refetched);
  });

  it("switching to a different character (id changes) reseeds the full draft", () => {
    const current: CharDraft = {
      id: 7,
      name_en: "A",
      role_type: "general",
      large_portrait: "",
      small_portrait: "",
      is_portrait_modified: false,
    };
    const other: CharDraft = {
      id: 9,
      name_en: "B",
      role_type: "scientist",
      large_portrait: "p9",
      small_portrait: "s9",
      is_portrait_modified: true,
    };
    const { draft, seededId } = nextDraftOnEntityRefetch(
      7,
      other.id,
      current,
      other,
      PORTRAIT_FIELDS,
    );
    expect(draft).toEqual(other);
    expect(seededId).toBe(9);
  });

  it("same-id refetch with NO syncFields (Event/Decision/Idea path) preserves the draft untouched", () => {
    const draft0 = {
      id: 3,
      title: "user edit",
      picture: "GFX_merged_by_onUploaded",
    };
    const refetched = { id: 3, title: "stale server", picture: "GFX_old" };
    const { draft } = nextDraftOnEntityRefetch(
      3,
      refetched.id,
      draft0,
      refetched,
    );
    expect(draft).toBe(draft0);
    expect(draft.title).toBe("user edit");
  });
});

describe("nextDraftOnEntityChange — 脚本 Apply 之后表单必须跟上", () => {
  type Char = {
    id: number;
    advisor_slot: string;
    name: string;
    large_portrait: string;
  };

  const fp = (c: Char) => stableFingerprint(c);

  it("🔴 同 id + 内容变了(脚本 Apply)→ 重播种,表单跟上", () => {
    const seeded: Char = {
      id: 7,
      advisor_slot: "economy_advisor",
      name: "A",
      large_portrait: "",
    };

    const applied: Char = { ...seeded, advisor_slot: "intelligence_advisor" };

    const r = nextDraftOnEntityChange<Char, number>({
      prevSeededId: 7,
      prevFingerprint: fp(seeded),
      entityId: applied.id,
      entity: applied,
      currentDraft: seeded,
      fingerprintOf: fp,
    });

    expect(r.reseeded).toBe(true);
    expect(r.draft.advisor_slot).toBe("intelligence_advisor");

    expect(r.clobberedDirtyDraft).toBe(false);
    expect(r.fingerprint).toBe(fp(applied));
  });

  it("同 id + 内容变了 + 草稿是脏的 → 仍重播种,但要报告碾过了(D1 = b)", () => {
    const seeded: Char = {
      id: 7,
      advisor_slot: "economy_advisor",
      name: "A",
      large_portrait: "",
    };
    const dirtyDraft: Char = { ...seeded, name: "用户改了名字还没保存" };
    const applied: Char = { ...seeded, advisor_slot: "intelligence_advisor" };

    const r = nextDraftOnEntityChange<Char, number>({
      prevSeededId: 7,
      prevFingerprint: fp(seeded),
      entityId: applied.id,
      entity: applied,
      currentDraft: dirtyDraft,
      fingerprintOf: fp,
    });

    expect(r.reseeded).toBe(true);
    expect(r.draft.advisor_slot).toBe("intelligence_advisor");
    expect(r.clobberedDirtyDraft).toBe(true);
  });

  it("同 id + 内容没变 → 保住草稿", () => {
    const seeded: Char = {
      id: 7,
      advisor_slot: "economy_advisor",
      name: "A",
      large_portrait: "",
    };
    const dirtyDraft: Char = { ...seeded, name: "未保存的编辑" };

    const r = nextDraftOnEntityChange<Char, number>({
      prevSeededId: 7,
      prevFingerprint: fp(seeded),
      entityId: 7,
      entity: seeded,
      currentDraft: dirtyDraft,
      fingerprintOf: fp,
    });

    expect(r.reseeded).toBe(false);
    expect(r.draft).toBe(dirtyDraft);
    expect(r.clobberedDirtyDraft).toBe(false);
  });

  it("同 id + 内容没变 + 有 syncFields → 只同步服务端托管字段,其余草稿不动", () => {
    const seeded: Char = {
      id: 7,
      advisor_slot: "economy_advisor",
      name: "A",
      large_portrait: "old.png",
    };
    const dirtyDraft: Char = { ...seeded, name: "未保存的编辑" };

    const refetched: Char = { ...seeded, large_portrait: "new.png" };

    const fpNoPortrait = (c: Char) => stableFingerprint(c, ["large_portrait"]);

    const r = nextDraftOnEntityChange<Char, number>({
      prevSeededId: 7,
      prevFingerprint: fpNoPortrait(seeded),
      entityId: 7,
      entity: refetched,
      currentDraft: dirtyDraft,

      fingerprintOf: fpNoPortrait,
      syncFieldsOnSameId: ["large_portrait"],
    });

    expect(r.reseeded).toBe(false);
    expect(r.draft.large_portrait).toBe("new.png");
    expect(r.draft.name).toBe("未保存的编辑");
  });

  it("换实体 → 整表重播种,且不算「覆盖」", () => {
    const a: Char = {
      id: 7,
      advisor_slot: "army_chief",
      name: "A",
      large_portrait: "",
    };
    const b: Char = {
      id: 9,
      advisor_slot: "theorist",
      name: "B",
      large_portrait: "",
    };
    const dirtyDraft: Char = { ...a, name: "未保存" };

    const r = nextDraftOnEntityChange<Char, number>({
      prevSeededId: 7,
      prevFingerprint: fp(a),
      entityId: b.id,
      entity: b,
      currentDraft: dirtyDraft,
      fingerprintOf: fp,
    });

    expect(r.reseeded).toBe(true);
    expect(r.seededId).toBe(9);
    expect(r.draft).toEqual(b);
    expect(r.clobberedDirtyDraft).toBe(false);
  });
});

describe("stableFingerprint", () => {
  it("🔴 键顺序不同但内容相同 → 指纹相同(否则每次刷新都假重播种)", () => {
    const a = { id: 1, name: "x", slot: "y" };
    const b = { slot: "y", id: 1, name: "x" };
    expect(stableFingerprint(a)).toBe(stableFingerprint(b));
  });

  it("嵌套对象的键顺序同样归一", () => {
    const a = { id: 1, locs: { zh: "中", en: "e" } };
    const b = { locs: { en: "e", zh: "中" }, id: 1 };
    expect(stableFingerprint(a)).toBe(stableFingerprint(b));
  });

  it("数组顺序是内容的一部分,不排序", () => {
    expect(stableFingerprint({ t: ["a", "b"] })).not.toBe(
      stableFingerprint({ t: ["b", "a"] }),
    );
  });

  it("omit 剔除顶层字段", () => {
    const a = { id: 1, name: "x", updated_at: "T1" };
    const b = { id: 1, name: "x", updated_at: "T2" };
    expect(stableFingerprint(a, ["updated_at"])).toBe(
      stableFingerprint(b, ["updated_at"]),
    );
    expect(stableFingerprint(a)).not.toBe(stableFingerprint(b));
  });

  it("omit 只作用于顶层 —— 嵌套里的同名 key 是内容,不许被抹掉", () => {
    const a = { id: 1, meta: { updated_at: "T1" } };
    const b = { id: 1, meta: { updated_at: "T2" } };
    expect(stableFingerprint(a, ["updated_at"])).not.toBe(
      stableFingerprint(b, ["updated_at"]),
    );
  });

  it("内容真的变了 → 指纹变", () => {
    expect(stableFingerprint({ slot: "economy_advisor" })).not.toBe(
      stableFingerprint({ slot: "intelligence_advisor" }),
    );
  });
});
