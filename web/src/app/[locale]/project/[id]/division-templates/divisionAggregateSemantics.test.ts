import { readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const pageSource = readFileSync(join(here, "page.tsx"), "utf8");
const messagesRoot = resolve(here, "../../../../../../messages");
const en = JSON.parse(
  readFileSync(join(messagesRoot, "en", "divisionTemplate.json"), "utf8"),
) as Record<string, string>;
const zh = JSON.parse(
  readFileSync(join(messagesRoot, "zh", "divisionTemplate.json"), "utf8"),
) as Record<string, string>;

const declaration = (name: string) => {
  const match = pageSource.match(
    new RegExp(`const ${name} = \\[([\\s\\S]*?)\\] as const;`),
  );
  return match?.[1] ?? "";
};

describe("division-template aggregate semantics", () => {
  it("keeps absolute values separate from equipment-relative modifiers", () => {
    const absolute = declaration("ABSOLUTE_AGG_FIELDS");
    const modifiers = declaration("EQUIPMENT_MODIFIER_FIELDS");

    expect(absolute).toContain('"combat_width"');
    expect(absolute).toContain('"max_strength"');
    expect(absolute).toContain('"max_organisation"');
    expect(absolute).toContain('"manpower"');
    expect(absolute).not.toMatch(
      /"soft_attack"|"hard_attack"|"defense"|"breakthrough"/,
    );

    expect(modifiers).toContain('"soft_attack"');
    expect(modifiers).toContain('"hard_attack"');
    expect(modifiers).toContain('"defense"');
    expect(modifiers).toContain('"breakthrough"');
  });

  it("renders relative modifiers as percentages instead of raw decimals", () => {
    expect(pageSource).toMatch(/function formatEquipmentModifier/);
    expect(pageSource).toMatch(/value \* 10000/);
    expect(pageSource).toMatch(/\$\{percent\}%/);
    expect(pageSource).toMatch(
      /formatEquipmentModifier\(aggregate\.totals\[k\]/,
    );

    const body = pageSource.match(
      /function formatEquipmentModifier\(value: number\): string \{([\s\S]*?)\n\}/,
    )?.[1];
    expect(body).toBeTruthy();
    const format = new Function("value", body ?? "") as (
      value: number,
    ) => string;
    expect([format(-0.9), format(-0.4), format(-0.3)]).toEqual([
      "-90%",
      "-40%",
      "-30%",
    ]);
  });

  it("labels every relative field explicitly as a modifier in both locales", () => {
    for (const key of [
      "aggSoftAttack",
      "aggHardAttack",
      "aggDefense",
      "aggBreakthrough",
    ]) {
      expect(en[key].toLowerCase()).toContain("modifier");
      expect(zh[key]).toContain("修正");
    }
    expect(en.modifierGroupTitle.toLowerCase()).toContain("modifier");
    expect(zh.modifierGroupTitle).toContain("修正");
  });

  it("does not claim the simplified aggregate matches the game", () => {
    expect(en.aggHint.toLowerCase()).not.toContain("matches the game");
    expect(zh.aggHint).not.toContain("贴近游戏数值");
    expect(en.aggHint.toLowerCase()).toContain("equipment-relative");
    expect(zh.aggHint).toContain("装备数值相对");
  });
});
