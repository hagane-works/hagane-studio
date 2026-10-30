import { describe, it, expect } from "vitest";
import { createDirHandleFromFileList } from "@/lib/fileListDirHandle";
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

const TNO_DESCRIPTOR = [
  'version="1.10.0b"',
  "tags={",
  '\t"Alternative History"',
  "}",
  'name="The New Order: Last Days of Europe"',
  'supported_version="1.14.*"',
  'remote_file_id="2438003901"',
].join("\n");

const MOD: File[] = [
  makeFile("TNO/descriptor.mod", TNO_DESCRIPTOR),
  makeFile("TNO/common/characters/GER.txt", "characters = {}"),
  makeFile("TNO/localisation/english/tno_l_english.yml", "l_english:\n"),
];

describe("performScan · root descriptor.mod collection", () => {
  it("collects the mod-root descriptor.mod at path 'descriptor.mod' with its real bytes", async () => {
    const dir = createDirHandleFromFileList(MOD);
    const scan = await performScan(dir, "GER", () => {});

    expect(scan.descriptor).not.toBeNull();

    expect(scan.descriptor!.path).toBe("descriptor.mod");

    await expect(scan.descriptor!.file.text()).resolves.toContain(
      'name="The New Order: Last Days of Europe"',
    );
  });

  it("never surfaces the descriptor as a toggleable category (it holds no workbench content)", async () => {
    const dir = createDirHandleFromFileList(MOD);
    const scan = await performScan(dir, "GER", () => {});

    const everyCategoryPath = Object.values(scan)
      .filter(
        (v): v is { files: { path: string }[] } =>
          !!v && Array.isArray((v as { files?: unknown }).files),
      )
      .flatMap((c) => c.files.map((f) => f.path));
    expect(everyCategoryPath).not.toContain("descriptor.mod");
  });

  it("returns descriptor=null for a folder with no descriptor.mod (hint simply never fires)", async () => {
    const dir = createDirHandleFromFileList([
      makeFile("Partial/common/characters/GER.txt", "characters = {}"),
    ]);
    const scan = await performScan(dir, "GER", () => {});

    expect(scan.descriptor).toBeNull();
  });
});
