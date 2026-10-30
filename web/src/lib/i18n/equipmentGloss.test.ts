import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  equipmentStatGlossZh,
  equipmentTokenLabel,
  equipmentTypeGlossZh,
} from "./equipmentGloss";
import { EQUIPMENT_NUMERIC_FIELDS } from "@/app/[locale]/project/[id]/equipment/equipmentForm";

const here = dirname(fileURLToPath(import.meta.url));

const zhEquipmentMessages = JSON.parse(
  readFileSync(
    join(here, "../../..", "messages", "zh", "equipmentEditor.json"),
    "utf8",
  ),
) as Record<string, string>;

const NAMES = {
  infantry_equipment: {
    en: "Infantry Equipment",
    zh: "步兵装备",
    de: "Infanterieausrüstung",
  },
  support_equipment: { zh: "支援装备（游戏）" },
};

const camelize = (token: string): string =>
  token.replace(/_([a-z0-9])/g, (_, c: string) => c.toUpperCase());

describe("equipmentGloss", () => {
  it("glosses stat tokens with the team's zh terms and equipment with the game's names", () => {
    expect(equipmentStatGlossZh["soft_attack"]).toBe("软攻");
    expect(equipmentStatGlossZh["breakthrough"]).toBe("突破");
    expect(equipmentTokenLabel("soft_attack", "zh", NAMES)).toBe(
      "soft_attack · 软攻",
    );
    expect(equipmentTokenLabel("infantry_equipment", "zh", NAMES)).toBe(
      "infantry_equipment · 步兵装备",
    );

    expect(equipmentTokenLabel("infantry_equipment", "de", NAMES)).toBe(
      "infantry_equipment · Infanterieausrüstung",
    );

    expect(equipmentTokenLabel("support_equipment", "zh", NAMES)).toBe(
      "support_equipment · 支援装备（游戏）",
    );
    expect(equipmentTokenLabel("cas", "zh", NAMES)).toBe(
      `cas · ${equipmentTypeGlossZh.cas}`,
    );
  });

  it("returns unknown (mod-added) tokens unchanged, and bare tokens in en", () => {
    expect(equipmentTokenLabel("tno_super_rifle_equipment", "zh", NAMES)).toBe(
      "tno_super_rifle_equipment",
    );
    expect(equipmentTokenLabel("", "zh", NAMES)).toBe("");
    expect(equipmentTokenLabel("infantry_equipment", "en", NAMES)).toBe(
      "infantry_equipment",
    );

    expect(equipmentTokenLabel("infantry_equipment", "ja", NAMES)).toBe(
      "infantry_equipment",
    );
    expect(equipmentTokenLabel("infantry_equipment", "zh", {})).toBe(
      "infantry_equipment",
    );

    expect(equipmentTokenLabel("cas", "fr", NAMES)).toBe("cas");
  });

  it("covers every equipment numeric stat field", () => {
    for (const token of EQUIPMENT_NUMERIC_FIELDS) {
      expect(
        equipmentStatGlossZh[token],
        `missing stat gloss: ${token}`,
      ).toMatch(/[一-鿿]/);
    }
  });

  it("zh equipmentEditor.json stat labels are `中文 (token)` using this gloss", () => {
    for (const token of EQUIPMENT_NUMERIC_FIELDS) {
      const key = camelize(token);
      expect(
        zhEquipmentMessages[key],
        `messages/zh/equipmentEditor.json missing key: ${key}`,
      ).toBe(`${equipmentStatGlossZh[token]} (${token})`);
    }
  });

  it("every gloss key is a plain HOI4 token and every value is ICU-safe Chinese", () => {
    for (const map of [equipmentStatGlossZh, equipmentTypeGlossZh]) {
      for (const [token, gloss] of Object.entries(map)) {
        expect(token).toMatch(/^[A-Za-z0-9_]+$/);

        expect(gloss).toMatch(/[一-鿿]/);
        expect(gloss).not.toMatch(/[{}<>]/);
      }
    }
  });
});
