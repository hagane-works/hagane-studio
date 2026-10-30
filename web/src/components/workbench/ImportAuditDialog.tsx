"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { TriangleAlert } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { API_BASE } from "@/lib/api";
import { useProjectStore } from "@/lib/store/projectStore";

interface EntityPreview {
  characters: number;
  focus_files: number;
  events: number;
  ideas: number;
  decisions: number;
  history: number;
}

interface DetectedTag {
  tag: string;
  file_path: string;
  is_dynamic: boolean;
  already_exists: boolean;
  entity_preview: EntityPreview;

  name_preview?: { en: string | null; zh: string | null };
  color_preview?: number[] | null;
}

function rgbCss(color: number[] | null | undefined): string | null {
  if (!color || color.length < 3) return null;
  const [r, g, b] = color;
  return `rgb(${r}, ${g}, ${b})`;
}

function displayName(d: DetectedTag): string {
  return d.name_preview?.en || d.name_preview?.zh || d.tag;
}

interface AuditSummary {
  declared: number;
  new: number;
  existing: number;
  dynamic: number;
  static: number;
}

interface WarningItem {
  code: string;
  params: Record<string, string | number>;
}

const PARSER_RECOVERY_CODES = new Set([
  "parser_recovery_stray_rbrace",
  "parser_recovery_empty_value",
  "parser_recovery_unclosed_block",
]);

interface AuditResponse {
  country_tags_files: string[];
  summary: AuditSummary;
  detected: DetectedTag[];
  warnings: string[];

  warning_items?: WarningItem[];
  descriptor_name?: string | null;
}

interface ImportAuditDialogProps {
  projectId: number;
  open: boolean;

  files: File[];
  paths: string[];

  onConfirmed: (createdTags: string[]) => void;

  onCancel: () => void;
}

