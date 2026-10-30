import { afterEach, describe, expect, it, vi } from "vitest";
import {
  analyzeErrorLog,
  analyzePlaytestBundle,
  getPerfProfile,
  getStartDateLint,
} from "./modHealthApi";

afterEach(() => {
  vi.unstubAllGlobals();
});

function stubFetch() {
  const fetchMock = vi.fn<typeof fetch>(() =>
    Promise.resolve({ ok: true, json: () => Promise.resolve({}) } as Response),
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

describe("mod-health Go routing", () => {
  it("routes perf + start-dates reads to Go", async () => {
    const fetchMock = stubFetch();
    await getPerfProfile(42);
    await getStartDateLint(42);
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "/api/go/lint/perf/42",
      "/api/go/lint/start-dates/42",
    ]);
  });

  it("routes the error.log multipart upload to Go under the `file` field", async () => {
    const fetchMock = stubFetch();
    await analyzeErrorLog(42, new File(["[1][2][x.cpp:3]: boom"], "error.log"));
    expect(fetchMock).toHaveBeenCalledOnce();
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/go/lint/error-log/42");
    expect(init?.method).toBe("POST");
    expect(((init?.body as FormData).get("file") as File).name).toBe(
      "error.log",
    );

    expect(
      (init?.headers as Record<string, string> | undefined)?.["Content-Type"],
    ).toBeUndefined();
  });

  it("routes the playtest bundle upload to Go with all five evidence fields", async () => {
    const fetchMock = stubFetch();
    await analyzePlaytestBundle(42, {
      gameLog: new File(["Start-date: 1936.1.1.12"], "game.log"),
      exceptionTxt: new File(
        ["Unhandled Exception 0x1 (Boom)"],
        "exception.txt",
      ),
      dlcLoadJson: new File(['{"enabled_mods":[]}'], "dlc_load.json"),
      systemLog: new File(["Active Mod Count: 1"], "system.log"),
      modInventoryJson: new File(["[]"], "mods.json"),
    });
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/go/lint/playtest-bundle/42");
    const form = init?.body as FormData;
    for (const field of [
      "game_log",
      "exception_txt",
      "dlc_load_json",
      "system_log",
      "mod_inventory_json",
    ]) {
      expect(form.get(field)).toBeInstanceOf(File);
    }
  });
});
