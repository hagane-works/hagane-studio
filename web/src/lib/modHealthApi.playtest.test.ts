import { afterEach, describe, expect, it, vi } from "vitest";
import { analyzePlaytestBundle } from "./modHealthApi";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("analyzePlaytestBundle", () => {
  it("sends system log, inventory and trimmed expected identity fields", async () => {
    const responseBody = {
      findings: [],
      totals: { error: 0, warning: 0, info: 0 },
      count: 0,
      files_present: {
        game_log: false,
        exception_txt: false,
        dlc_load_json: false,
        system_log: true,
        mod_inventory_json: true,
      },
      built_date: null,
      loaded_date: null,
      runtime_diagnostics: null,
      runtime_diagnostics_error: null,
    };
    const fetchMock = vi.fn<typeof fetch>(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(responseBody),
      } as Response),
    );
    vi.stubGlobal("fetch", fetchMock);

    const systemLog = new File(["Active Mod Count: 1"], "system.log", {
      type: "text/plain",
    });
    const inventory = new File(['{"descriptors":[]}'], "mods.json", {
      type: "application/json",
    });
    await analyzePlaytestBundle(
      42,
      { systemLog, modInventoryJson: inventory },
      {
        name: "  Expected Mod  ",
        path: "  mod/expected.mod  ",
        buildId: "  hagane-build-123  ",
      },
    );

    expect(fetchMock).toHaveBeenCalledOnce();
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/go/lint/playtest-bundle/42");
    expect(init?.method).toBe("POST");
    const form = init?.body as FormData;
    expect((form.get("system_log") as File).name).toBe("system.log");
    expect((form.get("mod_inventory_json") as File).name).toBe("mods.json");
    expect(form.get("expected_mod_name")).toBe("Expected Mod");
    expect(form.get("expected_mod_path")).toBe("mod/expected.mod");
    expect(form.get("expected_build_id")).toBe("hagane-build-123");
  });

  it("does not invent expected identity fields when the form is blank", async () => {
    const fetchMock = vi.fn<typeof fetch>(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve({}),
      } as Response),
    );
    vi.stubGlobal("fetch", fetchMock);

    await analyzePlaytestBundle(
      7,
      { systemLog: new File(["Active Mod Count: 0"], "system.log") },
      { name: " ", path: "", buildId: "  " },
    );

    const form = fetchMock.mock.calls[0][1]?.body as FormData;
    expect(form.has("expected_mod_name")).toBe(false);
    expect(form.has("expected_mod_path")).toBe(false);
    expect(form.has("expected_build_id")).toBe(false);
  });
});
