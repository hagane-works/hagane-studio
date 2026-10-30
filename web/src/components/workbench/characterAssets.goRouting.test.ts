import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("character artwork Go routing", () => {
  it("keeps editor and in-game previews off the Python asset path", () => {
    const preview = readFileSync(
      resolve(
        process.cwd(),
        "src/components/workbench/CharacterInGamePreview.tsx",
      ),
      "utf8",
    );
    const upload = readFileSync(
      resolve(process.cwd(), "src/components/workbench/PortraitUpload.tsx"),
      "utf8",
    );

    expect(preview).toContain(
      "/go/projects/${projectId}/characters/${character.id}/assets/",
    );
    expect(preview).not.toContain(
      "${API_BASE}/projects/${projectId}/characters/",
    );
    expect(upload).toContain(
      "charactersApi.resetPortrait(projectId, characterId)",
    );
    expect(upload).toContain(
      "`${API_BASE}/go/projects/${projectId}/characters/${characterId}/assets/",
    );
    expect(upload).not.toContain("${API_BASE}/api/go/projects/");
    expect(upload).not.toContain(
      "/api/projects/${projectId}/characters/${characterId}/portrait",
    );
  });
});
