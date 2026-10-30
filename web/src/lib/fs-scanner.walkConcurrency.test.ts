import { describe, expect, it, vi } from "vitest";
import {
  collectDeclaredCountryTagCount,
  collectGfxAndGuiFiles,
  collectGfxFiles,
  collectGuiFiles,
  collectLocFiles,
  countDeclaredCountryTags,
  createMainThreadCheckpoint,
  walkFiles,
  yieldToMainThread,
  type WalkFile,
} from "./fs-scanner";

interface FakeNode {
  [name: string]: FakeNode | string;
}

function makeTree(tree: FakeNode) {
  let inFlight = 0;
  let peak = 0;

  function dir(node: FakeNode, name: string): FileSystemDirectoryHandle {
    return {
      kind: "directory",
      name,
      async *values() {
        for (const [key, val] of Object.entries(node)) {
          yield typeof val === "string" ? file(key, val) : dir(val, key);
        }
      },
    } as unknown as FileSystemDirectoryHandle;
  }

  function file(name: string, content: string): FileSystemFileHandle {
    return {
      kind: "file",
      name,
      async getFile() {
        inFlight += 1;
        peak = Math.max(peak, inFlight);

        await new Promise((r) => setTimeout(r, 0));
        inFlight -= 1;
        return new File([content], name);
      },
    } as unknown as FileSystemFileHandle;
  }

  return { root: dir(tree, ""), peak: () => peak };
}

function makeImmediateTree(tree: FakeNode): FileSystemDirectoryHandle {
  function dir(node: FakeNode, name: string): FileSystemDirectoryHandle {
    return {
      kind: "directory",
      name,
      async *values() {
        for (const [key, val] of Object.entries(node)) {
          yield typeof val === "string" ? file(key, val) : dir(val, key);
        }
      },
    } as unknown as FileSystemDirectoryHandle;
  }

  function file(name: string, content: string): FileSystemFileHandle {
    return {
      kind: "file",
      name,
      async getFile() {
        return new File([content], name);
      },
    } as unknown as FileSystemFileHandle;
  }

  return dir(tree, "");
}

async function walkFilesSerial(
  dirHandle: FileSystemDirectoryHandle,
  prefix: string,
  opts: {
    predicate: (name: string, fullPath: string) => boolean;
    skipSubtrees?: string[];
  },
): Promise<WalkFile[]> {
  const results: WalkFile[] = [];
  const shouldSkip = (p: string) =>
    !!opts.skipSubtrees?.some((s) => p === s || p.startsWith(s + "/"));
  async function walk(dir: FileSystemDirectoryHandle, p: string) {
    if (shouldSkip(p)) return;
    for await (const entry of dir.values()) {
      const entryPath = p ? `${p}/${entry.name}` : entry.name;
      if (entry.kind === "file") {
        if (opts.predicate(entry.name, entryPath)) {
          const f = await (entry as FileSystemFileHandle).getFile();
          results.push({ file: f, path: entryPath });
        }
      } else if (entry.kind === "directory") {
        await walk(entry as FileSystemDirectoryHandle, entryPath);
      }
    }
  }
  await walk(dirHandle, prefix);
  return results;
}

function bigTree(): FakeNode {
  const states: FakeNode = {};
  for (let i = 0; i < 50; i++)
    states[`state_${String(i).padStart(3, "0")}.txt`] = "x";
  return {
    "a.txt": "x",
    nested: {
      "b.txt": "x",
      deeper: { "c.txt": "x", "skipme.txt": "x" },
    },
    "d.txt": "x",
    states,
    categories: { "cat.txt": "x" },
    "e.txt": "x",
    "ignored.dds": "binary",
  };
}

const txtOnly = { predicate: (name: string) => name.endsWith(".txt") };

