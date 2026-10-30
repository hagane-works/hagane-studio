"use client";

import { useCallback } from "react";
import useSWR from "swr";

import { lintApi, type DeadFlagReport } from "@/lib/api";

export function useDeadFlagLint(projectId: number | null): {
  report: DeadFlagReport | null;
  refresh: () => Promise<void>;
} {
  const { data, mutate } = useSWR(
    projectId ? ["lint/dead-flags", projectId] : null,
    () => lintApi.deadFlags(projectId!),
    {
      dedupingInterval: 30_000,
    },
  );

  const refresh = useCallback(async () => {
    await mutate();
  }, [mutate]);

  return { report: data ?? null, refresh };
}
