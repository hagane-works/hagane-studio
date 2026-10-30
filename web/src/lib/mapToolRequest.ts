import { xhrUploadWithProgress } from "@/lib/api/_client";

export function mapToolFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  return fetch(input, init);
}

export async function mapToolUpload(
  url: string,
  body: FormData,
  onProgress: (percent: number) => void,
): Promise<Response> {
  const out = await xhrUploadWithProgress(url, body, {
    onUploadProgress: ({ loaded, total, lengthComputable }) => {
      if (lengthComputable && total > 0)
        onProgress(Math.min(100, Math.floor((loaded / total) * 100)));
    },
  });
  const text = await out.text();
  return new Response(text || null, {
    status: out.status,
    headers: { "Content-Type": "application/json" },
  });
}

export interface MapProblem {
  code?: string;
  detail: string;
  values?: Record<string, string | number>;
  provinces?: number[];
}

export async function mapResponseProblem(
  response: Response,
): Promise<MapProblem> {
  const out = (await response.json().catch(() => ({}))) as {
    code?: unknown;
    detail?: unknown;
    values?: unknown;
    provinces?: unknown;
  };
  const values: Record<string, string | number> = {};
  if (out.values && typeof out.values === "object") {
    for (const [key, value] of Object.entries(out.values)) {
      if (typeof value === "string" || typeof value === "number")
        values[key] = value;
    }
  }
  return {
    code: typeof out.code === "string" && out.code ? out.code : undefined,
    detail:
      typeof out.detail === "string" ? out.detail : `HTTP ${response.status}`,
    values,
    provinces: Array.isArray(out.provinces)
      ? out.provinces.filter((id): id is number => Number.isInteger(id))
      : [],
  };
}
