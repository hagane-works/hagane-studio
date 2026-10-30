import type { ScanResults, ScanCategory, ScanStatusTranslator } from "./types";
import {
  collectDeclaredCountryTagCount,
  collectGfxAndGuiFiles,
  collectGfxFiles,
  collectLocFiles,
  collectFlagFiles,
  collectMapFiles,
  collectTxtFiles,
  createMainThreadCheckpoint,
  FLAT_COMMON_TEXT_DIRS,
  readFileByPath,
  tryGetDir,
  yieldToMainThread,
} from "@/lib/fs-scanner";

export function emptyScan(): ScanResults {
  return {
    characters: { files: [], fileCount: 0, enabled: true },
    focus_trees: { files: [], fileCount: 0, enabled: true },
    technologies: { files: [], fileCount: 0, enabled: true },
    ideas: { files: [], fileCount: 0, enabled: true },
    events: { files: [], fileCount: 0, enabled: true },
    decisions: { files: [], fileCount: 0, enabled: true },
    bookmarks: { files: [], fileCount: 0, enabled: true },
    history_files: { files: [], fileCount: 0, enabled: true },
    states: { files: [], fileCount: 0, enabled: true },
    country_tags: { files: [], fileCount: 0, enabled: false },
    gfx: { files: [], fileCount: 0, enabled: true },
    gui: { files: [], fileCount: 0, enabled: true },
    loc: { files: [], fileCount: 0, enabled: true },
    raids: { files: [], fileCount: 0, enabled: true },
    units: { files: [], fileCount: 0, enabled: true },
    common_rules: { files: [], fileCount: 0, enabled: true },
    map: { files: [], fileCount: 0, enabled: true },
    flags: { files: [], fileCount: 0, enabled: true },
    descriptor: null,
  };
}

export const COMMON_RULE_DIRS = [
  "dynamic_modifiers",
  "modifier_definitions",
  "ai_strategy_plans",
  "ai_templates",
  "wargoals",
  "autonomous_states",
  "intelligence_agencies",
  "scripted_effects",
  "scripted_triggers",
  "scripted_localisation",
  "doctrines",

  "factions",

  "on_actions",

  "countries",

  "ideologies",
] as const;

export const COUNTRY_AUDIT_PREFIXES = [
  "common/country_tags/",
  "common/countries/",
] as const;

export function selectCountryAuditFiles(
  scan: Pick<ScanResults, "country_tags" | "common_rules">,
) {
  const candidates = [...scan.country_tags.files, ...scan.common_rules.files];
  return candidates.filter((entry) => {
    const path = entry.path.replace(/\\/g, "/").toLowerCase();
    return COUNTRY_AUDIT_PREFIXES.some((prefix) => path.startsWith(prefix));
  });
}

const CHARACTER_SIBLING_DIRS = [
  "unit_leader",
  "scientist_traits",
  "opinion_modifiers",
] as const;

export type ScanSection =
  | "characters"
  | "focus"
  | "tech"
  | "ideas"
  | "events"
  | "decisions"
  | "bookmarks"
  | "history"
  | "states"
  | "raids"
  | "units"
  | "commonRules"
  | "map"
  | "flags"
  | "countryTags"
  | "localisation"
  | "gfx"
  | "descriptor";

export interface ScanStatusEvent {
  key: string;
  fallback: string;
  values?: Record<string, string | number>;
}

export interface PerformScanOptions {
  sections?: readonly ScanSection[];
  onStatusEvent?: (event: ScanStatusEvent) => void;

  signal?: AbortSignal;
}

export const PREVIEW_SECTIONS: readonly ScanSection[] = [
  "characters",
  "focus",
  "ideas",
  "events",
  "decisions",
  "countryTags",
];

