export const API_BASE = "/api";

export class ApiError extends Error {
  readonly status: number;
  readonly detail: string;

  readonly payload: unknown;

  constructor(status: number, detail: string, payload: unknown = detail) {
    super(detail);
    this.name = "ApiError";
    this.status = status;
    this.detail = detail;
    this.payload = payload;
  }
}

export function readableError(reason: unknown): string {
  if (reason instanceof ApiError) return reason.detail;
  if (reason instanceof Error) return reason.message;
  return String(reason);
}

export async function fetchWithTimeoutRetry(
  url: string,
  init: RequestInit,
  opts: { timeoutMs?: number; maxRetries?: number } = {},
): Promise<Response> {
  const timeoutMs = opts.timeoutMs ?? 60_000;
  const maxRetries = opts.maxRetries ?? 3;
  let lastErr: unknown;
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    try {
      const res = await fetch(url, {
        credentials: "include",
        ...init,
        signal: controller.signal,
      });
      clearTimeout(timer);
      if (res.status >= 500 && attempt < maxRetries - 1) {
        lastErr = new Error(`Server ${res.status}`);
        await new Promise((r) => setTimeout(r, 1000 * 2 ** attempt));
        continue;
      }
      return res;
    } catch (err) {
      clearTimeout(timer);
      lastErr = err;
      if (attempt < maxRetries - 1) {
        await new Promise((r) => setTimeout(r, 1000 * 2 ** attempt));
        continue;
      }
      throw err;
    }
  }
  throw lastErr ?? new Error("Request failed after retries");
}

export interface UploadProgressEvent {
  loaded: number;
  total: number;
  lengthComputable: boolean;
}

export interface XhrUploadOptions {
  timeoutMs?: number;

  onUploadProgress?: (ev: UploadProgressEvent) => void;

  signal?: AbortSignal;
}

export interface XhrUploadResponse {
  ok: boolean;
  status: number;
  json: () => Promise<unknown>;
  text: () => Promise<string>;
}

export function xhrUploadWithProgress(
  url: string,
  body: FormData,
  opts: XhrUploadOptions = {},
): Promise<XhrUploadResponse> {
  const timeoutMs = opts.timeoutMs ?? 30 * 60_000;
  return new Promise<XhrUploadResponse>((resolve, reject) => {
    if (opts.signal?.aborted) {
      reject(new DOMException("upload aborted", "AbortError"));
      return;
    }
    const xhr = new XMLHttpRequest();
    xhr.open("POST", url, true);
    xhr.withCredentials = true;
    xhr.timeout = timeoutMs;

    if (opts.onUploadProgress && xhr.upload) {
      xhr.upload.onprogress = (e: ProgressEvent) => {
        opts.onUploadProgress!({
          loaded: e.loaded,
          total: e.lengthComputable ? e.total : 0,
          lengthComputable: e.lengthComputable,
        });
      };
    }

    const onAbort = () => {
      try {
        xhr.abort();
      } catch {}
    };
    const cleanup = () => {
      if (opts.signal) opts.signal.removeEventListener("abort", onAbort);
    };
    if (opts.signal)
      opts.signal.addEventListener("abort", onAbort, { once: true });

    xhr.onload = () => {
      cleanup();
      const status = xhr.status;
      const textBody = xhr.responseText ?? "";
      resolve({
        ok: status >= 200 && status < 300,
        status,
        json: async () => JSON.parse(textBody),
        text: async () => textBody,
      });
    };
    xhr.onerror = () => {
      cleanup();
      reject(new Error("network error during upload"));
    };
    xhr.ontimeout = () => {
      cleanup();
      reject(new DOMException("upload timed out", "TimeoutError"));
    };
    xhr.onabort = () => {
      cleanup();
      reject(new DOMException("upload aborted", "AbortError"));
    };

    xhr.send(body);
  });
}

const SESSION_RELOAD_KEY = "haganeworks:session-reload";

function reloadForNewSession() {
  try {
    const last = Number(sessionStorage.getItem(SESSION_RELOAD_KEY) || 0);
    if (Date.now() - last < 10_000) return;
    sessionStorage.setItem(SESSION_RELOAD_KEY, String(Date.now()));
  } catch {
    return;
  }
  window.location.reload();
}

export async function request<T>(
  path: string,
  options?: RequestInit & {
    onResponse?: (response: Response) => void;

    redirectOnUnauthorized?: boolean;
  },
): Promise<T> {
  const {
    onResponse,
    redirectOnUnauthorized = true,
    ...fetchOptions
  } = options ?? {};
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options?.headers as Record<string, string>) || {}),
  };
  const res = await fetch(`${API_BASE}${path}`, {
    credentials: "include",
    ...fetchOptions,
    headers,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ detail: res.statusText }));

    if (
      res.status === 401 &&
      redirectOnUnauthorized &&
      typeof window !== "undefined"
    ) {
      reloadForNewSession();
    }

    if (
      res.status === 409 &&
      typeof window !== "undefined" &&
      typeof err.detail === "string" &&
      err.detail.includes("already exists") &&
      err.detail.includes("Pick a unique")
    ) {
      window.dispatchEvent(
        new CustomEvent("api-collision-error", {
          detail: { path, message: err.detail },
        }),
      );
    }
    const rawDetail = err?.detail;
    const detail =
      typeof rawDetail === "string"
        ? rawDetail
        : rawDetail &&
            typeof rawDetail === "object" &&
            typeof (rawDetail as { message?: unknown }).message === "string"
          ? (rawDetail as { message: string }).message
          : `Request failed: ${res.status}`;

    const payload =
      err &&
      typeof err === "object" &&
      Object.keys(err as Record<string, unknown>).some(
        (key) => key !== "detail",
      )
        ? err
        : rawDetail;
    throw new ApiError(res.status, detail, payload);
  }
  onResponse?.(res);
  if (res.status === 204) return undefined as T;
  const body = await res.json();

  const _bodyWarnings =
    body && typeof body === "object"
      ? (body as { warnings?: unknown }).warnings
      : undefined;
  if (
    Array.isArray(_bodyWarnings) &&
    _bodyWarnings.length > 0 &&
    _bodyWarnings.every((w) => typeof w === "string")
  ) {
    const warnings = _bodyWarnings as string[];
    console.warn(`[API warning ${path}]`, warnings);
    if (typeof window !== "undefined") {
      window.dispatchEvent(
        new CustomEvent("api-backend-warnings", {
          detail: { path, warnings },
        }),
      );
    }
  }
  return body;
}
