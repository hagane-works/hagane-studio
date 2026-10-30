import { trackGuiRead } from "@/lib/guiLoadActivity";
import type { WorldMapPayload } from "./worldMapModel";
import { withGameLanguage } from "@/lib/guiRenderApi";

export async function loadWorldMap(
  projectId: number,
  date: string,
): Promise<WorldMapPayload> {
  return trackGuiRead(
    (async () => {
      const response = await fetch(
        withGameLanguage(
          `/api/go/projects/${projectId}/gui-render/world-map?date=${encodeURIComponent(date)}`,
        ),
      );
      if (!response.ok) {
        const out = (await response.json().catch(() => ({}))) as {
          detail?: string;
        };
        throw new Error(out.detail || `HTTP ${response.status}`);
      }
      return response.json() as Promise<WorldMapPayload>;
    })(),
  );
}
