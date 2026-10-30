import { useMemo } from "react";
import { API_BASE } from "@/lib/api";
import { slugify } from "@/lib/wiki";
import { useProjectStore } from "@/lib/store/projectStore";

export function depKeyFromName(name: string): string {
  return slugify(name);
}

export function depKeysFromDependencies(
  deps: readonly string[] | null | undefined,
): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const d of deps ?? []) {
    const key = slugify(d);
    if (key && !seen.has(key)) {
      seen.add(key);
      out.push(key);
    }
  }
  return out;
}

export function gfxUrl(
  projectId: number,
  depKey: string,
  name: string,
): string {
  return `${API_BASE}/go/projects/${projectId}/basemod/${encodeURIComponent(depKey)}/gfx/${encodeURIComponent(name)}`;
}

export function artUrl(
  projectId: number,
  depKey: string,
  relPath: string,
): string {
  const cleaned = relPath.replace(/^\/+/, "");
  const encoded = cleaned.split("/").map(encodeURIComponent).join("/");
  return `${API_BASE}/go/projects/${projectId}/basemod/${encodeURIComponent(depKey)}/art/${encoded}`;
}

const EMPTY_DEP_KEYS: string[] = [];

export function useBaseModDepKeys(): string[] {
  const dependencies = useProjectStore((s) => s.currentProject?.dependencies);
  return useMemo(
    () =>
      dependencies && dependencies.length
        ? depKeysFromDependencies(dependencies)
        : EMPTY_DEP_KEYS,
    [dependencies],
  );
}
