import { withGameLanguage } from "@/lib/guiRenderApi";

export type NameKind = "state" | "city" | "region";

export function nameKey(kind: NameKind, id: number): string {
  if (kind === "state") return `STATE_${id}`;
  if (kind === "city") return `VICTORY_POINTS_${id}`;
  return `STRATEGICREGION_${id}`;
}

export function nameLanguages(
  exportLanguages: readonly string[] | null | undefined,
): string[] {
  const out = ["en"];
  for (const lang of exportLanguages ?? []) {
    const code = lang.trim().toLowerCase();
    if (code && !out.includes(code)) out.push(code);
  }
  return out;
}

async function answer<T>(response: Response): Promise<T> {
  const out = (await response.json().catch(() => ({}))) as T & {
    detail?: string;
  };
  if (!response.ok) throw new Error(out.detail || `HTTP ${response.status}`);
  return out;
}

export async function loadRewrites(
  projectId: number,
  lang: string,
): Promise<Record<string, string>> {
  const out = await answer<{ overrides: Record<string, string> }>(
    await fetch(
      withGameLanguage(
        `/api/go/projects/${projectId}/gui-render/text?lang=${encodeURIComponent(lang)}`,
      ),
    ),
  );
  return out.overrides ?? {};
}

export async function setRewrite(
  projectId: number,
  lang: string,
  key: string,
  value: string,
): Promise<void> {
  await answer(
    await fetch(`/api/go/projects/${projectId}/gui-render/text`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key, value, lang }),
    }),
  );
}

export async function clearRewrite(
  projectId: number,
  lang: string,
  key: string,
): Promise<void> {
  await answer(
    await fetch(
      `/api/go/projects/${projectId}/gui-render/text?key=${encodeURIComponent(key)}&lang=${encodeURIComponent(lang)}`,
      {
        method: "DELETE",
      },
    ),
  );
}

export function nameChanges(
  draft: Record<string, string>,
  saved: Record<string, string | undefined>,
): Record<string, string | null> {
  const out: Record<string, string | null> = {};
  for (const [lang, raw] of Object.entries(draft)) {
    const value = raw.trim();
    const was = saved[lang] ?? "";
    if (value === was) continue;
    out[lang] = value === "" ? null : value;
  }
  return out;
}
