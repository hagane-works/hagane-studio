"use client";

import { useCallback } from "react";
import useSWR from "swr";

import { focusIconsApi, technologiesApi } from "@/lib/api";

const EMPTY_ICONS: Set<string> = new Set();

function useIconInventory(
  key: string,
  projectId: number | null,
  list: (projectId: number) => Promise<{ icons: string[] }>,
  warnLabel: string,
): { icons: Set<string>; refresh: () => Promise<void> } {
  const { data, mutate } = useSWR(
    projectId ? [key, projectId] : null,
    async () => new Set((await list(projectId!)).icons),
    {
      onError: () =>
        console.warn(
          `[${warnLabel}] 图标清单刷新失败,画布图标可能显示占位符。刷新页面可重试。`,
        ),
    },
  );

  const refresh = useCallback(async () => {
    await mutate();
  }, [mutate]);

  return { icons: data ?? EMPTY_ICONS, refresh };
}

export function useFocusIconInventory(projectId: number | null) {
  return useIconInventory(
    "focus/icons",
    projectId,
    focusIconsApi.list,
    "focus-tree",
  );
}

export function useTechIconInventory(projectId: number | null) {
  return useIconInventory(
    "tech/icons",
    projectId,
    technologiesApi.listIcons,
    "technology",
  );
}
