import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import { vanillaDataApi, vanillaEventsApi, vanillaIdeasApi } from "./vanilla";
import { request } from "./_client";

describe("vanilla data Go routing", () => {
  it("routes vanilla JSON index endpoints to the Go service", async () => {
    vi.mocked(request).mockClear();
    await vanillaDataApi.equipmentArchetypes();
    await vanillaDataApi.subunits();
    await vanillaEventsApi.list({ q: "usa", limit: 5 });
    await vanillaEventsApi.detail("usa", 1);
    await vanillaIdeasApi.list({ category: "country", has_picture: true });
    await vanillaIdeasApi.detail("country", "neutrality_idea");
    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/vanilla/equipment-archetypes",
      "/go/vanilla/subunits",
      "/go/vanilla/events?q=usa&limit=5",
      "/go/vanilla/events/usa/1",
      "/go/vanilla/ideas?category=country&has_picture=true",
      "/go/vanilla/ideas/country/neutrality_idea",
    ]);
  });
});

describe("vanilla direct-fetch modules Go routing", () => {
  it("vanillaCollisions + vanillaScriptNames hit /go/vanilla/*", async () => {
    const fs = await import("node:fs");
    const collisions = fs.readFileSync(
      new URL("../vanillaCollisions.ts", import.meta.url),
      "utf-8",
    );
    expect(collisions).toContain('"/go/vanilla/id-collisions"');
    const scriptNames = fs.readFileSync(
      new URL("../vanillaScriptNames.ts", import.meta.url),
      "utf-8",
    );
    expect(scriptNames).toContain('"/go/vanilla/script-names"');
    expect(scriptNames).toContain('"/go/vanilla/structural-fields"');
  });

  it("icon URL builders hit /go/vanilla (切片 B · OBS 302/DDS 兜底在 Go)", async () => {
    const fs = await import("node:fs");
    const pins: Array<[string, string]> = [
      ["../techGraph.ts", "/go/vanilla/tech-icon/"],
      ["../focusTreeGraph.ts", "/go/vanilla/focus-icon/"],
      [
        "../../components/workbench/FocusNodeEditor.tsx",
        "/go/vanilla/focus-icon/",
      ],
      [
        "../../components/workbench/focus-graph/FocusIconLibraryDialog.tsx",
        "/api/go/vanilla/focus-icon/",
      ],
      [
        "../../components/workbench/VanillaIdeasTab.tsx",
        "/go/vanilla/focus-icon/",
      ],
    ];
    for (const [rel, needle] of pins) {
      const src = fs.readFileSync(new URL(rel, import.meta.url), "utf-8");
      expect(src, rel).toContain(needle);
    }
  });
});
