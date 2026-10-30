"use client";

import { readProjectEditRevisions } from "../save-diagnostics/revisionRequest";
import { editRevisionErrorCode } from "../save-diagnostics/editRevisionError";
import type { EditRevisions } from "../save-diagnostics/editRevisions";
import { useState, useEffect, useMemo, useCallback, useRef } from "react";
import { useTranslations } from "use-intl";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { API_BASE, focusIconsApi, type FocusIconLibraryEntry } from "@/lib/api";

const SERVER_ORIGIN = API_BASE.replace(/\/api\/?$/, "");

const FAVORITES_LS_KEY_PREFIX = "focus_icon_favorites:v1:";
const FAVORITES_MAX = 10;

const GFX_NAME_RE = /^[A-Za-z0-9_]+$/;

type LibraryTab = "all" | "vanilla" | "project" | "favorites";

interface FocusIconLibraryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: number;

  currentIcon?: string;

  onSelect: (gfxName: string) => void;

  title?: string;
  description?: string;
  uploadDescription?: string;
  uploadNamePlaceholder?: string;

  projectOnly?: boolean;
}

function _loadFavorites(projectId: number): string[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(
      FAVORITES_LS_KEY_PREFIX + projectId,
    );
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed)
      ? parsed
          .filter((v): v is string => typeof v === "string")
          .slice(0, FAVORITES_MAX)
      : [];
  } catch {
    return [];
  }
}

function _saveFavorites(projectId: number, list: string[]): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(
      FAVORITES_LS_KEY_PREFIX + projectId,
      JSON.stringify(list.slice(0, FAVORITES_MAX)),
    );
  } catch {}
}

