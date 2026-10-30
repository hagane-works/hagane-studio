import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const dialogSource = readFileSync(
  join(__dirname, "LoadExistingModDialog.tsx"),
  "utf8",
);
const apiSource = readFileSync(
  join(__dirname, "../../lib/api/project.ts"),
  "utf8",
);
const projectTypeSource = readFileSync(
  join(__dirname, "../../types/project.ts"),
  "utf8",
);

describe("load-existing descriptor replace_path carry", () => {
  it("accepts repeated replace paths in the auto-detect response and resets them per pick", () => {
    expect(dialogSource).toContain("replace_paths?: string[];");
    expect(dialogSource).toContain("Array.isArray(parsed.replace_paths)");
    expect(dialogSource).toContain("setParsedReplacePaths([]);");
    expect(dialogSource).toContain("!scanning &&");
  });

  it("sends parsed paths as durable imported metadata on project creation", () => {
    expect(dialogSource).toContain(
      "imported_descriptor_replace_paths: parsedReplacePaths",
    );
    expect(apiSource).toContain(
      "imported_descriptor_replace_paths?: string[];",
    );
    expect(projectTypeSource).toContain(
      "imported_descriptor_replace_paths?: string[];",
    );
  });
});
