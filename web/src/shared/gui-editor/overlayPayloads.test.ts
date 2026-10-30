import { describe, expect, it } from "vitest";

import {
  buildSetTextPayload,
  buildSwapSpritePayload,
  readSetTextLangTexts,
  stableGuiLocalisationKey,
} from "./overlayPayloads";

describe("GUI editor overlay payloads", () => {
  it("stores sprite replacements as a GFX reference only", () => {
    expect(buildSwapSpritePayload("GFX_country_flag")).toEqual({
      gfx_ref: "GFX_country_flag",
    });
    expect(buildSwapSpritePayload("GFX_country_flag")).not.toHaveProperty(
      "override_file",
    );
  });

  it.each(["loc", "key"] as const)(
    "reuses an original %s localisation key",
    (originalKind) => {
      expect(
        buildSetTextPayload({
          target: "Root/Header/Title",
          originalText: "ORIGINAL_TITLE",
          originalKind,
          langTexts: { en: "Title", zh: "标题" },
        }),
      ).toEqual({
        key: "ORIGINAL_TITLE",
        is_new: false,
        lang_texts: { en: "Title", zh: "标题" },
      });
    },
  );

  it("gives literal text a stable safe key derived from its NAME-PATH", () => {
    const target = "Root/政治 标题/label.one";
    const key = stableGuiLocalisationKey(target);
    expect(key).toMatch(/^GUI_[A-Z0-9_]+_[A-F0-9]{8}$/);
    expect(stableGuiLocalisationKey(target)).toBe(key);
    expect(stableGuiLocalisationKey(`${target}/other`)).not.toBe(key);
    expect(
      buildSetTextPayload({
        target,
        originalText: "Literal title",
        originalKind: "literal",
        langTexts: { en: "New title", zh: "新标题" },
      }),
    ).toEqual({
      key,
      is_new: true,
      lang_texts: { en: "New title", zh: "新标题" },
    });
  });

  it("keeps the persisted key semantics when editing a previewed new key", () => {
    expect(
      buildSetTextPayload({
        target: "Root/Title",
        originalText: "GUISUB_GUI_ROOT_TITLE_12345678",
        originalKind: "loc",
        langTexts: { en: "Changed again" },
        existingPayload: {
          key: "GUI_ROOT_TITLE_12345678",
          is_new: true,
          lang_texts: { en: "Changed" },
        },
      }),
    ).toEqual({
      key: "GUI_ROOT_TITLE_12345678",
      is_new: true,
      lang_texts: { en: "Changed again" },
    });
  });

  it("preserves legacy missing-is_new behavior for an existing keyed edit", () => {
    expect(
      buildSetTextPayload({
        target: "Root/Title",
        originalText: "GUISUB_OLD_KEY",
        originalKind: "loc",
        langTexts: { en: "Updated" },
        existingPayload: { key: "OLD_KEY", text: "Old" },
      }),
    ).toMatchObject({ key: "OLD_KEY", is_new: true });
  });

  it("reads modern and legacy text payloads without dropping language values", () => {
    expect(
      readSetTextLangTexts(
        { lang_texts: { zh: "中文", fr: "Français" }, text: "English" },
        "Rendered fallback",
      ),
    ).toEqual({ zh: "中文", fr: "Français", en: "English" });
    expect(
      readSetTextLangTexts({ text: "Legacy" }, "Rendered fallback"),
    ).toEqual({
      en: "Legacy",
    });
    expect(readSetTextLangTexts({ key: "KEY_ONLY" }, "Resolved text")).toEqual({
      en: "Resolved text",
    });
  });
});
