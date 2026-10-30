import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

describe("PortraitUpload asset URLs", () => {
  const source = readFileSync(join(__dirname, "PortraitUpload.tsx"), "utf8");

  it("uses the shared API_BASE, not a local one", () => {
    expect(source).toMatch(
      /import \{[^}]*\bAPI_BASE\b[^}]*\} from "@\/lib\/api"/,
    );
    expect(source).not.toMatch(/const API_BASE\s*=/);
  });

  it("builds every character asset URL on it", () => {
    const urls =
      source.match(/`[^`]*\/characters\/\$\{characterId\}\/assets\/[^`]*`/g) ??
      [];
    expect(urls.length).toBeGreaterThan(0);
    for (const url of urls)
      expect(url.startsWith("`${API_BASE}/go/projects/")).toBe(true);
  });
});
