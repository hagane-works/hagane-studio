import { afterEach, describe, expect, it, vi } from "vitest";

import {
  chunkNameOf,
  claimChunkReload,
  isChunkLoadError,
} from "./chunkLoadRecovery";

const turbopack = Object.assign(
  new Error(
    "Failed to load chunk /_next/static/chunks/15p87dy5e2ywy.js from module 964893",
  ),
  { name: "ChunkLoadError" },
);

describe("chunk load recovery", () => {
  afterEach(() => {
    window.sessionStorage.clear();
    vi.restoreAllMocks();
  });

  it("recognises a page file that did not load, in the forms browsers report it", () => {
    expect(isChunkLoadError(turbopack)).toBe(true);
    expect(isChunkLoadError(new Error("Loading chunk 412 failed."))).toBe(true);
    expect(
      isChunkLoadError(
        new TypeError(
          "Failed to fetch dynamically imported module: https://x/a.js",
        ),
      ),
    ).toBe(true);
    expect(
      isChunkLoadError(new TypeError("Importing a module script failed.")),
    ).toBe(true);
    expect(
      isChunkLoadError(
        new TypeError("Cannot read properties of undefined (reading 'find')"),
      ),
    ).toBe(false);
    expect(isChunkLoadError(null)).toBe(false);
    expect(isChunkLoadError("Failed to load chunk")).toBe(false);
  });

  it("names the file without host or query", () => {
    expect(chunkNameOf(turbopack)).toBe("15p87dy5e2ywy.js");
    expect(chunkNameOf(new Error("Loading chunk 412 failed."))).toBe("");
  });

  it("reloads once, then not again within a minute, and never over unsaved edits", () => {
    expect(claimChunkReload(true, 1_000_000)).toBe(false);
    expect(claimChunkReload(false, 1_000_000)).toBe(true);
    expect(claimChunkReload(false, 1_030_000)).toBe(false);
    expect(claimChunkReload(false, 1_061_000)).toBe(true);
  });

  it("does not reload without session storage, the loop guard", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    expect(claimChunkReload(false)).toBe(false);
  });
});
