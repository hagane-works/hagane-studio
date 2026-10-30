import type { CountryRow } from "@/lib/api/scenario";
import {
  deriveGraphicalCulture2d,
  hexToRgb,
  rgbToHex,
} from "@/lib/graphicalCultures";

export interface GameCountryProfile {
  tag: string;
  name: string;
  adjective: string;
  names: Record<string, string>;
  groups: { id: string; label: string }[];
  ruling: string;
  graphical_culture: string;
  shown: string;
  capital?: { id: number; name?: string };
  flag?: { url: string; width: number; height: number; project?: boolean };
}

export interface ProfileDraft {
  name: string;
  adjective: string;

  names: Record<string, string>;
  color: string;

  colorUi: string;
  culture: string;
  capital: number;
}

type ProfileRow = Pick<
  CountryRow,
  | "tag"
  | "name_en"
  | "display_name_locs"
  | "adjective_en"
  | "adjective_locs"
  | "ideology_name_locs"
  | "color"
  | "color_ui"
  | "graphical_culture"
>;

function own(value: string | undefined, tag: string): string {
  const text = (value ?? "").trim();
  return text && text.toUpperCase() !== tag.toUpperCase() ? text : "";
}

export function profileDraft(
  row: ProfileRow,
  game: GameCountryProfile,
  capital: number | null,
): ProfileDraft {
  const tag = row.tag;
  const names: Record<string, string> = {};
  for (const group of game.groups) {
    names[group.id] =
      own(row.ideology_name_locs?.[group.id]?.en, tag) ||
      game.names[group.id] ||
      "";
  }
  return {
    name:
      own(row.display_name_locs?.en, tag) || own(row.name_en, tag) || game.name,
    adjective:
      (row.adjective_locs?.en ?? "").trim() ||
      (row.adjective_en ?? "").trim() ||
      game.adjective,
    names,
    color: row.color?.length === 3 ? rgbToHex(row.color) : "#808080",
    colorUi: row.color_ui?.length === 3 ? rgbToHex(row.color_ui) : "",
    culture: (row.graphical_culture ?? "").trim() || game.graphical_culture,
    capital: capital ?? game.capital?.id ?? 0,
  };
}

export type ProfileProblem =
  "nameRequired" | "adjectiveRequired" | "capitalRequired";

export function profileProblem(draft: ProfileDraft): ProfileProblem | null {
  if (!draft.name.trim()) return "nameRequired";
  if (!draft.adjective.trim()) return "adjectiveRequired";
  if (!Number.isInteger(draft.capital) || draft.capital <= 0)
    return "capitalRequired";
  return null;
}

export function profilePatch(
  draft: ProfileDraft,
  start: ProfileDraft,
  row: ProfileRow,
): { country: Partial<CountryRow> | null; capital: number | null } {
  const namesChanged =
    draft.name.trim() !== start.name ||
    draft.adjective.trim() !== start.adjective ||
    Object.keys(draft.names).some(
      (id) => (draft.names[id] ?? "").trim() !== (start.names[id] ?? ""),
    );
  const colorChanged = draft.color !== start.color;
  const colorUiChanged = draft.colorUi !== start.colorUi;
  const cultureChanged = draft.culture !== start.culture;
  let country: Partial<CountryRow> | null = null;
  if (namesChanged || colorChanged || colorUiChanged || cultureChanged) {
    const ideologyNames: Record<string, Record<string, string>> = {};
    for (const [id, locs] of Object.entries(row.ideology_name_locs ?? {}))
      ideologyNames[id] = { ...locs };
    for (const [id, name] of Object.entries(draft.names)) {
      if (name.trim())
        ideologyNames[id] = { ...(ideologyNames[id] ?? {}), en: name.trim() };
    }
    country = {
      name_en: draft.name.trim(),
      display_name_locs: {
        ...(row.display_name_locs ?? {}),
        en: draft.name.trim(),
      },
      adjective_en: draft.adjective.trim(),
      adjective_locs: {
        ...(row.adjective_locs ?? {}),
        en: draft.adjective.trim(),
      },
      ideology_name_locs: ideologyNames,
    };
    if (colorChanged) country.color = hexToRgb(draft.color);
    if (colorUiChanged)
      country.color_ui = draft.colorUi ? hexToRgb(draft.colorUi) : [];
    if (cultureChanged) {
      country.graphical_culture = draft.culture;
      country.graphical_culture_2d = deriveGraphicalCulture2d(draft.culture);
    }
  }
  return {
    country,
    capital: draft.capital !== start.capital ? draft.capital : null,
  };
}
