"use client";

import { useEffect, useMemo, useState } from "react";
import { Eye, ImageIcon } from "lucide-react";
import { useTranslations } from "use-intl";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { API_BASE } from "@/lib/api";
import { useLocaleStore } from "@/lib/store/localeStore";

interface FullModFileEntry {
  path: string;
  is_binary: boolean;
  size: number;
  content?: string;
  content_b64?: string;
}

interface FullModFilesResponse {
  mod_name: string;
  primary_tag: string;
  total_files: number;
  include_imported?: boolean;
  files: FullModFileEntry[];
}

interface FullModPreviewDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: number;
}

const LS_INCLUDE_IMPORTED_KEY = "fullModFiles.includeImported";

function readIncludeImportedDefault(): boolean {
  if (typeof window === "undefined") return true;
  const v = window.localStorage.getItem(LS_INCLUDE_IMPORTED_KEY);
  if (v === "false") return false;
  return true;
}

function persistIncludeImported(next: boolean): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(LS_INCLUDE_IMPORTED_KEY, next ? "true" : "false");
}

function _topLevelDir(path: string): string {
  const idx = path.indexOf("/");
  if (idx === -1) return path;

  const after = path.slice(idx + 1);
  const idx2 = after.indexOf("/");
  if (idx2 === -1) return after;
  return after.slice(0, idx2);
}

function _humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
}

