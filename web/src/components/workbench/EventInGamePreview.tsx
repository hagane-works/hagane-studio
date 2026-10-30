"use client";

import { useTranslations } from "use-intl";
import type { GameEvent, EventOption } from "@/types";
import { eventsApi } from "@/lib/api";
import { CountryTagPreview } from "@/components/CountryTagPreview";
import { AssetPreviewImg } from "./AssetPreviewImg";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import {
  resolveDisplayName,
  resolveDisplayDesc,
} from "@/lib/i18n/localizedName";

interface EventInGamePreviewProps {
  gameEvent: GameEvent;
  projectId: number;
  isDraft?: boolean;
  activeTarget?: EventPreviewTarget | null;

  onSelectTarget?: (target: EventPreviewTarget, subIndex?: number) => void;

  activeOptionIndex?: number | null;
}

export type EventPreviewTarget =
  "picture" | "title" | "description" | "options";

function flattenLocField(val: GameEvent["title"]): string {
  if (typeof val === "string") return val;
  if (Array.isArray(val) && val.length > 0) {
    const first = val[0];
    return typeof first?.text === "string" ? first.text : "";
  }
  return "";
}

function eventTypeBadge(eventType: string): string {
  switch (eventType) {
    case "news_event":
      return "border-rose-400/40 bg-rose-500/15 text-rose-200";
    case "state_event":
      return "border-sky-400/40 bg-sky-500/15 text-sky-200";
    case "unit_leader_event":
    case "operative_leader_event":
      return "border-violet-400/40 bg-violet-500/15 text-violet-200";
    default:
      return "border-stone-500/40 bg-stone-700/40 text-stone-200";
  }
}

function editableTargetClass(
  target: EventPreviewTarget,
  activeTarget: EventPreviewTarget | null | undefined,
  enabled: boolean,
): string {
  if (!enabled) return "";
  const active =
    activeTarget === target
      ? "ring-2 ring-accent/80 bg-accent/10"
      : "ring-1 ring-transparent hover:ring-accent/60";
  return `cursor-pointer rounded-sm outline-none transition-[box-shadow,background-color] ${active}`;
}

