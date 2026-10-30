import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import {
  createDirHandleFromFileList,
  isImportRelevant,
} from "@/lib/fileListDirHandle";
import { isModMapFile } from "@/lib/fs-scanner";
import { performScan } from "./scan";

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
  makeFile("KR/descriptor.mod", 'name="KR"'),
  makeFile("KR/map/provinces.bmp", "BM"),
  makeFile("KR/map/definition.csv", "0;0;0;0;land;false;unknown;0\n"),
  makeFile("KR/map/default.map", 'provinces = "provinces.bmp"\n'),
  makeFile("KR/map/strategicregions/1-A.txt", "strategic_region={}"),
  makeFile("KR/map/terrain/colormap_rgb_cityemissivemask_a.dds", "DDS"),
  makeFile("KR/map/terrain/atlas0.dds", "DDS"),
];

describe("import · the mod's own map", () => {
  it("collects the files that define the map, not the render-only textures", async () => {
    const scan = await performScan(
      createDirHandleFromFileList(MOD),
      "",
      () => {},
    );
    expect(scan.map.files.map((f) => f.path).sort()).toEqual([
      "map/default.map",
      "map/definition.csv",
      "map/provinces.bmp",
      "map/strategicregions/1-A.txt",
      "map/terrain/colormap_rgb_cityemissivemask_a.dds",
    ]);
    expect(scan.map.fileCount).toBe(5);
  });

  it("keeps map files through the browser fallback's extension filter", () => {
    expect(isImportRelevant(makeFile("KR/map/provinces.bmp"))).toBe(true);
    expect(isImportRelevant(makeFile("KR/map/definition.csv"))).toBe(true);
    expect(isImportRelevant(makeFile("KR/map/default.map"))).toBe(true);

    expect(isImportRelevant(makeFile("KR/gfx/loadingscreens/a.bmp"))).toBe(
      false,
    );
    expect(isModMapFile("map/terrain/atlas0.dds")).toBe(false);
  });

  it("uploads the map and counts it in the size pre-flight", () => {
    const source = fs.readFileSync(
      path.resolve(
        process.cwd(),
        "src/components/workbench/import-wizard/UnifiedModImport.tsx",
      ),
      "utf8",
    );
    expect(source).toContain("addFiles(scan.map);");
    expect(source).toContain("accumulate(scan.map);");
  });
});
