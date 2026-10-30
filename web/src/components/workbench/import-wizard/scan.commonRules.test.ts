import { describe, it, expect } from "vitest";
import { createDirHandleFromFileList } from "@/lib/fileListDirHandle";
import { performScan } from "./scan";

function makeFile(relPath: string, content: BlobPart = "x = {}"): File {
  const name = relPath.split("/").pop()!;
  const f = new File([content], name);
  Object.defineProperty(f, "webkitRelativePath", {
    value: relPath,
    configurable: true,
  });
  return f;
}

const MOD: File[] = [
  makeFile("TNO/common/scripted_effects/TNO_generic_effects.txt"),
  makeFile("TNO/common/scripted_triggers/TNO_generic_triggers.txt"),
  makeFile("TNO/common/scripted_localisation/TNO_scripted_loc.txt"),
  makeFile("TNO/common/dynamic_modifiers/TNO_dynamic_modifiers.txt"),
  makeFile("TNO/common/autonomous_states/00_autonomous_states.txt"),
  makeFile("TNO/common/intelligence_agencies/00_intelligence_agencies.txt"),
  makeFile("TNO/common/doctrines/folders/00_doctrine_folders.txt"),
  makeFile("TNO/common/doctrines/subdoctrines/land_subdoctrines.txt"),
  makeFile("TNO/common/unit_leader/00_traits.txt"),
  makeFile("TNO/common/unit_leader/GNG_traits.txt"),
  makeFile("TNO/common/scientist_traits/00_traits.txt"),
  makeFile("TNO/common/opinion_modifiers/00_opinion_modifiers.txt"),

  makeFile("TNO/common/on_actions/00_on_actions.txt"),
  makeFile("TNO/common/on_actions/TNO_Reich_on_actions.txt"),

  makeFile("TNO/common/characters/GER.txt"),
  makeFile("TNO/common/characters/USA.txt"),
];

function allPaths(scan: Awaited<ReturnType<typeof performScan>>): string[] {
  return Object.values(scan)
    .filter(
      (v): v is { files: { path: string }[] } =>
        !!v && Array.isArray((v as { files?: unknown }).files),
    )
    .flatMap((c) => c.files.map((f) => f.path));
}

describe("performScan · common/ rule + script directories", () => {
  it("collects every backend-routed common/ rule dir (the Q1 #1 gap)", async () => {
    const scan = await performScan(
      createDirHandleFromFileList(MOD),
      "GER",
      () => {},
    );
    const paths = allPaths(scan);

    for (const p of [
      "common/scripted_effects/TNO_generic_effects.txt",
      "common/scripted_triggers/TNO_generic_triggers.txt",
      "common/scripted_localisation/TNO_scripted_loc.txt",
      "common/dynamic_modifiers/TNO_dynamic_modifiers.txt",
      "common/autonomous_states/00_autonomous_states.txt",
      "common/intelligence_agencies/00_intelligence_agencies.txt",
      "common/doctrines/folders/00_doctrine_folders.txt",
      "common/doctrines/subdoctrines/land_subdoctrines.txt",
      "common/unit_leader/00_traits.txt",
      "common/unit_leader/GNG_traits.txt",
      "common/scientist_traits/00_traits.txt",
      "common/opinion_modifiers/00_opinion_modifiers.txt",
      "common/on_actions/00_on_actions.txt",
      "common/on_actions/TNO_Reich_on_actions.txt",
    ]) {
      expect(paths, `${p} must be uploaded`).toContain(p);
    }
  });

  it("does NOT TAG-filter them — these hold mod-global symbols", async () => {
    const scan = await performScan(
      createDirHandleFromFileList(MOD),
      "GER",
      () => {},
    );
    const paths = allPaths(scan);

    expect(paths).toContain("common/unit_leader/GNG_traits.txt");
    expect(paths).toContain("common/characters/GER.txt");
    expect(paths).not.toContain("common/characters/USA.txt");
  });

  it("routes the three characters-flag-gated dirs into the characters bucket", async () => {
    const scan = await performScan(
      createDirHandleFromFileList(MOD),
      "GER",
      () => {},
    );
    const charPaths = scan.characters.files.map((f) => f.path);

    expect(charPaths).toContain("common/unit_leader/00_traits.txt");
    expect(charPaths).toContain("common/scientist_traits/00_traits.txt");
    expect(charPaths).toContain(
      "common/opinion_modifiers/00_opinion_modifiers.txt",
    );
    expect(scan.characters.fileCount).toBe(charPaths.length);

    const rulePaths = scan.common_rules.files.map((f) => f.path);
    expect(rulePaths.some((p) => p.startsWith("common/unit_leader/"))).toBe(
      false,
    );
    expect(
      rulePaths.some((p) => p.startsWith("common/opinion_modifiers/")),
    ).toBe(false);
  });

  it("keeps a per-directory breakdown so the bucket is not uploaded-but-invisible", async () => {
    const scan = await performScan(
      createDirHandleFromFileList(MOD),
      "GER",
      () => {},
    );

    expect(scan.common_rules.dirCounts).toEqual({
      dynamic_modifiers: 1,
      autonomous_states: 1,
      intelligence_agencies: 1,
      scripted_effects: 1,
      scripted_triggers: 1,
      scripted_localisation: 1,
      doctrines: 2,
      on_actions: 2,
    });
    expect(scan.common_rules.fileCount).toBe(10);

    const summed = Object.values(scan.common_rules.dirCounts!).reduce(
      (a, b) => a + b,
      0,
    );
    expect(summed).toBe(scan.common_rules.files.length);
  });

  it("uploads the mod's own common/ideologies, nested and untagged", async () => {
    const scan = await performScan(
      createDirHandleFromFileList([
        makeFile("Sandbox/common/ideologies/00_ideologies.txt"),
        makeFile("Sandbox/common/ideologies/MC/MC_ideologies.txt"),
        makeFile("Sandbox/common/ideologies/notes.md"),
        makeFile("Sandbox/common/characters/HJM.txt"),
      ]),
      "HJM",
      () => {},
    );

    const paths = allPaths(scan);
    expect(paths).toContain("common/ideologies/00_ideologies.txt");
    expect(paths).toContain("common/ideologies/MC/MC_ideologies.txt");
    expect(paths).not.toContain("common/ideologies/notes.md");
    expect(scan.common_rules.dirCounts).toEqual({ ideologies: 2 });
  });

  it("omits absent directories entirely (a mod without them shows nothing)", async () => {
    const scan = await performScan(
      createDirHandleFromFileList([
        makeFile("Small/common/characters/GER.txt"),
      ]),
      "GER",
      () => {},
    );

    expect(scan.common_rules.fileCount).toBe(0);
    expect(scan.common_rules.dirCounts).toEqual({});
  });
});
