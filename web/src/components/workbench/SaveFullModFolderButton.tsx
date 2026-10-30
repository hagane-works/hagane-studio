"use client";

import { useEffect, useRef, useState } from "react";
import { Save } from "lucide-react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { InfoTooltip, TooltipProvider } from "@/components/ui/tooltip";
import { API_BASE } from "@/lib/api";
import { useLocaleStore } from "@/lib/store/localeStore";

import { canWriteToFolder } from "@/lib/fsAccess";
import { BrowserCompatHint } from "@/components/workbench/BrowserCompatHint";

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

let cachedDirHandle: FileSystemDirectoryHandle | null = null;
let cachedDirName = "";

type FSAccessWindow = Window & {
  showDirectoryPicker?: (options?: {
    mode?: "read" | "readwrite";
  }) => Promise<FileSystemDirectoryHandle>;
};

async function getDirSubpath(
  root: FileSystemDirectoryHandle,
  path: string,
): Promise<{ dir: FileSystemDirectoryHandle; filename: string }> {
  const parts = path.split("/").filter(Boolean);
  const filename = parts.pop()!;
  let current = root;
  for (const part of parts) {
    if (part === "." || part === "..") {
      throw new Error(`__ILLEGAL_SEGMENT__:${part}`);
    }
    current = await current.getDirectoryHandle(part, { create: true });
  }
  return { dir: current, filename };
}

