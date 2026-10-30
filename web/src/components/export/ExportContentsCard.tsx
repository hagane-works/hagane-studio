"use client";

import { Package } from "lucide-react";
import { useTranslations } from "use-intl";

export interface ExportSummary {
  characters: number;
  focus_trees: number;
  focus_nodes: number;
  technologies: number;
  ideas: number;
  event_namespaces?: number;
  events?: number;
  decision_categories?: number;
  decisions?: number;
  custom_traits?: number;
  opinion_modifiers?: number;
  dynamic_modifiers?: number;
  super_events?: number;
  bookmarks?: number;
  history_files?: number;

  raw_files?: number;
}

export interface AssetAudit {
  idea_pictures_missing: number;
  event_pictures_missing: number;
  decision_icons_missing: number;
  tech_icons_missing: number;
  focus_icons_missing: number;
  portraits_missing: number;
  total_missing: number;
}

export function summaryHasContent(summary: ExportSummary | null): boolean {
  return (
    !!summary &&
    Object.values(summary).some((v) => typeof v === "number" && v > 0)
  );
}

function formatSize(bytes: number): string {
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function ExportContentsCard({
  pending,
  summary,
  summaryUnavailable,
  assetAudit,
  builtFileCount,
  builtSizeBytes,
  identityDescKey,
  legacyMapNote,
}: {
  pending: "idle" | "running" | null;
  summary: ExportSummary | null;

  summaryUnavailable: boolean;
  assetAudit: AssetAudit | null;

  builtFileCount: number | null;
  builtSizeBytes: number | null;

  identityDescKey: string;

  legacyMapNote: string | null;
}) {
  const t = useTranslations("export");

  const totalEntities = summary
    ? Object.values(summary).reduce(
        (acc, v) => acc + (typeof v === "number" && v > 0 ? v : 0),
        0,
      )
    : 0;
  const hasContent = summaryHasContent(summary);

  const rows: { label: string; value: string }[] = [];
  if (summary) {
    if (summary.focus_trees > 0) {
      rows.push({
        label: t("summaryFocusTrees"),
        value: t("ui.contents.rowFocusValue", {
          trees: summary.focus_trees,
          nodes: summary.focus_nodes,
        }),
      });
    }
    const simple: { key: keyof ExportSummary; label: string }[] = [
      { key: "characters", label: t("summaryCharacters") },
      { key: "events", label: t("summaryEvents") },
      { key: "ideas", label: t("summaryIdeas") },
      { key: "technologies", label: t("summaryTechnologies") },
      { key: "decisions", label: t("ui.contents.rowDecisions") },
    ];
    for (const row of simple) {
      const v = summary[row.key];
      if (typeof v === "number" && v > 0) {
        rows.push({ label: row.label, value: String(v) });
      }
    }
    const counted = new Set([
      "focus_trees",
      "focus_nodes",
      "characters",
      "events",
      "ideas",
      "technologies",
      "decisions",
    ]);
    const other = Object.entries(summary).reduce(
      (acc, [key, v]) =>
        acc + (!counted.has(key) && typeof v === "number" && v > 0 ? v : 0),
      0,
    );
    if (other > 0) {
      rows.push({ label: t("ui.contents.rowOther"), value: String(other) });
    }
  }

  return (
    <aside
      aria-labelledby="export-contents-title"
      className="flex flex-col rounded-lg border border-border bg-card p-4"
    >
      <h3
        id="export-contents-title"
        className="flex items-center justify-between text-sm font-semibold text-foreground"
      >
        {t("ui.contents.title")}
        <Package className="h-4 w-4 text-muted-foreground" aria-hidden />
      </h3>

      {pending ? (
        <p className="mt-3 text-xs text-muted-foreground">
          {pending === "running"
            ? t("ui.stepper.step1Running")
            : t("ui.contents.notChecked")}
        </p>
      ) : summaryUnavailable ? (
        <p className="mt-3 text-xs text-muted-foreground">
          {t("ui.contents.unavailable")}
        </p>
      ) : !hasContent ? (
        <p className="mt-3 text-xs text-muted-foreground">
          {t("ui.contents.empty")}
        </p>
      ) : (
        <>
          <p className="mt-3 text-2xl font-semibold tracking-tight text-foreground">
            {builtFileCount !== null
              ? t("ui.contents.filesHeadline", { count: builtFileCount })
              : t("ui.contents.entitiesHeadline", { count: totalEntities })}
            {builtSizeBytes !== null ? (
              <span className="ml-1.5 text-xs font-medium text-muted-foreground">
                {formatSize(builtSizeBytes)}
              </span>
            ) : null}
          </p>
          <p className="mt-1.5 text-xs text-muted-foreground">
            {t(identityDescKey)}
          </p>
          {rows.length > 0 ? (
            <ul className="mt-3 grid gap-1.5 border-t border-border pt-3">
              {rows.map((row) => (
                <li
                  key={row.label}
                  className="flex items-baseline justify-between gap-3 text-xs"
                >
                  <span className="text-muted-foreground">{row.label}</span>
                  <span className="font-semibold text-foreground">
                    {row.value}
                  </span>
                </li>
              ))}
            </ul>
          ) : null}
        </>
      )}

      <div className="mt-auto space-y-2 pt-3">
        {assetAudit && assetAudit.total_missing > 0 ? (
          <p className="rounded-md bg-muted px-3 py-2 text-xs text-muted-foreground">
            {t("ui.contents.assetNote", { count: assetAudit.total_missing })}
          </p>
        ) : null}
        {legacyMapNote ? (
          <p className="rounded-md bg-muted px-3 py-2 text-xs text-muted-foreground">
            {legacyMapNote}
          </p>
        ) : null}
      </div>
    </aside>
  );
}
