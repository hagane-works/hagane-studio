import { useSyncExternalStore } from "react";
import { requestGameIndexed, sessionGameData } from "@/lib/api/gameIndex";

export interface VanillaSubSection {
  name: string;
  sub_key: string;
  y: number;
  text_key: string;

  names: Record<string, string>;
}

interface FolderInfo {
  sub_sections: VanillaSubSection[];
  count: number;
}

export type TechSections = Readonly<Record<string, FolderInfo>>;

const NO_SECTIONS: TechSections = {};

const techSections = sessionGameData(
  () =>
    requestGameIndexed<{ folders: Record<string, FolderInfo> }>(
      "/go/vanilla/tech-sections",
    ).then((data): TechSections => data.folders ?? {}),
  NO_SECTIONS,
);

export function useTechSections(): TechSections {
  return useSyncExternalStore(
    techSections.subscribe,
    techSections.current,
    techSections.current,
  );
}

const CAT_TO_SUB_KEY: Record<string, string> = {
  cat_light_armor: "light",
  cat_medium_armor: "medium",
  cat_heavy_armor: "heavy",
  cat_amphibious: "amphibious",

  cat_fighter: "fighter",
  cat_heavy_fighter: "h_fighter",
  cat_cas: "cas",
  cat_naval_bomber: "nav",
  cat_medium_air: "tac",
  cat_strategic_bomber: "strat",
  cat_scout_plane: "scout",
  cat_air_transport: "transport",

  cat_air_engine: "engines",
  cat_air_weapon: "weapons",

  cat_destroyers: "destroyer",
  cat_cruisers: "cruiser",
  cat_battlecruisers: "battlecruiser",
  cat_battleships: "battleship",
  cat_carriers: "carrier",
  cat_submarines: "sub",
  cat_transport_ships: "transport",
  bc_tech: "battlecruiser",

  industry: "production",
  cat_construction_tech: "construction",
  synthetic_oil: "synthetic_oil",
  cat_synthetic_resources: "synthetic_oil",

  cat_artillery: "artillery",
  cat_anti_air: "anti_air",
  cat_anti_tank: "anti_tank",
  cat_rocket: "rocket",

  infantry_weapons: "weapons",
  cat_infantry_weapons: "weapons",
  cat_machine_gun: "weapons",
  cat_motorized_equipment: "motorized",
  cat_mechanized_equipment: "motorized",
  marine: "specialists",
  paratroopers: "specialists",
  mountaineers: "specialists",
  cat_special_forces: "specialists",

  cat_train: "trains",
};

const TECH_ID_KEYWORDS: { kw: string; sub_key: string }[] = [
  { kw: "light", sub_key: "light" },
  { kw: "medium", sub_key: "medium" },
  { kw: "heavy", sub_key: "heavy" },
  { kw: "amphibious", sub_key: "amphibious" },
  { kw: "interwar_tank", sub_key: "medium" },
  { kw: "fighter", sub_key: "fighter" },
  { kw: "cas", sub_key: "cas" },
  { kw: "bomber", sub_key: "tac" },
  { kw: "transport", sub_key: "transport" },
  { kw: "destroyer", sub_key: "destroyer" },
  { kw: "cruiser", sub_key: "cruiser" },
  { kw: "battleship", sub_key: "battleship" },
  { kw: "carrier", sub_key: "carrier" },
  { kw: "submarine", sub_key: "sub" },
  { kw: "rocket", sub_key: "rocket" },
  { kw: "atomic", sub_key: "" },
];

export interface ResolvedSection {
  subKey: string;

  names: Record<string, string>;

  via: "y-band" | "category" | "tech_id" | "fallback";
}

export const FALLBACK_SECTION: ResolvedSection = {
  subKey: "_other",
  names: { en: "Other", zh: "其他" },
  via: "fallback",
};

interface MinimalTech {
  tech_id: string;
  position_y: number;
  categories: string[];
  folder?: string;
}

export function resolveVisualSection(
  tech: MinimalTech,
  folder: string,
  sections: TechSections,
): ResolvedSection {
  const folderToUse = tech.folder || folder;
  const folderInfo = sections[folderToUse];
  const subSections = folderInfo?.sub_sections ?? [];

  const buildFromSubSection = (
    s: VanillaSubSection,
    via: ResolvedSection["via"],
  ): ResolvedSection => ({
    subKey: s.sub_key,
    names: s.names ?? {},
    via,
  });

  if (subSections.length > 0 && typeof tech.position_y === "number") {
    let chosen: VanillaSubSection | null = null;
    for (const s of subSections) {
      if (tech.position_y >= s.y) {
        chosen = s;
      } else {
        break;
      }
    }
    if (!chosen) chosen = subSections[0];
    return buildFromSubSection(chosen, "y-band");
  }

  if (Array.isArray(tech.categories) && tech.categories.length > 0) {
    for (const cat of tech.categories) {
      const sub_key = CAT_TO_SUB_KEY[cat];
      if (sub_key) {
        const match = subSections.find((s) => s.sub_key === sub_key);
        if (match) return buildFromSubSection(match, "category");

        return {
          subKey: sub_key,
          names: {},
          via: "category",
        };
      }
    }
  }

  const lowerId = tech.tech_id.toLowerCase();
  for (const { kw, sub_key } of TECH_ID_KEYWORDS) {
    if (sub_key && lowerId.includes(kw)) {
      const match = subSections.find((s) => s.sub_key === sub_key);
      if (match) return buildFromSubSection(match, "tech_id");
    }
  }

  return FALLBACK_SECTION;
}

export function getFolderSubSections(
  folder: string,
  sections: TechSections,
): VanillaSubSection[] {
  return sections[folder]?.sub_sections ?? [];
}
