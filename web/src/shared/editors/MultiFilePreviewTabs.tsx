"use client";

import { ReactNode, useEffect, useState } from "react";
import { useTranslations } from "use-intl";
import { charactersApi } from "@/lib/api";
import { useProjectStore } from "@/lib/store/projectStore";
import {
  isContentLocale,
  type ContentLocale,
} from "@/lib/store/contentLocaleStore";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

type TabKey = "script" | `loc_${ContentLocale}` | "gfx";

interface PreviewFile {
  path: string;
  content: string;
}

interface MultiFilePreviewTabsProps {
  scriptEditor: ReactNode;

  projectId: number;
  characterId: number | null;
}

export function MultiFilePreviewTabs({
  scriptEditor,
  projectId,
  characterId,
}: MultiFilePreviewTabsProps) {
  const t = useTranslations("multiFilePreviewTabs");
  const [activeTab, setActiveTab] = useState<TabKey>("script");
  const [files, setFiles] = useState<Record<string, PreviewFile> | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const exportLanguages = useProjectStore(
    (state) => state.currentProject?.export_languages,
  );

  useEffect(() => {
    let cancelled = false;
    Promise.resolve().then(() => {
      if (cancelled) return;
      setFiles(null);
      setError(null);
    });
    return () => {
      cancelled = true;
    };
  }, [characterId]);

  useEffect(() => {
    if (activeTab === "script") return;
    if (characterId === null) return;
    if (files !== null) return;
    let cancelled = false;

    Promise.resolve().then(() => {
      if (cancelled) return;
      setLoading(true);
      setError(null);
    });
    charactersApi
      .previewFiles(projectId, characterId)
      .then((res) => {
        if (cancelled) return;
        setFiles(res.files);
      })
      .catch((e) => {
        if (cancelled) return;
        setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (cancelled) return;
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [activeTab, characterId, projectId, files]);

  const contentLanguages: ContentLocale[] = ["en"];
  for (const candidate of exportLanguages ?? []) {
    if (
      isContentLocale(candidate) &&
      candidate !== "en" &&
      !contentLanguages.includes(candidate)
    ) {
      contentLanguages.push(candidate);
    }
  }
  const tabs: TabKey[] = [
    "script",
    ...contentLanguages.map((lang): TabKey => `loc_${lang}`),
    "gfx",
  ];

  return (
    <TooltipProvider>
      <div className="flex flex-col h-full">
        <div className="flex items-center border-b border-border/40 bg-card">
          {tabs.map((tab) => (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              className={`px-3 py-1.5 text-xs whitespace-nowrap ui-interactive ${
                activeTab === tab
                  ? "bg-background text-foreground font-medium border-b-2 border-accent -mb-px"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted/40"
              }`}
              disabled={tab !== "script" && characterId === null}
            >
              {tab === "script"
                ? "Script"
                : tab === "gfx"
                  ? "GFX"
                  : `Loc ${tab.slice(4).toUpperCase()}`}
            </button>
          ))}
        </div>

        {activeTab !== "script" && files?.[activeTab] && (
          <div className="px-3 py-1 bg-muted/20 border-b border-border/40 shrink-0">
            <Tooltip>
              <TooltipTrigger
                render={
                  <span className="text-xs text-muted-foreground font-mono truncate block cursor-help">
                    {files[activeTab].path.split("/").pop()}
                  </span>
                }
              />
              <TooltipContent side="bottom" align="start">
                <span className="font-mono text-xs">
                  {files[activeTab].path}
                </span>
              </TooltipContent>
            </Tooltip>
          </div>
        )}

        <div className="flex-1 min-h-0 overflow-hidden">
          {activeTab === "script" ? (
            scriptEditor
          ) : (
            <div className="h-full overflow-auto p-3 bg-background">
              {loading && (
                <p className="text-xs text-muted-foreground">{t("loading")}</p>
              )}
              {error && (
                <p className="text-xs text-destructive">
                  {t("previewFailed", { error })}
                </p>
              )}
              {!loading && !error && files?.[activeTab] && (
                <pre className="text-xs font-mono whitespace-pre-wrap leading-relaxed">
                  {files[activeTab].content}
                </pre>
              )}
              {!loading && !error && !files && characterId !== null && (
                <p className="text-xs text-muted-foreground">
                  {t("noContentYet")}
                </p>
              )}
              {characterId === null && (
                <p className="text-xs text-muted-foreground">
                  {t("selectCharacterHint")}
                </p>
              )}
            </div>
          )}
        </div>
      </div>
    </TooltipProvider>
  );
}
