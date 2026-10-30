"use client";

import { useTranslations } from "use-intl";
import type { Decision, DecisionCategory } from "@/types";
import { decisionsApi } from "@/lib/api";
import { CountryTagPreview } from "@/components/CountryTagPreview";
import { AssetPreviewImg } from "./AssetPreviewImg";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import {
  resolveDisplayName,
  resolveDisplayDesc,
} from "@/lib/i18n/localizedName";

interface DecisionInGamePreviewProps {
  decision: Decision;

  category?: DecisionCategory | null;
  projectId: number;
}

function costBadgeTone(cost: number): string {
  if (cost < 0) return "text-sky-300 border-sky-400/40 bg-sky-500/15";
  if (cost === 0)
    return "text-emerald-300 border-emerald-400/40 bg-emerald-500/15";
  if (cost <= 50) return "text-amber-200 border-amber-400/40 bg-amber-500/20";
  if (cost <= 150)
    return "text-orange-200 border-orange-400/40 bg-orange-500/25";
  return "text-rose-200 border-rose-400/50 bg-rose-500/30";
}

function dayBadge(label: string, days: number): React.ReactElement | null {
  if (!days || days <= 0) return null;
  return (
    <span
      key={label}
      className="px-1.5 py-0.5 rounded border border-ingame-edge/40 bg-ingame-band/40 text-micro font-mono text-ingame-text"
      title={`${label}: ${days} days`}
    >
      {label} {days}d
    </span>
  );
}

export function DecisionInGamePreview({
  decision,
  category,
  projectId,
}: DecisionInGamePreviewProps) {
  const t = useTranslations("decisionInGamePreview");
  const locale = useContentLocaleStore((s) => s.locale);

  const name = resolveDisplayName(
    locale,
    decision.display_name_locs,
    decision.name_en,
    decision.name_zh,
    decision.decision_id,
  );
  const desc = resolveDisplayDesc(
    locale,
    decision.description_locs,
    decision.desc_en,
    decision.desc_zh,
  );
  const catName = resolveDisplayName(
    locale,
    category?.display_name_locs,
    category?.name_en,
    category?.name_zh,
    category?.category_id || "Uncategorized",
  );
  const tone = costBadgeTone(decision.cost ?? 0);

  const isMission =
    decision.selectable_mission ||
    (decision.days_mission_timeout ?? 0) > 0 ||
    !!decision.activation?.trim();
  const showIsGoodNo = isMission && decision.is_good === false;

  return (
    <div className="rounded-md border border-ingame-edge/60 bg-gradient-to-b from-ingame-panel/85 to-ingame-panel-deep/95 text-ingame-title shadow-sm overflow-hidden">
      <div className="flex items-center gap-2 px-3 py-1.5 bg-ingame-band/80 border-b border-ingame-edge/40">
        {category?.icon && category?.id != null && (
          <AssetPreviewImg
            url={decisionsApi.categoryIconUrl(projectId, category.id)}
            className="w-5 h-4 object-contain shrink-0"
          />
        )}
        <span className="text-micro font-mono uppercase tracking-wide text-ingame-gold/90 truncate">
          {catName}
        </span>
        <span className="ml-auto text-micro font-mono text-ingame-text/70">
          {t("approximateLabel")}
        </span>
      </div>

      <div className="flex items-stretch gap-4 px-4 py-4">
        <div className="shrink-0 w-12 h-10 rounded border border-ingame-edge/50 bg-ingame-band/60 flex items-center justify-center overflow-hidden">
          {decision.icon ? (
            <AssetPreviewImg
              url={decisionsApi.iconUrl(projectId, decision.id)}
              className="max-h-12 max-w-full object-contain"
            />
          ) : (
            <span className="text-xs text-ingame-text/70">⚪</span>
          )}
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-medium text-sm truncate text-ingame-title">
              <CountryTagPreview text={name} chipStyle={false} />
            </span>
            <span
              className={`shrink-0 px-1.5 py-0.5 rounded border text-micro font-mono ${tone}`}
            >
              {(decision.cost ?? 0) < 0
                ? t("ppRefund", { amount: Math.abs(decision.cost ?? 0) })
                : t("ppCost", { amount: decision.cost ?? 0 })}
            </span>
            {decision.fire_only_once && (
              <span className="px-1.5 py-0.5 rounded border border-purple-400/40 bg-purple-500/20 text-micro font-mono text-purple-200">
                fire-once
              </span>
            )}
            {dayBadge("activate", decision.days_remove ?? 0)}
            {dayBadge("cooldown", decision.days_re_enable ?? 0)}
          </div>
          <p className="text-micro font-mono text-ingame-text/75 mt-0.5 truncate">
            {decision.decision_id}
          </p>
          {desc && (
            <p className="text-xs text-ingame-text/90 mt-1 line-clamp-2">
              <CountryTagPreview text={desc} />
            </p>
          )}
        </div>
      </div>

      {(decision.custom_cost_text ||
        decision.selectable_mission ||
        showIsGoodNo) && (
        <div className="px-3 py-1.5 border-t border-ingame-edge/30 bg-ingame-band/60 text-micro font-mono text-ingame-text/75 flex flex-wrap gap-x-3 gap-y-0.5">
          {decision.custom_cost_text && (
            <span title={t("costTextTooltip")}>
              cost-text · {decision.custom_cost_text}
            </span>
          )}
          {decision.selectable_mission && (
            <span
              className="text-sky-300/80"
              title={t("selectableMissionTooltip")}
            >
              mission · selectable
            </span>
          )}
          {showIsGoodNo && (
            <span className="text-rose-300/80">is_good = no</span>
          )}
        </div>
      )}
    </div>
  );
}
