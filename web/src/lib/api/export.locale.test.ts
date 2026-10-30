import { afterEach, describe, expect, it, vi } from "vitest";
import { exportApi } from "./export";
import { useLocaleStore } from "@/lib/store/localeStore";

describe("exports carry the interface language", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    useLocaleStore.setState({ locale: "en" });
  });

  function stubFetch(body: unknown) {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(body), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    return fetchMock;
  }

  it("sends the locale with an export job", async () => {
    useLocaleStore.setState({ locale: "zh" });
    const fetchMock = stubFetch({
      job_id: "j1",
      status: "complete",
      files: [],
    });
    await exportApi.buildArtifact(7, false, [], true, false);
    expect(fetchMock.mock.calls[0][0]).toBe("/api/go/export/jobs");
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toMatchObject({
      project_id: 7,
      locale: "zh",
    });
  });

  it("sends the locale with an install", async () => {
    useLocaleStore.setState({ locale: "ko" });
    const fetchMock = stubFetch({
      mod_folder: "m",
      descriptor: "m.mod",
      files: 1,
    });
    await exportApi.install(7);
    expect(fetchMock.mock.calls[0][0]).toBe("/api/app/projects/7/install");
    expect(JSON.parse(fetchMock.mock.calls[0][1].body).locale).toBe("ko");
  });
});
