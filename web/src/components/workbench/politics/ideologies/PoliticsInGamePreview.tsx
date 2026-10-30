"use client";

import type { ReactNode } from "react";
import { useTranslations } from "use-intl";

export type PoliticsPreviewTarget = "name" | "description" | "color" | "icon";

export interface PoliticsPreviewGroup {
  id: string;
  label: string;
  color: string;
  value: number;
}

export interface PoliticsPreviewCountry {
  tag: string;
  name: string;
  hasPoliticsState: boolean;
  rulingParty: string;
  popularities: Record<string, number>;
  electionsAllowed: boolean | null;
  lastElection: string;
}

interface PoliticsInGamePreviewProps {
  scriptId: string;
  displayName: string;
  description: string;
  groupId: string;
  groupLabel: string;
  groupColor: string;
  stanceCount: number;
  iconUrl?: string | null;
  countryFlag?: ReactNode;
  groups: PoliticsPreviewGroup[];
  country: PoliticsPreviewCountry | null;
  isDraft?: boolean;
  activeTarget?: PoliticsPreviewTarget | null;
  editableTargets?: readonly PoliticsPreviewTarget[];
  onSelectTarget?: (target: PoliticsPreviewTarget) => void;
}

function editableTargetClass(
  target: PoliticsPreviewTarget,
  activeTarget: PoliticsPreviewTarget | null | undefined,
  enabled: boolean,
): string {
  if (!enabled) return "";
  const active =
    activeTarget === target
      ? "ring-2 ring-accent/80 bg-accent/10"
      : "ring-1 ring-transparent hover:ring-accent/60";
  return `ui-focus-ring cursor-pointer rounded-sm outline-none transition-[box-shadow,background-color] ${active}`;
}

