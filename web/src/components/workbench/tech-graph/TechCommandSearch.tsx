"use client";

import { useMemo, useState } from "react";
import { ListPanelHint } from "../ListPanelHint";
import { useTranslations } from "use-intl";
import type { Technology } from "@/types";
import { searchTechs } from "@/lib/techGraph";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import type { TechLineOption } from "./LedgerTabs";
import { Input } from "@/components/ui/input";

interface TechCommandSearchProps {
  techs: Technology[];
  techLines: TechLineOption[];
  onSelectTech: (techDbId: number, techLine: string) => void;
}

export function TechCommandSearch({
  techs,
  techLines,
  onSelectTech,
}: TechCommandSearchProps) {
  const t = useTranslations("techCommandSearch");
  const locale = useContentLocaleStore((s) => s.locale);
  const [query, setQuery] = useState("");

  const lineLabel = useMemo(() => {
    const map: Record<string, string> = {};
    for (const l of techLines) map[l.value] = l.label;
    return map;
  }, [techLines]);

  const results = useMemo(() => {
    if (!query.trim()) return [];
    return searchTechs(techs, query).slice(0, 50);
  }, [techs, query]);

  const totalCount = techs.length;

  return (
    <div className="p-3 flex flex-col h-full min-h-0">
      <div className="mb-2 shrink-0">
        <h3 className="text-sm font-semibold mb-2">{t("heading")}</h3>
        <Input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t("placeholder")}
          className="py-2 bg-muted/30 focus:ring-2 focus:ring-primary/40 placeholder:text-muted-foreground"
          aria-label={t("inputAriaLabel")}
        />
      </div>

      {query.trim() === "" ? (
        <ListPanelHint className="mt-4 shrink-0 space-y-2">
          <p className="font-semibold text-foreground">{t("paletteTitle")}</p>
          <p className="leading-relaxed">{t("paletteHint")}</p>
          <p className="border-t border-border pt-2">{t("navHint")}</p>
          <p className="pt-1 font-mono text-micro">
            {t("projectTotal", {
              techCount: totalCount,
              lineCount: techLines.length,
            })}
          </p>
        </ListPanelHint>
      ) : (
        <div className="flex-1 overflow-y-auto space-y-0.5 mt-1 min-h-0">
          {results.length === 0 ? (
            <p className="text-xs text-muted-foreground p-2">
              {t("noMatches", { query })}
            </p>
          ) : (
            <>
              <p className="text-micro text-muted-foreground px-2 pb-1 font-mono shrink-0">
                {results.length === 50
                  ? t("firstMatches", { count: 50 })
                  : t("matches", { count: results.length })}
              </p>
              {results.map((tech) => (
                <button
                  key={tech.id}
                  onClick={() => onSelectTech(tech.id, tech.tech_line)}
                  className="w-full text-left px-2 py-1.5 rounded hover:bg-muted/60 text-xs transition-colors group"
                >
                  <div className="font-medium truncate">
                    {resolveDisplayName(
                      locale,
                      tech.display_name_locs,
                      tech.name_en,
                      tech.name_zh,
                      tech.name || tech.tech_id,
                    )}
                  </div>
                  <div className="text-muted-foreground text-micro truncate font-mono">
                    {tech.tech_id} ·{" "}
                    <span>{lineLabel[tech.tech_line] ?? tech.tech_line}</span>
                  </div>
                </button>
              ))}
            </>
          )}
        </div>
      )}
    </div>
  );
}
