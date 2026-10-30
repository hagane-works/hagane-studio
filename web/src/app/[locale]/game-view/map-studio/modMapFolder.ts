import { createMainThreadCheckpoint } from "@/lib/fs-scanner";

export interface ModFile {
  path: string;
  file: File;
}

export type ModMapPicture = "provinces" | "heightmap" | "terrain" | "rivers";

export type ModMapNeed =
  "provinces" | "heightmap" | "terrain" | "definition" | "states" | "regions";

export interface ModMapFolder {
  root: string;

  name: string;
  pictures: Partial<Record<ModMapPicture, File>>;
  definition?: File;
  regions: ModFile[];
  states: ModFile[];
  localisation: ModFile[];
  missing: ModMapNeed[];
}

const PICTURES: Record<string, ModMapPicture> = {
  "map/provinces.bmp": "provinces",
  "map/heightmap.bmp": "heightmap",
  "map/terrain.bmp": "terrain",
  "map/rivers.bmp": "rivers",
};

export function scanModFolder(
  files: readonly { path: string; file: File }[],
): ModMapFolder[] {
  const norm = files.map(({ path, file }) => ({
    path: path.replace(/\\/g, "/").replace(/^\/+/, ""),
    file,
  }));
  const roots: string[] = [];
  for (const { path } of norm) {
    const lower = path.toLowerCase();
    if (lower === "map/provinces.bmp" || lower.endsWith("/map/provinces.bmp"))
      roots.push(path.slice(0, path.length - "map/provinces.bmp".length));
  }
  roots.sort(
    (a, b) => a.split("/").length - b.split("/").length || a.localeCompare(b),
  );
  const owner = (path: string) => {
    let best: string | null = null;
    for (const root of roots)
      if (
        path.toLowerCase().startsWith(root.toLowerCase()) &&
        (best === null || root.length > best.length)
      )
        best = root;
    return best;
  };
  const folders = new Map<string, ModMapFolder>(
    roots.map((root) => [
      root,
      {
        root,
        name: root.replace(/\/+$/, "").split("/").pop() || root,
        pictures: {},
        regions: [],
        states: [],
        localisation: [],
        missing: [],
      },
    ]),
  );
  for (const { path, file } of norm) {
    const root = owner(path);
    if (root === null) continue;
    const folder = folders.get(root)!;
    const rel = path.slice(root.length);
    const lower = rel.toLowerCase();
    if (PICTURES[lower]) folder.pictures[PICTURES[lower]] = file;
    else if (lower === "map/definition.csv") folder.definition = file;
    else if (
      lower.startsWith("map/strategicregions/") &&
      lower.endsWith(".txt")
    )
      folder.regions.push({ path: rel, file });
    else if (lower.startsWith("history/states/") && lower.endsWith(".txt"))
      folder.states.push({ path: rel, file });
    else if (lower.startsWith("localisation/") && lower.endsWith(".yml"))
      folder.localisation.push({ path: rel, file });
  }
  for (const folder of folders.values()) {
    for (const need of ["provinces", "heightmap", "terrain"] as const)
      if (!folder.pictures[need]) folder.missing.push(need);
    if (!folder.definition) folder.missing.push("definition");
    if (!folder.states.length) folder.missing.push("states");
    if (!folder.regions.length) folder.missing.push("regions");
  }
  return [...folders.values()];
}

export function namesStatesOrRegions(text: string): boolean {
  return /\b(STATE_\d+|STRATEGICREGION_\d+|VICTORY_POINTS_\d+)\s*:/.test(text);
}

export async function namingLocalisation(
  files: readonly ModFile[],
  onRead: () => void,
  signal?: AbortSignal,
): Promise<ModFile[]> {
  const naming: boolean[] = new Array(files.length).fill(false);
  const checkpoint = createMainThreadCheckpoint();
  const checkAbort = () => {
    if (signal?.aborted)
      throw new DOMException("The folder scan was cancelled", "AbortError");
  };
  checkAbort();
  let next = 0;
  let stopped = false;
  const reader = async () => {
    try {
      while (!stopped && next < files.length) {
        checkAbort();
        const i = next++;
        const text = await files[i].file.text();
        if (stopped) return;
        checkAbort();
        naming[i] = namesStatesOrRegions(text);
        onRead();
        checkAbort();
        const pause = checkpoint();
        if (pause) await pause;
        checkAbort();
      }
    } catch (error) {
      stopped = true;
      throw error;
    }
  };
  await Promise.all(Array.from({ length: Math.min(8, files.length) }, reader));
  return files.filter((_, i) => naming[i]);
}

export function bundleModFiles(files: readonly ModFile[]): {
  index: string;
  bundle: Blob;
} {
  return {
    index: JSON.stringify(
      files.map(({ path, file }) => ({ path, size: file.size })),
    ),
    bundle: new Blob(files.map(({ file }) => file)),
  };
}

export function tagMapField(map: ReadonlyMap<string, string>): string {
  return [...map]
    .filter(([from, to]) => from && to)
    .map(([from, to]) => `${from}=${to}`)
    .join(",");
}