async function writeFileEntry(
  root: FileSystemDirectoryHandle,
  entry: FullModFileEntry,
): Promise<void> {
  const { dir, filename } = await getDirSubpath(root, entry.path);
  const handle = await dir.getFileHandle(filename, { create: true });
  const writable = await handle.createWritable();
  if (entry.is_binary && entry.content_b64) {
    const binary = atob(entry.content_b64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
    await writable.write(bytes);
  } else {
    const content = entry.content ?? "";

    if (entry.path.endsWith(".yml")) {
      await writable.write(new Uint8Array([0xef, 0xbb, 0xbf]));
    }
    await writable.write(content);
  }
  await writable.close();
}

interface SaveFullModFolderButtonProps {
  projectId: number;

  label?: string;

  compact?: boolean;

  variant?: "outline" | "ghost" | "secondary";
}

export function SaveFullModFolderButton({
  projectId,
  label,
  compact = false,
  variant = "outline",
}: SaveFullModFolderButtonProps) {
  const t = useTranslations("saveFullModFolderButton");
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<"" | "saved" | "partial" | "error">("");
  const [error, setError] = useState<string | null>(null);
  const [lastCount, setLastCount] = useState<number>(0);

  const [partial, setPartial] = useState<{
    ok: number;
    total: number;
    failed: number;
    firstError: string;
  } | null>(null);
  const [includeImported, setIncludeImported] = useState<boolean>(true);
  const dirHandleRef = useRef<FileSystemDirectoryHandle | null>(
    cachedDirHandle,
  );
  const dirNameRef = useRef<string>(cachedDirName);

  useEffect(() => {
    setIncludeImported(readIncludeImportedDefault());
  }, []);

  const onToggleIncludeImported = (next: boolean) => {
    setIncludeImported(next);
    persistIncludeImported(next);
  };

  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const fsAccessSupported = canWriteToFolder();

  const pickFolder = async () => {
    const picker = (window as FSAccessWindow).showDirectoryPicker;
    if (!picker) throw new Error("File System Access API unavailable");
    const handle = await picker({ mode: "readwrite" });
    dirHandleRef.current = handle;
    cachedDirHandle = handle;
    dirNameRef.current = handle.name;
    cachedDirName = handle.name;
  };

  const handleSave = async () => {
    if (!fsAccessSupported) {
      return;
    }
    setSaving(true);
    setError(null);
    setStatus("");
    setPartial(null);
    try {
      if (!dirHandleRef.current) await pickFolder();
      const dir = dirHandleRef.current;
      if (!dir) throw new Error("No mod folder selected");

      const r = await fetch(
        `${API_BASE}/go/projects/${projectId}/full-mod-files?include_imported=${
          includeImported ? "true" : "false"
        }&locale=${useLocaleStore.getState().locale}`,
      );
      if (!r.ok) throw new Error(`HTTP ${r.status}`);
      const body: FullModFilesResponse = await r.json();

      let ok = 0;
      const failures: string[] = [];
      for (const entry of body.files) {
        try {
          await writeFileEntry(dir, entry);
          ok += 1;
        } catch (fileErr) {
          if ((fileErr as Error).name === "AbortError") throw fileErr;
          const raw =
            fileErr instanceof Error ? fileErr.message : String(fileErr);
          const m = raw.match(/^__ILLEGAL_SEGMENT__:(.*)$/);
          const msg = m
            ? t("illegalPathSegment", { segment: m[1], path: entry.path })
            : `${entry.path}: ${raw}`;
          failures.push(msg);
        }
      }

      setLastCount(ok);
      if (failures.length === 0) {
        setStatus("saved");
        setTimeout(() => setStatus(""), 2500);
      } else {
        setPartial({
          ok,
          total: body.files.length,
          failed: failures.length,
          firstError: failures[0],
        });
        setStatus("partial");
      }
    } catch (e) {
      if ((e as Error).name === "AbortError") {
        setSaving(false);
        return;
      }
      setError(e instanceof Error ? e.message : String(e));
      setStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const changeFolder = async () => {
    setError(null);
    setStatus("");
    try {
      await pickFolder();
    } catch (e) {
      if ((e as Error).name === "AbortError") return;
      setError(e instanceof Error ? e.message : String(e));
      setStatus("error");
    }
  };

  const btnLabel =
    label ??
    (saving
      ? t("syncing")
      : dirHandleRef.current
        ? t("syncToNamed", { dir: dirNameRef.current })
        : t("syncToFolder"));

  if (mounted && !fsAccessSupported) {
    return <BrowserCompatHint mode="folderWrite" />;
  }

  return (
    <TooltipProvider>
      <div
        className={
          compact
            ? "flex w-full flex-col items-stretch gap-1"
            : "inline-flex items-center gap-2 flex-wrap"
        }
      >
        <Button
          size={compact ? "sm" : "default"}
          variant={variant}
          onClick={handleSave}
          disabled={saving}
          title={
            dirHandleRef.current
              ? t("titleWithFolder", { dir: dirNameRef.current })
              : t("titleNoFolder")
          }
          className={
            compact ? "h-7 w-full justify-start gap-2.5 px-3 text-xs" : ""
          }
        >
          <Save className="h-4 w-4 shrink-0" aria-hidden />
          {btnLabel}
        </Button>

        <div className="flex items-center gap-1.5 px-3">
          <label
            className="inline-flex items-center gap-1 text-xs text-muted-foreground cursor-pointer select-none"
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
              disabled={saving}
              className="h-3 w-3 accent-[var(--color-accent)] cursor-pointer"
            />
            {t("includeImportedLabel")}
          </label>
          <InfoTooltip
            title={t("tooltipTitle")}
            content={t.raw("tooltipContent")}
          />
        </div>
        {dirHandleRef.current && !saving && (
          <button
            type="button"
            onClick={() => void changeFolder()}
            className="px-3 text-left text-micro text-muted-foreground hover:text-foreground underline-offset-2 hover:underline ui-interactive"
            title={t("changeFolderTitle")}
          >
            {t("changeFolder")}
          </button>
        )}
        {status === "saved" && (
          <span className="px-3 text-xs text-success">
            {t("wroteFiles", { count: lastCount })}
          </span>
        )}
        {status === "partial" && partial && (
          <span
            className="px-3 text-xs text-warning truncate max-w-[420px]"
            title={partial.firstError}
          >
            {t("wrotePartial", {
              ok: partial.ok,
              total: partial.total,
              failed: partial.failed,
              firstError: partial.firstError,
            })}
          </span>
        )}
        {status === "error" && error && (
          <span
            className="px-3 text-xs text-destructive truncate max-w-[300px]"
            title={error}
          >
            {error}
          </span>
        )}

        <p className="basis-full inline-flex px-3 items-center gap-1 text-xs text-muted-foreground leading-relaxed">
          {t("additiveCaption")}
          <InfoTooltip
            title={t("additiveTitle")}
            content={t.raw("additiveWarning")}
          />
        </p>
      </div>
    </TooltipProvider>
  );
}
