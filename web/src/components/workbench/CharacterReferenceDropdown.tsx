"use client";

import { useState, useEffect, useMemo, useRef } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { Character, BaseModRef } from "@/types";
import { charactersApi, basemodIndexApi } from "@/lib/api";
import { visibleCharacterName } from "@/lib/i18n/characterDisplayName";

interface CharacterReferenceDropdownProps {
  projectId: number;

  effectType: string;
  onSelect: (code: string) => void;
  buttonLabel: string;
}

interface DisplayItem {
  id: string;
  source: "project" | "base";

  displayName: string;

  dependencyName?: string;
}

const BASE_SEARCH_LIMIT = 25;

export function CharacterReferenceDropdown({
  projectId,
  effectType,
  onSelect,
  buttonLabel,
}: CharacterReferenceDropdownProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");

  const [debouncedSearch, setDebouncedSearch] = useState("");
  const dropdownRef = useRef<HTMLDivElement>(null);
  const t = useTranslations("characterReferenceDropdown");

  const { data: projectChars = [] } = useSWR<Character[]>(
    open && projectId ? ["characters/dropdown", projectId] : null,
    () => charactersApi.list(projectId),
    { keepPreviousData: true },
  );

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search.trim()), 250);
    return () => clearTimeout(timer);
  }, [search]);

  const { data: baseResult, isLoading: baseLoading } = useSWR(
    open && projectId
      ? ["basemod-refs/character", projectId, debouncedSearch]
      : null,
    () =>
      basemodIndexApi.searchRefs(
        projectId,
        "character",
        debouncedSearch,
        BASE_SEARCH_LIMIT,
      ),
    { keepPreviousData: true },
  );

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
    const results: DisplayItem[] = [];
    const baseRefs: BaseModRef[] = baseResult?.results ?? [];

    for (const c of projectChars) {
      const name =
        visibleCharacterName(c.display_name_locs?.en) ||
        visibleCharacterName(c.name_en) ||
        visibleCharacterName(c.name) ||
        c.tag;
      if (
        !q ||
        c.tag.toLowerCase().includes(q) ||
        name.toLowerCase().includes(q)
      ) {
        results.push({ id: c.tag, source: "project", displayName: name });
      }
    }

    for (const ref of baseRefs) {
      if (results.some((r) => r.id === ref.id)) continue;
      results.push({
        id: ref.id,
        source: "base",
        displayName: ref.display_name,
        dependencyName: ref.dependency_name,
      });
    }

    return results;
  }, [search, projectChars, baseResult]);

  const handleSelect = (id: string) => {
    onSelect(`${effectType} = ${id}`);
    setOpen(false);
    setSearch("");
  };

  const basePending = open && baseLoading && !baseResult;

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

          <div className="max-h-60 overflow-y-auto">
            {items.length === 0 ? (
              <p className="p-3 text-xs text-muted-foreground">
                {basePending
                  ? t("searchingBase")
                  : search
                    ? t("noResults")
                    : t("empty")}
              </p>
            ) : (
              <>
                {items.map((item) => (
                  <button
                    key={`${item.source}-${item.id}`}
                    className="w-full text-left px-3 py-1.5 text-xs hover:bg-muted transition-colors flex items-center justify-between gap-2"
                    onClick={() => handleSelect(item.id)}
                  >
                    <span className="flex flex-col min-w-0">
                      <span className="font-mono truncate">{item.id}</span>
                      {item.displayName && item.displayName !== item.id && (
                        <span className="text-muted-foreground truncate">
                          {item.displayName}
                        </span>
                      )}
                    </span>
                    <span className="ml-2 shrink-0 text-right">
                      {item.source === "project" ? (
                        <span className="text-success">
                          {t("projectBadge")}
                        </span>
                      ) : (
                        <span className="text-muted-foreground">
                          {item.dependencyName || t("baseBadge")}
                        </span>
                      )}
                    </span>
                  </button>
                ))}
                {basePending && (
                  <p className="p-2 text-xs text-muted-foreground">
                    {t("searchingBase")}
                  </p>
                )}
              </>
            )}
          </div>

          <div className="border-t p-2">
            <button
              className="w-full text-left px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
              onClick={() => {
                const id = prompt(t("promptManual"));
                if (id) handleSelect(id.trim());
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
