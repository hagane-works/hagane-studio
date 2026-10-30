import {
  TOOLTIP,
  hasTooltip,
  type GameTooltipContent,
  type TooltipRect,
} from "./tooltipText";

export interface TooltipElement {
  readonly isConnected: boolean;
  readonly offsetWidth: number;
  readonly offsetHeight: number;
  getBoundingClientRect(): {
    left: number;
    top: number;
    width: number;
    height: number;
  };
  contains(other: TooltipElement | null): boolean;
  compareDocumentPosition(other: TooltipElement): number;
}

export function domElement(el: Element | null): TooltipElement | null {
  return el as unknown as TooltipElement | null;
}

export interface TooltipHit {
  key: string;
  content: GameTooltipContent;

  rect: TooltipRect;
}

export interface TooltipSource {
  id: string;
  element(): TooltipElement | null;

  hitTest(x: number, y: number): TooltipHit | null;
}

export interface TooltipHost {
  id: string;

  explicit: boolean;
  element(): TooltipElement | null;
}

export interface ShownTooltip {
  key: string;
  content: GameTooltipContent;

  delayed: boolean;

  cursor: { x: number; y: number };
  anchorRect: TooltipRect | null;
  bounds: { w: number; h: number };
}

export interface TooltipSnapshot {
  hostId: string | null;
  shown: ShownTooltip | null;
}

interface Candidate {
  key: string;
  content: GameTooltipContent;
  clientX: number;
  clientY: number;

  anchorClient: TooltipRect | null;
}

interface Current extends Candidate {
  since: number;
  delayed: boolean;
}

export interface TooltipClock {
  now(): number;
  set(fn: () => void, ms: number): unknown;
  clear(handle: unknown): void;
}

const realClock: TooltipClock = {
  now: () => Date.now(),
  set: (fn, ms) => setTimeout(fn, ms),
  clear: (handle) => clearTimeout(handle as ReturnType<typeof setTimeout>),
};

const FOLLOWING = 4;

const EMPTY: TooltipSnapshot = { hostId: null, shown: null };

function toClient(el: TooltipElement): {
  left: number;
  top: number;
  scale: number;
} {
  const r = el.getBoundingClientRect();
  return {
    left: r.left,
    top: r.top,
    scale: el.offsetWidth ? r.width / el.offsetWidth || 1 : 1,
  };
}

