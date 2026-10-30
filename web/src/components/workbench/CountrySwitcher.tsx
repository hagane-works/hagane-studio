"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { ChevronDown, ChevronRight, Network, Trash2 } from "lucide-react";
import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import {
  useProjectStore,
  useActiveCountry,
  useProjectCountries,
} from "@/lib/store/projectStore";
import { AddCountryDialog } from "@/components/workbench/AddCountryDialog";
import { CountryRelationsSheet } from "@/components/workbench/CountryRelationsSheet";
import {
  DeleteCountryDialog,
  type DeleteCountryTarget,
} from "@/components/workbench/DeleteCountryDialog";
import {
  CountryFlag,
  isResolvedCountryName,
} from "@/components/workbench/CountryFlag";
import { useVanillaTagName } from "@/lib/vanillaCountryTags";
import { countriesApi, type CountryClassificationItem } from "@/lib/api";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";

interface CountrySwitcherProps {
  projectId: number;
}

const RECENT_LIMIT = 5;
const recentKey = (projectId: number) =>
  `hoi4-hagane-recent-countries-${projectId}`;

const BADGE_KEYS = [
  "characters",
  "ideas",
  "events",
  "decisions",
  "focuses",
  "custom_traits",
  "event_namespaces",
  "decision_categories",
  "technologies",
] as const;

const BADGE_LABEL_KEY: Record<string, string> = {
  characters: "badgeCharacters",
  ideas: "badgeIdeas",
  events: "badgeEvents",
  decisions: "badgeDecisions",
  focuses: "badgeFocuses",
  custom_traits: "badgeCustomTraits",
  event_namespaces: "badgeEventNamespaces",
  decision_categories: "badgeDecisionCategories",
  technologies: "badgeTechnologies",
};

const BADGE_GLYPH: Record<string, string> = {
  characters: "C",
  ideas: "I",
  events: "E",
  decisions: "D",
  focuses: "F",
  custom_traits: "T",
  event_namespaces: "N",
  decision_categories: "K",

  technologies: "R",
};

function loadRecent(projectId: number): string[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(recentKey(projectId));
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter((x): x is string => typeof x === "string")
      .slice(0, RECENT_LIMIT);
  } catch {
    return [];
  }
}

function saveRecent(projectId: number, tag: string) {
  if (typeof window === "undefined") return;
  try {
    const cur = loadRecent(projectId).filter((t) => t !== tag);
    cur.unshift(tag);
    const trimmed = cur.slice(0, RECENT_LIMIT);
    window.localStorage.setItem(recentKey(projectId), JSON.stringify(trimmed));
  } catch {}
}

const EMPTY_ITEMS: CountryClassificationItem[] = [];

