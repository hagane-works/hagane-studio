import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const frontendRoot = join(here, "..", "..");
const read = (path: string) => readFileSync(join(frontendRoot, path), "utf8");

const exportPage = read("src/app/[locale]/project/[id]/export/page.tsx");
const settingsDialog = read("src/components/export/ExportSettingsDialog.tsx");
const appendDialog = read(
  "src/components/workbench/AppendToExistingModDialog.tsx",
);
const diffDialog = read("src/components/workbench/DiffMergeExportDialog.tsx");

function stripComments(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, "").replace(/^\s*\/\/.*$/gm, "");
}

function conditionalGuidance(source: string, testId: string): string {
  const match = source.match(
    new RegExp(
      `\\{hasTechnologySurface \\? \\(\\s*<section[\\s\\S]*?data-testid="${testId}"[\\s\\S]*?<\\/section>\\s*\\) : null\\}`,
    ),
  );
  expect(match).not.toBeNull();
  return match?.[0] ?? "";
}

describe("technology surface export guidance wiring", () => {
  it("derives the project signal from non-empty surface branches and passes it explicitly", () => {
    expect(exportPage).toMatch(
      /const hasTechnologySurface =\s*\(currentProject\?\.tech_surface_manifest\?\.branches\?\.length \?\? 0\) > 0;/,
    );
    expect(
      exportPage.match(/hasTechnologySurface=\{hasTechnologySurface\}/g),
    ).toHaveLength(3);

    for (const dialog of [settingsDialog, appendDialog, diffDialog]) {
      expect(dialog).toContain("hasTechnologySurface: boolean;");
      expect(dialog).toContain("hasTechnologySurface,");
    }
  });

  it("renders the settings explanation only for technology surface projects", () => {
    const guidance = conditionalGuidance(
      settingsDialog,
      "technology-additive-export-guidance",
    );

    for (const key of [
      "technologyAdditiveTitle",
      "technologyAdditiveBody",
      "technologyAdditiveFiles",
      "technologyReplacePathNote",
    ]) {
      expect(guidance).toContain(`t("${key}")`);
    }
    expect(guidance).toContain("border-border");
    expect(guidance).toContain("bg-muted/20");
    expect(guidance).toContain("<Info");
    expect(guidance).not.toMatch(/warning|destructive|AlertTriangle/);
  });

  it("renders technology-specific append and diff boundaries as neutral information", () => {
    const appendGuidance = conditionalGuidance(
      appendDialog,
      "technology-append-guidance",
    );
    const diffGuidance = conditionalGuidance(
      diffDialog,
      "technology-diff-merge-guidance",
    );

    for (const guidance of [appendGuidance, diffGuidance]) {
      expect(guidance).toContain('t("technologyMergeTitle")');
      expect(guidance).toContain('t("technologyMergeBody")');
      expect(guidance).toContain("border-border");
      expect(guidance).toContain("bg-muted/20");
      expect(guidance).toContain("<Info");
      expect(guidance).not.toMatch(/warning|destructive|AlertTriangle/);
    }
  });

  it("does not auto-enable or auto-open a technology replace_path", () => {
    const executablePage = stripComments(exportPage);
    const executableSettings = stripComments(settingsDialog);

    expect(executablePage).toContain(
      "const [replacePaths, setReplacePaths] = useState<Record<string, boolean>>({});",
    );
    expect(executablePage).not.toContain("showReplacePathSection");
    expect(executableSettings).not.toMatch(
      /hasDependencies|showReplacePathSection/,
    );
    expect(executablePage).not.toMatch(/technologies\s*:\s*true/);
    expect(executablePage).not.toContain(
      'onToggleReplacePath("technologies", true)',
    );
    expect(executableSettings).not.toContain("setReplaceOpen(true)");
    expect(executableSettings).not.toContain(
      'onToggleReplacePath("technologies", true)',
    );
  });
});