export function FullModPreviewDialog({
  open,
  onOpenChange,
  projectId,
}: FullModPreviewDialogProps) {
  const t = useTranslations("fullModPreviewDialog");
  const [data, setData] = useState<FullModFilesResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);

  const [includeImported, setIncludeImported] = useState<boolean>(
    readIncludeImportedDefault,
  );

  useEffect(() => {
    if (!open) return;

    let cancelled = false;
    Promise.resolve().then(() => {
      if (cancelled) return;
      setLoading(true);
      setError(null);
      setSelectedPath(null);
    });
    fetch(
      `${API_BASE}/go/projects/${projectId}/full-mod-files?include_imported=${
        includeImported ? "true" : "false"
      }&locale=${useLocaleStore.getState().locale}`,
    )
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json() as Promise<FullModFilesResponse>;
      })
      .then((j) => {
        setData(j);

        const firstText = j.files.find((f) => !f.is_binary);
        if (firstText) setSelectedPath(firstText.path);
      })
      .catch((e: unknown) =>
        setError(e instanceof Error ? e.message : String(e)),
      )
      .finally(() => setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [open, projectId, includeImported]);

  const onToggleIncludeImported = (next: boolean) => {
    setIncludeImported(next);
    persistIncludeImported(next);
  };

  const grouped = useMemo(() => {
    if (!data) return new Map<string, FullModFileEntry[]>();
    const m = new Map<string, FullModFileEntry[]>();
    for (const f of data.files) {
      const k = _topLevelDir(f.path);
      const arr = m.get(k) ?? [];
      arr.push(f);
      m.set(k, arr);
    }
    return m;
  }, [data]);

  const groupedSorted = useMemo(
    () => Array.from(grouped.entries()).sort(([a], [b]) => a.localeCompare(b)),
    [grouped],
  );

  const selected = data?.files.find((f) => f.path === selectedPath) ?? null;
  const totalSize = useMemo(
    () => data?.files.reduce((s, f) => s + f.size, 0) ?? 0,
    [data],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent className="!max-w-5xl !w-[90vw] h-[80vh] flex flex-col p-0">
        <DialogHeader className="px-4 pt-4 shrink-0">
          <DialogTitle>
            <Eye className="inline h-4 w-4 mr-1.5 align-[-3px]" aria-hidden />
            Full Mod Preview · {data?.mod_name || ""}
          </DialogTitle>
          <DialogDescription>{t("description")}</DialogDescription>
          <label
            className="inline-flex items-center gap-1 text-xs text-muted-foreground cursor-pointer select-none mt-1"
            title={
              includeImported
                ? t("includeImportedTitleOn")
                : t("includeImportedTitleOff")
            }
          >
            <input
              type="checkbox"
              checked={includeImported}
              onChange={(e) => onToggleIncludeImported(e.target.checked)}
              disabled={loading}
              className="h-3 w-3 accent-[var(--color-accent)] cursor-pointer"
            />
            {t("includeImportedLabel")}
          </label>
        </DialogHeader>

        <div className="flex-1 min-h-0 flex border-t border-border">
          <div className="w-72 shrink-0 border-r border-border overflow-y-auto bg-muted/20">
            {loading && (
              <p className="text-sm text-muted-foreground p-3">
                {t("loading")}
              </p>
            )}
            {error && <p className="text-sm text-destructive p-3">{error}</p>}
            {data && (
              <div className="px-2 py-2 border-b border-border text-micro text-muted-foreground">
                {data.total_files} files · {grouped.size} dirs ·{" "}
                {_humanSize(totalSize)}
              </div>
            )}
            {data &&
              groupedSorted.map(([dir, files]) => (
                <div key={dir} className="border-b border-border/60">
                  <div className="px-2 py-1 text-micro uppercase font-mono tracking-wider text-muted-foreground bg-muted/40">
                    {dir} ({files.length})
                  </div>
                  {files.map((f) => (
                    <button
                      key={f.path}
                      type="button"
                      onClick={() => setSelectedPath(f.path)}
                      className={
                        "w-full text-left px-2 py-1 text-xs font-mono hover:bg-muted transition-colors truncate flex items-center gap-1 " +
                        (selectedPath === f.path
                          ? "bg-primary/10 text-primary font-semibold"
                          : "")
                      }
                      title={`${f.path} · ${_humanSize(f.size)}`}
                    >
                      {f.is_binary && (
                        <ImageIcon className="h-3.5 w-3.5" aria-hidden />
                      )}
                      <span className="truncate flex-1">
                        {f.path.split("/").pop()}
                      </span>
                      <span className="text-micro text-muted-foreground shrink-0">
                        {_humanSize(f.size)}
                      </span>
                    </button>
                  ))}
                </div>
              ))}
          </div>

          <div className="flex-1 min-w-0 flex flex-col">
            {selected ? (
              <>
                <div className="px-3 py-2 border-b border-border bg-muted/20 shrink-0 flex items-center gap-2">
                  <span className="text-xs font-mono truncate flex-1">
                    {selected.path}
                  </span>
                  <span className="text-micro text-muted-foreground">
                    {_humanSize(selected.size)}
                  </span>
                  {selected.is_binary && (
                    <span className="text-micro text-amber-700 dark:text-amber-300 px-1.5 py-0.5 rounded bg-amber-50 dark:bg-amber-950/30">
                      binary
                    </span>
                  )}
                </div>
                <div className="flex-1 min-h-0 overflow-auto">
                  {selected.is_binary ? (
                    <div className="p-4 text-sm text-muted-foreground italic">
                      {t("binaryNote", { size: _humanSize(selected.size) })}
                    </div>
                  ) : (
                    <pre className="p-3 text-xs font-mono whitespace-pre-wrap break-all">
                      {selected.content ?? ""}
                    </pre>
                  )}
                </div>
              </>
            ) : (
              <div className="flex-1 flex items-center justify-center text-muted-foreground text-sm">
                {t("selectFileHint")}
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export function FullModPreviewButton({
  projectId,
  compact = false,
  variant = "outline",
}: {
  projectId: number;
  compact?: boolean;

  variant?: "outline" | "ghost" | "secondary";
}) {
  const t = useTranslations("fullModPreviewDialog");
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        size={compact ? "sm" : "default"}
        variant={variant}
        onClick={() => {
          setOpen(true);
        }}
        title={t("buttonTitle")}
        className={
          compact ? "h-7 w-full justify-start gap-2.5 px-3 text-xs" : ""
        }
      >
        <Eye className="h-4 w-4 shrink-0" aria-hidden />
        {t("buttonLabel")}
      </Button>
      <FullModPreviewDialog
        open={open}
        onOpenChange={setOpen}
        projectId={projectId}
      />
    </>
  );
}