export function createTooltipHub(clock: TooltipClock = realClock) {
  const sources = new Map<string, TooltipSource>();
  const hosts: TooltipHost[] = [];
  const listeners = new Set<() => void>();
  let snapshot: TooltipSnapshot = EMPTY;
  let current: Current | null = null;
  let pending: (Candidate & { since: number }) | null = null;
  let explicit: Candidate | null = null;
  let showTimer: unknown = null;
  let hideTimer: unknown = null;
  let delayedTimer: unknown = null;

  const emit = () => listeners.forEach((listener) => listener());

  let elected: TooltipHost | null = null;
  function elect(): TooltipHost | null {
    for (let i = hosts.length - 1; i >= 0; i--)
      if (hosts[i].explicit) return hosts[i];
    let best: TooltipHost | null = null;
    let bestArea = -1;
    for (const host of hosts) {
      const el = host.element();
      const area = el ? el.offsetWidth * el.offsetHeight : 0;
      if (area > bestArea) {
        best = host;
        bestArea = area;
      }
    }
    return best;
  }
  function electedHost(): TooltipHost | null {
    if (!elected || !elected.element()?.isConnected) elected = elect();
    return elected;
  }

  function publish() {
    const host = electedHost();
    const el = host?.element() ?? null;
    if (!current || !host || !el) {
      snapshot = { hostId: host?.id ?? null, shown: null };
      emit();
      return;
    }
    const { left, top, scale } = toClient(el);
    const a = current.anchorClient;
    snapshot = {
      hostId: host.id,
      shown: {
        key: current.key,
        content: current.content,
        delayed: current.delayed,
        cursor: {
          x: (current.clientX - left) / scale,
          y: (current.clientY - top) / scale,
        },
        anchorRect: a
          ? {
              x: (a.x - left) / scale,
              y: (a.y - top) / scale,
              w: a.w / scale,
              h: a.h / scale,
            }
          : null,
        bounds: { w: el.offsetWidth, h: el.offsetHeight },
      },
    };
    emit();
  }

  function stop(handle: unknown) {
    if (handle !== null) clock.clear(handle);
  }

  function setCurrent(next: Current | null) {
    stop(delayedTimer);
    delayedTimer = null;
    stop(hideTimer);
    hideTimer = null;
    current = next;
    if (next && next.content.delayed && !next.delayed) {
      const key = next.key;
      delayedTimer = clock.set(
        () => {
          delayedTimer = null;
          if (current?.key !== key) return;
          current = { ...current, delayed: true };
          publish();
        },
        Math.max(0, next.since + TOOLTIP.delayedDelayMs - clock.now()),
      );
    }
    publish();
  }

  function clearPending() {
    stop(showTimer);
    showTimer = null;
    pending = null;
  }

  function offer(candidate: Candidate | null) {
    if (candidate && current && candidate.key === current.key) {
      stop(hideTimer);
      hideTimer = null;
      clearPending();
      current = { ...current, ...candidate };
      publish();
      return;
    }
    if (candidate && pending && candidate.key === pending.key) {
      pending = { ...pending, ...candidate };
      return;
    }
    if (candidate) {
      clearPending();
      pending = { ...candidate, since: clock.now() };
      showTimer = clock.set(() => {
        showTimer = null;
        const next = pending;
        pending = null;
        if (next) setCurrent({ ...next, delayed: false });
      }, TOOLTIP.showDelayMs);
      return;
    }
    clearPending();
    if (current && hideTimer === null) {
      hideTimer = clock.set(() => {
        hideTimer = null;
        setCurrent(null);
      }, TOOLTIP.hideDelayMs);
    }
  }

  function resolve(
    clientX: number,
    clientY: number,
    target: TooltipElement | null,
  ): Candidate | null {
    const live: { source: TooltipSource; el: TooltipElement }[] = [];
    for (const source of sources.values()) {
      const el = source.element();
      if (el && el.isConnected) live.push({ source, el });
    }

    live.sort((a, b) =>
      a.el.compareDocumentPosition(b.el) & FOLLOWING ? 1 : -1,
    );
    for (const { source, el } of live) {
      const inside = target !== null && el.contains(target);

      if (
        target &&
        !inside &&
        !(target.compareDocumentPosition(el) & FOLLOWING)
      )
        return null;
      const { left, top, scale } = toClient(el);
      const hit = source.hitTest(
        (clientX - left) / scale,
        (clientY - top) / scale,
      );
      if (hit) {
        return {
          key: `${source.id}:${hit.key}`,
          content: hit.content,
          clientX,
          clientY,
          anchorClient: {
            x: left + hit.rect.x * scale,
            y: top + hit.rect.y * scale,
            w: hit.rect.w * scale,
            h: hit.rect.h * scale,
          },
        };
      }

      if (inside) return null;
    }
    return null;
  }

  return {
    subscribe(listener: () => void): () => void {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    getSnapshot: (): TooltipSnapshot => snapshot,

    registerSource(source: TooltipSource): () => void {
      sources.set(source.id, source);
      return () => {
        if (sources.get(source.id) !== source) return;
        sources.delete(source.id);

        const prefix = `${source.id}:`;
        if (pending?.key.startsWith(prefix)) clearPending();
        if (current?.key.startsWith(prefix)) setCurrent(null);
      };
    },
    registerHost(host: TooltipHost): () => void {
      hosts.push(host);
      elected = null;
      publish();
      return () => {
        const at = hosts.indexOf(host);
        if (at >= 0) hosts.splice(at, 1);
        elected = null;
        publish();
      };
    },
    hasSources: () => sources.size > 0,

    pointerMove(
      clientX: number,
      clientY: number,
      target: TooltipElement | null,
    ) {
      if (explicit) {
        explicit = { ...explicit, clientX, clientY };
        offer(explicit);
        return;
      }
      offer(resolve(clientX, clientY, target));
    },

    clear() {
      explicit = null;
      clearPending();
      setCurrent(null);
    },

    showExplicit(
      key: string,
      content: GameTooltipContent,
      clientX: number,
      clientY: number,
      rect: TooltipRect | null,
    ) {
      if (!hasTooltip(content)) {
        explicit = null;
        offer(null);
        return;
      }
      explicit = {
        key: `explicit:${key}`,
        content,
        clientX,
        clientY,
        anchorClient: rect,
      };
      offer(explicit);
    },
    hideExplicit(key: string) {
      if (explicit?.key !== `explicit:${key}`) return;
      explicit = null;
      offer(null);
    },

    resolve,
  };
}

export type TooltipHub = ReturnType<typeof createTooltipHub>;

export const tooltipHub = createTooltipHub();

let attached = false;

export function attachTooltipHub(): void {
  if (attached || typeof window === "undefined") return;
  attached = true;
  window.addEventListener(
    "pointermove",
    (e) => {
      if (e.pointerType === "touch") return;
      tooltipHub.pointerMove(
        e.clientX,
        e.clientY,
        domElement(e.target instanceof Element ? e.target : null),
      );
    },
    { capture: true, passive: true },
  );
  window.addEventListener("blur", () => tooltipHub.clear());
  document.documentElement.addEventListener("pointerleave", () =>
    tooltipHub.clear(),
  );
}
