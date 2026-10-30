import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const ideaSource = readFileSync(join(__dirname, "IdeaEditor.tsx"), "utf8");

const superEventSource = readFileSync(
  join(__dirname, "SuperEventEditor.tsx"),
  "utf8",
);
const doctrineSource = readFileSync(
  join(__dirname, "DoctrinesContent.tsx"),
  "utf8",
);

describe("content editor authoring UX contract", () => {
  it("routes Idea trigger and effect blocks through the shared script editor", () => {
    expect(
      ideaSource.match(/<ScriptBlockEditor/g)?.length,
    ).toBeGreaterThanOrEqual(10);
    expect(ideaSource).toContain("templates={CONDITION_TEMPLATES}");
    expect(ideaSource).toContain("templates={EFFECT_TEMPLATES}");
    expect(ideaSource).not.toMatch(
      /<Textarea[^>]*>[\s\S]{0,300}form\.available/,
    );
  });

  it("uses the shared effect editor and an unrestricted numeric step for SuperEvents", () => {
    expect(superEventSource).toContain("<ScriptBlockEditor");
    expect(superEventSource).toContain("templates={EFFECT_TEMPLATES}");
    expect(superEventSource).toContain('step="any"');
    expect(superEventSource).toContain(
      "parseSuperEventAiChance(e.target.value)",
    );
    expect(superEventSource).not.toContain('step="0.1"');
  });

  it("offers doctrine authors the existing project icon library without removing manual GFX entry", () => {
    expect(doctrineSource).toContain("<FocusIconLibraryDialog");
    expect(doctrineSource).toContain(
      "onClick={() => setIconLibraryOpen(true)}",
    );
    expect(doctrineSource).toContain(
      'onSelect={(gfxName) => update("icon", gfxName)}',
    );
    expect(doctrineSource).toContain(
      'onChange={(e) => update("icon", e.target.value)}',
    );
    expect(doctrineSource).toContain("projectOnly");
    expect(doctrineSource).toContain('title={t("iconDialogTitle")}');
  });
});
