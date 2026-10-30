"use client";

import useSWR from "swr";
import { useTranslations } from "use-intl";

import { projectsApi } from "@/lib/api/project";
import { SELECTABLE_DLCS, normalizeDlcChoice } from "@/lib/dlcCatalog";

import { HoverTip } from "./HoverTip";

export function DlcBadge({ projectId }: { projectId: number }) {
  const t = useTranslations("dlcPicker");
  const project = useSWR(
    ["game-view-project", projectId],
    () => projectsApi.get(projectId),
    { revalidateOnFocus: false },
  );
  if (!project.data) return null;
  const owned = normalizeDlcChoice(project.data.owned_dlcs);
  const count = owned?.length ?? SELECTABLE_DLCS.length;
  const summary =
    owned == null || count === SELECTABLE_DLCS.length
      ? t("summaryAll")
      : count === 0
        ? t("summaryNone")
        : t("summarySome", { count, total: SELECTABLE_DLCS.length });
  const names =
    owned == null || count === SELECTABLE_DLCS.length
      ? t("summaryAll")
      : count === 0
        ? t("summaryNone")
        : owned.join(", ");
  return (
    <HoverTip tip={t("badgeTip", { names })} align="left">
      {(describedBy) => (
        <span
          tabIndex={0}
          aria-describedby={describedBy}
          className="rounded-md border border-ingame-edge/70 bg-ingame-band px-2 py-1.5 text-xs text-ingame-text shadow-lg"
        >
          {t("badge", { summary })}
        </span>
      )}
    </HoverTip>
  );
}
