"use client";

import { useState, useEffect, useMemo, useRef } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { Idea } from "@/types";
import { ideasApi } from "@/lib/api";
import {
  loadVanillaIdeas,
  isVanillaIdeasLoaded,
  searchVanillaIdeas,
  getVanillaIdeasForCountry,
  getVanillaIdeasByGroup,
} from "@/lib/vanillaIdeas";

type Tab =
  | "project"
  | "country"
  | "shared"
  | "laws"
  | "design"
  | "old_advisors"
  | "other";

const TABS: { value: Tab; labelKey: string }[] = [
  { value: "project", labelKey: "tabProject" },
  { value: "country", labelKey: "tabCountry" },
  { value: "shared", labelKey: "tabShared" },
  { value: "laws", labelKey: "tabLaws" },
  { value: "design", labelKey: "tabDesign" },
  { value: "old_advisors", labelKey: "tabOldAdvisors" },
  { value: "other", labelKey: "tabOther" },
];

interface IdeaReferenceDropdownProps {
  projectId: number;
  countryTag: string;

  effectType: "add_ideas" | "remove_ideas" | "swap_ideas" | "has_idea";
  onSelect: (code: string) => void;
  buttonLabel: string;
}

interface DisplayItem {
  id: string;
  source: "project" | "vanilla";
  group?: string;
  country?: string;
  label?: string;
}

export function IdeaReferenceDropdown({
  projectId,
  countryTag,
  effectType,
  onSelect,
  buttonLabel,
}: IdeaReferenceDropdownProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [activeTab, setActiveTab] = useState<Tab>("project");

  const [vReady, setVReady] = useState<boolean>(isVanillaIdeasLoaded);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const t = useTranslations("ideaReferenceDropdown");

  const { data: projectIdeas = [] } = useSWR<Idea[]>(
    open && projectId ? ["ideas/dropdown", projectId] : null,
    () => ideasApi.list(projectId),
    { keepPreviousData: true },
  );
  useEffect(() => {
    if (open && !vReady) {
      loadVanillaIdeas()
        .then(() => setVReady(true))
        .catch(() => {});
    }
  }, [open, vReady]);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const items = useMemo((): DisplayItem[] => {
    const q = search.toLowerCase().trim();

    if (q) {
      const results: DisplayItem[] = [];

      for (const pi of projectIdeas) {
        if (
          pi.idea_id.toLowerCase().includes(q) ||
          pi.name_en.toLowerCase().includes(q)
        ) {
          results.push({ id: pi.idea_id, source: "project" });
        }
      }

      const vanillaResults = vReady ? searchVanillaIdeas(q, undefined, 40) : [];
      for (const vr of vanillaResults) {
        if (!results.some((r) => r.id === vr.id)) {
          results.push({
            id: vr.id,
            source: "vanilla",
            group: vr.group,
            country: vr.country,
            label: vr.label,
          });
        }
      }

      return results;
    }

    if (!vReady && activeTab !== "project") return [];

    switch (activeTab) {
      case "project":
        return projectIdeas.map((pi) => ({
          id: pi.idea_id,
          source: "project" as const,
        }));

      case "country": {
        const vanilla = getVanillaIdeasForCountry(countryTag);
        return vanilla.map((v) => ({
          id: v.id,
          source: "vanilla" as const,
          group: v.group,
          label: v.label,
        }));
      }

      case "shared": {
        const shared = getVanillaIdeasForCountry("shared").filter(
          (i) => i.group === "country_spirits" || i.group === "hidden_spirits",
        );
        return shared.map((v) => ({
          id: v.id,
          source: "vanilla" as const,
          group: v.group,
        }));
      }

      case "laws":
        return getVanillaIdeasByGroup("laws").map((v) => ({
          id: v.id,
          source: "vanilla" as const,
          group: "laws",
        }));

      case "design":
        return getVanillaIdeasByGroup("design_companies")
          .slice(0, 100)
          .map((v) => ({
            id: v.id,
            source: "vanilla" as const,
            group: "design_companies",
            country: v.country,
          }));

      case "old_advisors":
        return getVanillaIdeasByGroup("old_advisors")
          .slice(0, 100)
          .map((v) => ({
            id: v.id,
            source: "vanilla" as const,
            group: "old_advisors",
            country: v.country,
            label: `${v.id} (old advisor)`,
          }));

      case "other":
        return [];

      default:
        return [];
    }
  }, [search, activeTab, projectIdeas, countryTag, vReady]);

  const vanillaPending =
    !vReady && (search.length > 0 || activeTab !== "project");

  const handleSelect = (ideaId: string) => {
    const code = `${effectType} = ${ideaId}`;
    onSelect(code);
    setOpen(false);
    setSearch("");
  };

  return (
    <div className="relative inline-block" ref={dropdownRef}>
      <Button
        size="sm"
        variant="outline"
        className="text-xs h-6 px-2"
        onClick={() => setOpen(!open)}
      >
        {buttonLabel} ▾
      </Button>

      {open && (
        <div className="absolute z-50 top-7 left-0 w-80 bg-popover border rounded-md shadow-lg">
          <div className="p-2 border-b">
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("searchPlaceholder")}
              className="text-xs h-7"
              autoFocus
            />
          </div>

          {!search && (
            <div className="flex flex-wrap gap-0.5 p-1.5 border-b">
              {TABS.map((tab) => (
                <button
                  key={tab.value}
                  className={`px-2 py-0.5 rounded text-xs transition-colors ${
                    activeTab === tab.value
                      ? "bg-primary text-primary-foreground"
                      : "bg-muted hover:bg-muted/80"
                  }`}
                  onClick={() => setActiveTab(tab.value)}
                >
                  {tab.value === "country" ? countryTag : t(tab.labelKey)}
                </button>
              ))}
            </div>
          )}

          <div className="max-h-60 overflow-y-auto">
            {activeTab === "other" && !search ? (
              <p className="p-3 text-xs text-muted-foreground">
                {t("otherHint")}
              </p>
            ) : vanillaPending ? (
              <p className="p-3 text-xs text-muted-foreground">
                {t("loadingVanilla")}
              </p>
            ) : items.length === 0 ? (
              <p className="p-3 text-xs text-muted-foreground">
                {search ? t("noResults") : t("emptyCategory")}
              </p>
            ) : (
              items.map((item) => (
                <button
                  key={`${item.source}-${item.id}`}
                  className="w-full text-left px-3 py-1.5 text-xs hover:bg-muted transition-colors flex items-center justify-between"
                  onClick={() => handleSelect(item.id)}
                >
                  <span className="font-mono truncate">{item.id}</span>
                  <span className="text-muted-foreground ml-2 shrink-0">
                    {item.source === "project" ? (
                      <span className="text-success">{t("projectBadge")}</span>
                    ) : item.country ? (
                      item.country
                    ) : item.group ? (
                      item.group.replace("_", " ")
                    ) : null}
                  </span>
                </button>
              ))
            )}
          </div>

          <div className="border-t p-2">
            <button
              className="w-full text-left px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
              onClick={() => {
                const name = prompt(t("promptManual"));
                if (name) handleSelect(name.trim());
              }}
            >
              {t("enterManually")}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
