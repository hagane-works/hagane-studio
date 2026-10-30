import { describe, it, expect } from "vitest";
import {
  createDirHandleFromPreparedFileList,
  createDirHandleFromFileList,
  countImportRelevant,
  isImportRelevant,
  prepareFileListImport,
} from "@/lib/fileListDirHandle";
import {
  tryGetDir,
  collectTxtFiles,
  collectGfxFiles,
  readFileByPath,
  findImageFile,
  collectDeclaredCountryTagCount,
} from "@/lib/fs-scanner";

function makeFile(relPath: string, content: BlobPart = "x"): File {
  const name = relPath.split("/").pop()!;
  const f = new File([content], name);
  Object.defineProperty(f, "webkitRelativePath", {
    value: relPath,
    configurable: true,
  });
  return f;
}

const MOD: File[] = [
  makeFile("MyMod/descriptor.mod", 'name="My Mod"\nsupported_version="1.16.*"'),
  makeFile("MyMod/common/characters/USA.txt"),
  makeFile("MyMod/common/characters/generals/USA_generals.txt"),
  makeFile("MyMod/common/national_focus/usa.txt"),
  makeFile("MyMod/events/usa.txt"),
  makeFile("MyMod/interface/mymod.gfx"),
  makeFile("MyMod/gfx/leaders/USA/portrait.dds", new Uint8Array([1, 2, 3])),
  makeFile("MyMod/localisation/english/mymod_l_english.yml"),
  makeFile(
    "MyMod/common/country_tags/00_countries.txt",
    'USA = "countries/USA.txt"\nFRA = "countries/FRA.txt"',
  ),

  makeFile("MyMod/map/provinces.bmp", new Uint8Array([0])),

  makeFile("MyMod/music/song.ogg"),
  makeFile("MyMod/readme.md"),
];

describe("createDirHandleFromFileList shim", () => {
  it("derives the root name from the first path segment", () => {
    const dir = createDirHandleFromFileList(MOD);
    expect(dir.name).toBe("MyMod");
    expect(dir.kind).toBe("directory");
  });

  it("pre-filters to import-relevant files (keeps images and the mod's map; drops music/docs)", () => {
    expect(isImportRelevant(makeFile("a/b.txt"))).toBe(true);
    expect(isImportRelevant(makeFile("a/b.gfx"))).toBe(true);
    expect(isImportRelevant(makeFile("a/b.yml"))).toBe(true);
    expect(isImportRelevant(makeFile("a/b.dds"))).toBe(true);
    expect(isImportRelevant(makeFile("a/b.png"))).toBe(true);
    expect(isImportRelevant(makeFile("a/b.tga"))).toBe(true);
    expect(isImportRelevant(makeFile("a/descriptor.mod"))).toBe(true);
    expect(isImportRelevant(makeFile("a/b.ogg"))).toBe(false);
    expect(isImportRelevant(makeFile("a/b.bmp"))).toBe(false);
    expect(isImportRelevant(makeFile("a/map/provinces.bmp"))).toBe(true);
    expect(isImportRelevant(makeFile("a/b.md"))).toBe(false);
    expect(countImportRelevant(MOD)).toBe(10);
  });

  it("getDirectoryHandle navigates and rejects on missing (tryGetDir contract)", async () => {
    const dir = createDirHandleFromFileList(MOD);
    expect(await tryGetDir(dir, "common", "characters")).not.toBeNull();
    expect(await tryGetDir(dir, "common", "nope")).toBeNull();

    expect(await tryGetDir(dir, "music")).toBeNull();
    expect(await tryGetDir(dir, "map")).not.toBeNull();
  });

  it("getFileHandle reads descriptor.mod at the root", async () => {
    const dir = createDirHandleFromFileList(MOD);
    const fh = await dir.getFileHandle("descriptor.mod");
    expect(await (await fh.getFile()).text()).toContain("My Mod");
    await expect(dir.getFileHandle("missing.txt")).rejects.toBeTruthy();
  });

  it("values() iterates root children (descriptor.mod is reachable like a native handle)", async () => {
    const dir = createDirHandleFromFileList(MOD);
    const names: string[] = [];
    for await (const entry of dir.values()) names.push(entry.name);
    expect(names).toContain("descriptor.mod");
    expect(names).toContain("common");
    expect(names).toContain("events");
    expect(names).not.toContain("music");
  });

  it("collectTxtFiles recurses — matches performScan's character scan", async () => {
    const dir = createDirHandleFromFileList(MOD);
    const charsDir = await tryGetDir(dir, "common", "characters");
    const files = await collectTxtFiles(charsDir!, "common/characters", {});
    expect(files.map((f) => f.path).sort()).toEqual([
      "common/characters/USA.txt",
      "common/characters/generals/USA_generals.txt",
    ]);
  });

  it("collectGfxFiles finds sprite mappings under interface/", async () => {
    const dir = createDirHandleFromFileList(MOD);
    const iface = await tryGetDir(dir, "interface");
    const gfx = await collectGfxFiles(iface!, "interface");
    expect(gfx.map((f) => f.path)).toEqual(["interface/mymod.gfx"]);
  });

  it("readFileByPath resolves images (Stage 2 retains .dds) and returns null for filtered files", async () => {
    const dir = createDirHandleFromFileList(MOD);
    expect(
      await readFileByPath(dir, "gfx/leaders/USA/portrait.dds"),
    ).not.toBeNull();
    expect(await readFileByPath(dir, "music/song.ogg")).toBeNull();
  });

  it("findImageFile resolves via extension swap (png requested, dds present)", async () => {
    const dir = createDirHandleFromFileList(MOD);
    const found = await findImageFile(dir, "gfx/leaders/USA/portrait.png");
    expect(found).not.toBeNull();
    expect(found!.path).toBe("gfx/leaders/USA/portrait.dds");
  });

  it("country_tags declaration count matches the parser (preview parity)", async () => {
    const dir = createDirHandleFromFileList(MOD);
    const ctags = await tryGetDir(dir, "common", "country_tags");
    const files = await collectTxtFiles(ctags!, "common/country_tags", {});
    expect(await collectDeclaredCountryTagCount(files)).toBe(2);
  });

  it("falls back to a flat root name when webkitRelativePath is absent", () => {
    const flat = [new File(["x"], "loose.txt")];
    const dir = createDirHandleFromFileList(flat);
    expect(dir.name).toBe("loose.txt");
  });
});

