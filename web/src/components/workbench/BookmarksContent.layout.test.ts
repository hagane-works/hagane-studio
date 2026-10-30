import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const source = readFileSync(join(__dirname, "BookmarksContent.tsx"), "utf8");

const editor = readFileSync(join(__dirname, "BookmarkEditor.tsx"), "utf8");

describe("bookmark workbench layout contract", () => {
  it("keeps WorkbenchShell as the sole editor scroll owner", () => {
    const editorStart = source.indexOf("editorPanel={");
    const previewStart = source.indexOf("previewPanel={", editorStart);
    const editorPanel = source.slice(editorStart, previewStart);

    expect(editorStart).toBeGreaterThan(-1);
    expect(previewStart).toBeGreaterThan(editorStart);
    expect(editorPanel).not.toContain("{campaignSetupPanel}");
    expect(editorPanel).not.toContain("overflow-y-auto");
    expect(editorPanel).not.toContain('className="h-full"');
    expect(source).toContain("{campaignSetupPanel}");
    expect(source.indexOf("{campaignSetupPanel}")).toBeLessThan(editorStart);
    expect(source).toContain("flex-col overflow-clip");

    expect(source).toContain("<BookmarkEditor");
    expect(editor).not.toContain("overflow-y-auto");
  });

  it("keeps the backdrop uploader discoverable while a new bookmark is locked", () => {
    expect(editor).toContain("disabled={!isExisting}");
    expect(editor).toContain('t("backdropUploadCreateFirst")');
    expect(editor).toContain("key={`bookmark-backdrop:${activeId}`}");
    expect(editor).not.toContain(
      '{typeof activeId === "number" ? (\n                    <div className="pt-1">\n                      <ImageUpload',
    );
  });

  it("uses the vanilla 180x104 scenario-card slot instead of a full-screen backdrop", () => {
    expect(editor).toContain("targetSize={{ width: 180, height: 104 }}");
    expect(editor).not.toContain("targetSize={{ width: 1810, height: 540 }}");
  });
});
