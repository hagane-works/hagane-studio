import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("project catalog reads use fixed Go routes", () => {
  it("keeps technology folders and known traits off Python without a fallback", () => {
    const characterEditor = readFileSync(
      resolve(process.cwd(), "src/components/workbench/CharacterEditor.tsx"),
      "utf8",
    );
    const technologyPage = readFileSync(
      resolve(
        process.cwd(),
        "src/app/[locale]/project/[id]/technology/page.tsx",
      ),
      "utf8",
    );

    expect(characterEditor).toContain(
      "/api/go/projects/${projectId}/custom-traits/known?trait_type=country_leader",
    );
    expect(characterEditor).toContain(
      "/api/go/projects/${projectId}/custom-traits/known?trait_type=unit_leader",
    );
    expect(characterEditor).not.toContain(
      "${API_BASE}/projects/${projectId}/custom-traits/known",
    );
    expect(technologyPage).toContain(
      "/api/go/projects/${projectId}/technologies/known-folders",
    );
    expect(technologyPage).not.toContain(
      "${API_BASE}/projects/${projectId}/technologies/known-folders",
    );
    expect(`${characterEditor}\n${technologyPage}`).not.toContain(
      "NEXT_PUBLIC_GO",
    );
  });
});