describe("walkFiles", () => {
  it("单目录内的 getFile 并发起来(串行版峰值恒为 1)", async () => {
    const { root, peak } = makeTree(bigTree());
    await walkFiles(root, "", txtOnly);
    expect(peak()).toBeGreaterThan(1);
  });

  it("并发有上限,不会把整个目录一次性放出去", async () => {
    const { root, peak } = makeTree(bigTree());
    await walkFiles(root, "", txtOnly);

    expect(peak()).toBeLessThanOrEqual(16);
  });

  it("返回顺序与串行实现逐项一致", async () => {
    const tree = bigTree();
    const fast = await walkFiles(makeTree(tree).root, "mod", txtOnly);
    const slow = await walkFilesSerial(makeTree(tree).root, "mod", txtOnly);
    expect(fast.map((f) => f.path)).toEqual(slow.map((f) => f.path));

    expect(fast.map((f) => f.file.name)).toEqual(slow.map((f) => f.file.name));
  });

  it("prefix 与 skipSubtrees 的行为不变", async () => {
    const tree = bigTree();
    const opts = {
      predicate: (n: string) => n.endsWith(".txt"),
      skipSubtrees: ["mod/nested/deeper"],
    };
    const fast = await walkFiles(makeTree(tree).root, "mod", opts);
    const slow = await walkFilesSerial(makeTree(tree).root, "mod", opts);
    expect(fast.map((f) => f.path)).toEqual(slow.map((f) => f.path));
    expect(fast.some((f) => f.path.includes("deeper"))).toBe(false);
    expect(fast.every((f) => f.path.startsWith("mod/"))).toBe(true);
  });

  it("predicate 过滤掉的文件不会被 getFile(别为不要的文件付 IPC)", async () => {
    let opened = 0;
    const { root } = makeTree({
      "keep.txt": "x",
      "drop.dds": "x",
      "drop2.dds": "x",
    });
    const counting = {
      predicate: (n: string) => {
        if (!n.endsWith(".txt")) return false;
        opened += 1;
        return true;
      },
    };
    const out = await walkFiles(root, "", counting);
    expect(out).toHaveLength(1);
    expect(opened).toBe(1);
  });

  it("立即 resolve 的大目录也会定期让出 macrotask,不独占渲染线程", async () => {
    const flat: FakeNode = {};
    for (let i = 0; i < 600; i++) {
      flat[`GER_${String(i).padStart(4, "0")}.txt`] = String(i);
    }

    let timerFired = false;
    const timerDone = new Promise<void>((resolve) => {
      setTimeout(() => {
        timerFired = true;
        resolve();
      }, 0);
    });

    const out = await walkFiles(makeImmediateTree(flat), "events", txtOnly);

    expect(timerFired).toBe(true);
    expect(out).toHaveLength(600);
    expect(out[0].path).toBe("events/GER_0000.txt");
    expect(out.at(-1)?.path).toBe("events/GER_0599.txt");
    await timerDone;
  });

  it("大量不匹配条目的枚举本身也会让出 macrotask", async () => {
    const images: FakeNode = {};
    for (let i = 0; i < 600; i++) {
      images[`portrait_${String(i).padStart(4, "0")}.dds`] = "binary";
    }

    let timerFired = false;
    const timerDone = new Promise<void>((resolve) => {
      setTimeout(() => {
        timerFired = true;
        resolve();
      }, 0);
    });

    const out = await walkFiles(
      makeImmediateTree(images),
      "interface",
      txtOnly,
    );
    expect(out).toEqual([]);
    expect(timerFired).toBe(true);
    await timerDone;
  });
});

