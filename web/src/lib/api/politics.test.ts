import { beforeEach, describe, expect, it, vi } from "vitest";

const requestMock = vi.hoisted(() => vi.fn());

vi.mock("./_client", () => ({ request: requestMock }));

import { politicsApi } from "./politics";

beforeEach(() => requestMock.mockReset());

describe("politicsApi", () => {
  it("uses the authenticated project request seam for reference coverage", async () => {
    requestMock.mockResolvedValue({ project_id: "p", ideologies: [] });
    const controller = new AbortController();

    await politicsApi.references(17, controller.signal);

    expect(requestMock).toHaveBeenCalledWith(
      "/go/projects/17/politics/references",
      { signal: controller.signal },
    );
  });

  it("keeps delete preview narrow and sends the one-time preview_token unchanged", async () => {
    requestMock.mockResolvedValue({});
    await politicsApi.deletePreview(17, {
      kind: "subtype",
      script_id: "civic_reformism",
    });
    await politicsApi.deleteApply(17, { preview_token: "one-time-token" });

    expect(requestMock).toHaveBeenNthCalledWith(
      1,
      "/go/projects/17/politics/delete-preview",
      {
        method: "POST",
        body: JSON.stringify({ kind: "subtype", script_id: "civic_reformism" }),
      },
    );
    expect(requestMock).toHaveBeenNthCalledWith(
      2,
      "/go/projects/17/politics/delete-apply",
      {
        method: "POST",
        body: JSON.stringify({ preview_token: "one-time-token" }),
      },
    );
  });

  it("routes authoring through the dedicated political commands", async () => {
    requestMock.mockResolvedValue({});
    await politicsApi.createRoot(17, {
      script_id: "civic_coalition",
      body: "types = { }",
      name: { en: "Civic Coalition" },
    });
    await politicsApi.createSubtype(17, {
      script_id: "civic_reformism",
      root_id: "civic_coalition",
      name: { en: "Civic Reformism" },
      random_disabled: false,
    });
    await politicsApi.updateDefinition(
      17,
      { kind: "root", script_id: "civic/coalition" },
      {
        name: { en: "Renewed Civic Coalition" },
        body: "types = { }\ncolor = { 45 95 135 }",
      },
    );

    expect(requestMock).toHaveBeenNthCalledWith(
      1,
      "/go/projects/17/politics/roots",
      expect.objectContaining({ method: "POST" }),
    );
    expect(requestMock).toHaveBeenNthCalledWith(
      2,
      "/go/projects/17/politics/subtypes",
      expect.objectContaining({ method: "POST" }),
    );
    expect(requestMock).toHaveBeenNthCalledWith(
      3,
      "/go/projects/17/politics/definitions/root/civic%2Fcoalition",
      {
        method: "PATCH",
        body: JSON.stringify({
          name: { en: "Renewed Civic Coalition" },
          body: "types = { }\ncolor = { 45 95 135 }",
        }),
      },
    );
  });
});
