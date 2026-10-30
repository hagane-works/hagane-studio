"use client";

import useSWR from "swr";
import { useProjectStore } from "@/lib/store/projectStore";

export function changedKeys<T extends object>(
  row: T,
  data: Partial<T>,
): string[] {
  return Object.keys(data).filter(
    (key) =>
      JSON.stringify(row[key as keyof T] ?? null) !==
      JSON.stringify(data[key as keyof T] ?? null),
  );
}

export function useWorkbenchProject(projectId: number): boolean {
  const ready = useProjectStore((s) => s.currentProject?.id === projectId);
  useSWR(
    ready ? null : ["game-view-project-store", projectId],
    () => useProjectStore.getState().fetchProject(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  return ready;
}