describe("scanner cooperative checkpoints", () => {
  it("prefers Scheduler.yield when Chromium exposes it", async () => {
    const schedulerYield = vi.fn(async () => {});
    vi.stubGlobal("scheduler", { yield: schedulerYield });
    try {
      await yieldToMainThread();
      expect(schedulerYield).toHaveBeenCalledTimes(1);
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it("time budget can yield before the item-count safety ceiling", async () => {
    let timerFired = false;
    const timerDone = new Promise<void>((resolve) => {
      setTimeout(() => {
        timerFired = true;
        resolve();
      }, 0);
    });
    const checkpoint = createMainThreadCheckpoint(0);

    const pendingYield = checkpoint();
    expect(pendingYield).not.toBeNull();
    await pendingYield;
    expect(timerFired).toBe(true);
    await timerDone;
  });
});

describe("collectDeclaredCountryTagCount", () => {
  function countryTagFile(
    index: number,
    text: () => Promise<string>,
    path = `common/country_tags/${String(index).padStart(3, "0")}.txt`,
  ): WalkFile {
    return {
      path,
      file: {
        name: path.split("/").at(-1) ?? "tags.txt",
        text,
      } as unknown as File,
    };
  }

  it("bounds concurrent File.text reads while preserving the declared-TAG count", async () => {
    let inFlight = 0;
    let peak = 0;
    const files = Array.from({ length: 25 }, (_, index) =>
      countryTagFile(index, async () => {
        inFlight += 1;
        peak = Math.max(peak, inFlight);
        await new Promise((resolve) => setTimeout(resolve, 0));
        inFlight -= 1;
        return `T${index.toString(36).padStart(3, "0")} = "countries/${index}.txt"`;
      }),
    );

    expect(await collectDeclaredCountryTagCount(files)).toBe(25);
    expect(peak).toBeGreaterThan(1);
    expect(peak).toBeLessThanOrEqual(8);
  });

  it("reads files in sorted path order and matches the synchronous parser's separator semantics", async () => {
    const started: string[] = [];
    const separators = [
      "\r\n",
      "\r",
      "\x0b",
      "\x0c",
      "\x1c",
      "\x1d",
      "\x1e",
      "\x85",
      "\u2028",
      "\u2029",
    ];
    const content =
      "\ufeff" +
      [
        'GER = "countries/Germany.txt"',
        "dynamic_tags = yes",
        "# comment",
        'FRA = "countries/France.txt"',
        'ger = "countries/Duplicate.txt"',
      ].join(separators[0]);
    const files = [
      countryTagFile(
        2,
        async () => {
          started.push("z");
          return content;
        },
        "z.txt",
      ),
      countryTagFile(
        0,
        async () => {
          started.push("a");
          return separators
            .map(
              (separator, index) =>
                `X${index.toString(36).padStart(2, "0")} = "countries/${index}.txt"${separator}`,
            )
            .join("");
        },
        "a.txt",
      ),
    ];
    const rawContents = [
      await files[1].file.text(),
      await files[0].file.text(),
    ];
    started.length = 0;

    expect(await collectDeclaredCountryTagCount(files)).toBe(
      countDeclaredCountryTags(rawContents),
    );
    expect(started).toEqual(["a", "z"]);
  });

  it("yields a macrotask while parsing immediately-resolved large contents", async () => {
    const content = Array.from(
      { length: 1_024 },
      (_, index) =>
        `T${index.toString(36).padStart(4, "0")} = "countries/${index}.txt"`,
    ).join("\n");
    let timerFired = false;
    const timerDone = new Promise<void>((resolve) => {
      setTimeout(() => {
        timerFired = true;
        resolve();
      }, 0);
    });

    const count = await collectDeclaredCountryTagCount([
      countryTagFile(0, async () => content),
    ]);

    expect(count).toBe(1_024);
    expect(timerFired).toBe(true);
    await timerDone;
  });
});

describe("collectGfxAndGuiFiles", () => {
  it("单次 interface walk 与两个旧 collector 的过滤和各自顺序完全一致", async () => {
    const tree: FakeNode = {
      "z.gui": "z",
      "a.gfx": "a",
      nested: {
        "b.gfx": "b",
        "c.txt": "ignored",
        deeper: {
          "d.gui": "d",
          "upper.GFX": "ignored-case",
          "e.gfx": "e",
        },
      },
      "f.gui": "f",
      "portrait.dds": "ignored",
    };

    const combined = await collectGfxAndGuiFiles(
      makeImmediateTree(tree),
      "interface",
    );
    const legacyGfx = await collectGfxFiles(
      makeImmediateTree(tree),
      "interface",
    );
    const legacyGui = await collectGuiFiles(
      makeImmediateTree(tree),
      "interface",
    );

    expect(combined.gfx.map((item) => item.path)).toEqual(
      legacyGfx.map((item) => item.path),
    );
    expect(combined.gui.map((item) => item.path)).toEqual(
      legacyGui.map((item) => item.path),
    );
    expect(combined.gfx.map((item) => item.file.name)).toEqual(
      legacyGfx.map((item) => item.file.name),
    );
    expect(combined.gui.map((item) => item.file.name)).toEqual(
      legacyGui.map((item) => item.file.name),
    );
  });
});

describe("collectLocFiles", () => {
  it("keeps localisation packs regardless of filename or active country", async () => {
    const { root } = makeTree({
      english: {
        "GER_events_l_english.yml": 'l_english:\n GER_event: "German event"',
        "totally_unrelated_pack.yml": 'l_english:\n GER_ADJ: "German"',
        "notes.txt": "not localisation",
      },
      german: {
        "author_custom_names.YML": 'l_german:\n GER_ADJ: "Deutsch"',
      },
    });

    const files = await collectLocFiles(root, "localisation");
    expect(files.map((item) => item.path)).toEqual([
      "localisation/english/GER_events_l_english.yml",
      "localisation/english/totally_unrelated_pack.yml",
      "localisation/german/author_custom_names.YML",
    ]);
  });
});
