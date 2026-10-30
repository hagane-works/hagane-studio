import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const HERE = path.dirname(
  new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1"),
);

const SRC = fs
  .readFileSync(path.join(HERE, "layout.tsx"), "utf8")
  .replace(/\/\*[\s\S]*?\*\//g, "")
  .replace(/(^|[^:])\/\/.*$/gm, "$1");

describe("sidebar · a way back from an expanded 深化 section", () => {
  it("groups still default to collapsed", () => {
    const table = SRC.slice(
      SRC.indexOf("TOOLS_GROUP_DEFAULT_COLLAPSED"),
      SRC.indexOf("toolsGroupStorageKey ="),
    );
    for (const id of ["content", "military", "localisation", "misc"]) {
      expect(table).toMatch(new RegExp(`${id}:\\s*true`));
    }
  });

  it("a collapse-all control exists and persists to the same storage keys", () => {
    expect(SRC).toMatch(/const collapseAllToolGroups = \(\) => \{/);

    const fn = SRC.slice(
      SRC.indexOf("const collapseAllToolGroups"),
      SRC.indexOf("const toggleToolGroup"),
    );
    expect(fn).toMatch(/for \(const id of TOOLS_GROUP_IDS\)/);
    expect(fn).toMatch(/toolsGroupStorageKey\(id\),\s*"1"/);
    expect(fn).toMatch(/setToolGroupCollapsed\(/);
  });

  it("the control is conditional — no permanent clutter in a tidy sidebar", () => {
    expect(SRC).toMatch(/const expandedGroupCount = TOOLS_GROUP_IDS\.filter\(/);
    expect(SRC).toMatch(/expandedGroupCount > 0 \? \(/);

    expect(SRC).toMatch(/!sidebarCollapsed && expandedGroupCount > 0/);
  });

  it("it hangs off the 深化 label, not somewhere the user has to hunt for", () => {
    const deepen = SRC.slice(
      SRC.indexOf('label={t("sectionDeepen")}'),
      SRC.indexOf('label={t("sectionDeepen")}') + 1400,
    );
    expect(deepen).toMatch(/collapseAllToolGroups/);
    expect(deepen).toMatch(/t\("collapseAllGroups"\)/);
  });

  it("group headers read as controls at REST, not only on hover", () => {
    const header = SRC.slice(
      SRC.indexOf("function ToolsGroupHeader"),
      SRC.indexOf("export default function ProjectLayout"),
    );

    expect(header).toMatch(/collapsed\s*\?\s*"bg-sidebar-accent\//);
    expect(header).toMatch(/:\s*"bg-sidebar-accent\//);

    expect(header).toMatch(/border-border\/60/);

    expect(header).toMatch(/ChevronRight/);
    expect(header).toMatch(/ChevronDown/);
  });
});
