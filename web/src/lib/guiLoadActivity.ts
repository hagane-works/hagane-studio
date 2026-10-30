import { useSyncExternalStore } from "react";

export type GuiLoadSnapshot = {
  pending: number;

  finished: number;

  slow: boolean;
};

export const SLOW_AFTER_MS = 400;

let snapshot: GuiLoadSnapshot = { pending: 0, finished: 0, slow: false };
const listeners = new Set<() => void>();
let slowTimer: ReturnType<typeof setTimeout> | null = null;

function publish(next: GuiLoadSnapshot) {
  snapshot = next;
  for (const listener of listeners) listener();
}

function begin() {
  const wasIdle = snapshot.pending === 0;
  publish({ ...snapshot, pending: snapshot.pending + 1 });
  if (wasIdle) {
    if (slowTimer) clearTimeout(slowTimer);
    slowTimer = setTimeout(() => {
      slowTimer = null;
      if (snapshot.pending > 0) publish({ ...snapshot, slow: true });
    }, SLOW_AFTER_MS);
  }
}

function end() {
  const pending = Math.max(0, snapshot.pending - 1);
  if (pending === 0 && slowTimer) {
    clearTimeout(slowTimer);
    slowTimer = null;
  }
  publish({
    pending,
    finished: snapshot.finished + 1,
    slow: pending > 0 && snapshot.slow,
  });
}

export function trackGuiRead<T>(request: Promise<T>): Promise<T> {
  begin();
  return request.finally(end);
}

export function getGuiLoadSnapshot(): GuiLoadSnapshot {
  return snapshot;
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

const serverSnapshot: GuiLoadSnapshot = {
  pending: 0,
  finished: 0,
  slow: false,
};

export function useGuiLoadActivity(): GuiLoadSnapshot {
  return useSyncExternalStore(
    subscribe,
    getGuiLoadSnapshot,
    () => serverSnapshot,
  );
}
