import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  ApiError,
  WRITE_BLOCKED_EVENT,
  request,
  responseError,
  writeBlockedFolder,
} from "./_client";

const blocked = {
  detail: "could not write C:\\Users\\me\\Documents\\mods\\files\\p.png: Access is denied.",
  code: "write_blocked",
  folder: "C:\\Users\\me\\Documents\\mods\\files",
};

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("write-blocked errors", () => {
  let events: CustomEvent[];
  let spy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    events = [];
    spy = vi.spyOn(window, "dispatchEvent").mockImplementation((e: Event) => {
      if (e.type === WRITE_BLOCKED_EVENT) events.push(e as CustomEvent);
      return true;
    });
  });

  afterEach(() => {
    spy.mockRestore();
    vi.unstubAllGlobals();
  });

  it("turns an upload response into an ApiError that names the folder", async () => {
    const err = await responseError(jsonResponse(403, blocked), "Upload failed");
    expect(err).toBeInstanceOf(ApiError);
    expect(err.message).toBe(`Upload failed: ${blocked.detail}`);
    expect(writeBlockedFolder(err)).toBe(blocked.folder);
    expect(events.map((e) => e.detail)).toEqual([{ folder: blocked.folder }]);
  });

  it("raises the notice from request() too", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(403, blocked)) as unknown as typeof fetch,
    );
    const err = await request("/go/projects/1/characters").catch((e) => e);
    expect(writeBlockedFolder(err)).toBe(blocked.folder);
    expect(events).toHaveLength(1);
  });

  it("leaves other failures alone", async () => {
    const err = await responseError(
      jsonResponse(500, { detail: "disk full", code: "write_failed" }),
    );
    expect(err.message).toBe("disk full");
    expect(writeBlockedFolder(err)).toBeNull();
    expect(writeBlockedFolder(new Error("x"))).toBeNull();
    const plain = await responseError(new Response("oops", { status: 502 }));
    expect(plain.message).toBe("oops");
    expect(events).toHaveLength(0);
  });
});
