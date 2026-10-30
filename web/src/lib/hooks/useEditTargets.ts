"use client";

import { useEffect, useState } from "react";
import { API_BASE } from "@/lib/api";

export interface EditTargetKindEntry {
  label: string;
  count: number;
  modes: Record<string, number>;
  paths: string[];
}

export interface EditTargetsResponse {
  is_loaded_existing_mod: boolean;
  primary_tag: string;
  mod_name: string;
  by_kind: Record<string, EditTargetKindEntry>;
  total_files: number;
  error?: string;
}

export function useEditTargets(
  projectId: number | null,
  refreshKey: number = 0,
) {
  const [data, setData] = useState<EditTargetsResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!projectId) {
        if (!cancelled) setData(null);
        return;
      }
      if (cancelled) return;
      setLoading(true);
      try {
        const r = await fetch(
          `${API_BASE}/go/projects/${projectId}/edit-targets`,
        );
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        const json: EditTargetsResponse = await r.json();
        if (!cancelled) {
          setData(json);
          setError(null);
        }
      } catch (e: unknown) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : String(e));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [projectId, refreshKey]);

  return { data, loading, error };
}