export function PoliticsInGamePreview({
  scriptId,
  displayName,
  description,
  groupId,
  groupLabel,
  groupColor,
  stanceCount,
  iconUrl,
  countryFlag,
  groups,
  country,
  isDraft = false,
  activeTarget,
  editableTargets = [],
  onSelectTarget,
}: PoliticsInGamePreviewProps) {
  const t = useTranslations("politicsPanel.workbench");
  const tPanel = useTranslations("politicsPanel");
  const canEditTarget = (target: PoliticsPreviewTarget) =>
    Boolean(onSelectTarget && editableTargets.includes(target));
  const total = groups.reduce(
    (sum, group) => sum + Math.max(0, group.value),
    0,
  );
  const segments = groups
    .filter((group) => group.value > 0 && total > 0)
    .reduce<{
      offset: number;
      values: Array<PoliticsPreviewGroup & { share: number; offset: number }>;
    }>(
      (result, group) => {
        const share = (group.value / total) * 100;
        return {
          offset: result.offset + share,
          values: [
            ...result.values,
            { ...group, share, offset: result.offset },
          ],
        };
      },
      { offset: 0, values: [] },
    ).values;
  const hasPoliticsState = Boolean(country?.hasPoliticsState);
  const selectedSupport = hasPoliticsState
    ? (country?.popularities[groupId] ?? 0)
    : 0;
  const selectedRules = groups.find(
    (group) => group.id === country?.rulingParty,
  );

  return (
    <div className="space-y-3 p-3" data-testid="politics-live-preview">
      <div className="overflow-hidden rounded-md border border-ingame-edge/60 bg-ingame-panel text-ingame-title shadow-sm">
        <div className="flex items-center justify-between gap-2 border-b border-ingame-edge/40 bg-ingame-band/80 px-3 py-2">
          <div className="flex min-w-0 items-center gap-2">
            {countryFlag && (
              <div
                data-testid="politics-preview-country-flag"
                className="shrink-0 overflow-hidden rounded-sm"
              >
                {countryFlag}
              </div>
            )}
            <div className="min-w-0">
              <p className="font-mono text-sm font-semibold uppercase tracking-wide">
                {t("livePreviewWindowTitle")}
              </p>
              <p className="mt-0.5 truncate text-micro text-ingame-text/75">
                {country
                  ? `${country.tag} · ${country.name}`
                  : t("livePreviewNoCountryShort")}
              </p>
            </div>
          </div>
          {isDraft && (
            <span className="rounded border border-accent/50 bg-accent/20 px-1.5 py-0.5 text-micro font-mono text-accent">
              {t("liveDraft")}
            </span>
          )}
        </div>

        <div className="grid grid-cols-[5.5rem_minmax(0,1fr)] gap-3 p-3">
          <div className="min-w-0">
            <button
              type="button"
              className={`flex aspect-square w-full items-center justify-center overflow-hidden border border-ingame-edge/50 bg-ingame-band/50 ${editableTargetClass("icon", activeTarget, canEditTarget("icon"))}`}
              disabled={!canEditTarget("icon")}
              onClick={() => canEditTarget("icon") && onSelectTarget?.("icon")}
              title={
                canEditTarget("icon") ? t("livePreviewEditIcon") : undefined
              }
            >
              {iconUrl ? (
                <img src={iconUrl} alt="" className="size-full object-cover" />
              ) : (
                <span className="px-2 text-center text-micro text-ingame-text/70">
                  {t("livePreviewNoIcon")}
                </span>
              )}
            </button>
            <p className="mt-1 truncate text-center text-micro text-ingame-text/65">
              {tPanel("visualIdentityTitle")}
            </p>
          </div>

          <div className="min-w-0 space-y-2">
            <button
              type="button"
              className={`block w-full px-1 py-0.5 text-left ${editableTargetClass("name", activeTarget, canEditTarget("name"))}`}
              disabled={!canEditTarget("name")}
              onClick={() => canEditTarget("name") && onSelectTarget?.("name")}
              title={
                canEditTarget("name") ? t("livePreviewEditName") : undefined
              }
            >
              <strong className="block truncate text-base text-ingame-title">
                {displayName || scriptId}
              </strong>
              <span className="block truncate font-mono text-micro text-ingame-text/70">
                {scriptId}
              </span>
            </button>
            <button
              type="button"
              className={`block w-full px-1 py-0.5 text-left ${editableTargetClass("description", activeTarget, canEditTarget("description"))}`}
              disabled={!canEditTarget("description")}
              onClick={() =>
                canEditTarget("description") && onSelectTarget?.("description")
              }
              title={
                canEditTarget("description")
                  ? t("livePreviewEditDescription")
                  : undefined
              }
            >
              <span className="line-clamp-3 text-xs leading-relaxed text-ingame-text/90">
                {description || t("livePreviewNoDescription")}
              </span>
            </button>
          </div>
        </div>

        <div className="border-t border-ingame-edge/40 bg-ingame-panel-deep/70 p-3">
          <div className="grid grid-cols-[7rem_minmax(0,1fr)] items-center gap-3">
            <div className="relative mx-auto size-28">
              <svg
                viewBox="0 0 100 100"
                role="img"
                aria-label={t("livePreviewSupportChart")}
                className="size-full -rotate-90"
              >
                <circle
                  cx="50"
                  cy="50"
                  r="38"
                  pathLength="100"
                  fill="none"
                  stroke="var(--ingame-edge)"
                  strokeWidth="18"
                  opacity="0.55"
                />
                {segments.map((segment) => (
                  <circle
                    key={segment.id}
                    cx="50"
                    cy="50"
                    r="38"
                    pathLength="100"
                    fill="none"
                    stroke={segment.color}
                    strokeWidth={segment.id === groupId ? "20" : "18"}
                    strokeDasharray={`${segment.share} ${100 - segment.share}`}
                    strokeDashoffset={-segment.offset}
                    opacity={segment.id === groupId ? "1" : "0.72"}
                  />
                ))}
              </svg>
              <div
                data-testid="politics-support-chart-center"
                className="pointer-events-none absolute inset-0 grid place-content-center text-center"
              >
                <strong className="text-lg tabular-nums text-ingame-title">
                  {hasPoliticsState ? `${selectedSupport}%` : "—"}
                </strong>
              </div>
            </div>

            <dl className="min-w-0 space-y-2 text-xs">
              <div>
                <dt className="text-ingame-text/65">
                  {t("livePreviewRulingGroup")}
                </dt>
                <dd className="truncate font-medium text-ingame-title">
                  {hasPoliticsState
                    ? selectedRules?.label ||
                      country?.rulingParty ||
                      t("livePreviewNotSet")
                    : t("livePreviewNoCountryState")}
                </dd>
              </div>
              <div>
                <dt className="text-ingame-text/65">
                  {t("livePreviewSelectedGroup")}
                </dt>
                <dd>
                  <button
                    type="button"
                    className={`mt-0.5 inline-flex max-w-full items-center gap-2 px-1 py-0.5 text-left ${editableTargetClass("color", activeTarget, canEditTarget("color"))}`}
                    disabled={!canEditTarget("color")}
                    onClick={() =>
                      canEditTarget("color") && onSelectTarget?.("color")
                    }
                    title={
                      canEditTarget("color")
                        ? t("livePreviewEditColor")
                        : undefined
                    }
                  >
                    <span
                      className="size-3 shrink-0 rounded-full border border-ingame-edge/60"
                      style={{ backgroundColor: groupColor }}
                      aria-hidden="true"
                    />
                    <span className="truncate font-medium text-ingame-title">
                      {groupLabel}
                    </span>
                  </button>
                </dd>
              </div>
              <div>
                <dt className="text-ingame-text/65">
                  {t("livePreviewStances")}
                </dt>
                <dd className="font-medium text-ingame-title">
                  {t("livePreviewStanceCount", { count: stanceCount })}
                </dd>
              </div>
            </dl>
          </div>
        </div>
      </div>

      <p className="rounded-md border border-border bg-muted/20 p-2.5 text-xs leading-relaxed text-muted-foreground">
        {hasPoliticsState
          ? t("livePreviewCountryHint", { tag: country?.tag ?? "" })
          : t("livePreviewNoCountryHint")}
      </p>
    </div>
  );
}
