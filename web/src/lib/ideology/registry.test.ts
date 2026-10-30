import { beforeEach, describe, expect, it } from "vitest";

import { REFERENCE_TABLES } from "@/lib/__fixtures__/referenceTables";
import { setReferenceTables } from "@/lib/referenceTables";
import {
  countryFlagIdeologyGroups,
  groupIdeologyOptions,
  mergeWithVanilla,
  normalizeRegistry,
  subIdeologyOptions,
} from "./registry";

beforeEach(() => setReferenceTables(REFERENCE_TABLES));

const customRoot = {
  id: "market_socialism",
  body: "types = {\n}\n\ncolor = { 128 32 32 }",
  name: { en: "Market Socialism", zh: "市场社会主义" },
};

describe("custom ROOT ideology registry", () => {
  it("normalizes root definitions and exposes their ids as groups", () => {
    const normalized = normalizeRegistry({ root_definitions: [customRoot] });

    expect(normalized.root_definitions).toEqual([customRoot]);
    expect(normalized.groups).toEqual(["market_socialism"]);
  });

  it("keeps custom roots when merging the vanilla vocabulary", () => {
    const merged = mergeWithVanilla({
      root_definitions: [customRoot],
      sub_ideologies: [{ id: "syndicalism", group: customRoot.id }],
    });

    expect(merged.groups).toContain(customRoot.id);
    expect(merged.root_definitions).toEqual([customRoot]);
    expect(merged.sub_ideologies).toContainEqual({
      id: "syndicalism",
      group: customRoot.id,
      name: undefined,
      random_disabled: undefined,
    });
  });

  it("uses the localized custom ROOT name while keeping the raw token as its value", () => {
    const option = groupIdeologyOptions(
      { root_definitions: [customRoot] },
      "zh",
    ).find((entry) => entry.value === customRoot.id);

    expect(option).toMatchObject({
      value: "market_socialism",
      label: "市场社会主义",
    });
  });

  it("uses the selected Mod content localisation as the visible vanilla ROOT label", () => {
    const options = groupIdeologyOptions(undefined, "zh");

    expect(options.find((entry) => entry.value === "democratic")?.label).toBe(
      "民主主义",
    );
    expect(options.find((entry) => entry.value === "communism")?.label).toBe(
      "共产主义",
    );
    expect(options.find((entry) => entry.value === "fascism")?.label).toBe(
      "法西斯主义",
    );
    expect(options.find((entry) => entry.value === "neutrality")?.label).toBe(
      "中立主义",
    );
  });

  it("offers registered custom ROOTs as country-flag variants, but not subtypes", () => {
    const groups = countryFlagIdeologyGroups({
      root_definitions: [customRoot],
      sub_ideologies: [{ id: "syndicalism", group: customRoot.id }],
    });

    expect(groups).toEqual([
      "democratic",
      "communism",
      "fascism",
      "neutrality",
      customRoot.id,
    ]);
    expect(groups).not.toContain("syndicalism");
  });

  it("preserves and resolves non-EN/ZH authored localisation maps", () => {
    const localizedRoot = {
      ...customRoot,
      name: { ...customRoot.name, de: "Marktsozialismus" },
      desc: { en: "Custom root", de: "Eigene Ideologiegruppe" },
    };
    const normalized = normalizeRegistry({
      root_definitions: [localizedRoot],
      sub_ideologies: [
        {
          id: "councilism",
          group: localizedRoot.id,
          name: { en: "Councilism", de: "Rätekommunismus" },
          desc: { en: "Council rule", de: "Herrschaft der Räte" },
        },
      ],
    });

    expect(normalized.root_definitions?.[0].desc?.de).toBe(
      "Eigene Ideologiegruppe",
    );
    expect(normalized.sub_ideologies[0].desc?.de).toBe("Herrschaft der Räte");
    expect(
      groupIdeologyOptions(normalized, "de").find(
        (entry) => entry.value === localizedRoot.id,
      )?.label,
    ).toBe("Marktsozialismus");
  });
});

describe("an emptied vanilla ideology table", () => {
  beforeEach(() =>
    setReferenceTables({ ideologies: { groups: [], sub_ideologies: [] } }),
  );

  it("offers only the project's own ideologies", () => {
    const project = {
      root_definitions: [customRoot],
      sub_ideologies: [{ id: "syndicalism", group: customRoot.id }],
    };

    expect(
      subIdeologyOptions(project, "en").map((option) => option.value),
    ).toEqual(["syndicalism"]);
    expect(
      groupIdeologyOptions(project, "en").map((option) => option.value),
    ).toEqual([customRoot.id]);
    expect(countryFlagIdeologyGroups(undefined)).toEqual([]);
  });
});
