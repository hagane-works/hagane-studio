import { describe, expect, it } from "vitest";

import {
  CUSTOM_VOCAB_SENTINEL,
  baseModOnlyOptions,
  resolveVocabularySelection,
} from "./vocabularyMode";

const VANILLA = [
  "political_advisor",
  "army_chief",
  "navy_chief",
  "air_chief",
  "high_command",
  "theorist",
] as const;

const KAISERREICH = [
  "political_advisor",
  "economy_advisor",
  "foreign_advisor",
  "intelligence_advisor",
];

describe("resolveVocabularySelection", () => {
  it("原版取值:正常选中、不标自定义、不提示", () => {
    const r = resolveVocabularySelection("army_chief", VANILLA, KAISERREICH);
    expect(r.selectValue).toBe("army_chief");
    expect(r.freeText).toBe(false);
    expect(r.isCustom).toBe(false);
    expect(r.showUnknownNote).toBe(false);
  });

  it("基 mod 的取值:可直接选中(不进自由输入),标自定义,不提示", () => {
    const r = resolveVocabularySelection(
      "economy_advisor",
      VANILLA,
      KAISERREICH,
    );
    expect(r.selectValue).toBe("economy_advisor");
    expect(r.freeText).toBe(false);
    expect(r.isCustom).toBe(true);
    expect(r.showUnknownNote).toBe(false);
  });

  it("🔴 谁也不认识的取值:自动进自定义模式并把值显示出来 —— 绝不留白", () => {
    const r = resolveVocabularySelection("my_new_slot", VANILLA, KAISERREICH);

    expect(r.freeText).toBe(true);
    expect(r.isCustom).toBe(true);
    expect(r.selectValue).toBe(CUSTOM_VOCAB_SENTINEL);
    expect(r.showUnknownNote).toBe(true);
  });

  it("空值:干净占位,不标不提示", () => {
    const r = resolveVocabularySelection("", VANILLA, KAISERREICH);
    expect(r.selectValue).toBe("");
    expect(r.freeText).toBe(false);
    expect(r.isCustom).toBe(false);
    expect(r.showUnknownNote).toBe(false);
  });

  it("null / undefined 与空值同解", () => {
    for (const v of [null, undefined]) {
      const r = resolveVocabularySelection(v, VANILLA, KAISERREICH);
      expect(r.freeText).toBe(false);
      expect(r.isCustom).toBe(false);
      expect(r.showUnknownNote).toBe(false);
    }
  });

  it("用户主动点「自定义…」:即便当前还没填,也进自由输入", () => {
    const r = resolveVocabularySelection("", VANILLA, KAISERREICH, true);
    expect(r.freeText).toBe(true);
    expect(r.selectValue).toBe(CUSTOM_VOCAB_SENTINEL);

    expect(r.showUnknownNote).toBe(false);
  });

  it("🔴 没有基 mod(词表为空)时永远静默 —— 从零起的项目自造 slot 不该被提示", () => {
    const r = resolveVocabularySelection("my_new_slot", VANILLA, []);
    expect(r.freeText).toBe(true);
    expect(r.isCustom).toBe(true);
    expect(r.showUnknownNote).toBe(false);
  });

  it("explicitCustomMode 只能开、不能关 —— 值不认识时 false 也照样进自定义", () => {
    const r = resolveVocabularySelection(
      "my_new_slot",
      VANILLA,
      KAISERREICH,
      false,
    );
    expect(r.freeText).toBe(true);
  });

  it("取值两端的空白不影响判定", () => {
    const r = resolveVocabularySelection(
      "  army_chief  ",
      VANILLA,
      KAISERREICH,
    );
    expect(r.isCustom).toBe(false);
    expect(r.freeText).toBe(false);
  });
});

describe("baseModOnlyOptions", () => {
  it("去掉与原版重复的取值 —— 否则 political_advisor 会在下拉里出现两次", () => {
    const out = baseModOnlyOptions(VANILLA, KAISERREICH);
    expect(out).not.toContain("political_advisor");
    expect(out).toEqual([
      "economy_advisor",
      "foreign_advisor",
      "intelligence_advisor",
    ]);
  });

  it("基 mod 词表内部的重复也只保留一份", () => {
    const out = baseModOnlyOptions(VANILLA, [
      "economy_advisor",
      "economy_advisor",
    ]);
    expect(out).toEqual(["economy_advisor"]);
  });

  it("空串 / 纯空白项被丢弃(后端 DISTINCT 已过滤,这里是第二道)", () => {
    const out = baseModOnlyOptions(VANILLA, ["", "   ", "economy_advisor"]);
    expect(out).toEqual(["economy_advisor"]);
  });

  it("没有基 mod 时返回空数组,调用方据此不渲染第二组", () => {
    expect(baseModOnlyOptions(VANILLA, [])).toEqual([]);
  });
});
