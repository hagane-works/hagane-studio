import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  fetchMock.mockResolvedValue({
    ok: true,
    status: 200,
    statusText: "OK",
    json: async () => ({ windows: [], edits: [] }),
    blob: async () => new Blob(),
  });
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(async () => {
  const { guiRenderApi } = await import("./guiRenderApi");
  guiRenderApi.setProjectId(null);
  vi.unstubAllGlobals();
});

describe("project GUI routes are permanently Go-owned", () => {
  it("sends the five project snapshot reads directly to /api/go", async () => {
    const { guiRenderApi } = await import("./guiRenderApi");
    guiRenderApi.setProjectId(182);

    await guiRenderApi.status();
    await guiRenderApi.listWindows();
    await guiRenderApi.getWindow("countrytechtreeview", "interface/demo.gui");
    await guiRenderApi.loadOverlay("interface/demo.gui");

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "/api/go/projects/182/gui-render/status",
      "/api/go/projects/182/gui-render/windows",
      "/api/go/projects/182/gui-render/window?name=countrytechtreeview&file=interface%2Fdemo.gui",
      "/api/go/projects/182/gui-render/overlay?file=interface%2Fdemo.gui",
    ]);

    const techHook = readFileSync(
      resolve(
        process.cwd(),
        "src/components/workbench/tech-graph/useTechTreeGeometry.ts",
      ),
      "utf8",
    );
    const panelHook = readFileSync(
      resolve(
        process.cwd(),
        "src/components/workbench/ui-panels/canvas/usePanelWindow.ts",
      ),
      "utf8",
    );

    expect(techHook).toContain("/go/projects/");
    expect(panelHook).toContain("/go/projects/");
  });

  it("sends overlay writes, refresh and bundle export directly to /api/go", async () => {
    const { guiRenderApi } = await import("./guiRenderApi");
    guiRenderApi.setProjectId(182);
    await guiRenderApi.saveOverlay({
      file: "demo.gui",
      edits: [],
      scaffold_hash: "a".repeat(64),
      expected_scaffold_hash: "a".repeat(64),
      expected_revision: "0",
    });
    await guiRenderApi.refresh();
    await guiRenderApi.exportBundle("demo.gui");
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "/api/go/projects/182/gui-render/overlay",
      "/api/go/projects/182/gui-render/refresh",
      "/api/go/projects/182/gui-render/export?file=demo.gui",
    ]);
  });

  it("uses Go for project assets", async () => {
    const { guiRenderApi } = await import("./guiRenderApi");
    guiRenderApi.setProjectId(182);

    expect(guiRenderApi.spriteUrl("GFX_demo")).toBe(
      "/api/go/projects/182/gui-render/sprite?name=GFX_demo",
    );
    expect(guiRenderApi.fontUrl("aldrich")).toBe(
      "/api/go/projects/182/gui-render/font?family=aldrich",
    );
    await guiRenderApi.getInjectionPoints();
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/go/projects/182/gui-render/scripted-gui-injection-points",
    );
  });

  it("surfaces sprite override and restore details from the service", async () => {
    const { guiRenderApi } = await import("./guiRenderApi");
    guiRenderApi.setProjectId(182);
    fetchMock
      .mockResolvedValueOnce({
        ok: false,
        status: 422,
        json: async () => ({ detail: "图片尺寸不符合目标" }),
      })
      .mockResolvedValueOnce({
        ok: false,
        status: 503,
        json: async () => ({ detail: "原图存储暂时不可用" }),
      });

    await expect(
      guiRenderApi.override(
        "GFX_demo",
        new File(["x"], "demo.png", { type: "image/png" }),
      ),
    ).rejects.toThrow("图片尺寸不符合目标");
    await expect(guiRenderApi.revert("GFX_demo")).rejects.toThrow(
      "原图存储暂时不可用",
    );
  });
});

describe("game view data version", () => {
  it("asks the project's Go route and returns the version", async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      statusText: "OK",
      json: async () => ({ version: "abc|k|t" }),
    });
    const { guiRenderApi } = await import("./guiRenderApi");
    guiRenderApi.setProjectId(76);
    await expect(guiRenderApi.dataVersion()).resolves.toBe("abc|k|t");
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/go/projects/76/gui-render/data-version",
    );
  });

  it("is checked before a quiet refresh re-requests the screens", () => {
    const page = readFileSync(
      resolve(__dirname, "../app/[locale]/game-view/page.tsx"),
      "utf8",
    );
    const start = page.indexOf("const refreshGameView = useCallback(");
    const body = page.slice(start, page.indexOf("}, [t, projectId]);", start));
    const check = body.indexOf("guiRenderApi.dataVersion()");
    const same = body.indexOf(
      "canSkipGameViewRefresh(quiet, version, checkedVersion.current)",
    );
    const leave = body.indexOf("return;", same);
    const mutate = body.indexOf("isGameViewRefreshKey(key, false)");
    expect(start).toBeGreaterThan(0);
    expect(check).toBeGreaterThan(0);
    expect(same).toBeGreaterThan(check);
    expect(leave).toBeGreaterThan(same);
    expect(mutate).toBeGreaterThan(leave);
  });
});
