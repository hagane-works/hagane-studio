import { describe, expect, it } from "vitest";
import {
  bundleModFiles,
  namesStatesOrRegions,
  namingLocalisation,
  scanModFolder,
  tagMapField,
} from "./modMapFolder";

const file = (name: string) => new File(["x"], name.split("/").pop() ?? name);
const tree = (paths: string[]) =>
  paths.map((path) => ({ path, file: file(path) }));

describe("a MOD's map in a picked folder (导入 MOD 地图)", () => {
  it("finds the MOD by its map and reads what its map is made of", () => {
    const [mod, ...rest] = scanModFolder(
      tree([
        "MyMod/descriptor.mod",
        "MyMod/map/provinces.bmp",
        "MyMod/map/heightmap.bmp",
        "MyMod/map/terrain.bmp",
        "MyMod/map/rivers.bmp",
        "MyMod/map/definition.csv",
        "MyMod/map/strategicregions/1-West.txt",
        "MyMod/history/states/1-West.txt",
        "MyMod/history/states/2-East.txt",
        "MyMod/localisation/english/states_l_english.yml",
        "MyMod/common/national_focus/x.txt",
      ]),
    );
    expect(rest).toEqual([]);
    expect(mod.root).toBe("MyMod/");
    expect(mod.name).toBe("MyMod");
    expect(Object.keys(mod.pictures).sort()).toEqual([
      "heightmap",
      "provinces",
      "rivers",
      "terrain",
    ]);
    expect(mod.definition).toBeDefined();
    expect(mod.states.map((f) => f.path)).toEqual([
      "history/states/1-West.txt",
      "history/states/2-East.txt",
    ]);
    expect(mod.regions.map((f) => f.path)).toEqual([
      "map/strategicregions/1-West.txt",
    ]);
    expect(mod.localisation.map((f) => f.path)).toEqual([
      "localisation/english/states_l_english.yml",
    ]);
    expect(mod.missing).toEqual([]);
  });

  it("says what is missing, and finds each MOD of a folder of MODs", () => {
    const mods = scanModFolder(
      tree([
        "mod/A/map/provinces.bmp",
        "mod/A/map/definition.csv",
        "mod/B/map/provinces.bmp",
        "mod/B/history/states/1-x.txt",
        "mod/C/common/x.txt",
      ]),
    );
    expect(mods.map((m) => m.name)).toEqual(["A", "B"]);
    expect(mods[0].missing).toEqual([
      "heightmap",
      "terrain",
      "states",
      "regions",
    ]);
    expect(mods[1].missing).toEqual([
      "heightmap",
      "terrain",
      "definition",
      "regions",
    ]);
    expect(mods[1].states).toHaveLength(1);
  });

  it("finds nothing in a folder with no map of its own", () => {
    expect(
      scanModFolder(
        tree(["MyMod/common/x.txt", "MyMod/history/states/1-x.txt"]),
      ),
    ).toEqual([]);
  });

  it("sends only the localisation that names states, regions or towns", () => {
    expect(namesStatesOrRegions('l_english:\n STATE_12:0 "Hesse"\n')).toBe(
      true,
    );
    expect(namesStatesOrRegions(' VICTORY_POINTS_6521:0 "Berlin"')).toBe(true);
    expect(namesStatesOrRegions(' GER_focus:0 "x"')).toBe(false);
  });

  it("writes the country map as the server reads it", () => {
    expect(
      tagMapField(
        new Map([
          ["KOR", "SHU"],
          ["BBB", ""],
          ["PRU", "WEI"],
        ]),
      ),
    ).toBe("KOR=SHU,PRU=WEI");
  });

  it("sends the MOD's text files in one bundle with their index", async () => {
    const files = [
      {
        path: "history/states/1-West.txt",
        file: new File(["state={ id=1 }"], "1-West.txt"),
      },
      {
        path: "localisation/english/s_l_english.yml",
        file: new File([`l_english:\n STATE_1:0 "西境"`], "s_l_english.yml"),
      },
    ];
    const { index, bundle } = bundleModFiles(files);
    const listed = JSON.parse(index) as { path: string; size: number }[];
    expect(listed.map((entry) => entry.path)).toEqual(
      files.map((entry) => entry.path),
    );
    const bytes = new Uint8Array(await bundle.arrayBuffer());
    expect(bytes.length).toBe(
      listed.reduce((sum, entry) => sum + entry.size, 0),
    );
    const second = bytes.slice(listed[0].size);
    expect(new Uint8Array(await files[1].file.arrayBuffer())).toEqual(second);
  });

  it("keeps the localisation that names states or regions, counting each file read", async () => {
    const loc = (path: string, text: string) => ({
      path,
      file: new File([text], path),
    });
    const files = [
      loc(
        "localisation/english/states_l_english.yml",
        `l_english:\n STATE_1:0 "West"`,
      ),
      loc(
        "localisation/english/focus_l_english.yml",
        `l_english:\n GER_focus:0 "Focus"`,
      ),
      loc(
        "localisation/english/regions_l_english.yml",
        `l_english:\n STRATEGICREGION_1:0 "Sea"`,
      ),
    ];
    let read = 0;
    const named = await namingLocalisation(files, () => {
      read += 1;
    });
    expect(read).toBe(3);
    expect(named.map((entry) => entry.path)).toEqual([
      files[0].path,
      files[2].path,
    ]);
  });

  it("lets the browser process timers while checking immediately available localisation texts", async () => {
    const files = Array.from({ length: 3000 }, (_, i) => ({
      path: `localisation/${i}.yml`,
      file: { text: async () => `STATE_${i}:0 "Name"` } as File,
    }));
    let heartbeats = 0;
    let reads = 0;
    const heartbeat = setInterval(() => {
      heartbeats += 1;
    }, 0);
    try {
      expect(
        await namingLocalisation(files, () => {
          reads += 1;
        }),
      ).toHaveLength(files.length);
      expect(reads).toBe(files.length);
      expect(heartbeats).toBeGreaterThan(0);
    } finally {
      clearInterval(heartbeat);
    }
  });

  it("stops localisation reads and progress when cancelled by the progress callback", async () => {
    const controller = new AbortController();
    let reads = 0;
    let opened = 0;
    const files = Array.from({ length: 100 }, (_, i) => ({
      path: `localisation/${i}.yml`,
      file: {
        text: async () => {
          opened += 1;
          return `STATE_${i}:0 "Name"`;
        },
      } as File,
    }));
    await expect(
      namingLocalisation(
        files,
        () => {
          reads += 1;
          controller.abort();
        },
        controller.signal,
      ),
    ).rejects.toMatchObject({ name: "AbortError" });
    expect(reads).toBe(1);
    expect(opened).toBeLessThanOrEqual(8);
    await expect(
      namingLocalisation(
        files,
        () => {
          reads += 1;
        },
        controller.signal,
      ),
    ).rejects.toMatchObject({ name: "AbortError" });
    expect(reads).toBe(1);
  });

  it("does not turn a localisation read failure into a successful empty scan", async () => {
    const failure = new DOMException("File is unreadable", "NotReadableError");
    const file = new File(["x"], "a.yml");
    Object.defineProperty(file, "text", {
      value: async () => {
        throw failure;
      },
    });
    const files = [{ path: "localisation/a.yml", file }];
    await expect(namingLocalisation(files, () => undefined)).rejects.toBe(
      failure,
    );
  });

  it("stops the other localisation readers after one file fails without issuing stale progress", async () => {
    const failure = new DOMException("Unreadable file", "NotReadableError");
    const finish: (() => void)[] = [];
    let opened = 0;
    let progress = 0;
    const files = Array.from({ length: 100 }, (_, i) => ({
      path: `localisation/${i}.yml`,
      file: {
        text: async () => {
          opened += 1;
          if (i === 0) throw failure;
          await new Promise<void>((resolve) => {
            finish.push(resolve);
          });
          return `STATE_${i}:0 "Name"`;
        },
      } as File,
    }));
    await expect(
      namingLocalisation(files, () => {
        progress += 1;
      }),
    ).rejects.toBe(failure);
    for (const resolve of finish) resolve();
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(opened).toBe(8);
    expect(progress).toBe(0);
  });
});
