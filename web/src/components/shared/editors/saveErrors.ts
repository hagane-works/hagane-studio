export const GENERIC_SAVE_ERROR = "Save failed";

export function extractSaveErrorMessage(
  err: unknown,
  fallback: string = GENERIC_SAVE_ERROR,
): string {
  if (err == null) return fallback;
  if (typeof err === "string") {
    const s = err.trim();
    return s.length > 0 ? s : fallback;
  }
  if (err instanceof Error) {
    const m = err.message?.trim();
    return m && m.length > 0 ? m : fallback;
  }
  if (typeof err === "object") {
    const o = err as Record<string, unknown>;

    const responseDetail = (
      (o.response as Record<string, unknown> | undefined)?.data as
        Record<string, unknown> | undefined
    )?.detail;
    const candidate =
      (typeof o.detail === "string" && o.detail) ||
      (typeof responseDetail === "string" && responseDetail) ||
      (typeof o.message === "string" && o.message) ||
      "";
    const s = candidate.trim();
    return s.length > 0 ? s : fallback;
  }
  return fallback;
}

export function validateRequiredId(
  rawId: string | null | undefined,
  label: string,
): string | null {
  const trimmed = (rawId ?? "").trim();
  if (trimmed.length === 0) {
    return `${label} is required. Enter an id before saving (a blank id would be silently renamed).`;
  }
  return null;
}
