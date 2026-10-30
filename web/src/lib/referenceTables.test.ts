import fs from "node:fs";
import path from "node:path";
import { afterEach, describe, expect, it, vi } from "vitest";
import { REFERENCE_TABLES } from "./__fixtures__/referenceTables";
import {
  fontTextColors,
  loadReferenceTables,
  setReferenceTables,
  textColors,
  vanillaIdeologies,
} from "./referenceTables";

const fetchMock = vi.fn();
vi.stubGlobal("fetch", fetchMock);

afterEach(() => {
  fetchMock.mockReset();
  setReferenceTables({});
});

describe("the reference tables", () => {
  it("are empty until loaded", () => {
    expect(vanillaIdeologies()).toEqual({ groups: [], sub_ideologies: [] });
    expect(textColors()).toEqual({});
    expect(fontTextColors("hoi_18mbs")).toEqual({});
  });

  it("load once from the program and are read synchronously after", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => REFERENCE_TABLES,
    });
    await expect(loadReferenceTables()).resolves.toEqual(REFERENCE_TABLES);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/app/reference-tables",
      expect.anything(),
    );
    expect(vanillaIdeologies().sub_ideologies).toHaveLength(22);
    expect(textColors().G).toBe("rgb(0, 159, 3)");
    expect(fontTextColors("hoi_18mbs").G).toBe("rgb(86, 172, 91)");
    expect(fontTextColors("cg_16b")).toEqual({});
  });

  it("fails the load when the program cannot answer", async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      json: async () => ({ detail: "broken" }),
    });
    await expect(loadReferenceTables()).rejects.toThrow("broken");
  });

  it("reads emptied or malformed parts as empty", () => {
    setReferenceTables({
      ideologies: {
        groups: "democratic",
        sub_ideologies: [
          { id: 3 },
          "liberalism",
          { id: "rexism", group: 1, name: "x" },
        ],
      },
      text_colors: {
        global: { G: "rgb(0, 159, 3)", R: 5 },
        fonts: { hoi_18mbs: null },
      },
    });
    expect(vanillaIdeologies()).toEqual({
      groups: [],
      sub_ideologies: [{ id: "rexism", group: "" }],
    });
    expect(textColors()).toEqual({ G: "rgb(0, 159, 3)" });
    expect(fontTextColors("hoi_18mbs")).toEqual({});
  });
});

const DEFAULTS = path.join(
  path.dirname(
    new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1"),
  ),
  "../../../internal/reftables/defaults",
);
const HAVE_DEFAULTS = fs.existsSync(path.join(DEFAULTS, "ideologies.json"));

describe.skipIf(!HAVE_DEFAULTS)("the test fixture", () => {
  it("matches the tables the program ships", () => {
    const read = (name: string) =>
      JSON.parse(fs.readFileSync(path.join(DEFAULTS, name), "utf8"));
    expect(REFERENCE_TABLES.ideologies).toEqual(read("ideologies.json"));
    expect(REFERENCE_TABLES.text_colors).toEqual(read("text_colors.json"));
  });
});
