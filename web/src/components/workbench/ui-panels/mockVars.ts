import { create } from "zustand";

interface MockVarsState {
  enabled: boolean;

  vars: Record<string, number>;
  setEnabled: (enabled: boolean) => void;
  setVar: (name: string, value: number) => void;
  removeVar: (name: string) => void;
}

export const useMockVarsStore = create<MockVarsState>((set) => ({
  enabled: false,
  vars: {},
  setEnabled: (enabled) => set({ enabled }),
  setVar: (name, value) => set((s) => ({ vars: { ...s.vars, [name]: value } })),
  removeVar: (name) =>
    set((s) => {
      const next = { ...s.vars };
      delete next[name];
      return { vars: next };
    }),
}));

const RUNTIME_TOKEN_RE = /\[\?([A-Za-z0-9_.@^]+)(\|[^\]]*)?\]/g;

export function evalRuntimeTokens(
  text: string,
  vars: Record<string, number>,
): string {
  return text.replace(RUNTIME_TOKEN_RE, (_m, name: string, fmt?: string) => {
    const raw = vars[name] ?? 0;
    const f = (fmt ?? "").slice(1);
    const plus = f.includes("+") && raw > 0 ? "+" : "";
    const decimals = /\d/.test(f)
      ? Number.parseInt(f.replace(/\D/g, ""), 10)
      : 0;
    const pct = f.includes("%");
    const value = (pct ? raw * 100 : raw).toFixed(
      Number.isFinite(decimals) ? decimals : 0,
    );
    return `${plus}${value}${pct ? "%" : ""}`;
  });
}

export function collectReferencedVars(
  controls: {
    frame_var?: string;
    text?: string;
    text_locs?: Record<string, string>;
    children?: unknown[];
  }[],
): string[] {
  const found = new Set<string>();
  const scanText = (text: string | undefined) => {
    if (!text) return;
    for (const m of text.matchAll(RUNTIME_TOKEN_RE)) found.add(m[1]);
  };
  const walk = (list: typeof controls) => {
    for (const c of list) {
      if (c.frame_var) found.add(c.frame_var);
      scanText(c.text);
      for (const v of Object.values(c.text_locs ?? {})) scanText(v);
      if (Array.isArray(c.children)) walk(c.children as typeof controls);
    }
  };
  walk(controls);
  return [...found].sort();
}
