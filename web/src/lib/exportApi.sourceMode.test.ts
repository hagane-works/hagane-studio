import { beforeEach, describe, expect, it, vi } from "vitest";

const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }));

vi.mock("./api/_client", () => ({
  API_BASE: "/api",
  request: requestMock,
}));

import { exportApi } from "./api/export";

function callTo(prefix: string): [string, { body?: string } | undefined] {
  const hit = requestMock.mock.calls.find(([url]) => {
    const s = String(url);
    return s.startsWith(prefix) || s.startsWith(`/go${prefix}`);
  });
  if (!hit) {
    throw new Error(
      `no request to ${prefix}; saw: ${requestMock.mock.calls
        .map(([url]) => String(url))
        .join(", ")}`,
    );
  }
  return hit as [string, { body?: string } | undefined];
}

describe("export source identity wiring", () => {
  beforeEach(() => {
    requestMock.mockReset();

    requestMock
      .mockResolvedValueOnce({})
      .mockResolvedValueOnce({})
      .mockResolvedValueOnce({})
      .mockResolvedValueOnce({ job_id: "job-1", status: "queued" })
      .mockResolvedValueOnce({
        job_id: "job-1",
        status: "complete",
        files: [],
      });
  });

  it("keeps check, validation, and structure preview on the full-package slice", async () => {
    await exportApi.validate(7, true);
    await exportApi.check(7, true);
    await exportApi.previewStructure(7, true);

    expect(callTo("/projects/7/export/validate")[0]).toBe(
      "/go/projects/7/export/validate?include_imported=true",
    );

    expect(callTo("/projects/7/export/check")[0]).toBe(
      "/go/projects/7/export/check?include_imported=true",
    );
    expect(callTo("/projects/7/export/check")[1]).toEqual({ method: "POST" });
    expect(requestMock).toHaveBeenCalledWith(
      "/go/export/jobs",
      expect.objectContaining({ method: "POST" }),
    );
    expect(JSON.parse(callTo("/go/export/jobs")[1]!.body!)).toMatchObject({
      project_id: 7,
      include_imported: true,
    });
    expect(requestMock).toHaveBeenCalledWith("/go/export/jobs/7/job-1");
  });

  it("asks the start-date lint as part of the check", async () => {
    await exportApi.check(7, true);
    expect(callTo("/go/lint/start-dates/7")[0]).toBe("/go/lint/start-dates/7");
  });

  it("preserves the created-only defaults", async () => {
    await exportApi.validate(7);
    await exportApi.check(7);
    await exportApi.previewStructure(7);

    expect(
      JSON.parse(callTo("/go/export/jobs")[1]!.body!).include_imported,
    ).toBe(false);
  });

  it("returns a completed backend-cached artifact without polling again", async () => {
    requestMock.mockReset();
    requestMock
      .mockResolvedValueOnce({
        enabled: false,
        job_kinds: ["parse_full", "export_full"],
      })
      .mockResolvedValueOnce({
        job_id: "cached-job",
        status: "complete",
        files: ["descriptor.mod"],
      });

    const result = await exportApi.previewStructure(7, true);

    expect(result.job_id).toBe("cached-job");
    expect(result.files).toEqual(["descriptor.mod"]);
    expect(requestMock).toHaveBeenCalledTimes(2);
  });

  it("keeps polling when one export status read is temporarily unavailable", async () => {
    vi.useFakeTimers();
    requestMock.mockReset();
    requestMock
      .mockResolvedValueOnce({
        enabled: false,
        job_kinds: ["parse_full", "export_full"],
      })
      .mockResolvedValueOnce({ job_id: "job-1", status: "queued" })
      .mockRejectedValueOnce(
        Object.assign(new Error("temporary"), { status: 503 }),
      )
      .mockResolvedValueOnce({
        job_id: "job-1",
        status: "complete",
        files: ["descriptor.mod"],
      });

    const resultPromise = exportApi.previewStructure(7);
    await vi.runAllTimersAsync();
    const result = await resultPromise;

    expect(result.status).toBe("complete");
    expect(result.files).toEqual(["descriptor.mod"]);
    expect(requestMock).toHaveBeenCalledTimes(4);
    vi.useRealTimers();
  });

  it("threads one descriptor selection through all diagnostic entrypoints", async () => {
    const replacePaths = ["common/national_focus", "events"];
    await exportApi.validate(7, false, replacePaths);
    await exportApi.check(7, false, replacePaths);
    await exportApi.previewStructure(7, false, replacePaths);

    expect(callTo("/projects/7/export/validate")[0]).toContain(
      "replace_paths=common%2Fnational_focus%2Cevents",
    );
    expect(callTo("/projects/7/export/check")[0]).toContain(
      "replace_paths=common%2Fnational_focus%2Cevents",
    );
    expect(
      JSON.parse(callTo("/go/export/jobs")[1]!.body!).replace_paths,
    ).toEqual(replacePaths);
  });
});
