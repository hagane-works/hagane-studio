import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import { customTraitsApi } from "./characters";
import { miosApi } from "./military";
import { countriesApi, superEventsApi } from "./scenario";
import { ideologyIconsApi } from "./tooling";

describe("final creator assets use fixed Go routes", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({
        ok: true,
        status: 200,
        statusText: "OK",
        json: async () => ({}),
        text: async () => "",
      })),
    );
  });

  afterEach(() => vi.unstubAllGlobals());

  it("routes custom traits and MIO trait icons to object-only Go handlers", async () => {
    await customTraitsApi.uploadIcon(182, 7, new File(["x"], "trait.png"));
    await miosApi.uploadTraitIcon(
      182,
      8,
      "fast track",
      new File(["x"], "mio.png"),
    );

    expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe(
      "/api/go/projects/182/custom-traits/7/upload-icon",
    );
    expect(vi.mocked(fetch).mock.calls[1]?.[0]).toBe(
      "/api/go/projects/182/mios/8/traits/fast%20track/upload-icon",
    );
    expect(customTraitsApi.iconUrl(182, 7)).toBe(
      "/api/go/projects/182/custom-traits/7/icon",
    );
    expect(miosApi.traitIconUrl(182, 8, "fast track")).toBe(
      "/api/go/projects/182/mios/8/traits/fast%20track/icon",
    );
  });

  it("routes super-event, ideology and base/variant country pictures to Go", async () => {
    await superEventsApi.uploadPicture(182, 9, new File(["x"], "event.png"));
    await ideologyIconsApi.upload(
      182,
      "democratic",
      new File(["x"], "ideology.png"),
    );
    await countriesApi.uploadFlag(182, 10, new File(["x"], "base.png"));
    await countriesApi.uploadFlag(
      182,
      10,
      new File(["x"], "variant.png"),
      "fascism",
    );

    expect(vi.mocked(fetch).mock.calls.map(([path]) => path)).toEqual([
      "/api/go/projects/182/super-events/9/upload-picture",
      "/api/go/projects/182/ideology-icons/democratic/upload",
      "/api/go/projects/182/countries/10/upload-flag",
      "/api/go/projects/182/countries/10/upload-flag?ideology=fascism",
    ]);
    expect(superEventsApi.picturePreviewUrl(182, 9)).toBe(
      "/api/go/projects/182/super-events/9/picture",
    );
    expect(ideologyIconsApi.previewUrl(182, "democratic")).toBe(
      "/api/go/projects/182/ideology-icons/democratic",
    );
    expect(countriesApi.flagPreviewUrl(182, 10)).toBe(
      "/api/go/projects/182/countries/10/flag",
    );
    expect(countriesApi.flagPreviewUrl(182, 10, "fascism")).toBe(
      "/api/go/projects/182/countries/10/flag?ideology=fascism",
    );
  });
});
