import { gameViewHref, gameViewWindowName } from "@/lib/gameViewLinks";
import type { Dispatch, SetStateAction } from "react";
import { useLocale, useTranslations } from "use-intl";
import { useRouter } from "@/i18n/navigation";
import type { ScanAndParseResult } from "@/lib/api";
import { formatTriple, renderImportWarning } from "./format";
import type { WarnTranslator, Phase } from "./types";
import { CrossRefBanner } from "../CrossRefBanner";

export interface ImportSuccessCardProps {
  phase: Phase;
  reImporting: boolean;
  result: ScanAndParseResult | null;
  t: ReturnType<typeof useTranslations>;
  tag: string;
  countryTag: string;
  router: ReturnType<typeof useRouter>;
  projectId: number;
  setOpen: Dispatch<SetStateAction<boolean>>;
  imageStatus: string;
  whatsNextDetailOpen: boolean;
  setWhatsNextDetailOpen: Dispatch<SetStateAction<boolean>>;
  imageExtSwaps: number;
  projectCountries: string[];
  handleAddCountryAndReImport: (newTag: string) => Promise<void>;
  scanStatus: string;
}

export function ImportSuccessCard({
  phase,
  reImporting,
  result,
  t,
  tag,
  countryTag,
  router,
  projectId,
  setOpen,
  imageStatus,
  whatsNextDetailOpen,
  setWhatsNextDetailOpen,
  imageExtSwaps,
  projectCountries,
  handleAddCountryAndReImport,
  scanStatus,
}: ImportSuccessCardProps) {
  const locale = useLocale();
  return (
    <>
      {(phase === "done" || reImporting) &&
        result &&
        (() => {
          const summary = [
            {
              key: "characters",
              label: t("labelCharacters"),
              count:
                (result.characters?.created ?? 0) +
                (result.characters?.updated ?? 0),
            },
            {
              key: "focus_trees",
              label: t("labelFocusTrees"),
              count:
                (result.focus_trees?.created_trees ?? 0) +
                (result.focus_trees?.updated_trees ?? 0),
            },
            {
              key: "technologies",
              label: t("labelTechnologies"),
              count:
                (result.technologies?.created ?? 0) +
                (result.technologies?.updated ?? 0),
            },
            {
              key: "ideas",
              label: t("labelIdeas"),
              count:
                (result.ideas?.created ?? 0) + (result.ideas?.updated ?? 0),
            },
            {
              key: "events",
              label: t("labelEvents"),
              count:
                (result.events?.created_events ?? 0) +
                (result.events?.updated_events ?? 0),
            },
            {
              key: "decisions",
              label: t("labelDecisions"),
              count:
                (result.decisions?.created_decisions ?? 0) +
                (result.decisions?.updated_decisions ?? 0),
            },
          ].filter((item) => item.count > 0);
          const totalImpacted = summary.reduce((n, item) => n + item.count, 0);

          const handleOpenGameView = () => {
            setOpen(false);
            window.open(
              `/${locale}${gameViewHref(projectId, { tag: tag || countryTag, panel: "countrypoliticsview" })}`,
              gameViewWindowName(projectId),
            );
          };
          const handleOpenDashboard = () => {
            setOpen(false);

            router.push(`/project/${projectId}/characters`);
          };
          const aggregatedWarnings = [
            ...(result.warnings ?? []),
            ...(result.characters?.warnings ?? []),
            ...(result.focus_trees?.warnings ?? []),
            ...(result.technologies?.warnings ?? []),
            ...(result.ideas?.warnings ?? []),
            ...(result.events?.warnings || []),
            ...(result.decisions?.warnings || []),
          ];

          const hasNotes =
            aggregatedWarnings.length > 0 || !!result.cross_ref_report;

          const triWords = {
            created: t("triCreated"),
            updated: t("triUpdated"),
            skipped: t("triSkipped"),
          };
          return (
            <div className="rounded-lg border border-emerald-500/50 bg-emerald-500/5 p-4 space-y-3">
              <div className="flex items-center justify-between gap-2">
                <p className="text-base font-semibold text-emerald-600 dark:text-emerald-400">
                  {t("whatsNextTitle")}
                </p>
                <span className="text-xs text-muted-foreground">
                  policy: {result.conflict_policy}
                </span>
              </div>

              {summary.length === 0 ? (
                <p className="text-xs text-muted-foreground">
                  {t("whatsNextSummaryEmpty")}
                </p>
              ) : (
                <div className="text-sm">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground mb-1">
                    {t("whatsNextSummaryHeader")}
                  </p>
                  <p className="text-foreground">
                    {summary.map((item, idx) => (
                      <span key={item.key}>
                        {idx > 0 && (
                          <span className="text-muted-foreground"> · </span>
                        )}
                        <span className="font-semibold">{item.count}</span>
                        <span className="text-muted-foreground">
                          {" "}
                          {item.label}
                        </span>
                      </span>
                    ))}
                  </p>
                  {imageStatus && (
                    <p className="text-xs text-muted-foreground mt-1">
                      {imageStatus}
                    </p>
                  )}
                </div>
              )}

              {hasNotes && (
                <p className="text-xs text-muted-foreground">
                  {t("whatsNextReassure")}
                </p>
              )}

              {totalImpacted > 0 && (
                <div>
                  <button
                    type="button"
                    onClick={() => setWhatsNextDetailOpen((v) => !v)}
                    className="text-xs text-muted-foreground hover:text-foreground underline-offset-2 hover:underline"
                  >
                    {whatsNextDetailOpen
                      ? `▼ ${t("whatsNextDetailHide")}`
                      : `▶ ${t("whatsNextDetailToggle")}`}
                  </button>
                  {whatsNextDetailOpen && (
                    <div className="mt-2 space-y-1 text-xs border-l-2 border-emerald-500/30 pl-3">
                      {result.characters.created +
                        result.characters.updated +
                        result.characters.skipped >
                        0 && (
                        <p>
                          {t("itemCharacters")}:{" "}
                          {formatTriple(
                            result.characters.created,
                            result.characters.updated,
                            result.characters.skipped,
                            "",
                            triWords,
                          )}
                        </p>
                      )}
                      {result.focus_trees.created_trees +
                        result.focus_trees.updated_trees +
                        result.focus_trees.skipped_trees >
                        0 && (
                        <p>
                          {t("itemFocusTrees")}:{" "}
                          {formatTriple(
                            result.focus_trees.created_trees,
                            result.focus_trees.updated_trees,
                            result.focus_trees.skipped_trees,
                            " " + t("unitTrees"),
                            triWords,
                          )}{" "}
                          ·{" "}
                          {t("nodesCount", {
                            count: result.focus_trees.created_nodes,
                          })}
                        </p>
                      )}
                      {result.technologies.created +
                        result.technologies.updated +
                        result.technologies.skipped >
                        0 && (
                        <p>
                          {t("itemTechnologies")}:{" "}
                          {formatTriple(
                            result.technologies.created,
                            result.technologies.updated,
                            result.technologies.skipped,
                            "",
                            triWords,
                          )}
                        </p>
                      )}
                      {result.ideas.created +
                        result.ideas.updated +
                        result.ideas.skipped >
                        0 && (
                        <p>
                          {t("itemIdeas")}:{" "}
                          {formatTriple(
                            result.ideas.created,
                            result.ideas.updated,
                            result.ideas.skipped,
                            "",
                            triWords,
                          )}
                          {(result.ideas.character_slot_count ?? 0) > 0 && (
                            <span className="text-amber-500 ml-2">
                              · {result.ideas.character_slot_count}{" "}
                              {t("characterSlotNote")}
                            </span>
                          )}
                        </p>
                      )}
                      {result.events?.created_events +
                        result.events?.updated_events +
                        result.events?.skipped_events >
                        0 && (
                        <p>
                          {t("itemEvents")}:{" "}
                          {formatTriple(
                            result.events.created_events,
                            result.events.updated_events,
                            result.events.skipped_events,
                            " " + t("unitEvents"),
                            triWords,
                          )}{" "}
                          ·{" "}
                          {t("namespacesCount", {
                            count:
                              result.events.created_namespaces +
                              (result.events.reused_namespaces ?? 0),
                          })}
                          {(result.events.reused_namespaces ?? 0) > 0
                            ? ` ${t("namespacesNewReused", { new: result.events.created_namespaces, reused: result.events.reused_namespaces ?? 0 })}`
                            : ""}
                        </p>
                      )}
                      {result.decisions?.created_decisions +
                        result.decisions?.updated_decisions +
                        result.decisions?.skipped_decisions >
                        0 && (
                        <p>
                          {t("itemDecisions")}:{" "}
                          {formatTriple(
                            result.decisions.created_decisions,
                            result.decisions.updated_decisions,
                            result.decisions.skipped_decisions,
                            " " + t("unitDecisions"),
                            triWords,
                          )}{" "}
                          ·{" "}
                          {t("categoriesCount", {
                            count:
                              result.decisions.created_categories +
                              (result.decisions.updated_categories ?? 0),
                          })}
                          {(result.decisions.updated_categories ?? 0) > 0
                            ? ` ${t("categoriesNewUpdated", { new: result.decisions.created_categories, updated: result.decisions.updated_categories ?? 0 })}`
                            : ""}
                        </p>
                      )}

                      {(result.needed_images?.length ?? 0) > 0 && (
                        <p className="text-muted-foreground">
                          {t("backendRequestedImages", {
                            images: result.needed_images?.length ?? 0,
                            mappings: result.gfx_mappings_count,
                          })}
                        </p>
                      )}
                    </div>
                  )}
                </div>
              )}

              <div className="space-y-2 pt-1">
                <button
                  type="button"
                  onClick={handleOpenDashboard}
                  disabled={reImporting}
                  className="w-full text-left rounded-md border border-border bg-muted/30 hover:bg-muted/50 px-3 py-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium">
                        {t("whatsNextDashboardTitle")}
                      </p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {t("whatsNextDashboardDesc")}
                      </p>
                    </div>
                    <span className="text-xs text-muted-foreground shrink-0">
                      {t("whatsNextDashboardButton")} →
                    </span>
                  </div>
                </button>

                <button
                  type="button"
                  onClick={handleOpenGameView}
                  disabled={reImporting}
                  className="w-full text-left rounded-md border border-border bg-muted/30 hover:bg-muted/50 px-3 py-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium">
                        {t("whatsNextGameViewTitle")}
                      </p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {t("whatsNextGameViewDesc")}
                      </p>
                    </div>
                    <span className="text-xs text-muted-foreground shrink-0">
                      {t("whatsNextGameViewButton")} →
                    </span>
                  </div>
                </button>

                <button
                  type="button"
                  onClick={() => setOpen(false)}
                  disabled={reImporting}
                  className="w-full text-center text-xs text-muted-foreground hover:text-foreground py-1.5 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t("whatsNextDismiss")}
                </button>
              </div>

              {aggregatedWarnings.length > 0 && (
                <details className="text-xs border-t border-emerald-500/20 pt-2">
                  <summary className="cursor-pointer select-none text-muted-foreground hover:text-foreground">
                    {t("importNotesToggle", {
                      count: aggregatedWarnings.length,
                    })}
                  </summary>
                  <div className="mt-1.5 space-y-1 text-muted-foreground max-h-32 overflow-y-auto pl-2">
                    {aggregatedWarnings.map((w, i) => (
                      <p key={i}>
                        {renderImportWarning(w, t as unknown as WarnTranslator)}
                      </p>
                    ))}
                  </div>
                </details>
              )}

              {imageExtSwaps > 0 && (
                <div className="text-muted-foreground text-xs border-t border-emerald-500/20 pt-2">
                  <p>{t("assetExtSwapNote", { count: imageExtSwaps })}</p>
                </div>
              )}

              {result.cross_ref_report && (
                <div className="border-t border-emerald-500/20 pt-2">
                  <CrossRefBanner
                    report={result.cross_ref_report}
                    existingCountryTags={projectCountries}
                    onAddCountryAndReImport={handleAddCountryAndReImport}
                    disabled={reImporting}
                  />
                  {reImporting && (
                    <p className="mt-1 text-xs text-muted-foreground text-center">
                      {scanStatus || t("reImporting")}
                    </p>
                  )}
                </div>
              )}
            </div>
          );
        })()}
    </>
  );
}
