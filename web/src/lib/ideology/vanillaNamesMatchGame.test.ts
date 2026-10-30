import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const GAME_ROOT =
  process.env.HOI4_GAME_ROOT ?? "G:/Steam/steamapps/common/Hearts of Iron IV";
const LOC_ROOT = path.join(GAME_ROOT, "localisation");
const HAVE_GAME = fs.existsSync(path.join(LOC_ROOT, "english"));

function loadLoc(language: string): Map<string, string> {
  const out = new Map<string, string>();
  const dir = path.join(LOC_ROOT, language);
  if (!fs.existsSync(dir)) return out;
  for (const file of fs.readdirSync(dir)) {
    if (!file.endsWith(".yml")) continue;
    const raw = fs
      .readFileSync(path.join(dir, file), "utf8")
      .replace(/^\uFEFF/, "");
    for (const line of raw.split(/\r?\n/)) {
      const m = /^\s*([a-z_0-9]+):\s*\d*\s*"([^"]*)"/.exec(line);
      if (!m || m[1].endsWith("_desc") || out.has(m[1])) continue;
      out.set(m[1], m[2]);
    }
  }
  return out;
}

const HERE = path.dirname(
  new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1"),
);

const SHIPPED_TABLE = path.join(
  HERE,
  "../../../../internal/reftables/defaults/ideologies.json",
);
const HAVE_TABLE = fs.existsSync(SHIPPED_TABLE);

describe.skipIf(!HAVE_GAME || !HAVE_TABLE)(
  "vanilla sub-ideology names match the game",
  () => {
    it("ideologies.json 的每条译名都逐字等于游戏 localisation", () => {
      const english = loadLoc("english");
      const chinese = loadLoc("simp_chinese");
      const table = JSON.parse(fs.readFileSync(SHIPPED_TABLE, "utf8")) as {
        sub_ideologies: { id: string; name?: { en?: string; zh?: string } }[];
      };
      expect(table.sub_ideologies.length).toBeGreaterThanOrEqual(22);

      const drift: string[] = [];
      for (const entry of table.sub_ideologies) {
        const officialEn = english.get(entry.id);
        const officialZh = chinese.get(entry.id);
        if (officialEn === undefined || officialZh === undefined) {
          drift.push(
            `${entry.id}: 游戏里没有这个 key(原版表不该出现自定义 id)`,
          );
          continue;
        }
        if (entry.name?.en !== officialEn) {
          drift.push(
            `${entry.id} en: 我们「${entry.name?.en}」官方「${officialEn}」`,
          );
        }
        if (entry.name?.zh !== officialZh) {
          drift.push(
            `${entry.id} zh: 我们「${entry.name?.zh}」官方「${officialZh}」`,
          );
        }
      }
      expect(drift, "ideologies.json 与游戏译名不一致").toEqual([]);
    });
  },
);
