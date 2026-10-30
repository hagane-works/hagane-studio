import { ApiError, request } from "./_client";

const FIRST_RETRY_MS = 500;
const MAX_RETRY_MS = 5000;

export function isGameFolderMissing(reason: unknown): boolean {
  return (
    reason instanceof ApiError &&
    (reason.payload as { code?: unknown } | null)?.code ===
      "game_folder_missing"
  );
}

export function isGameIndexBuilding(reason: unknown): boolean {
  return (
    reason instanceof ApiError &&
    reason.status === 503 &&
    (reason.payload as { code?: unknown } | null)?.code ===
      "game_index_building"
  );
}

export async function requestGameIndexed<T>(path: string): Promise<T> {
  for (
    let pause = FIRST_RETRY_MS;
    ;
    pause = Math.min(pause * 2, MAX_RETRY_MS)
  ) {
    try {
      return await request<T>(path);
    } catch (reason) {
      if (!isGameIndexBuilding(reason)) throw reason;
      await new Promise((resolve) => setTimeout(resolve, pause));
    }
  }
}

export function sessionGameData<T>(fetchData: () => Promise<T>, empty: T) {
  let data = empty;
  let inflight: Promise<T> | null = null;
  const listeners = new Set<() => void>();

  function load(): Promise<T> {
    if (data !== empty) return Promise.resolve(data);
    inflight ??= fetchData().then(
      (value) => {
        data = value;
        inflight = null;
        for (const listener of listeners) listener();
        return value;
      },
      (reason) => {
        inflight = null;
        throw reason;
      },
    );
    return inflight;
  }

  return {
    load,
    current: () => data,
    subscribe(listener: () => void) {
      listeners.add(listener);

      load().catch(() => undefined);
      return () => {
        listeners.delete(listener);
      };
    },
  };
}
