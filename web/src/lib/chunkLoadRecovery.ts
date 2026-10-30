const RELOAD_KEY = "hoi4.chunkReloadAt";
const RELOAD_WINDOW_MS = 60_000;
const CHUNK_MESSAGE =
  /Failed to load chunk|Loading chunk [\w./~-]+ failed|Loading CSS chunk|Failed to fetch dynamically imported module|Importing a module script failed|error loading dynamically imported module/i;

export function isChunkLoadError(error: unknown): boolean {
  if (!error || typeof error !== "object") return false;
  const { name, message } = error as { name?: unknown; message?: unknown };
  return (
    name === "ChunkLoadError" ||
    (typeof message === "string" && CHUNK_MESSAGE.test(message))
  );
}

export function chunkNameOf(error: unknown): string {
  const message = String(
    (error as { message?: unknown } | null)?.message ?? "",
  );
  return (
    message.match(/\/_next\/static\/chunks\/([\w.~-]+\.(?:js|css))/)?.[1] ?? ""
  );
}

export function claimChunkReload(
  hasUnsavedEdits: boolean,
  now: number = Date.now(),
): boolean {
  if (hasUnsavedEdits || typeof window === "undefined") return false;
  try {
    const last = Number(window.sessionStorage.getItem(RELOAD_KEY) ?? 0);
    if (Number.isFinite(last) && now - last < RELOAD_WINDOW_MS) return false;
    window.sessionStorage.setItem(RELOAD_KEY, String(now));
    return true;
  } catch {
    return false;
  }
}

export function reloadPage(): void {
  window.location.reload();
}
