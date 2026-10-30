"use client";

import { useTranslations } from "use-intl";
import type { Idea } from "@/types";
import { ideasApi } from "@/lib/api";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import { CountryTagPreview } from "@/components/CountryTagPreview";
import { AssetPreviewImg } from "./AssetPreviewImg";

interface IdeaInGamePreviewProps {
  idea: Idea;
  projectId: number;
  isDraft?: boolean;
}

function pickLocalizedText(
  locs: Record<string, string> | null | undefined,
  legacyEn?: string,
  legacyZh?: string,
): string {
  const anyLoc = Object.values(locs ?? {})
    .find((value) => value.trim())
    ?.trim();
  return (
    locs?.en?.trim() ||
    locs?.zh?.trim() ||
    anyLoc ||
    legacyEn?.trim() ||
    legacyZh?.trim() ||
    ""
  );
}

function titleCaseToken(value: string | null | undefined): string {
  const text = (value || "").replace(/_/g, " ").trim();
  if (!text) return "None";
  return text.replace(/\b\w/g, (m) => m.toUpperCase());
}

function formatModifierValue(value: number | string): string {
  if (typeof value === "string") return value;
  if (!Number.isFinite(value)) return String(value);
  const sign = value > 0 ? "+" : "";
  if (Math.abs(value) > 0 && Math.abs(value) < 2) {
    return `${sign}${Math.round(value * 1000) / 10}%`;
  }
  return `${sign}${value}`;
}

function modifierTone(value: number | string): string {
  if (typeof value !== "number") return "text-ingame-text";
  if (value > 0) return "text-emerald-300";
  if (value < 0) return "text-rose-300";
  return "text-ingame-text";
}

export function IdeaInGamePreview({
  idea,
  projectId,
  isDraft = false,
}: IdeaInGamePreviewProps) {
  const t = useTranslations("ideaInGamePreview");
  const title =
    pickLocalizedText(idea.display_name_locs, idea.name_en, idea.name_zh) ||
    idea.name ||
    idea.idea_id;
  const description =
    pickLocalizedText(idea.description_locs, idea.desc_en, idea.desc_zh) || "";
  const modifiers = Object.entries(idea.modifier || {}).filter(
    ([key]) => !key.startsWith("__"),
  );
  const research = Object.entries(idea.research_bonus || {});
  const designerTraits = idea.designer_traits || [];
  const picture = idea.picture?.trim() ?? "";
  const showPicture = Boolean(picture);

  const baseModDepKeys = useBaseModDepKeys();
  const baseModPictures = baseModDepKeys.map((dk) =>
    gfxUrl(
      projectId,
      dk,
      picture.startsWith("GFX_") ? picture : `GFX_idea_${picture}`,
    ),
  );

  return (
    <div className="overflow-hidden rounded-md border border-ingame-edge/60 bg-ingame-panel text-ingame-title shadow-sm">
      <div className="border-b border-ingame-edge/40 bg-ingame-band/80 px-3 py-2">
        <div className="flex items-start gap-3">
          <div className="h-20 w-20 shrink-0 overflow-hidden rounded-sm border border-ingame-edge/50 bg-ingame-panel-deep">
            {showPicture ? (
              <AssetPreviewImg
                url={ideasApi.pictureUrl(projectId, idea.id)}
                fallbackUrls={baseModPictures}
                className="h-full w-full object-cover"
              />
            ) : (
              <div className="flex h-full items-center justify-center text-micro font-mono text-ingame-text/70">
                {t("noIcon")}
              </div>
            )}
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0">
                <div className="truncate font-mono text-lg font-semibold text-ingame-title">
                  <CountryTagPreview text={title} />
                </div>
                <div className="mt-1 flex flex-wrap items-center gap-1.5 text-micro font-mono uppercase tracking-wide text-ingame-text/80">
                  <span>{idea.country_tag || "TAG"}</span>
                  <span className="text-ingame-text/50">/</span>
                  <span>{titleCaseToken(idea.idea_type)}</span>
                  {idea.ledger && (
                    <>
                      <span className="text-ingame-text/50">/</span>
                      <span>{titleCaseToken(idea.ledger)}</span>
                    </>
                  )}
                  {idea.is_designer && (
                    <span className="rounded border border-ingame-edge/50 bg-ingame-band/70 px-1.5 py-0.5 text-ingame-gold/90">
                      {t("designer")}
                    </span>
                  )}
                  {isDraft && (
                    <span className="rounded border border-accent/50 bg-accent/20 px-1.5 py-0.5 text-accent">
                      {t("draft")}
                    </span>
                  )}
                </div>
              </div>
            </div>
            <div className="mt-1 truncate text-micro font-mono text-ingame-text/70">
              {idea.idea_id}
            </div>
          </div>
        </div>
      </div>

      {description && (
        <div className="border-b border-ingame-edge/30 px-3 py-2 text-sm leading-relaxed text-ingame-text">
          <CountryTagPreview text={description} />
        </div>
      )}

      <div className="space-y-3 px-3 py-3">
        {modifiers.length > 0 ? (
          <div>
            <div className="mb-1 text-micro font-mono uppercase tracking-wide text-ingame-text/70">
              {t("effects")}
            </div>
            <div className="space-y-1 rounded-sm border border-ingame-edge/40 bg-black/25 px-2 py-1.5">
              {modifiers.slice(0, 8).map(([key, value]) => (
                <div
                  key={key}
                  className="flex items-center justify-between gap-3 text-xs font-mono"
                >
                  <span className="min-w-0 truncate text-ingame-text">
                    {titleCaseToken(key)}
                  </span>
                  <span className={`shrink-0 ${modifierTone(value)}`}>
                    {formatModifierValue(value)}
                  </span>
                </div>
              ))}
              {modifiers.length > 8 && (
                <div className="text-micro font-mono text-ingame-text/70">
                  {t("moreModifiers", { count: modifiers.length - 8 })}
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="rounded-sm border border-ingame-edge/40 bg-black/20 px-2 py-2 text-xs font-mono text-ingame-text/70">
            {t("noModifiers")}
          </div>
        )}

        {(designerTraits.length > 0 || research.length > 0) && (
          <div className="grid gap-2 sm:grid-cols-2">
            {designerTraits.length > 0 && (
              <div>
                <div className="mb-1 text-micro font-mono uppercase tracking-wide text-ingame-text/70">
                  {t("traits")}
                </div>
                <div className="flex flex-wrap gap-1">
                  {designerTraits.slice(0, 6).map((trait) => (
                    <span
                      key={trait}
                      className="rounded-sm border border-ingame-edge/50 bg-ingame-band/70 px-1.5 py-0.5 text-micro font-mono text-ingame-gold/90"
                    >
                      {trait}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {research.length > 0 && (
              <div>
                <div className="mb-1 text-micro font-mono uppercase tracking-wide text-ingame-text/70">
                  {t("researchBonus")}
                </div>
                <div className="flex flex-wrap gap-1">
                  {research.slice(0, 6).map(([key, value]) => (
                    <span
                      key={key}
                      className="rounded-sm border border-ingame-edge/50 bg-ingame-band/70 px-1.5 py-0.5 text-micro font-mono text-ingame-gold/90"
                    >
                      {key}: {formatModifierValue(value)}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