describe("cooperative webkitdirectory preparation", () => {
  it("yields to real event-loop tasks before filtering and between chunks", async () => {
    const files = [
      makeFile("BigMod/a.txt"),
      makeFile("BigMod/b.txt"),
      makeFile("BigMod/c.txt"),
    ];
    let taskBeforeFirstFilterRan = false;
    let taskBetweenChunksRan = false;
    let filterCalls = 0;
    let secondFilterObservedTask = false;

    setTimeout(() => {
      taskBeforeFirstFilterRan = true;
    }, 0);

    const prepared = await prepareFileListImport(files, {
      chunkSize: 1,
      filter: (file) => {
        filterCalls += 1;
        if (filterCalls === 1) {
          expect(taskBeforeFirstFilterRan).toBe(true);
          setTimeout(() => {
            taskBetweenChunksRan = true;
          }, 0);
        } else if (filterCalls === 2) {
          secondFilterObservedTask = taskBetweenChunksRan;
        }
        return isImportRelevant(file);
      },
    });

    expect(prepared.relevantCount).toBe(3);
    expect(secondFilterObservedTask).toBe(true);
  });

  it("does not consume a FileList iterator or make an Array.from-style full copy", async () => {
    const first = makeFile("NoCopy/a.txt");
    const second = makeFile("NoCopy/b.ogg");
    const indexedOnly = {
      0: first,
      1: second,
      length: 2,
      [Symbol.iterator](): never {
        throw new Error("the FileList iterator must not be consumed");
      },
    } as unknown as FileList;

    const prepared = await prepareFileListImport(indexedOnly, { chunkSize: 1 });
    expect(prepared.rootName).toBe("NoCopy");
    expect(prepared.files).toEqual([first]);
  });

  it("yields between tree-build chunks and preserves duplicate-path overwrite/order", async () => {
    let taskBetweenFilesRan = false;
    let secondFileObservedTask = false;
    const older = new File(["old"], "same.txt");
    Object.defineProperty(older, "webkitRelativePath", {
      configurable: true,
      get: () => {
        setTimeout(() => {
          taskBetweenFilesRan = true;
        }, 0);
        return "BigMod/common/same.txt";
      },
    });
    const sibling = makeFile("BigMod/common/sibling.txt");
    const newer = new File(["new"], "same.txt");
    Object.defineProperty(newer, "webkitRelativePath", {
      configurable: true,
      get: () => {
        secondFileObservedTask = taskBetweenFilesRan;
        return "BigMod/common/same.txt";
      },
    });

    const dir = await createDirHandleFromPreparedFileList(
      {
        rootName: "BigMod",
        files: [older, sibling, newer],
        relevantCount: 3,
      },
      { chunkSize: 1 },
    );

    expect(secondFileObservedTask).toBe(true);
    expect(await readFileByPath(dir, "common/same.txt")).toBe(newer);
    const common = await tryGetDir(dir, "common");
    const names: string[] = [];
    for await (const entry of common!.values()) names.push(entry.name);

    expect(names).toEqual(["same.txt", "sibling.txt"]);
  });
});