export function FocusIconLibraryDialog({
  open,
  onOpenChange,
  projectId,
  currentIcon,
  onSelect,
  title,
  description,
  uploadDescription,
  uploadNamePlaceholder,
  projectOnly = false,
}: FocusIconLibraryDialogProps) {
  const t = useTranslations("focusIconLibraryDialog");
  const [tab, setTab] = useState<LibraryTab>("all");
  const [searchInput, setSearchInput] = useState("");
  const [debouncedQ, setDebouncedQ] = useState("");

  const [fetchedIcons, setFetchedIcons] = useState<FocusIconLibraryEntry[]>([]);
  const [fetchedTotal, setFetchedTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(currentIcon ?? null);
  const [favorites, setFavorites] = useState<string[]>([]);

  const uploadRevisions = useRef<EditRevisions>({});
  const [uploadOpen, setUploadOpen] = useState(false);
  const [uploadName, setUploadName] = useState("");
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    let active = true;
    uploadRevisions.current = {};
    if (open)
      void readProjectEditRevisions(projectId)
        .then((value) => {
          if (active) uploadRevisions.current = value;
        })
        .catch((cause: unknown) => {
          if (active)
            setUploadError(
              cause instanceof Error ? cause.message : String(cause),
            );
        });
    return () => {
      active = false;
    };
  }, [open, projectId]);

  useEffect(() => {
    if (open) {
      setFavorites(_loadFavorites(projectId));
      setSelected(currentIcon ?? null);
      setSearchInput("");
      setDebouncedQ("");
      setTab(projectOnly ? "project" : "all");
      setUploadOpen(false);
      setUploadName("");
      setUploadFile(null);
      setUploadError(null);
    }
  }, [open, projectId, currentIcon, projectOnly]);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setDebouncedQ(searchInput.trim());
    }, 200);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [searchInput]);

  const fetchTokenRef = useRef(0);

  const fetchLibrary = useCallback(
    async (sourceTab: LibraryTab, q: string) => {
      if (sourceTab === "favorites") return;
      const myToken = ++fetchTokenRef.current;
      setLoading(true);
      setError(null);
      const source = sourceTab === "all" ? "all" : sourceTab;
      try {
        const resp = await focusIconsApi.library(projectId, {
          source,
          q: q || undefined,
          limit: 2000,
        });
        if (fetchTokenRef.current !== myToken) return;
        setFetchedIcons(resp.icons);
        setFetchedTotal(resp.total);
      } catch (err) {
        if (fetchTokenRef.current !== myToken) return;
        setError(String((err as Error)?.message || err));
        setFetchedIcons([]);
        setFetchedTotal(0);
      } finally {
        if (fetchTokenRef.current === myToken) {
          setLoading(false);
        }
      }
    },
    [projectId],
  );

  useEffect(() => {
    if (!open) return;
    void fetchLibrary(tab, debouncedQ);
  }, [open, tab, debouncedQ, fetchLibrary]);

  const favoriteEntries = useMemo<FocusIconLibraryEntry[]>(
    () =>
      favorites
        .filter(
          (name) =>
            !debouncedQ ||
            name.toLowerCase().includes(debouncedQ.toLowerCase()),
        )
        .map((name) => ({
          name,
          thumbnail_url: `/api/go/vanilla/focus-icon/${encodeURIComponent(name)}`,
          source: "vanilla" as const,
          dlc: null,
          category: name.startsWith("GFX_focus_")
            ? "focus"
            : name.startsWith("GFX_goal_")
              ? "goal"
              : null,
        })),
    [favorites, debouncedQ],
  );

  const displayedIcons = tab === "favorites" ? favoriteEntries : fetchedIcons;
  const displayedTotal =
    tab === "favorites" ? favoriteEntries.length : fetchedTotal;

  const toggleFavorite = useCallback(
    (gfxName: string) => {
      setFavorites((prev) => {
        const exists = prev.includes(gfxName);
        const next = exists
          ? prev.filter((n) => n !== gfxName)
          : [gfxName, ...prev].slice(0, FAVORITES_MAX);
        _saveFavorites(projectId, next);
        return next;
      });
    },
    [projectId],
  );

  const handleCommit = useCallback(() => {
    if (!selected) return;
    onSelect(selected);
    onOpenChange(false);
  }, [selected, onSelect, onOpenChange]);

  const handleCardDoubleClick = useCallback(
    (name: string) => {
      onSelect(name);
      onOpenChange(false);
    },
    [onSelect, onOpenChange],
  );

  const handleUploadNamed = useCallback(async () => {
    const name = uploadName.trim();
    if (!name || !uploadFile) return;
    setUploading(true);
    setUploadError(null);
    try {
      const res = await focusIconsApi.uploadNamed(
        projectId,
        name,
        uploadFile,
        uploadRevisions.current,
      );

      const stored = res.gfx_name;

      setUploadName("");
      setUploadFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
      setUploadOpen(false);
      setSelected(stored);
      setTab("project");

      void fetchLibrary("project", debouncedQ);
    } catch (err) {
      const raw = String((err as Error)?.message || err);
      if (editRevisionErrorCode(err)) {
        setUploadError(raw);
      } else if (
        /failed to fetch|networkerror|network request failed/i.test(raw)
      ) {
        setUploadError(t("uploadNamedErrNetwork"));
      } else if (/\b5\d\d\b/.test(raw)) {
        setUploadError(t("uploadNamedErrServer"));
      } else {
        setUploadError(t("uploadNamedErrGeneric"));
      }
    } finally {
      setUploading(false);
    }
  }, [uploadName, uploadFile, projectId, fetchLibrary, debouncedQ, t]);

  const nameInvalid =
    uploadName.trim().length > 0 && !GFX_NAME_RE.test(uploadName.trim());

  const tabs: { value: LibraryTab; label: string }[] = useMemo(
    () =>
      projectOnly
        ? [{ value: "project", label: t("tabProject") }]
        : [
            { value: "all", label: t("tabAll") },
            { value: "vanilla", label: "Vanilla" },
            { value: "project", label: t("tabProject") },
            {
              value: "favorites",
              label: `${t("tabFavorites")}${favorites.length ? ` (${favorites.length})` : ""}`,
            },
          ],
    [favorites.length, projectOnly, t],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent className="sm:max-w-3xl lg:max-w-5xl max-h-[85vh] flex flex-col gap-3">
        <div className="space-y-2">
          <h2 className="font-heading text-base leading-none font-medium">
            {title ?? t("title")}
          </h2>
          <p className="text-xs text-muted-foreground">
            {description ?? t("description")}
          </p>
        </div>

        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <Input
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder={t("searchPlaceholder")}
            className="font-mono text-xs sm:max-w-md"
          />
          <div className="flex gap-1 flex-wrap">
            {tabs.map((tb) => (
              <Button
                key={tb.value}
                size="sm"
                variant={tab === tb.value ? "default" : "outline"}
                onClick={() => setTab(tb.value)}
                type="button"
              >
                {tb.label}
              </Button>
            ))}
          </div>
        </div>

        <div className="text-xs text-muted-foreground flex items-center justify-between">
          <span>
            {loading ? (
              t("loading")
            ) : error ? (
              <span className="text-destructive">
                {t("loadFailed")} {error}
              </span>
            ) : (
              `${t("iconCount", { n: displayedTotal })}${displayedIcons.length < displayedTotal ? t("showingFirst", { n: displayedIcons.length }) : ""}`
            )}
          </span>
          {selected ? (
            <span className="font-mono">
              {t("selectedLabel")} {selected}
            </span>
          ) : (
            <span className="opacity-60">{t("notSelected")}</span>
          )}
        </div>

        <div className="flex-1 overflow-y-auto border rounded bg-muted/20 p-3">
          {!loading && displayedIcons.length === 0 ? (
            <div className="text-center text-sm text-muted-foreground py-12">
              {tab === "favorites"
                ? t("noFavorites")
                : tab === "project"
                  ? t("noProjectIcons")
                  : t("noMatch")}
            </div>
          ) : (
            <div className="grid grid-cols-4 sm:grid-cols-5 lg:grid-cols-7 gap-2">
              {displayedIcons.map((icon) => {
                const isSelected = selected === icon.name;
                const isFav = favorites.includes(icon.name);
                const thumbnailSrc = /^https?:\/\//i.test(icon.thumbnail_url)
                  ? icon.thumbnail_url
                  : `${SERVER_ORIGIN}${icon.thumbnail_url}`;
                return (
                  <div
                    key={`${icon.source}:${icon.name}`}
                    className={`relative group rounded border cursor-pointer transition-colors p-2 flex flex-col items-center gap-1 hover:bg-accent ${
                      isSelected
                        ? "border-primary ring-2 ring-primary/40 bg-accent"
                        : "border-border bg-background"
                    }`}
                    onClick={() => setSelected(icon.name)}
                    onDoubleClick={() => handleCardDoubleClick(icon.name)}
                    title={`${icon.name}${icon.dlc ? ` · ${icon.dlc}` : ""}${icon.category ? ` · ${icon.category}` : ""}`}
                  >
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        toggleFavorite(icon.name);
                      }}
                      className={`ui-focus-ring ui-interactive absolute top-1 right-1 text-sm leading-none rounded px-1 ${
                        isFav
                          ? "opacity-100"
                          : "opacity-40 group-hover:opacity-80"
                      }`}
                      aria-label={isFav ? t("unfavorite") : t("favorite")}
                    >
                      {isFav ? "★" : "☆"}
                    </button>

                    <img
                      src={thumbnailSrc}
                      alt={icon.name}
                      loading="lazy"
                      decoding="async"
                      className="w-16 h-16 object-contain"
                      onError={(e) => {
                        (e.currentTarget as HTMLImageElement).style.visibility =
                          "hidden";
                      }}
                    />
                    <span className="text-micro font-mono leading-tight text-center break-all line-clamp-2 max-w-full">
                      {icon.name.replace(/^GFX_(focus|goal)_/, "")}
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <div className="border-t pt-2">
          <button
            type="button"
            className="ui-focus-ring ui-interactive flex items-center gap-1.5 rounded text-xs text-muted-foreground hover:text-foreground"
            onClick={() => setUploadOpen((o) => !o)}
          >
            <span aria-hidden>{uploadOpen ? "▼" : "▶"}</span>
            <span>{t("uploadNamedToggle")}</span>
          </button>
          {uploadOpen && (
            <div className="mt-2 space-y-2 rounded border bg-muted/20 p-3">
              <p className="text-xs text-muted-foreground">
                {uploadDescription ?? t("uploadNamedDesc")}
              </p>
              <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
                <div className="flex flex-col gap-1 sm:max-w-xs">
                  <Input
                    value={uploadName}
                    onChange={(e) => setUploadName(e.target.value)}
                    placeholder={uploadNamePlaceholder ?? "GFX_focus_my_custom"}
                    className={`font-mono text-xs ${nameInvalid ? "border-destructive" : ""}`}
                    aria-label={t("uploadNamedNameLabel")}
                    aria-invalid={nameInvalid || undefined}
                  />
                  <p
                    className={`text-xs ${nameInvalid ? "text-destructive" : "text-muted-foreground"}`}
                  >
                    {t("uploadNamedNameHint")}
                  </p>
                </div>

                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".png,.jpg,.jpeg,.dds,.tga,image/*"
                  onChange={(e) => setUploadFile(e.target.files?.[0] ?? null)}
                  className="hidden"
                  aria-label={t("uploadNamedFileLabel")}
                />
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    type="button"
                    variant="outline"
                    onClick={() => fileInputRef.current?.click()}
                    className="shrink-0"
                  >
                    {uploadFile
                      ? t("uploadNamedChangeFile")
                      : t("uploadNamedChooseFile")}
                  </Button>
                  {uploadFile && (
                    <span className="font-mono text-xs text-muted-foreground truncate max-w-48">
                      {uploadFile.name}
                    </span>
                  )}
                </div>
                <Button
                  size="sm"
                  type="button"
                  disabled={
                    !uploadName.trim() ||
                    nameInvalid ||
                    !uploadFile ||
                    uploading
                  }
                  onClick={handleUploadNamed}
                  className="shrink-0"
                >
                  {uploading
                    ? t("uploadNamedUploading")
                    : t("uploadNamedButton")}
                </Button>
              </div>
              {uploadError && (
                <p className="text-xs text-destructive">
                  {t("uploadNamedFailed")} {uploadError}
                </p>
              )}
            </div>
          )}
        </div>

        <div className="-mx-4 -mb-4 flex flex-col-reverse gap-2 rounded-b-xl border-t bg-muted/50 p-3 sm:flex-row sm:justify-end">
          <Button
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
            type="button"
          >
            {t("cancel")}
          </Button>
          <Button
            size="sm"
            disabled={!selected}
            onClick={handleCommit}
            type="button"
          >
            {t("confirm")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
