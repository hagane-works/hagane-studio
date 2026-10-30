import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const read = (path: string) =>
  readFileSync(resolve(process.cwd(), path), "utf8");

const listWorkbenchSources = [
  "src/components/workbench/BookmarksContent.tsx",
  "src/components/workbench/DoctrinesContent.tsx",
  "src/components/workbench/HistoryFilesContent.tsx",
  "src/components/workbench/MIOsDialog.tsx",
  "src/components/workbench/ScriptedLocalisationsContent.tsx",
  "src/components/workbench/ScriptedSnippetsContent.tsx",
  "src/components/workbench/SuperEventsContent.tsx",
  "src/app/[locale]/project/[id]/wargoals/page.tsx",
] as const;

describe("Extended-B workbench load-state contract", () => {
  it.each(listWorkbenchSources)(
    "renders load failure before a zero-data creator state: %s",
    (path) => {
      const source = read(path);

      expect(source).toContain(
        'import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";',
      );
      expect(source).toMatch(
        /(?:!loading && )?listLoadFailed && rows\.length === 0 \? \([\s\S]*?<WorkbenchLoadError/,
      );
      expect(source).not.toContain("setRows([])");
    },
  );
});
