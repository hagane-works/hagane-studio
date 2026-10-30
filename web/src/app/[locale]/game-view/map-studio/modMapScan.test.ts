import { describe, expect, it } from "vitest";
import {
  readModMapDirectory,
  readModMapFileList,
  type ModMapScanProgress,
} from "./modMapScan";

type Tree = { [name: string]: Tree | string };

function fixture(tree: Tree, read?: (path: string) => Promise<void>) {
  const reads: string[] = [];
  let active = 0;
  let peak = 0;
  const directory = (
    node: Tree,
    name: string,
    path: string,
  ): FileSystemDirectoryHandle =>
    ({
      kind: "directory",
      name,
      async *values() {
        for (const [name, content] of Object.entries(node)) {
          const childPath = `${path}/${name}`;
          if (typeof content !== "string")
            yield directory(content, name, childPath);
          else
            yield {
              kind: "file" as const,
              name,
              async getFile() {
                reads.push(childPath);
                peak = Math.max(peak, ++active);
                try {
                  await read?.(childPath);
                  return new File([content], name);
                } finally {
                  active -= 1;
                }
              },
            } as FileSystemFileHandle;
        }
      },
    }) as unknown as FileSystemDirectoryHandle;
  return { root: directory(tree, "picked", "picked"), reads, peak: () => peak };
}

function uploaded(path: string): File {
  const file = new File(["x"], path.split("/").pop()!);
  Object.defineProperty(file, "webkitRelativePath", { value: path });
  return file;
}

describe("Map Studio incremental folder discovery", () => {
  it("finds nested MOD map assets case-insensitively without opening artwork or music", async () => {
    const tree = fixture({
      mods: {
        A: {
          MAP: {
            "PROVINCES.BMP": "x",
            "heightmap.bmp": "x",
            "terrain.bmp": "x",
            "rivers.bmp": "x",
            "DEFINITION.CSV": "x",
            StrategicRegions: { nested: { "1-West.TXT": "x" } },
          },
          History: { States: { "1-West.TxT": "x" } },
          Localisation: { English: { "states.YML": "x" } },
          gfx: { "picture.dds": "x", "wrong.bmp": "x" },
          music: { "track.ogg": "x" },
        },
        B: { map: { "provinces.bmp": "x" } },
      },
      "readme.txt": "x",
    });
    const progress: ModMapScanProgress[] = [];
    const result = await readModMapDirectory(tree.root, (p) =>
      progress.push(p),
    );
    expect(result.count).toBe(13);
    expect(result.files).toHaveLength(9);
    expect(tree.reads).toEqual(result.files.map((f) => f.path));
    expect(
      tree.reads.some((path) => /\/gfx\/|\/music\/|readme/.test(path)),
    ).toBe(false);
    expect(result.files.map((f) => f.path)).toContain(
      "picked/mods/A/MAP/StrategicRegions/nested/1-West.TXT",
    );
    expect(progress[0]).toEqual({ files: 0, directories: 1, path: "picked" });
    expect(progress.at(-1)).toMatchObject({ files: 13, path: "picked" });
    expect(progress.at(-1)?.directories).toBeGreaterThan(10);
    expect(progress.every((p) => !("total" in p))).toBe(true);
  });

  it("bounds getFile concurrency to eight and retains enumeration order despite reversed read completion", async () => {
    const states = Object.fromEntries(
      Array.from({ length: 32 }, (_, i) => [`${i}.txt`, "x"]),
    );
    const tree = fixture({ history: { states } }, async (path) => {
      const index = Number(path.split("/").pop()!.split(".")[0]);
      await new Promise((resolve) =>
        setTimeout(resolve, index % 8 === 0 ? 8 : 0),
      );
    });
    const result = await readModMapDirectory(tree.root, () => undefined);
    expect(tree.peak()).toBeGreaterThan(1);
    expect(tree.peak()).toBeLessThanOrEqual(8);
    expect(result.files.map((f) => f.path)).toEqual(
      Object.keys(states).map((name) => `picked/history/states/${name}`),
    );
    expect(result.files.map((f) => f.file.name)).toEqual(Object.keys(states));
  });

  it("reports intermediate discovered counts while real timer heartbeats run before enumeration completes", async () => {
    const tree = fixture(
      Object.fromEntries(
        Array.from({ length: 4000 }, (_, i) => [`ignored-${i}.dds`, "x"]),
      ),
    );
    const progress: ModMapScanProgress[] = [];
    let heartbeats = 0;
    const heartbeat = setInterval(() => {
      heartbeats += 1;
    }, 0);
    try {
      const result = await readModMapDirectory(tree.root, (p) =>
        progress.push(p),
      );
      expect(result.count).toBe(4000);
      expect(result.files).toEqual([]);
      expect(tree.reads).toEqual([]);
      expect(heartbeats).toBeGreaterThan(0);
      expect(progress.some((p) => p.files > 0 && p.files < 4000)).toBe(true);
      expect(progress.length).toBeLessThan(100);
    } finally {
      clearInterval(heartbeat);
    }
  });

  it("rejects AbortError while directory enumeration is blocked", async () => {
    const controller = new AbortController();
    let closed = 0;
    let entered!: () => void;
    const started = new Promise<void>((resolve) => {
      entered = resolve;
    });
    const root = {
      kind: "directory",
      name: "picked",
      values() {
        return {
          next() {
            entered();
            return new Promise<IteratorResult<FileSystemFileHandle>>(
              () => undefined,
            );
          },
          return() {
            closed += 1;
            return Promise.reject(new Error("cleanup after cancellation"));
          },
          [Symbol.asyncIterator]() {
            return this;
          },
        };
      },
    } as unknown as FileSystemDirectoryHandle;
    const run = readModMapDirectory(root, () => undefined, controller.signal);
    await started;
    controller.abort();
    await expect(run).rejects.toMatchObject({ name: "AbortError" });
    expect(closed).toBe(1);
  });

  it("rejects cancellation during getFile and starts no further file reads", async () => {
    const controller = new AbortController();
    let started!: () => void;
    const ready = new Promise<void>((resolve) => {
      started = resolve;
    });
    let calls = 0;
    const tree = fixture(
      {
        history: {
          states: Object.fromEntries(
            Array.from({ length: 40 }, (_, i) => [`${i}.txt`, "x"]),
          ),
        },
      },
      async () => {
        if (++calls === 8) started();
        await new Promise<void>(() => undefined);
      },
    );
    const run = readModMapDirectory(
      tree.root,
      () => undefined,
      controller.signal,
    );
    await ready;
    controller.abort();
    await expect(run).rejects.toMatchObject({ name: "AbortError" });
    expect(calls).toBe(8);
  });

  it("preserves permission and getFile errors", async () => {
    const denied = new DOMException("Permission denied", "NotAllowedError");

    const root = {
      name: "picked",
      kind: "directory",
      async *values() {
        throw denied;
      },
    } as unknown as FileSystemDirectoryHandle;
    await expect(readModMapDirectory(root, () => undefined)).rejects.toBe(
      denied,
    );
    const io = new DOMException("Read failed", "NotReadableError");
    const tree = fixture({ map: { "provinces.bmp": "x" } }, async () => {
      throw io;
    });
    await expect(readModMapDirectory(tree.root, () => undefined)).rejects.toBe(
      io,
    );
  });
});

