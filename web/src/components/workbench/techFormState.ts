import type { Technology } from "@/types";

export function parseTokenLines(value: string): string[] {
  return value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
}

export function parseDependencyLines(value: string): Record<string, number> {
  const out: Record<string, number> = {};
  for (const line of value.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    const eq = trimmed.indexOf("=");
    const key = (eq >= 0 ? trimmed.slice(0, eq) : trimmed).trim();
    if (!key) continue;
    const rawVal = eq >= 0 ? trimmed.slice(eq + 1).trim() : "";
    const num = rawVal === "" ? 1 : Number(rawVal);
    out[key] = Number.isFinite(num) ? num : 1;
  }
  return out;
}

export function formatDependencyLines(
  deps: Record<string, number> | undefined,
): string {
  return Object.entries(deps || {})
    .map(([k, v]) => `${k} = ${v}`)
    .join("\n");
}

export function buildTechSavePayload(form: Technology): Partial<Technology> {
  return {
    ...form,
    tech_id: (form.tech_id || "").trim(),
  };
}
