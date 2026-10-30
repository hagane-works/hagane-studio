import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

describe("country switcher unsaved-change guard", () => {
  it("checks the shared dirty guard before changing the active country", () => {
    const source = readFileSync(
      join(__dirname, "../CountrySwitcher.tsx"),
      "utf8",
    );
    const pick = source.match(
      /const pick = useCallback\([\s\S]*?\n\s*const activeItem/,
    )?.[0];

    expect(source).toContain('from "@/lib/hooks/useUnsavedGuard"');
    expect(pick).toBeDefined();
    expect(pick).toContain("tag !== activeCountry");
    expect(pick!.indexOf("confirmDiscardIfDirty()")).toBeGreaterThanOrEqual(0);
    expect(pick!.indexOf("confirmDiscardIfDirty()")).toBeLessThan(
      pick!.indexOf("setActiveCountryTag(tag)"),
    );
  });
});
