import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

const requireCentralSeams =
  process.env.HOI4_REQUIRE_POLITICS_WORKBENCH_SEAMS === "1";

describe.skipIf(!requireCentralSeams)(
  "politics ideology workbench central seams",
  () => {
    it("mounts the private workbench at the recoverable ideologies route", () => {
      const routeDirectory = path.join(
        process.cwd(),
        "src",
        "app",
        "[locale]",
        "project",
        "[id]",
        "ideologies",
      );
      expect(
        fs.existsSync(routeDirectory),
        "central must create src/app/[locale]/project/[id]/ideologies",
      ).toBe(true);

      const source = fs
        .readdirSync(routeDirectory)
        .filter((name) => name.endsWith(".ts") || name.endsWith(".tsx"))
        .map((name) => fs.readFileSync(path.join(routeDirectory, name), "utf8"))
        .join("\n");
      expect(source).toMatch(/IdeologyLibraryWorkbench/);
      expect(source).toMatch(/initialItemId/);
      expect(source).toMatch(/onItemChange/);
    });

    it("protects unsaved ideology edits through the shared route guard", () => {
      const workbench = fs.readFileSync(
        path.join(__dirname, "IdeologyLibrary.tsx"),
        "utf8",
      );
      expect(workbench).toContain(
        'useRegisterDirty("ideologies", editDirty || creatorDirty)',
      );
    });
  },
);
