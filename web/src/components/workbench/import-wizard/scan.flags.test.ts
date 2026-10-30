import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";
import {
  createDirHandleFromFileList,
  isImportRelevant,
} from "@/lib/fileListDirHandle";
import { isModFlagFile } from "@/lib/fs-scanner";
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
  makeFile("Sandbox/descriptor.mod", 'name="Sandbox"'),
  makeFile("Sandbox/gfx/flags/HJM.tga", "TGA"),
  makeFile("Sandbox/gfx/flags/HJM_feudalist.tga", "TGA"),
  makeFile("Sandbox/gfx/flags/W01.png", "PNG"),
  makeFile("Sandbox/gfx/flags/medium/HJM.tga", "TGA"),
  makeFile("Sandbox/gfx/flags/small/HJM.tga", "TGA"),
  makeFile("Sandbox/gfx/flags/notes.txt", "x"),
  makeFile("Sandbox/gfx/flags/AB.tga", "TGA"),
  makeFile("Sandbox/gfx/interface/goals/a.dds", "DDS"),
];

describe("import · the mod's own country flags", () => {
  it("collects the large flags, not medium/, small/ or anything that is not a tag's flag", async () => {
    const scan = await performScan(
      createDirHandleFromFileList(MOD),
      "",
      () => {},
    );
    expect(scan.flags.files.map((f) => f.path).sort()).toEqual([
      "gfx/flags/HJM.tga",
      "gfx/flags/HJM_feudalist.tga",
      "gfx/flags/W01.png",
    ]);
    expect(scan.flags.fileCount).toBe(3);
  });

  it("recognises a flag by its place and name", () => {
    expect(isModFlagFile("gfx/flags/HJM.tga")).toBe(true);
    expect(isModFlagFile("gfx\\flags\\hjm_libertarian_socialist.DDS")).toBe(
      true,
    );
    expect(isModFlagFile("gfx/flags/medium/HJM.tga")).toBe(false);
    expect(isModFlagFile("gfx/flags/AB.tga")).toBe(false);
    expect(isModFlagFile("gfx/interface/HJM.tga")).toBe(false);

    expect(isImportRelevant(makeFile("Sandbox/gfx/flags/HJM.tga"))).toBe(true);
  });

  it("uploads the flags and counts them in the size pre-flight", () => {
    const source = fs.readFileSync(
      path.resolve(
        process.cwd(),
        "src/components/workbench/import-wizard/UnifiedModImport.tsx",
      ),
      "utf8",
    );
    expect(source).toContain("addFiles(scan.flags);");
    expect(source).toContain("accumulate(scan.flags);");
  });
});
