import { readFileSync } from "node:fs";
import { join } from "node:path";
import { createTranslator } from "use-intl";
import { afterEach, describe, expect, it, vi } from "vitest";
import { saveProvince } from "@/components/workbench/world-map/ProvinceEditor";
import {
  layerTable,
  parseWorldMap,
  type WorldMapPayload,
} from "@/components/workbench/world-map/worldMapModel";
import {
  MAP_FINDING_CODES,
  MAP_FINDING_FIX_CODES,
  MAP_PROBLEM_CODES,
  MapProblemError,
  ONE_CLICK_FIXES,
  mapFindingFix,
  mapFindingMessage,
  mapProblemMessage,
  problemProvinceIndexes,
} from "./mapProblems";

const LOCALES = ["zh", "en", "de", "es", "fr", "ja", "ko", "ru"] as const;
type Translate = (
  key: string,
  values?: Record<string, string | number>,
) => string;
const translator = (locale: string): Translate => {
  const messages = JSON.parse(
    readFileSync(
      join(__dirname, `../../../../../messages/${locale}/mapStudio.json`),
      "utf8",
    ),
  );
  return createTranslator({
    locale,
    messages: { mapStudio: messages },
    namespace: "mapStudio",
  }) as unknown as Translate;
};

const VALUES = {
  id: 4927,
  tag: "MLI",
  tags: "MLI",
  state: "Bamako",
  count: 2,
  region: 12,
  x: 10,
  y: 20,
  width: 5632,
  height: 2048,
  pixels: 3,

  file: "heightmap.bmp",
  path: "common/x.txt",
  min: 400,
  max: 8,
  index: 3,
  states: "#5, #9",
  missing: 7,
  other: 11,
  got_width: 4096,
  got_height: 2048,
  map_width: 5632,
  map_height: 2048,
  provinces: "#1, #2",
  terrain: "tundra",
};

describe("map studio refusals in the reader's language", () => {
  it("has words for every code and finding in every language", () => {
    for (const locale of LOCALES) {
      const t = translator(locale);
      for (const code of [...MAP_PROBLEM_CODES, "other", "marked"]) {
        const text = t(`problem.${code}`, VALUES);
        expect(text, `${locale} problem.${code}`).not.toMatch(/^mapStudio\./);
        expect(text, `${locale} problem.${code}`).not.toMatch(/\{\w+\}/);
      }
      for (const code of [...MAP_FINDING_CODES, "other"]) {
        expect(t(`finding.${code}`), `${locale} finding.${code}`).not.toMatch(
          /^mapStudio\./,
        );
      }
    }
  });

  it("says the refusal from its code, with what it names", () => {
    const capital = {
      code: "sea_capital",
      detail: "省份 #4927 是 MLI 的首都(「Bamako」),不能变成海",
      values: { id: 4927, tags: "MLI", state: "Bamako" },
    };
    expect(mapProblemMessage(capital, translator("en"), "en")).toBe(
      "Province #4927 is the capital of MLI (Bamako) and cannot become sea.",
    );
    expect(mapProblemMessage(capital, translator("fr"), "fr")).toContain(
      "#4927",
    );
    expect(mapProblemMessage(capital, translator("fr"), "fr")).not.toMatch(
      /[一-龥]/,
    );
  });

  it("never shows the server's Chinese to anyone else", () => {
    const unknown = { code: "something_new", detail: "某个新的中文原因" };
    expect(mapProblemMessage(unknown, translator("en"), "en")).toBe(
      translator("en")("problem.other"),
    );
    expect(
      mapProblemMessage(
        { detail: "旧服务器的中文原因" },
        translator("de"),
        "de",
      ),
    ).not.toMatch(/[一-龥]/);

    expect(mapProblemMessage(unknown, translator("zh"), "zh")).toBe(
      "某个新的中文原因",
    );
  });

  it("says the map check's findings by their code", () => {
    const finding = { code: "small_province", message: "有省份不到 8 个像素" };
    expect(mapFindingMessage(finding, translator("en"), "en")).toBe(
      "Some provinces are very small",
    );
    expect(
      mapFindingMessage(
        { code: "new_rule", message: "新规则" },
        translator("ru"),
        "ru",
      ),
    ).toBe(translator("ru")("finding.other"));
  });
});

describe("the provinces a refusal names", () => {
  const payload: WorldMapPayload = {
    provinces: [
      [101, 0x010101, 0, 0, "plains", 1, 1, 1],
      [4927, 0x020202, 0, 0, "plains", 1, 1, 1],
    ],
    states: [[1, "Bamako", "MLI"]],
    regions: [[1, "Land"]],
    continents: ["Africa"],
    colors: { MLI: [100, 140, 60] },
  };
  const map = parseWorldMap(payload);

  it("are found by id and drawn red on the map", () => {
    expect(problemProvinceIndexes(map.provinces, [4927, 999])).toEqual([2]);
    const plain = layerTable(map, "political", { province: 0 }).colors;
    const marked = layerTable(
      { ...map, provinceFlags: new Set([2]) },
      "political",
      { province: 0 },
    ).colors;
    expect(marked[1]).toBe(plain[1]);
    expect(marked[2]).not.toBe(plain[2]);
    const r = marked[2] & 0xff;
    const g = (marked[2] >> 8) & 0xff;
    expect(r).toBeGreaterThan(g + 80);
  });
});

describe("the province panel's refusals", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("keep their code, said in the reader's language", async () => {
    const body = {
      code: "state_last_land",
      detail: "地块 #357 只剩这一个陆地省份",
      values: { state: 357 },
      provinces: [11662],
    };
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(JSON.stringify(body), { status: 422 })),
    );
    const err = await saveProvince(8, 11662, { state: 1082 }).catch(
      (e: unknown) => e,
    );
    expect(err).toBeInstanceOf(MapProblemError);
    const problem = (err as MapProblemError).problem;
    expect(problem.code).toBe("state_last_land");
    const en = mapProblemMessage(problem, translator("en"), "en");
    expect(en).toContain("#357");
    expect(en).not.toMatch(/[一-龥]/);
    expect(mapProblemMessage(problem, translator("ja"), "ja")).toContain(
      "#357",
    );
  });
});

describe("how to fix a map error", () => {
  it("says how in every language for every error it names, and every one-click fix has its sentence", () => {
    for (const locale of LOCALES) {
      const t = translator(locale);
      for (const code of MAP_FINDING_FIX_CODES) {
        const text = mapFindingFix(code, t);
        expect(text, `${locale} findingFix.${code}`).toBeTruthy();
        expect(text, `${locale} findingFix.${code}`).not.toMatch(
          /^mapStudio\./,
        );
        if (locale !== "zh" && locale !== "ja")
          expect(text, `${locale} findingFix.${code}`).not.toMatch(/[一-龥]/);
      }
      for (const key of ["fixOneClick", "fixedPixels", "fixedLakeStates"]) {
        expect(t(key, { count: 3 }), `${locale} ${key}`).not.toMatch(
          /^mapStudio\./,
        );
      }
      expect(
        mapFindingFix("x_crossing", t),
        "a warning needs no fix",
      ).toBeNull();
    }
    for (const code of ONE_CLICK_FIXES)
      expect(MAP_FINDING_FIX_CODES as readonly string[]).toContain(code);
    for (const code of MAP_FINDING_FIX_CODES)
      expect(MAP_FINDING_CODES as readonly string[]).toContain(code);
  });
});