export async function performScan(
  dirHandle: FileSystemDirectoryHandle,
  tag: string,
  onStatus: (msg: string) => void,
  t?: ScanStatusTranslator,
  opts?: PerformScanOptions,
): Promise<ScanResults> {
  opts?.signal?.throwIfAborted();
  const results = emptyScan();
  const upperTag = tag?.toUpperCase() || "";

  const status = async (
    key: string,
    fallback: string,
    values?: Record<string, string | number>,
  ): Promise<void> => {
    opts?.signal?.throwIfAborted();
    opts?.onStatusEvent?.({ key, fallback, values });
    onStatus(t ? t(key, values) : fallback);

    await yieldToMainThread();
    opts?.signal?.throwIfAborted();
  };

  const sections = opts?.sections;
  const want = (s: ScanSection): boolean => !sections || sections.includes(s);
  const assemblyCheckpoint = createMainThreadCheckpoint();
  const appendFiles = async (
    target: ScanCategory["files"],
    source: ScanCategory["files"],
    map?: (
      file: ScanCategory["files"][number],
    ) => ScanCategory["files"][number],
  ): Promise<void> => {
    for (const file of source) {
      opts?.signal?.throwIfAborted();
      target.push(map ? map(file) : file);
      const pendingYield = assemblyCheckpoint();
      if (pendingYield) await pendingYield;
    }
  };

  const sectionDir = async (
    sec: ScanSection,
    key: string,
    fallback: string,
    ...segments: string[]
  ): Promise<FileSystemDirectoryHandle | null> => {
    opts?.signal?.throwIfAborted();
    if (!want(sec)) return null;
    await status(key, fallback);
    const section = await tryGetDir(dirHandle, ...segments);
    opts?.signal?.throwIfAborted();
    return section;
  };

  if (want("characters"))
    await status("scanCharacters", `Scanning characters for ${upperTag}...`, {
      tag: upperTag,
    });
  const charsDir = want("characters")
    ? await tryGetDir(dirHandle, "common", "characters")
    : null;
  if (charsDir) {
    results.characters.files = await collectTxtFiles(
      charsDir,
      "common/characters",
      { filterTag: upperTag },
    );
    results.characters.fileCount = results.characters.files.length;
  }

  const clDir = want("characters")
    ? await tryGetDir(dirHandle, "common", "country_leader")
    : null;
  if (clDir) {
    const clFiles = await collectTxtFiles(clDir, "common/country_leader");
    await appendFiles(results.characters.files, clFiles);
    results.characters.fileCount = results.characters.files.length;
  }

  for (const sub of want("characters") ? CHARACTER_SIBLING_DIRS : []) {
    const subDir = await tryGetDir(dirHandle, "common", sub);
    if (!subDir) continue;
    const subFiles = await collectTxtFiles(subDir, `common/${sub}`);
    if (subFiles.length === 0) continue;
    await appendFiles(results.characters.files, subFiles);
    results.characters.fileCount = results.characters.files.length;
  }

  if (want("focus"))
    await status("scanFocus", `Scanning focus trees for ${upperTag}...`, {
      tag: upperTag,
    });
  const focusDir = want("focus")
    ? await tryGetDir(dirHandle, "common", "national_focus")
    : null;
  if (focusDir) {
    results.focus_trees.files = await collectTxtFiles(
      focusDir,
      "common/national_focus",
      { filterTag: upperTag, keepShared: true },
    );
    results.focus_trees.fileCount = results.focus_trees.files.length;
  }

  const techDir = await sectionDir(
    "tech",
    "scanTech",
    "Scanning technologies...",
    "common",
    "technologies",
  );
  if (techDir) {
    results.technologies.files = await collectTxtFiles(
      techDir,
      "common/technologies",
    );
    results.technologies.fileCount = results.technologies.files.length;
  }

  if (want("ideas"))
    await status("scanIdeas", `Scanning ideas for ${upperTag}...`, {
      tag: upperTag,
    });
  const ideasDir = want("ideas")
    ? await tryGetDir(dirHandle, "common", "ideas")
    : null;
  if (ideasDir) {
    results.ideas.files = await collectTxtFiles(ideasDir, "common/ideas", {
      filterTag: upperTag,
      keepShared: true,
    });
    results.ideas.fileCount = results.ideas.files.length;
  }

  if (want("events"))
    await status("scanEvents", `Scanning events for ${upperTag}...`, {
      tag: upperTag,
    });
  const eventsDir = want("events")
    ? await tryGetDir(dirHandle, "events")
    : null;
  if (eventsDir) {
    const eventsStats = { seen: 0, kept: 0 };
    results.events.files = await collectTxtFiles(eventsDir, "events", {
      filterTag: upperTag,
      keepShared: true,
      keepGlobalNamespace: true,
      stats: eventsStats,
    });
    results.events.fileCount = results.events.files.length;
    results.events.skippedLocal = Math.max(
      0,
      eventsStats.seen - eventsStats.kept,
    );
  }

  if (want("decisions"))
    await status("scanDecisions", `Scanning decisions for ${upperTag}...`, {
      tag: upperTag,
    });
  const decisionsDir = want("decisions")
    ? await tryGetDir(dirHandle, "common", "decisions")
    : null;
  if (decisionsDir) {
    results.decisions.files = await collectTxtFiles(
      decisionsDir,
      "common/decisions",
      {
        filterTag: upperTag,
        keepShared: true,
        skipSubtrees: ["common/decisions/categories"],
      },
    );
    const catDir = await tryGetDir(decisionsDir, "categories");
    if (catDir) {
      const catFiles = await collectTxtFiles(
        catDir,
        "common/decisions/categories",
      );
      await appendFiles(results.decisions.files, catFiles);
    }
    results.decisions.fileCount = results.decisions.files.length;
  }

  const bookmarksDir = await sectionDir(
    "bookmarks",
    "scanBookmarks",
    "Scanning bookmarks...",
    "common",
    "bookmarks",
  );
  if (bookmarksDir) {
    results.bookmarks.files = await collectTxtFiles(
      bookmarksDir,
      "common/bookmarks",
    );
    results.bookmarks.fileCount = results.bookmarks.files.length;
  }

  if (want("history"))
    await status("scanHistory", `Scanning history files for ${upperTag}...`, {
      tag: upperTag,
    });
  const historyDir = want("history")
    ? await tryGetDir(dirHandle, "history", "countries")
    : null;
  if (historyDir) {
    results.history_files.files = await collectTxtFiles(
      historyDir,
      "history/countries",
      { filterTag: upperTag },
    );
    results.history_files.fileCount = results.history_files.files.length;
  }

  const statesDir = await sectionDir(
    "states",
    "scanStates",
    "Scanning states...",
    "history",
    "states",
  );
  if (statesDir) {
    results.states.files = await collectTxtFiles(statesDir, "history/states");
    results.states.fileCount = results.states.files.length;
  }

  const raidsDir = await sectionDir(
    "raids",
    "scanRaids",
    "Scanning raids...",
    "common",
    "raids",
  );
  if (raidsDir) {
    results.raids.files = await collectTxtFiles(raidsDir, "common/raids");
    results.raids.fileCount = results.raids.files.length;
  }

  const unitsDir = await sectionDir(
    "units",
    "scanUnits",
    "Scanning units...",
    "common",
    "units",
  );
  if (unitsDir) {
    await appendFiles(
      results.units.files,
      await collectTxtFiles(unitsDir, "common/units"),
    );
  }
  const historyUnitsDir = want("units")
    ? await tryGetDir(dirHandle, "history", "units")
    : null;
  if (historyUnitsDir) {
    await appendFiles(
      results.units.files,
      await collectTxtFiles(historyUnitsDir, "history/units"),
    );
  }
  results.units.fileCount = results.units.files.length;

  const mapDir = await sectionDir("map", "scanMap", "Scanning map...", "map");
  if (mapDir) {
    results.map.files = await collectMapFiles(mapDir);
    results.map.fileCount = results.map.files.length;
  }

  const flagsDir = await sectionDir(
    "flags",
    "scanFlags",
    "Scanning flags...",
    "gfx",
    "flags",
  );
  if (flagsDir) {
    results.flags.files = await collectFlagFiles(flagsDir);
    results.flags.fileCount = results.flags.files.length;
  }

  const commonDir = await sectionDir(
    "commonRules",
    "scanCommonRules",
    "Scanning shared rules & scripts...",
    "common",
  );
  if (commonDir) {
    const dirCounts: Record<string, number> = {};
    for (const sub of COMMON_RULE_DIRS) {
      const subDir = await tryGetDir(commonDir, sub);
      if (!subDir) continue;
      const files = await collectTxtFiles(subDir, `common/${sub}`);
      let selected = files;
      if (FLAT_COMMON_TEXT_DIRS.has(sub)) {
        selected = [];
        for (const file of files) {
          if (file.path.split("/").length === 3) selected.push(file);
          const pendingYield = assemblyCheckpoint();
          if (pendingYield) await pendingYield;
        }
      }
      if (selected.length > 0) {
        await appendFiles(results.common_rules.files, selected);
        dirCounts[sub] = selected.length;
      }
    }
    results.common_rules.fileCount = results.common_rules.files.length;
    results.common_rules.dirCounts = dirCounts;
  }

  const countryTagsDir = await sectionDir(
    "countryTags",
    "scanCountryTags",
    "Scanning country_tags...",
    "common",
    "country_tags",
  );
  if (countryTagsDir) {
    results.country_tags.files = await collectTxtFiles(
      countryTagsDir,
      "common/country_tags",
    );
    results.country_tags.fileCount = results.country_tags.files.length;
    results.country_tags.declaredTagCount =
      await collectDeclaredCountryTagCount(results.country_tags.files);
  }

  const localisationDir = await sectionDir(
    "localisation",
    "scanLocalisation",
    "Scanning localisation...",
    "localisation",
  );
  if (localisationDir) {
    results.loc.files = await collectLocFiles(localisationDir, "localisation");
    results.loc.fileCount = results.loc.files.length;
  }

  const interfaceDir = await sectionDir(
    "gfx",
    "scanGfx",
    "Scanning GFX mappings...",
    "interface",
  );
  if (interfaceDir) {
    const interfaceFiles = await collectGfxAndGuiFiles(
      interfaceDir,
      "interface",
    );
    results.gfx.files = interfaceFiles.gfx;
    results.gui.files = interfaceFiles.gui;
    results.gui.fileCount = results.gui.files.length;
  }

  const technologyTagsDir = want("gfx")
    ? await tryGetDir(dirHandle, "common", "technology_tags")
    : null;
  if (technologyTagsDir) {
    await appendFiles(
      results.gui.files,
      await collectTxtFiles(technologyTagsDir, "common/technology_tags"),
    );
    results.gui.fileCount = results.gui.files.length;
  }

  const dlcDir = want("gfx") ? await tryGetDir(dirHandle, "dlc") : null;
  if (dlcDir) {
    for await (const entry of dlcDir.values()) {
      if (entry.kind === "directory") {
        const dlcInterfaceDir = await tryGetDir(entry, "interface");
        if (dlcInterfaceDir) {
          const dlcGfxFiles = await collectGfxFiles(
            dlcInterfaceDir,
            `dlc/${entry.name}/interface`,
          );
          await appendFiles(results.gfx.files, dlcGfxFiles, (file) => ({
            file: file.file,
            path: `interface/${file.file.name}`,
          }));
        }
      }
      const pendingYield = assemblyCheckpoint();
      if (pendingYield) await pendingYield;
    }
  }
  results.gfx.fileCount = results.gfx.files.length;

  const descriptor = want("descriptor")
    ? await readFileByPath(dirHandle, "descriptor.mod")
    : null;
  results.descriptor = descriptor
    ? { file: descriptor, path: "descriptor.mod" }
    : null;

  console.log("[scan] performScan results:", {
    tag: upperTag,
    characters: results.characters.fileCount,
    focus_trees: results.focus_trees.fileCount,
    technologies: results.technologies.fileCount,
    ideas: results.ideas.fileCount,
    events: results.events.fileCount,
    decisions: results.decisions.fileCount,
    bookmarks: results.bookmarks.fileCount,
    history_files: results.history_files.fileCount,
    states: results.states.fileCount,
    raids: results.raids.fileCount,
    units: results.units.fileCount,
    common_rules: results.common_rules.fileCount,
    common_rules_by_dir: results.common_rules.dirCounts,
    map: results.map.fileCount,
    flags: results.flags.fileCount,
    gfx: results.gfx.fileCount,
    gui: results.gui.fileCount,
    loc: results.loc.fileCount,
  });

  results.characters.enabled = results.characters.fileCount > 0;
  results.focus_trees.enabled = results.focus_trees.fileCount > 0;
  results.technologies.enabled = results.technologies.fileCount > 0;
  results.ideas.enabled = results.ideas.fileCount > 0;
  results.events.enabled = results.events.fileCount > 0;
  results.decisions.enabled = results.decisions.fileCount > 0;
  results.bookmarks.enabled = results.bookmarks.fileCount > 0;
  results.history_files.enabled = results.history_files.fileCount > 0;
  results.states.enabled = results.states.fileCount > 0;
  opts?.signal?.throwIfAborted();
  return results;
}

export function applyModeDefaults(
  raw: ScanResults,
  isLoadExistingMod: boolean,
  projectId: number,
): ScanResults {
  if (isLoadExistingMod) return raw;
  let lastSession: Record<string, boolean> = {};
  if (typeof window !== "undefined") {
    try {
      lastSession = JSON.parse(
        localStorage.getItem(`unified_import_last_selection_${projectId}`) ||
          "{}",
      );
    } catch {
      lastSession = {};
    }
  }
  const apply = (cat: ScanCategory, key: string): ScanCategory => ({
    ...cat,
    enabled: cat.fileCount > 0 && lastSession[key] === true,
  });
  return {
    ...raw,
    characters: apply(raw.characters, "characters"),
    focus_trees: apply(raw.focus_trees, "focus_trees"),
    technologies: apply(raw.technologies, "technologies"),
    ideas: apply(raw.ideas, "ideas"),
    events: apply(raw.events, "events"),
    decisions: apply(raw.decisions, "decisions"),
    bookmarks: apply(raw.bookmarks, "bookmarks"),
    history_files: apply(raw.history_files, "history_files"),
    states: apply(raw.states, "states"),
  };
}
