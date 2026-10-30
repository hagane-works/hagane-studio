import { useMemo } from "react";
import useSWR from "swr";

import { request } from "@/lib/api";
import {
  findFolderContainer,
  parseFolderGeometry,
  type FolderGeometry,
  type GuiNode,
} from "@/lib/techGraph";

export const TECH_TREE_WINDOW = "countrytechtreeview";

interface WindowResponse {
  node?: GuiNode | null;
}

export interface TechTreeGeometryState {
  geometries: Map<string, FolderGeometry>;
  loading: boolean;

  errorMessage: string | null;
}

export function geometryFromWindow(
  win: WindowResponse | null,
  folder: string,
): FolderGeometry | null {
  if (!win?.node || !folder) return null;
  const container = findFolderContainer(win.node, folder);
  return parseFolderGeometry(container, folder);
}

async function fetchWindow(projectId: number): Promise<WindowResponse | null> {
  try {
    return await request<WindowResponse>(
      `/go/projects/${projectId}/gui-render/window?name=${encodeURIComponent(TECH_TREE_WINDOW)}`,
    );
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);

    if (/window not found|failed: 404/i.test(msg)) return null;
    throw e;
  }
}

export function useTechTreeGeometry(
  projectId: number,
  folders: readonly string[],
): TechTreeGeometryState {
  const { data, error, isLoading } = useSWR(
    projectId ? ["tech-graph/geometry", projectId] : null,
    () => fetchWindow(projectId),
    { revalidateOnFocus: false, keepPreviousData: true },
  );

  const key = useMemo(() => [...folders].sort().join("\u0000"), [folders]);
  const geometries = useMemo(() => {
    const out = new Map<string, FolderGeometry>();
    if (!data) return out;
    for (const folder of key ? key.split("\u0000") : []) {
      const geo = geometryFromWindow(data, folder);
      if (geo) out.set(folder, geo);
    }
    return out;
  }, [data, key]);

  return {
    geometries,
    loading: isLoading && data == null,
    errorMessage: error
      ? error instanceof Error
        ? error.message
        : String(error)
      : null,
  };
}