export function ImportAuditDialog({
  projectId,
  open,
  files,
  paths,
  onConfirmed,
  onCancel,
}: ImportAuditDialogProps) {
  const t = useTranslations("import.auditDialog");
  const fetchProject = useProjectStore((s) => s.fetchProject);

  const [audit, setAudit] = useState<AuditResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState("");
  const [creating, setCreating] = useState(false);
  const [createProgress, setCreateProgress] = useState(0);

  useEffect(() => {
    if (!open) {
      setAudit(null);
      setError(null);
      setSelected(new Set());
      setSearch("");
      setCreateProgress(0);
    }
  }, [open]);

  useEffect(() => {
    if (!open || audit) return;
    let cancelled = false;
    const run = async () => {
      setLoading(true);
      setError(null);
      try {
        const formData = new FormData();
        files.forEach((f) => formData.append("files", f));
        paths.forEach((p) => formData.append("paths", p));
        const res = await fetch(
          `/api/go/projects/${projectId}/import/audit-tags`,
          { method: "POST", body: formData },
        );
        if (!res.ok) {
          const body = await res.json().catch(() => ({}));
          throw new Error(body.detail || `Audit failed: HTTP ${res.status}`);
        }
        const data: AuditResponse = await res.json();
        if (cancelled) return;
        setAudit(data);

        const defaults = new Set<string>();
        for (const d of data.detected) {
          if (!d.is_dynamic && !d.already_exists) {
            defaults.add(d.tag);
          }
        }
        setSelected(defaults);
      } catch (e) {
        if (!cancelled) setError((e as Error).message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    void run();
    return () => {
      cancelled = true;
    };
  }, [open, audit, files, paths, projectId]);

  const filtered = useMemo(() => {
    if (!audit) return [] as DetectedTag[];
    const q = search.trim().toUpperCase();
    if (!q) return audit.detected;
    return audit.detected.filter(
      (d) => d.tag.includes(q) || d.file_path.toUpperCase().includes(q),
    );
  }, [audit, search]);

  const warningGroups = useMemo(() => {
    const informational: number[] = [];
    const parserRecovery: number[] = [];
    for (let i = 0; i < (audit?.warnings.length ?? 0); i++) {
      const code = audit?.warning_items?.[i]?.code;
      (code && PARSER_RECOVERY_CODES.has(code)
        ? parserRecovery
        : informational
      ).push(i);
    }
    return { informational, parserRecovery };
  }, [audit]);

  const selectableInView = useMemo(
    () => filtered.filter((d) => !d.already_exists).map((d) => d.tag),
    [filtered],
  );

  const allInViewSelected = useMemo(
    () =>
      selectableInView.length > 0 &&
      selectableInView.every((t) => selected.has(t)),
    [selectableInView, selected],
  );

  const toggleAllInView = useCallback(() => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (allInViewSelected) {
        for (const t of selectableInView) next.delete(t);
      } else {
        for (const t of selectableInView) next.add(t);
      }
      return next;
    });
  }, [allInViewSelected, selectableInView]);

  const toggleOne = useCallback((tag: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(tag)) next.delete(tag);
      else next.add(tag);
      return next;
    });
  }, []);

  const selectStaticOnly = useCallback(() => {
    if (!audit) return;
    const next = new Set<string>();
    for (const d of audit.detected) {
      if (!d.is_dynamic && !d.already_exists) next.add(d.tag);
    }
    setSelected(next);
  }, [audit]);

  const selectAll = useCallback(() => {
    if (!audit) return;
    const next = new Set<string>();
    for (const d of audit.detected) {
      if (!d.already_exists) next.add(d.tag);
    }
    setSelected(next);
  }, [audit]);

  const selectNone = useCallback(() => {
    setSelected(new Set());
  }, []);

  const handleConfirm = async () => {
    if (!audit) return;
    setCreating(true);
    setError(null);
    setCreateProgress(0);
    const toCreate = audit.detected.filter((d) => selected.has(d.tag));
    const created: string[] = [];
    try {
      for (let i = 0; i < toCreate.length; i++) {
        const d = toCreate[i];

        const namePref = d.name_preview;
        const displayedName = namePref?.en || namePref?.zh || d.tag;
        const res = await fetch(
          `${API_BASE}/go/projects/${projectId}/countries`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              tag: d.tag,
              name: displayedName,
              name_en: namePref?.en || "",
              name_zh: namePref?.zh || "",
              color: d.color_preview || [],
              ideology: "neutrality",
            }),
          },
        );
        if (res.ok) {
          created.push(d.tag);
        } else if (res.status !== 409) {
          const body = await res.json().catch(() => ({}));
          throw new Error(
            `Failed to create ${d.tag}: ${body.detail || res.statusText}`,
          );
        } else {
          created.push(d.tag);
        }
        setCreateProgress(i + 1);
      }
      await fetchProject(projectId);
      onConfirmed(created);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setCreating(false);
    }
  };

  const handleSkip = () => {
    onConfirmed([]);
  };

  const renderWarning = useCallback(
    (i: number): string => {
      const item = audit?.warning_items?.[i];
      const raw = audit?.warnings?.[i] ?? "";
      if (!item) return raw;
      switch (item.code) {
        case "unrouted_dir":
          return t("warningUnroutedDir", {
            count: Number(item.params.count ?? 0),
            dir: String(item.params.dir ?? ""),
          });
        case "read_failed":
          return t("warningReadFailed", {
            path: String(item.params.path ?? ""),
            error: String(item.params.error ?? ""),
          });
        case "decode_failed":
          return t("warningDecodeFailed", {
            path: String(item.params.path ?? ""),
            error: String(item.params.error ?? ""),
          });
        case "parser_recovery_stray_rbrace":
          return t("warningParserRecoveryStrayBrace", {
            path: String(item.params.path ?? ""),
            line: Number(item.params.line ?? 0),
          });
        case "parser_recovery_empty_value":
          return t("warningParserRecoveryEmptyValue", {
            path: String(item.params.path ?? ""),
            line: Number(item.params.line ?? 0),
            key: String(item.params.key ?? ""),
            terminator: String(item.params.terminator ?? ""),
          });
        case "parser_recovery_unclosed_block":
          return t("warningParserRecoveryUnclosedBlock", {
            path: String(item.params.path ?? ""),
            line: Number(item.params.line ?? 0),
            eofLine: Number(item.params.eof_line ?? 0),
          });
        default:
          return raw;
      }
    },
    [audit, t],
  );

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && !creating) onCancel();
      }}
      disablePointerDismissal
    >
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
        </DialogHeader>

        {loading && (
          <div className="py-8 text-center text-sm text-muted-foreground">
            {t("scanning")}
          </div>
        )}

        {error && (
          <div className="rounded border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {t("genericError", { message: error })}
          </div>
        )}

        {audit && (
          <div className="space-y-3">
            <div className="text-sm text-muted-foreground">
              {t.rich("summary", {
                declared: audit.summary.declared,
                staticCount: audit.summary.static,
                dynamicCount: audit.summary.dynamic,
                existing: audit.summary.existing,
                code: (chunks) => <code>{chunks}</code>,
                strong: (chunks) => <strong>{chunks}</strong>,
                tagCode: (chunks) => <code>&lt;{chunks}&gt;_*</code>,
              })}
            </div>

            {warningGroups.informational.length > 0 && (
              <div className="rounded border border-border/60 bg-muted/40 px-3 py-2 text-sm text-muted-foreground">
                <p className="font-medium text-foreground/80">
                  {t("warningsHeading", {
                    n: warningGroups.informational.length,
                  })}
                </p>
                <ul className="mt-1 space-y-1">
                  {warningGroups.informational.map((i) => (
                    <li key={i} className="flex items-start gap-1.5">
                      <span aria-hidden className="select-none">
                        ℹ️
                      </span>
                      <span>{renderWarning(i)}</span>
                    </li>
                  ))}
                </ul>
                <p className="mt-1.5 text-xs text-muted-foreground">
                  {t("warningsNoAction")}
                </p>
              </div>
            )}

            {warningGroups.parserRecovery.length > 0 && (
              <div
                role="alert"
                className="rounded border border-warning/40 bg-warning/10 px-3 py-2 text-sm"
              >
                <div className="flex items-start gap-2">
                  <TriangleAlert
                    aria-hidden
                    className="mt-0.5 size-4 shrink-0 text-warning"
                  />
                  <div className="min-w-0">
                    <p className="font-medium text-foreground">
                      {t("parserRecoveryHeading", {
                        n: warningGroups.parserRecovery.length,
                      })}
                    </p>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {t("parserRecoveryHelp")}
                    </p>
                    <ul className="mt-2 list-disc space-y-1 pl-4 text-foreground/90">
                      {warningGroups.parserRecovery.map((i) => (
                        <li key={i}>{renderWarning(i)}</li>
                      ))}
                    </ul>
                  </div>
                </div>
              </div>
            )}

            <div className="flex flex-wrap items-center gap-2">
              <Input
                placeholder={t("searchPlaceholder")}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="max-w-xs"
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={selectStaticOnly}
              >
                {t("selectStaticOnly")}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={selectAll}
              >
                {t("selectAllIncludingDynamic")}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={selectNone}
              >
                {t("clearSelection")}
              </Button>
              <span className="ml-auto text-sm">
                {t.rich("selectedCountLabel", {
                  n: selected.size,
                  total: audit.summary.declared,
                  strong: (chunks) => <strong>{chunks}</strong>,
                })}
              </span>
            </div>

            <ScrollArea className="h-[400px] rounded border">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-muted/60">
                  <tr className="text-left">
                    <th className="w-10 px-2 py-2">
                      <input
                        type="checkbox"
                        checked={allInViewSelected}
                        onChange={toggleAllInView}
                        aria-label={t("selectAllInView")}
                      />
                    </th>

                    <th
                      className="w-6 px-1 py-2"
                      aria-label={t("colorColumn")}
                    />
                    <th className="px-2 py-2">TAG</th>
                    <th className="px-2 py-2">{t("nameColumn")}</th>
                    <th className="px-2 py-2">{t("statusColumn")}</th>
                    <th className="px-2 py-2 text-right">
                      {t("entityPreviewColumn")}
                    </th>
                    <th className="px-2 py-2">{t("sourceFileColumn")}</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((d) => {
                    const ep = d.entity_preview;
                    const epTotal =
                      ep.characters +
                      ep.focus_files +
                      ep.events +
                      ep.ideas +
                      ep.decisions +
                      ep.history;
                    const isSel = selected.has(d.tag);
                    const swatch = rgbCss(d.color_preview);
                    const displayed = displayName(d);
                    const fallback = displayed === d.tag;
                    return (
                      <tr
                        key={d.tag}
                        className={
                          d.already_exists
                            ? "border-t opacity-50"
                            : "border-t hover:bg-muted/30"
                        }
                      >
                        <td className="px-2 py-1.5">
                          <input
                            type="checkbox"
                            checked={isSel}
                            disabled={d.already_exists}
                            onChange={() => toggleOne(d.tag)}
                            aria-label={t("selectTag", { tag: d.tag })}
                          />
                        </td>
                        <td className="px-1 py-1.5">
                          {swatch ? (
                            <span
                              className="inline-block h-4 w-4 rounded-sm border border-border/60"
                              style={{ backgroundColor: swatch }}
                              title={`RGB ${d.color_preview!.join(", ")}`}
                            />
                          ) : (
                            <span
                              className="inline-block h-4 w-4 rounded-sm border border-dashed border-border/40"
                              title={t("noColorTooltip")}
                            />
                          )}
                        </td>
                        <td className="px-2 py-1.5 font-mono">{d.tag}</td>
                        <td className="px-2 py-1.5">
                          {fallback ? (
                            <span className="text-muted-foreground italic">
                              {d.tag}
                            </span>
                          ) : (
                            <span title={d.name_preview?.zh || undefined}>
                              {displayed}
                            </span>
                          )}
                        </td>
                        <td className="px-2 py-1.5 space-x-1">
                          {d.already_exists && (
                            <Badge variant="secondary">
                              {t("existsBadge")}
                            </Badge>
                          )}
                          {d.is_dynamic && !d.already_exists && (
                            <Badge variant="outline">{t("dynamicBadge")}</Badge>
                          )}
                          {!d.is_dynamic && !d.already_exists && (
                            <Badge>{t("staticBadge")}</Badge>
                          )}
                        </td>
                        <td className="px-2 py-1.5 text-right tabular-nums">
                          {epTotal === 0 ? (
                            <span className="text-muted-foreground">—</span>
                          ) : (
                            <span
                              title={`chars ${ep.characters} · focus ${ep.focus_files} · events ${ep.events} · ideas ${ep.ideas} · decisions ${ep.decisions} · history ${ep.history}`}
                            >
                              {epTotal}
                            </span>
                          )}
                        </td>
                        <td className="px-2 py-1.5 truncate text-xs text-muted-foreground">
                          {d.file_path}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </ScrollArea>

            {creating && (
              <div className="text-sm text-muted-foreground">
                {t("creatingProgress", {
                  n: createProgress,
                  total: selected.size,
                })}
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <Button
                type="button"
                variant="outline"
                onClick={handleSkip}
                disabled={creating}
              >
                {t("skipDirectImport")}
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={onCancel}
                disabled={creating}
              >
                {t("cancelEntireImport")}
              </Button>
              <Button
                type="button"
                onClick={handleConfirm}
                disabled={creating || selected.size === 0}
              >
                {creating
                  ? t("creatingButton", {
                      n: createProgress,
                      total: selected.size,
                    })
                  : t("createAndContinue", { n: selected.size })}
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
