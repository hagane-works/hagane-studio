import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(),
  xhrUploadWithProgress: vi.fn(),
  fetchWithTimeoutRetry: vi.fn(),
}));

import { importApi } from "./import";

const okResponse = {
  ok: true,
  status: 200,
  json: async () => ({
    total: 1,
    created: 1,
    updated: 0,
    skipped: 0,
    preserved_fields: {},
    warnings: [],
    conflict_policy: "overwrite",
  }),
};

beforeEach(() => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => okResponse),
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("single-entity import Go routing", () => {
  it("routes Characters and Focus Tree uploads to projectsvc", async () => {
    const file = new File(["characters = {}"], "source.txt", {
      type: "text/plain",
    });
    await importApi.characters(182, file, "FRA", "skip");
    await importApi.focusTree(182, file, "GER", "overwrite");

    const calls = vi.mocked(fetch).mock.calls;
    expect(calls[0]?.[0]).toBe(
      "/api/go/projects/182/import/characters?country_tag=FRA",
    );
    expect(calls[1]?.[0]).toBe(
      "/api/go/projects/182/import/focus-tree?country_tag=GER",
    );
    const characterForm = calls[0]?.[1]?.body as FormData;
    const focusForm = calls[1]?.[1]?.body as FormData;
    expect(characterForm.get("conflict_policy")).toBe("skip");
    expect(focusForm.get("conflict_policy")).toBe("overwrite");
  });

  it("routes Technologies upload to projectsvc and encodes query values", async () => {
    const file = new File(["technologies = {}"], "air tech.txt", {
      type: "text/plain",
    });
    await importApi.technologies(
      182,
      file,
      "air tech",
      "air techs/folder",
      "FRA",
    );

    const fetchMock = vi.mocked(fetch);
    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "/api/go/projects/182/import/technologies?tech_line=air%20tech&folder_name=air%20techs%2Ffolder&country_tag=FRA",
    );
    const init = fetchMock.mock.calls[0]?.[1];
    expect(init).toMatchObject({ method: "POST" });
    expect(init?.body).toBeInstanceOf(FormData);
  });

  it("routes Ideas upload to projectsvc", async () => {
    const file = new File(["ideas = {}"], "ideas.txt", { type: "text/plain" });
    await importApi.ideas(182, file, "FRA");

    expect(vi.mocked(fetch).mock.calls[0]?.[0]).toBe(
      "/api/go/projects/182/import/ideas?country_tag=FRA",
    );
  });

  it("routes Draw.io upload to projectsvc with its scope fields", async () => {
    const file = new File(["<mxGraphModel/>"], "tree.drawio", {
      type: "application/xml",
    });
    await importApi.drawio(182, file, "GER", "sketch");

    const call = vi.mocked(fetch).mock.calls[0];
    expect(call?.[0]).toBe("/api/go/projects/182/import/drawio");
    const form = call?.[1]?.body as FormData;
    expect(form.get("country_tag")).toBe("GER");
    expect(form.get("tree_name")).toBe("sketch");
  });

  it("contains no Python single-entity import URL in the API implementation", async () => {
    const file = new File(["technologies = {}"], "tech.txt");
    await importApi.characters(7, file);
    await importApi.focusTree(7, file);
    await importApi.technologies(7, file, "infantry", "infantry_folder");
    await importApi.ideas(7, file);
    await importApi.drawio(7, file);

    for (const [url] of vi.mocked(fetch).mock.calls) {
      expect(String(url)).not.toMatch(/^\/api\/projects\/7\/import\//);
    }
  });
});
