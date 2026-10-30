import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { xhrUploadWithProgress } from "./api";

type Scenario = "success" | "timeout" | "error" | "pending" | "server500";

class MockXHR {
  static scenario: Scenario = "success";
  static responseText = '{"ok":true,"needed_images":[]}';

  method = "";
  url = "";
  withCredentials = false;
  timeout = 0;
  upload: {
    onprogress:
      | ((e: {
          lengthComputable: boolean;
          loaded: number;
          total: number;
        }) => void)
      | null;
  } = {
    onprogress: null,
  };
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
  ontimeout: (() => void) | null = null;
  onabort: (() => void) | null = null;
  status = 0;
  responseText = "";
  private aborted = false;

  open(method: string, url: string) {
    this.method = method;
    this.url = url;
  }

  send() {
    queueMicrotask(() => {
      if (this.aborted) return;
      switch (MockXHR.scenario) {
        case "pending":
          return;
        case "timeout":
          this.ontimeout?.();
          return;
        case "error":
          this.onerror?.();
          return;
        default: {
          this.upload.onprogress?.({
            lengthComputable: true,
            loaded: 50,
            total: 100,
          });
          this.upload.onprogress?.({
            lengthComputable: true,
            loaded: 100,
            total: 100,
          });
          this.status = MockXHR.scenario === "server500" ? 500 : 200;
          this.responseText = MockXHR.responseText;
          this.onload?.();
        }
      }
    });
  }

  abort() {
    this.aborted = true;
    this.onabort?.();
  }
}

const origXHR = (globalThis as { XMLHttpRequest?: unknown }).XMLHttpRequest;

beforeEach(() => {
  (globalThis as { XMLHttpRequest?: unknown }).XMLHttpRequest =
    MockXHR as unknown;
  MockXHR.scenario = "success";
  MockXHR.responseText = '{"ok":true,"needed_images":[]}';
});

afterEach(() => {
  (globalThis as { XMLHttpRequest?: unknown }).XMLHttpRequest = origXHR;
});

describe("xhrUploadWithProgress", () => {
  it("fires onUploadProgress during the upload and resolves a Response-like object", async () => {
    MockXHR.scenario = "success";
    const events: {
      loaded: number;
      total: number;
      lengthComputable: boolean;
    }[] = [];
    const res = await xhrUploadWithProgress("/api/x", new FormData(), {
      onUploadProgress: (e) => events.push(e),
    });

    expect(events.length).toBeGreaterThanOrEqual(2);
    expect(events[0]).toMatchObject({
      loaded: 50,
      total: 100,
      lengthComputable: true,
    });
    expect(events.at(-1)).toMatchObject({ loaded: 100, total: 100 });
    expect(res.ok).toBe(true);
    expect(res.status).toBe(200);
    await expect(res.json()).resolves.toMatchObject({ ok: true });
  });

  it("returns ok=false (not a throw) on a 5xx so callers can read the error body", async () => {
    MockXHR.scenario = "server500";
    MockXHR.responseText = '{"detail":"boom"}';
    const res = await xhrUploadWithProgress("/api/x", new FormData());
    expect(res.ok).toBe(false);
    expect(res.status).toBe(500);
    await expect(res.json()).resolves.toMatchObject({ detail: "boom" });
  });

  it("rejects with a TimeoutError when the request times out", async () => {
    MockXHR.scenario = "timeout";
    await expect(
      xhrUploadWithProgress("/api/x", new FormData(), { timeoutMs: 10 }),
    ).rejects.toMatchObject({
      name: "TimeoutError",
    });
  });

  it("rejects with a network Error on transport failure", async () => {
    MockXHR.scenario = "error";
    await expect(
      xhrUploadWithProgress("/api/x", new FormData()),
    ).rejects.toThrow(/network error/i);
  });

  it("aborts an in-flight upload via AbortSignal and rejects with AbortError", async () => {
    MockXHR.scenario = "pending";
    const controller = new AbortController();
    const p = xhrUploadWithProgress("/api/x", new FormData(), {
      signal: controller.signal,
    });

    queueMicrotask(() => controller.abort());
    await expect(p).rejects.toMatchObject({ name: "AbortError" });
  });

  it("rejects synchronously if the signal is already aborted before send", async () => {
    MockXHR.scenario = "success";
    const controller = new AbortController();
    controller.abort();
    await expect(
      xhrUploadWithProgress("/api/x", new FormData(), {
        signal: controller.signal,
      }),
    ).rejects.toMatchObject({ name: "AbortError" });
  });
});
