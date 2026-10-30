import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";
import { guiArray, guiRecord, readGuiResponse } from "./guiResponse";

describe("readGuiResponse", () => {
  it("returns a successful gui-render payload", async () => {
    const response = new Response(JSON.stringify({ subjects: [] }), {
      status: 200,
      headers: { "content-type": "application/json" },
    });
    await expect(
      readGuiResponse<{ subjects: unknown[] }>(response),
    ).resolves.toEqual({ subjects: [] });
  });

  it("rejects an HTTP error instead of exposing its detail object as game data", async () => {
    const response = new Response(
      JSON.stringify({ detail: "not found: subjects" }),
      {
        status: 404,
        statusText: "Not Found",
        headers: { "content-type": "application/json" },
      },
    );
    await expect(readGuiResponse(response)).rejects.toThrow(
      "not found: subjects",
    );
  });

  it("reports a non-JSON HTTP failure without crashing a renderer later", async () => {
    const response = new Response("upstream unavailable", {
      status: 502,
      statusText: "Bad Gateway",
    });
    await expect(readGuiResponse(response)).rejects.toThrow("Bad Gateway");
  });

  it("normalises arrays and records omitted by an older successful payload", () => {
    expect(guiArray(undefined)).toEqual([]);
    expect(guiArray({ detail: "old route" } as never)).toEqual([]);
    expect(guiArray([1, 2])).toEqual([1, 2]);
    expect(guiRecord(undefined)).toEqual({});
    expect(guiRecord([] as never)).toEqual({});
    expect(guiRecord({ GFX_icon: { resolved: true } })).toEqual({
      GFX_icon: { resolved: true },
    });
  });

  it("keeps gui-render view readers from returning unchecked JSON", () => {
    const unchecked = /return\s+(?:await\s+)?response\.json\(\)\s*;/g;
    const offenders = readdirSync(__dirname)
      .filter(
        (name) =>
          /\.(?:ts|tsx)$/.test(name) &&
          !name.endsWith(".test.ts") &&
          !name.endsWith(".test.tsx"),
      )
      .flatMap((name) => {
        const source = readFileSync(join(__dirname, name), "utf8");
        if (!source.includes("/gui-render/")) return [];
        return [...source.matchAll(unchecked)].map(
          (match) =>
            `${name}:${source.slice(0, match.index).split("\n").length}`,
        );
      });

    expect(
      offenders,
      "Use readGuiResponse, or explicitly branch on response.ok when an empty fallback is intentional",
    ).toEqual([]);
  });
});
