import { describe, it, expect } from "vitest";

import {
  localizedName,
  localizedDesc,
  resolveDisplayName,
  resolveDisplayDesc,
} from "./localizedName";
import { APP_LOCALES } from "@/i18n/routing";
import { defaultContentLocaleFor } from "@/lib/store/localeStore";

describe("interface locale and Mod content locale are independent", () => {
  it("starts every interface locale from the English content source", () => {
    for (const locale of APP_LOCALES) {
      expect(defaultContentLocaleFor(locale)).toBe("en");
    }
  });

  it("shows English for a German user until German content is selected", () => {
    expect(localizedName("en", "Mobilization", "动员", "cat_id")).toBe(
      "Mobilization",
    );
  });

  it("shows an explicitly selected imported German localisation", () => {
    expect(
      resolveDisplayName(
        "de",
        { en: "Mobilization", de: "Mobilmachung" },
        "Legacy English",
        "旧中文",
        "cat_id",
      ),
    ).toBe("Mobilmachung");
  });
});

describe("localizedName · active locale wins when both set", () => {
  it("EN shows EN (the reported decisions-category bug)", () => {
    expect(localizedName("en", "Mobilization", "动员", "cat_id")).toBe(
      "Mobilization",
    );
  });

  it("ZH shows ZH", () => {
    expect(localizedName("zh", "Mobilization", "动员", "cat_id")).toBe("动员");
  });
});

describe("localizedName · English is the universal fallback", () => {
  it("EN never silently switches to a Chinese-only legacy value", () => {
    expect(localizedName("en", "", "动员", "cat_id")).toBe("cat_id");
  });

  it("ZH with only name_en set → EN", () => {
    expect(localizedName("zh", "Mobilization", "", "cat_id")).toBe(
      "Mobilization",
    );
  });

  it("treats null/undefined as empty", () => {
    expect(localizedName("en", null, "动员", "cat_id")).toBe("cat_id");
    expect(localizedName("zh", "Mobilization", undefined, "cat_id")).toBe(
      "Mobilization",
    );
  });
});

describe("localizedName · fallback + trimming", () => {
  it("both empty → fallback", () => {
    expect(localizedName("en", "", "", "cat_id")).toBe("cat_id");
    expect(localizedName("zh", null, undefined, "cat_id")).toBe("cat_id");
  });

  it("whitespace-only counts as empty and is skipped", () => {
    expect(localizedName("en", "   ", "动员", "cat_id")).toBe("cat_id");
    expect(localizedName("en", "  Mobilization  ", "动员", "cat_id")).toBe(
      "Mobilization",
    );
  });

  it("whitespace on both → fallback", () => {
    expect(localizedName("zh", "  ", "\t\n", "cat_id")).toBe("cat_id");
  });
});

describe("localizedDesc", () => {
  it("active locale wins", () => {
    expect(localizedDesc("en", "Lets us mobilize", "让我们动员", "")).toBe(
      "Lets us mobilize",
    );
    expect(localizedDesc("zh", "Lets us mobilize", "让我们动员", "")).toBe(
      "让我们动员",
    );
  });

  it("defaults to empty string (no id) when both blank", () => {
    expect(localizedDesc("en", "", "", undefined)).toBe("");
    expect(localizedDesc("zh", null, undefined)).toBe("");
  });
});

describe("resolveDisplayName · map-first read-side resolver (cluster-C)", () => {
  it("uses display_name_locs[locale] when present (editor wrote only the map)", () => {
    expect(
      resolveDisplayName(
        "en",
        { en: "Great Depression" },
        "New Idea",
        "",
        "idea_id",
      ),
    ).toBe("Great Depression");
    expect(resolveDisplayName("zh", { zh: "大萧条" }, "", "", "idea_id")).toBe(
      "大萧条",
    );
  });

  it("active map locale wins over legacy active column", () => {
    expect(
      resolveDisplayName(
        "en",
        { en: "Map EN" },
        "Legacy EN",
        "Legacy ZH",
        "id",
      ),
    ).toBe("Map EN");
  });

  it("falls back to legacy active column when the map slot is empty", () => {
    expect(resolveDisplayName("en", {}, "Legacy EN", "Legacy ZH", "id")).toBe(
      "Legacy EN",
    );
    expect(resolveDisplayName("zh", null, "Legacy EN", "Legacy ZH", "id")).toBe(
      "Legacy ZH",
    );
  });

  it("falls back to map['en'] (canonical) before the other legacy column", () => {
    expect(resolveDisplayName("zh", { en: "English Only" }, "", "", "id")).toBe(
      "English Only",
    );
  });

  it("never leaks a Chinese-only legacy column into another content locale", () => {
    expect(resolveDisplayName("en", {}, "", "动员", "cat_id")).toBe("cat_id");
    expect(resolveDisplayName("de", {}, "", "动员", "cat_id")).toBe("cat_id");
  });

  it("falls through to fallback when nothing is set", () => {
    expect(resolveDisplayName("en", {}, "", "", "cat_id")).toBe("cat_id");
    expect(resolveDisplayName("zh", null, null, undefined, "cat_id")).toBe(
      "cat_id",
    );
  });

  it("trims map + legacy and skips whitespace-only slots", () => {
    expect(
      resolveDisplayName("en", { en: "   " }, "  Legacy  ", "", "id"),
    ).toBe("Legacy");
    expect(resolveDisplayName("en", { en: "  Spaced  " }, "x", "", "id")).toBe(
      "Spaced",
    );
  });
});

describe("resolveDisplayDesc · map-first description resolver", () => {
  it("uses description_locs[locale] first", () => {
    expect(resolveDisplayDesc("en", { en: "A desc" }, "legacy", "", "")).toBe(
      "A desc",
    );
  });

  it("defaults to empty string (no id) when nothing is set", () => {
    expect(resolveDisplayDesc("en", {}, "", "")).toBe("");
    expect(resolveDisplayDesc("zh", null, null, undefined)).toBe("");
  });
});
