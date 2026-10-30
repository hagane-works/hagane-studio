import useSWR from "swr";

import { request } from "@/lib/api";
import { guiRenderApi } from "@/lib/guiRenderApi";
import type { GuiWindow } from "@/lib/guiRenderApi";
import { assignIds } from "@/app/[locale]/gui-renderer/GuiCanvas";

export interface PanelWindowState {
  win: GuiWindow | null;
  loading: boolean;

  notFound: boolean;

  errorMessage: string | null;
}

async function fetchWindow(
  projectId: number,
  windowName: string,
): Promise<GuiWindow | null> {
  guiRenderApi.setProjectId(projectId);
  try {
    const w = await request<GuiWindow>(
      `/go/projects/${projectId}/gui-render/window?name=${encodeURIComponent(windowName)}`,
    );
    assignIds(w.node);
    return w;
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);

    if (/window not found|failed: 404/i.test(msg)) return null;
    throw e;
  }
}

export function usePanelWindow(
  projectId: number,
  windowName: string | null,
  updatedAt: string,
): PanelWindowState {
  const { data, error, isLoading } = useSWR(
    windowName ? ["ui-panels/window", projectId, windowName, updatedAt] : null,

    () => fetchWindow(projectId, windowName as string),
    { revalidateOnFocus: false, keepPreviousData: true },
  );

  return {
    win: data ?? null,
    loading: isLoading && data == null,
    notFound: !isLoading && !error && data === null,
    errorMessage: error
      ? error instanceof Error
        ? error.message
        : String(error)
      : null,
  };
}