describe("Map Studio FileList fallback", () => {
  it("uses the same filter and retains root prefixes, file identities and input order", async () => {
    const files = [
      uploaded("Mods/B/MAP/PROVINCES.BMP"),
      uploaded("Mods/A/history/states/1.txt"),
      uploaded("Mods/A/gfx/x.dds"),
      uploaded("Mods/A/localisation/x.YML"),
    ];
    const progress: ModMapScanProgress[] = [];
    const result = await readModMapFileList(files, (p) => progress.push(p));
    expect(result.count).toBe(4);
    expect(result.files.map((f) => f.path)).toEqual([
      files[0].webkitRelativePath,
      files[1].webkitRelativePath,
      files[3].webkitRelativePath,
    ]);
    expect(result.files[0].file).toBe(files[0]);
    expect(progress[0]).toEqual({
      files: 0,
      directories: 0,
      path: "",
      total: 4,
    });
    expect(progress.at(-1)).toMatchObject({ files: 4, total: 4 });
    expect(
      (
        await readModMapFileList(
          [new File(["x"], "readme.txt")],
          () => undefined,
        )
      ).count,
    ).toBe(1);
  });

  it("yields during a large already-available FileList and reports true partial totals", async () => {
    const file = uploaded("MOD/gfx/ignored.dds");
    const files = Array.from({ length: 4000 }, () => file);
    const progress: ModMapScanProgress[] = [];
    let heartbeats = 0;
    const heartbeat = setInterval(() => {
      heartbeats += 1;
    }, 0);
    try {
      const result = await readModMapFileList(files, (p) => progress.push(p));
      expect(result).toEqual({ files: [], count: 4000 });
      expect(heartbeats).toBeGreaterThan(0);
      expect(progress.some((p) => p.files > 0 && p.files < p.total!)).toBe(
        true,
      );
      expect(progress.at(-1)).toMatchObject({
        files: 4000,
        total: 4000,
        directories: 2,
      });
    } finally {
      clearInterval(heartbeat);
    }
  });

  it("honours pre-cancelled and mid-scan cancellation without publishing completion", async () => {
    const controller = new AbortController();
    const files = Array.from({ length: 1000 }, () =>
      uploaded("MOD/map/provinces.bmp"),
    );
    const progress: ModMapScanProgress[] = [];
    await expect(
      readModMapFileList(
        files,
        (p) => {
          progress.push(p);
          if (p.files > 0) controller.abort();
        },
        controller.signal,
      ),
    ).rejects.toMatchObject({ name: "AbortError" });
    expect(progress.at(-1)!.files).toBeLessThan(files.length);
    await expect(
      readModMapFileList(
        files,
        () => {
          throw new Error("must not report");
        },
        controller.signal,
      ),
    ).rejects.toMatchObject({ name: "AbortError" });
  });
});