export function EventInGamePreview({
  gameEvent,
  projectId,
  isDraft = false,
  activeTarget,
  onSelectTarget,
  activeOptionIndex,
}: EventInGamePreviewProps) {
  const t = useTranslations("eventInGamePreview");
  const locale = useContentLocaleStore((s) => s.locale);
  const canSelectTarget = Boolean(onSelectTarget);

  const title =
    resolveDisplayName(
      locale,
      gameEvent.display_name_locs,
      gameEvent.title_en,
      gameEvent.title_zh,
      "",
    ) ||
    flattenLocField(gameEvent.title) ||
    gameEvent.event_id;
  const desc =
    resolveDisplayDesc(
      locale,
      gameEvent.description_locs,
      gameEvent.desc_en,
      gameEvent.desc_zh,
    ) ||
    flattenLocField(gameEvent.description) ||
    "";
  const options = gameEvent.options ?? [];
  const immediate =
    typeof gameEvent.immediate === "string" ? gameEvent.immediate.trim() : "";
  const after =
    typeof gameEvent.after === "string" ? gameEvent.after.trim() : "";

  return (
    <div
      className={`rounded-md border border-ingame-edge/60 bg-gradient-to-b from-ingame-panel/90 to-ingame-panel-deep/95 text-ingame-title shadow-sm overflow-hidden ${
        gameEvent.major ? "ring-1 ring-rose-500/40" : ""
      }`}
    >
      <button
        type="button"
        className={`relative block w-full bg-ingame-band/80 border-b border-ingame-edge/40 text-left ${editableTargetClass(
          "picture",
          activeTarget,
          canSelectTarget,
        )}`}
        onClick={() => onSelectTarget?.("picture")}
        title={canSelectTarget ? t("editPicture") : undefined}
      >
        {gameEvent.picture ? (
          <AssetPreviewImg
            url={eventsApi.pictureUrl(projectId, gameEvent.id)}
            className="w-full max-h-48 object-cover"
          />
        ) : (
          <div className="h-12 flex items-center justify-center text-micro font-mono text-ingame-text/70">
            {t("noPicture")} · {gameEvent.picture || t("empty")}
          </div>
        )}

        <div className="absolute top-1.5 right-1.5 flex gap-1 flex-wrap justify-end">
          {isDraft && (
            <span className="px-1.5 py-0.5 rounded border border-accent/50 bg-accent/20 text-micro font-mono text-accent">
              draft
            </span>
          )}
          <span
            className={`px-1.5 py-0.5 rounded border text-micro font-mono ${eventTypeBadge(gameEvent.event_type)}`}
          >
            {gameEvent.event_type.replace("_event", "")}
          </span>
          {gameEvent.major && (
            <span className="px-1.5 py-0.5 rounded border border-rose-400/50 bg-rose-500/30 text-micro font-mono text-rose-100">
              major
            </span>
          )}
          {gameEvent.hidden && (
            <span className="px-1.5 py-0.5 rounded border border-ingame-edge/40 bg-ingame-band/70 text-micro font-mono text-ingame-text">
              hidden
            </span>
          )}
          {gameEvent.fire_only_once && (
            <span className="px-1.5 py-0.5 rounded border border-purple-400/40 bg-purple-500/20 text-micro font-mono text-purple-200">
              fire-once
            </span>
          )}
        </div>
      </button>

      <button
        type="button"
        className={`block w-full px-3 pt-2.5 text-left ${editableTargetClass(
          "title",
          activeTarget,
          canSelectTarget,
        )}`}
        onClick={() => onSelectTarget?.("title")}
        title={canSelectTarget ? t("editTitle") : undefined}
      >
        <p className="text-base font-medium text-ingame-title leading-snug line-clamp-2">
          <CountryTagPreview text={title} chipStyle={false} />
        </p>
        <p className="text-micro font-mono text-ingame-text/75 mt-0.5">
          {gameEvent.event_id}
          {gameEvent.is_triggered_only ? " · triggered_only" : " · mtth"}
          {gameEvent.auto_fire_on_startup && " · auto-fire on_startup"}
          {gameEvent.on_action_attach &&
            gameEvent.on_action_attach !== "on_startup" && (
              <span> · {gameEvent.on_action_attach}</span>
            )}
        </p>
      </button>

      {desc && (
        <button
          type="button"
          className={`block w-full px-3 py-2 text-left text-xs text-ingame-text/90 whitespace-pre-wrap line-clamp-6 ${editableTargetClass(
            "description",
            activeTarget,
            canSelectTarget,
          )}`}
          onClick={() => onSelectTarget?.("description")}
          title={canSelectTarget ? t("editDescription") : undefined}
        >
          <CountryTagPreview text={desc} />
        </button>
      )}

      <div className="block w-full px-3 pb-3 pt-1 space-y-1.5 border-t border-ingame-edge/30 mt-1">
        {options.length === 0 ? (
          <div className="text-micro font-mono text-ingame-text/70 italic">
            {t("noOptions")}
          </div>
        ) : (
          options.map((opt: EventOption, i) => {
            const optName = resolveDisplayName(
              locale,
              opt.display_name_locs,
              opt.name_en,
              opt.name_zh,
              opt.name || `OPTION_${i + 1}`,
            );
            const hasTrigger = !!opt.trigger?.trim();

            const isActiveOption =
              activeTarget === "options" && activeOptionIndex === i;
            const affordance = !canSelectTarget
              ? ""
              : isActiveOption
                ? "cursor-pointer outline-none ring-2 ring-accent/80"
                : "cursor-pointer outline-none ring-1 ring-transparent hover:ring-accent/60";
            return (
              <button
                key={`${optName}-${i}`}
                type="button"
                onClick={() => onSelectTarget?.("options", i)}
                title={
                  canSelectTarget
                    ? t("editOption", { n: i + 1 })
                    : hasTrigger
                      ? t("conditionalOption")
                      : t("alwaysVisibleOption")
                }
                className={`block w-full text-left px-2.5 py-1.5 rounded border text-xs font-mono transition-[box-shadow,background-color,color] ${
                  hasTrigger
                    ? "border-ingame-button/50 bg-ingame-button/25 text-ingame-gold/90"
                    : "border-ingame-button/70 bg-ingame-button/40 text-ingame-title hover:bg-ingame-button/55"
                } ${isActiveOption ? "bg-accent/10" : ""} ${affordance}`}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate">{optName}</span>
                  <div className="flex items-center gap-1 shrink-0">
                    {hasTrigger && (
                      <span className="text-micro uppercase text-ingame-gold/70">
                        trig
                      </span>
                    )}
                    {opt.ai_chance &&
                      (opt.ai_chance.base != null ||
                        opt.ai_chance.factor != null) && (
                        <span
                          className="text-micro text-sky-300/80"
                          title="AI weight defined"
                        >
                          ai
                        </span>
                      )}
                  </div>
                </div>
              </button>
            );
          })
        )}
      </div>

      {(gameEvent.timeout_days > 0 || immediate || after) && (
        <div className="px-3 py-1.5 border-t border-ingame-edge/30 bg-ingame-band/60 text-micro font-mono text-ingame-text/75 flex flex-wrap gap-x-3 gap-y-0.5">
          {gameEvent.timeout_days > 0 && (
            <span>{t("timeout", { days: gameEvent.timeout_days })}</span>
          )}
          {immediate && <span>{t("immediateEffects")}</span>}
          {after && <span>{t("afterEffects")}</span>}
        </div>
      )}
    </div>
  );
}
