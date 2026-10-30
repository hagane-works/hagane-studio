export async function readGuiResponse<T>(response: Response): Promise<T> {
  let payload: unknown;
  try {
    payload = await response.json();
  } catch (cause) {
    if (!response.ok) {
      throw new Error(response.statusText || `HTTP ${response.status}`);
    }
    throw cause instanceof Error
      ? cause
      : new Error("Invalid gui-render response");
  }

  if (!response.ok) {
    const detail =
      payload && typeof payload === "object"
        ? ((payload as { detail?: unknown; message?: unknown }).detail ??
          (payload as { message?: unknown }).message)
        : payload;
    throw new Error(
      typeof detail === "string" && detail.trim()
        ? detail
        : response.statusText || `HTTP ${response.status}`,
    );
  }
  return payload as T;
}

export function guiArray<T>(value: readonly T[] | null | undefined): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}

export function guiRecord<T>(
  value: Readonly<Record<string, T>> | null | undefined,
): Record<string, T> {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, T>)
    : {};
}