export function CountrySwitcher({ projectId }: CountrySwitcherProps) {
  const countries = useProjectCountries();
  const activeCountry = useActiveCountry();
  const setActiveCountryTag = useProjectStore((s) => s.setActiveCountryTag);
  const [addOpen, setAddOpen] = useState(false);
  const [open, setOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [refExpanded, setRefExpanded] = useState(false);

  const [relationsTag, setRelationsTag] = useState<string | null>(null);
  const [relationsOpen, setRelationsOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<DeleteCountryTarget | null>(
    null,
  );

  const [recent, setRecent] = useState<string[]>(() => loadRecent(projectId));
  const t = useTranslations("countrySwitcher");
  const vanillaName = useVanillaTagName();

  const { data: grouped, mutate } = useSWR(
    projectId ? ["countries/grouped", projectId] : null,
    () => countriesApi.grouped(projectId),
  );

  const [lastProjectId, setLastProjectId] = useState(projectId);
  if (lastProjectId !== projectId) {
    setLastProjectId(projectId);
    setRecent(loadRecent(projectId));
    setRefExpanded(false);
  }

  const handleOpenChange = useCallback(
    (next: boolean) => {
      setOpen(next);
      if (next) {
        setRecent(loadRecent(projectId));
        void mutate();
      }
    },
    [projectId, mutate],
  );

  const handlePaletteChange = useCallback(
    (next: boolean) => {
      setPaletteOpen(next);
      if (next) {
        setRecent(loadRecent(projectId));
        void mutate();
      }
    },
    [projectId, mutate],
  );

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key.toLowerCase() === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        handlePaletteChange(!paletteOpen);
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [paletteOpen, handlePaletteChange]);

  useEffect(() => {
    const onOpen = () => handlePaletteChange(true);
    window.addEventListener("hoi4-open-country-palette", onOpen);
    return () =>
      window.removeEventListener("hoi4-open-country-palette", onOpen);
  }, [handlePaletteChange]);

  const nameOf = useCallback(
    (item: CountryClassificationItem): string => {
      const vanilla = vanillaName(item.tag);
      if (vanilla) return vanilla;
      return isResolvedCountryName(item.name, item.tag) ? item.name : item.tag;
    },
    [vanillaName],
  );

  const authored = grouped?.authored ?? EMPTY_ITEMS;
  const reference = grouped?.reference ?? EMPTY_ITEMS;

  const byTag = useMemo(() => {
    const m = new Map<string, CountryClassificationItem>();
    for (const it of grouped?.authored ?? EMPTY_ITEMS) m.set(it.tag, it);
    for (const it of grouped?.reference ?? EMPTY_ITEMS) m.set(it.tag, it);
    return m;
  }, [grouped]);

  const recentItems = useMemo(() => {
    const seen = new Set<string>();
    const out: CountryClassificationItem[] = [];
    for (const tag of recent) {
      if (seen.has(tag)) continue;
      const it = byTag.get(tag);
      if (!it) continue;
      seen.add(tag);
      out.push(it);
    }
    return out;
  }, [recent, byTag]);

  const pick = useCallback(
    (tag: string) => {
      if (tag !== activeCountry && !confirmDiscardIfDirty()) return;
      setActiveCountryTag(tag);
      saveRecent(projectId, tag);
      setRecent(loadRecent(projectId));
      setOpen(false);
      setPaletteOpen(false);
    },
    [activeCountry, projectId, setActiveCountryTag],
  );

  const activeItem = activeCountry ? byTag.get(activeCountry) : undefined;

  if (countries.length <= 1) return null;

  const flagBlock = (item: CountryClassificationItem) => (
    <CountryFlag color={item.color} className="w-3 h-3" />
  );

  const badgeCluster = (item: CountryClassificationItem) => {
    const present = BADGE_KEYS.filter((k) => (item.counts?.[k] ?? 0) > 0);
    if (present.length === 0) {
      return (
        <span className="text-micro text-muted-foreground italic">
          {t("noContent")}
        </span>
      );
    }
    return (
      <span className="flex items-center gap-1 flex-wrap">
        {present.map((k) => (
          <span
            key={k}
            title={`${t(BADGE_LABEL_KEY[k])}: ${item.counts[k]}`}
            className="inline-flex items-center gap-0.5 px-1 rounded bg-muted text-micro font-mono text-muted-foreground leading-tight"
          >
            <span aria-hidden>{BADGE_GLYPH[k]}</span>
            <span>{item.counts[k]}</span>
          </span>
        ))}
      </span>
    );
  };

  const popoverRow = (item: CountryClassificationItem, dim: boolean) => {
    const display = nameOf(item);
    const isActive = item.tag === activeCountry;
    return (
      <button
        key={item.tag}
        type="button"
        onClick={() => pick(item.tag)}
        className={`w-full text-left px-2 py-1.5 rounded text-xs hover:bg-muted flex items-center gap-2 ${
          isActive ? "bg-muted font-semibold" : ""
        } ${dim ? "opacity-60 hover:opacity-100" : ""}`}
        title={display !== item.tag ? display : undefined}
      >
        {flagBlock(item)}
        <span className="font-mono font-semibold shrink-0">{item.tag}</span>
        {display !== item.tag && (
          <span className="text-muted-foreground truncate">{display}</span>
        )}
        {isActive && (
          <span className="ml-auto text-micro text-muted-foreground shrink-0">
            {t("active")}
          </span>
        )}
      </button>
    );
  };

  const popoverSection = (
    label: string,
    items: CountryClassificationItem[],
    icon: string,
    opts?: { dim?: boolean; withBadges?: boolean },
  ) => {
    if (items.length === 0) return null;
    return (
      <div className="mb-1.5 last:mb-0">
        <div className="sticky top-0 z-10 bg-popover px-1 py-1 text-micro font-medium text-muted-foreground flex items-center gap-1">
          <span>{icon}</span>
          <span>{label}</span>
          <span className="text-muted-foreground/60">({items.length})</span>
        </div>
        {items.map((item) =>
          opts?.withBadges ? (
            <div
              key={item.tag}
              className={`group flex items-stretch rounded text-xs hover:bg-muted ${
                item.tag === activeCountry ? "bg-muted" : ""
              }`}
            >
              <button
                type="button"
                onClick={() => pick(item.tag)}
                className="flex-1 min-w-0 text-left px-2 py-1.5 rounded-l flex flex-col gap-1"
                title={nameOf(item) !== item.tag ? nameOf(item) : undefined}
              >
                <span className="flex items-center gap-2">
                  {flagBlock(item)}
                  <span className="font-mono font-semibold shrink-0">
                    {item.tag}
                  </span>
                  {nameOf(item) !== item.tag && (
                    <span className="text-muted-foreground truncate">
                      {nameOf(item)}
                    </span>
                  )}
                  {item.tag === activeCountry && (
                    <span className="ml-auto text-micro text-muted-foreground shrink-0">
                      {t("active")}
                    </span>
                  )}
                </span>
                <span className="pl-5">{badgeCluster(item)}</span>
              </button>
              <button
                type="button"
                onClick={() => {
                  setRelationsTag(item.tag);
                  setRelationsOpen(true);
                  setOpen(false);
                }}
                className="shrink-0 px-2 flex items-center text-muted-foreground/60 hover:text-foreground"
                title={t("relationsEntry")}
                aria-label={t("relationsEntry")}
              >
                <Network className="w-3.5 h-3.5" aria-hidden />
              </button>
              <button
                type="button"
                onClick={() => {
                  setDeleteTarget({
                    id: item.id,
                    tag: item.tag,
                    name: nameOf(item),
                  });
                  setOpen(false);
                }}
                className="shrink-0 px-2 flex items-center text-muted-foreground/60 hover:text-destructive rounded-r ui-interactive"
                title={t("deleteCountry.entry", { country: item.tag })}
                aria-label={t("deleteCountry.entry", { country: item.tag })}
              >
                <Trash2 className="w-3.5 h-3.5" aria-hidden />
              </button>
            </div>
          ) : (
            popoverRow(item, opts?.dim ?? false)
          ),
        )}
      </div>
    );
  };

  const empty = authored.length === 0 && reference.length === 0;

  const paletteItem = (item: CountryClassificationItem, dim: boolean) => {
    const display = nameOf(item);
    return (
      <CommandItem
        key={item.tag}

        value={`${item.tag} ${display} ${item.name_en} ${item.name_zh}`.trim()}
        onSelect={() => pick(item.tag)}
        className={dim ? "opacity-70 data-selected:opacity-100" : ""}
      >
        {flagBlock(item)}
        <span className="font-mono font-semibold">{item.tag}</span>
        {display !== item.tag && (
          <span className="text-muted-foreground truncate">{display}</span>
        )}
        {item.total_content > 0 && (
          <span className="ml-auto text-micro text-muted-foreground font-mono">
            {t("contentItems", { count: item.total_content })}
          </span>
        )}
        {item.tag === activeCountry && (
          <span className="ml-2 text-micro text-muted-foreground shrink-0">
            {t("active")}
          </span>
        )}
      </CommandItem>
    );
  };

  return (
    <>
      <Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger
          render={(props) => (
            <Button
              {...props}
              variant="outline"
              className="w-full justify-between text-left font-normal text-xs h-8"
            >
              <span className="flex items-center gap-1.5 min-w-0">
                {activeItem && flagBlock(activeItem)}
                <span className="font-mono font-semibold truncate">
                  {activeCountry || "—"}
                </span>
              </span>
              <span className="text-muted-foreground text-micro shrink-0">
                {t("countriesCount", { count: countries.length })}
              </span>
            </Button>
          )}
        />
        <PopoverContent align="start" className="w-72 p-1">
          <button
            type="button"
            onClick={() => {
              setOpen(false);
              handlePaletteChange(true);
            }}
            className="w-full text-left px-2 py-1.5 mb-1 rounded text-xs text-muted-foreground hover:bg-muted hover:text-foreground flex items-center justify-between gap-2"
          >
            <span>{t("openPalette")}</span>
            <kbd className="text-micro font-mono px-1 py-0.5 rounded border border-border bg-muted">
              ⌘K
            </kbd>
          </button>
          <div className="max-h-80 overflow-y-auto">
            {empty ? (
              <div className="px-2 py-3 text-xs text-muted-foreground text-center">
                {t("noMatch")}
              </div>
            ) : (
              <>
                {popoverSection(t("recent"), recentItems, "★")}
                {popoverSection(t("authored"), authored, "◆", {
                  withBadges: true,
                })}
                {reference.length > 0 && (
                  <div className="mb-1.5 last:mb-0">
                    <button
                      type="button"
                      onClick={() => setRefExpanded((v) => !v)}
                      className="w-full px-1 py-1 rounded text-micro font-medium text-muted-foreground hover:bg-muted/60 flex items-center gap-1"
                      aria-expanded={refExpanded}
                      title={t("referenceHint")}
                    >
                      {refExpanded ? (
                        <ChevronDown className="w-3 h-3" aria-hidden />
                      ) : (
                        <ChevronRight className="w-3 h-3" aria-hidden />
                      )}
                      <span>{t("reference")}</span>
                      <span className="text-muted-foreground/60">
                        ({reference.length})
                      </span>
                      {!refExpanded && (
                        <span className="ml-auto text-micro text-muted-foreground normal-case">
                          {t("showReference", { count: reference.length })}
                        </span>
                      )}
                    </button>
                    {refExpanded &&
                      reference.map((item) => popoverRow(item, true))}
                  </div>
                )}
              </>
            )}
          </div>
          <div className="border-t border-border my-1" />
          <button
            type="button"
            onClick={() => {
              setOpen(false);
              setAddOpen(true);
            }}
            className="w-full text-left px-2 py-1.5 rounded text-xs text-muted-foreground hover:bg-muted hover:text-foreground"
          >
            {t("addCountry")}
          </button>
        </PopoverContent>
      </Popover>

      <CommandDialog
        open={paletteOpen}
        onOpenChange={handlePaletteChange}
        title={t("paletteTitle")}
        description={t("paletteDescription")}
        className="max-w-lg"
      >
        <Command>
          <CommandInput placeholder={t("searchPlaceholder")} />
          <CommandList>
            <CommandEmpty>{t("noMatch")}</CommandEmpty>
            {recentItems.length > 0 && (
              <CommandGroup heading={t("recent")}>
                {recentItems.map((item) => paletteItem(item, false))}
              </CommandGroup>
            )}
            {authored.length > 0 && (
              <CommandGroup heading={t("authored")}>
                {authored.map((item) => paletteItem(item, false))}
              </CommandGroup>
            )}
            {reference.length > 0 && (
              <>
                <CommandSeparator />
                <CommandGroup
                  heading={`${t("reference")} (${reference.length})`}
                >
                  {refExpanded ? (
                    reference.map((item) => paletteItem(item, true))
                  ) : (
                    <CommandItem
                      value="__show_reference__"
                      onSelect={() => setRefExpanded(true)}
                      className="text-muted-foreground italic"
                    >
                      {t("showReference", { count: reference.length })}
                    </CommandItem>
                  )}
                </CommandGroup>
              </>
            )}
            <CommandSeparator />
            <CommandGroup>
              <CommandItem
                value="__add_country__"
                onSelect={() => {
                  setPaletteOpen(false);
                  setAddOpen(true);
                }}
                className="text-muted-foreground"
              >
                {t("addCountry")}
              </CommandItem>
            </CommandGroup>
          </CommandList>
        </Command>
      </CommandDialog>

      <AddCountryDialog
        projectId={projectId}
        open={addOpen}
        onOpenChange={setAddOpen}
      />

      <CountryRelationsSheet
        projectId={projectId}
        tag={relationsTag}
        open={relationsOpen}
        onOpenChange={setRelationsOpen}
      />

      <DeleteCountryDialog
        projectId={projectId}
        country={deleteTarget}
        open={deleteTarget !== null}
        onOpenChange={(next) => {
          if (!next) setDeleteTarget(null);
        }}
        onDeleted={() => void mutate()}
      />
    </>
  );
}
