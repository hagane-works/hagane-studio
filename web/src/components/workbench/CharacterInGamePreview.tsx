"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import type { Character } from "@/types";
import { API_BASE } from "@/lib/api";
import { useBaseModDepKeys, artUrl, gfxUrl } from "@/lib/api/basemod";
import { CountryTagPreview } from "@/components/CountryTagPreview";
import { visibleCharacterName } from "@/lib/i18n/characterDisplayName";

interface CharacterInGamePreviewProps {
  character: Character;
  projectId: number;
  isDraft?: boolean;

  portraitVersion?: number;
}

function PortraitWithFallback({
  candidates,
  emptyLabel,
}: {
  candidates: string[];
  emptyLabel: string;
}) {
  const [stage, setStage] = useState(0);
  const current = stage < candidates.length ? candidates[stage] : null;

  return current ? (
    <img
      src={current}
      alt=""
      className="h-full w-full object-cover"
      onError={() => setStage((value) => value + 1)}
    />
  ) : (
    <div className="flex h-full items-center justify-center px-2 text-center text-micro font-mono text-ingame-text/70">
      {emptyLabel}
    </div>
  );
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

function roleLabel(
  role: Character["role_type"],
  t: ReturnType<typeof useTranslations>,
): string {
  switch (role) {
    case "country_leader":
      return t("roleCountryLeader");
    case "field_marshal":
      return t("roleFieldMarshal");
    case "general":
      return t("roleCorpsCommander");
    case "admiral":
      return t("roleAdmiral");
    case "scientist":
      return t("roleScientist");
    case "operative":
      return t("roleOperative");
    case "advisor":
      return t("roleAdvisor");
    default:
      return titleCaseToken(role);
  }
}

function portraitUrl(
  projectId: number,
  character: Character,
  version?: number,
): string | null {
  const ref = (
    character.large_portrait ||
    character.small_portrait ||
    ""
  ).trim();
  if (!ref) return null;
  const encoded = encodeURI(ref).replace(/#/g, "%23");
  const base = `${API_BASE}/go/projects/${projectId}/characters/${character.id}/assets/${encoded}`;
  return version ? `${base}?t=${version}` : base;
}

function statRows(character: Character): Array<[string, number | string]> {
  if (character.role_type === "operative") {
    return [
      ["Skill", character.operative_skill_level ?? 1],
      ["Nationality", character.operative_nationalities?.join(", ") || "Any"],
      ["Background", character.operative_bg || "Default"],
    ];
  }

  if (character.role_type === "scientist") {
    const specs = character.scientist_data?.specializations ?? [];
    return [
      ["Skill", character.skill ?? 1],
      [
        "Specializations",
        specs.map((s) => `${titleCaseToken(s.spec)} ${s.level}`).join(", ") ||
          "None",
      ],
      [
        "Multiple roles",
        character.scientist_data?.allow_multiple ? "Yes" : "No",
      ],
    ];
  }

  if (character.role_type === "admiral") {
    return [
      ["Skill", character.skill ?? 1],
      ["Attack", character.attack_skill ?? 1],
      ["Defense", character.defense_skill ?? 1],
      ["Maneuver", character.maneuvering_skill ?? 1],
      ["Coordination", character.coordination_skill ?? 1],
    ];
  }

  return [
    ["Skill", character.skill ?? 1],
    ["Attack", character.attack_skill ?? 1],
    ["Defense", character.defense_skill ?? 1],
    ["Planning", character.planning_skill ?? 1],
    ["Logistics", character.logistics_skill ?? 1],
  ];
}

function modifierLines(raw: string | null | undefined): string[] {
  return (raw || "")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .slice(0, 4);
}

export function CharacterInGamePreview({
  character,
  projectId,
  isDraft = false,
  portraitVersion,
}: CharacterInGamePreviewProps) {
  const t = useTranslations("characterInGamePreview");
  const name =
    visibleCharacterName(
      pickLocalizedText(
        character.display_name_locs,
        character.name_en,
        character.name_zh,
      ),
    ) ||
    visibleCharacterName(character.name) ||
    character.tag;

  const description =
    pickLocalizedText(
      (character as { description_locs?: Record<string, string> })
        .description_locs,
      character.desc_en,
      character.desc_zh,
    ) ||
    character.description ||
    "";

  const baseModDepKeys = useBaseModDepKeys();
  const portraitRef = (
    character.large_portrait ||
    character.small_portrait ||
    ""
  ).trim();

  const baseModPortraitUrls = useMemo(
    () =>
      portraitRef && baseModDepKeys.length
        ? baseModDepKeys.map((dk) =>
            portraitRef.startsWith("GFX_")
              ? gfxUrl(projectId, dk, portraitRef)
              : artUrl(projectId, dk, portraitRef),
          )
        : [],
    [projectId, portraitRef, baseModDepKeys],
  );
  const projectImgUrl = portraitUrl(projectId, character, portraitVersion);
  const portraitCandidates = useMemo(() => {
    const c: string[] = [];
    if (projectImgUrl) c.push(projectImgUrl);
    c.push(...baseModPortraitUrls);
    return c;
  }, [projectImgUrl, baseModPortraitUrls]);
  const portraitCandidatesKey = JSON.stringify(portraitCandidates);
  const traits =
    character.role_type === "operative"
      ? (character.operative_traits ?? [])
      : character.role_type === "advisor"
        ? (character.advisor_traits ?? [])
        : (character.traits ?? []);
  const advisorLines = modifierLines(character.advisor_modifier);
  const showAdvisor =
    character.has_advisor_role || character.role_type === "advisor";

  return (
    <div className="overflow-hidden rounded-md border border-ingame-edge/60 bg-ingame-panel text-ingame-title shadow-sm">
      <div className="flex items-center justify-between gap-2 border-b border-ingame-edge/40 bg-ingame-band/80 px-3 py-2">
        <div className="min-w-0">
          <div className="truncate font-mono text-lg font-semibold text-ingame-title">
            <CountryTagPreview text={name} />
          </div>
          <div className="mt-0.5 flex flex-wrap items-center gap-1.5 text-micro font-mono uppercase tracking-wide text-ingame-text">
            <span>{character.country_tag || "TAG"}</span>
            <span className="text-ingame-text/60">/</span>
            <span>{roleLabel(character.role_type, t)}</span>
            {isDraft && (
              <span className="rounded border border-accent/50 bg-accent/20 px-1.5 py-0.5 text-accent">
                {t("draft")}
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="grid gap-4 p-4 sm:grid-cols-[150px_1fr]">
        <div className="overflow-hidden rounded-sm border border-ingame-edge/50 bg-ingame-panel-deep">
          <div className="aspect-[156/210] bg-gradient-to-b from-ingame-band to-ingame-panel-deep">
            <PortraitWithFallback
              key={portraitCandidatesKey}
              candidates={portraitCandidates}
              emptyLabel={t("noPortrait")}
            />
          </div>
          {character.large_portrait && (
            <div className="truncate border-t border-ingame-edge/40 px-2 py-1 text-micro font-mono text-ingame-text/70">
              {character.large_portrait}
            </div>
          )}
        </div>

        <div className="min-w-0 space-y-3">
          <div className="grid grid-cols-2 gap-1.5">
            {statRows(character).map(([label, value]) => (
              <div
                key={label}
                className="rounded-sm border border-ingame-edge/40 bg-ingame-band/60 px-2 py-1"
              >
                <div className="text-micro font-mono uppercase tracking-wide text-ingame-gold/80">
                  {label}
                </div>
                <div className="truncate text-base font-mono text-ingame-title">
                  {value}
                </div>
              </div>
            ))}
          </div>

          {traits.length > 0 && (
            <div>
              <div className="mb-1 text-micro font-mono uppercase tracking-wide text-ingame-text/70">
                {t("traits")}
              </div>
              <div className="flex flex-wrap gap-1">
                {traits.slice(0, 8).map((trait) => (
                  <span
                    key={trait}
                    className="rounded-sm border border-ingame-edge/50 bg-ingame-band/70 px-1.5 py-0.5 text-micro font-mono text-ingame-gold/90"
                  >
                    {trait}
                  </span>
                ))}
                {traits.length > 8 && (
                  <span className="rounded-sm border border-ingame-edge/50 px-1.5 py-0.5 text-micro font-mono text-ingame-text/80">
                    +{traits.length - 8}
                  </span>
                )}
              </div>
            </div>
          )}

          {description && (
            <div className="rounded-sm border border-ingame-edge/40 bg-black/25 px-2.5 py-2 text-sm leading-relaxed text-ingame-text">
              <CountryTagPreview text={description} />
            </div>
          )}
        </div>
      </div>

      {showAdvisor && (
        <div className="border-t border-ingame-edge/40 bg-ingame-band/50 px-3 py-2">
          <div className="mb-1 flex items-center justify-between gap-2 text-micro font-mono uppercase tracking-wide text-ingame-text/80">
            <span>
              {titleCaseToken(character.advisor_slot) === "None"
                ? t("advisorSlot")
                : titleCaseToken(character.advisor_slot)}
            </span>
            <span>{t("cost", { amount: character.advisor_cost ?? 0 })}</span>
          </div>
          <div className="flex flex-wrap gap-1">
            {(character.advisor_traits ?? []).slice(0, 6).map((trait) => (
              <span
                key={trait}
                className="rounded-sm border border-ingame-edge/50 bg-ingame-band/70 px-1.5 py-0.5 text-micro font-mono text-ingame-gold/90"
              >
                {trait}
              </span>
            ))}
          </div>
          {advisorLines.length > 0 && (
            <div className="mt-2 space-y-0.5 font-mono text-micro text-emerald-300/90">
              {advisorLines.map((line) => (
                <div key={line} className="truncate">
                  {line}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
