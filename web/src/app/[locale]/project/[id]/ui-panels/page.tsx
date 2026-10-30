"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import useSWR from "swr";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import { PanelsTopLeft, TriangleAlert } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select } from "@/components/ui/select";
import { uiPanelsApi } from "@/lib/api";
import type { UiPanel, UiPanelControl } from "@/types";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { useProjectStore } from "@/lib/store/projectStore";
import {
  confirmDiscardIfDirty,
  useRegisterDirty,
} from "@/lib/hooks/useUnsavedGuard";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { useUiPanelSelection } from "@/components/workbench/ui-panels/selectionStore";
import { CanvasArea } from "./CanvasArea";
import { InspectorArea } from "./InspectorArea";
import { WeldCard } from "./WeldCard";

const CONTEXT_TYPES = ["decision_category", "player_context"] as const;

interface DraftPanel {
  token: string;
  display_name_locs: Record<string, string>;
  context_type: string;
  parent_window_token: string;
  visible_trigger: string;
  window_props: Record<string, unknown>;
  controls: UiPanelControl[];
  raw_extras: Record<string, unknown>;
}

function panelToDraft(p: UiPanel): DraftPanel {
  return {
    token: p.token || "",
    display_name_locs: p.display_name_locs || {},
    context_type: p.context_type || "decision_category",
    parent_window_token: p.parent_window_token || "",
    visible_trigger: p.visible_trigger || "",
    window_props: p.window_props || {},
    controls: p.controls || [],
    raw_extras: p.raw_extras || {},
  };
}

export default function UiPanelsPage() {
  const params = useParams();
  const projectId = Number(params.id);
  const t = useTranslations("uiPanels");
  const tCommon = useTranslations("common");
  const confirmDialog = useConfirm();

  const [list, setList] = useState<UiPanel[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [draft, setDraft] = useState<DraftPanel | null>(null);
  const [savedDraft, setSavedDraft] = useState<DraftPanel | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );
  const [error, setError] = useState<string | null>(null);
  const [newToken, setNewToken] = useState("");
  const setSelectedPath = useUiPanelSelection((s) => s.setSelectedPath);

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );

  const refresh = useCallback(async () => {
    const res = await uiPanelsApi.list(projectId);
    setList(res.panels);
    return res.panels;
  }, [projectId]);

  const { error: listLoadError, mutate: retryList } = useSWR(
    projectId ? ["ui-panels/list", projectId] : null,
    () => refresh(),
  );

  const selected = list.find((p) => p.id === selectedId) ?? null;

  const seededIdRef = useRef<number | null>(null);
  useEffect(() => {
    if (selectedId == null || !selected) return;
    if (selectedId === seededIdRef.current) return;
    seededIdRef.current = selectedId;
    const d = panelToDraft(selected);
    setDraft(d);
    setSavedDraft(d);
    setSaveStatus("idle");
    setError(null);
    setSelectedPath(null);
  }, [selectedId, selected, setSelectedPath]);

  const isUnsaved =
    draft != null && JSON.stringify(draft) !== JSON.stringify(savedDraft);
  useRegisterDirty("ui-panels", isUnsaved);

  const listNav = useListboxNav();

  const handleCreate = async (tokenOverride?: string) => {
    const token = (tokenOverride ?? newToken).trim();
    if (!token) return;
    try {
      const created = await uiPanelsApi.create(projectId, { token });
      setNewToken("");
      await refresh();
      setSelectedId(created.id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setSaveStatus("error");
    }
  };

  const handleSave = async () => {
    if (!draft || selectedId == null) return;
    setSaving(true);
    setSaveStatus("idle");
    try {
      await uiPanelsApi.update(projectId, selectedId, draft);
      await refresh();
      setSavedDraft(draft);
      setSaveStatus("success");
      setError(null);
      setTimeout(() => setSaveStatus("idle"), 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (panel: UiPanel) => {
    const ok = await confirmDialog({ body: t("deleteConfirm"), danger: true });
    if (!ok) return;
    await uiPanelsApi.delete(projectId, panel.id);
    if (selectedId === panel.id) {
      setSelectedId(null);
      setDraft(null);
      setSavedDraft(null);
      seededIdRef.current = null;
    }
    await refresh();
  };

  const patchDraft = (partial: Partial<DraftPanel>) =>
    setDraft((d) => (d ? { ...d, ...partial } : d));

  if (listLoadError && list.length === 0) {
    return (
      <WorkbenchLoadError
        onRetry={() => retryList(undefined, { throwOnError: false })}
      />
    );
  }

  return (
    <div className="flex h-full min-w-0">
      <aside className="w-64 border-r border-border shrink-0 flex flex-col">
        <div className="p-3 border-b border-border space-y-2">
          <div className="flex items-center gap-2">
            <PanelsTopLeft className="w-4 h-4 text-accent" aria-hidden />
            <h2 className="text-base font-semibold tracking-tight">
              {t("title")}
            </h2>
            <span className="ml-auto bg-muted text-muted-foreground rounded-full px-2 text-xs">
              {list.length}
            </span>
          </div>
          <p className="text-xs text-muted-foreground">{t("intro")}</p>

          <p
            title={t("previewNoticeDetail")}
            className="flex items-start gap-1.5 rounded-md border border-warning/40 bg-warning/10 px-2 py-1.5 text-xs text-warning"
          >
            <TriangleAlert
              className="mt-0.5 h-3.5 w-3.5 shrink-0"
              aria-hidden
            />
            <span>{t("previewNotice")}</span>
          </p>
          <div className="flex gap-1.5">
            <Input
              value={newToken}
              onChange={(e) => setNewToken(e.target.value)}
              placeholder={t("tokenPlaceholder")}
              className="h-8 text-xs font-mono"
              onKeyDown={(e) => {
                if (e.key === "Enter") void handleCreate();
              }}
            />
            <Button size="sm" onClick={() => void handleCreate()}>
              {t("newPanel")}
            </Button>
          </div>
        </div>
        <div
          className="flex-1 overflow-y-auto p-2 space-y-0.5 outline-none"
          {...listNav.containerProps}
        >
          {list.map((p) => (
            <div
              key={p.id}
              {...listNav.optionProps(selectedId === p.id)}
              onClick={() => {
                if (p.id === selectedId || confirmDiscardIfDirty())
                  setSelectedId(p.id);
              }}
              className={`ui-card px-3 py-2 cursor-pointer group focus-visible:ring-1 focus-visible:ring-ring ${
                selectedId === p.id ? "border-border bg-muted/40" : ""
              }`}
            >
              <div className="flex items-center gap-2">
                <span className="text-xs font-mono truncate">{p.token}</span>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    void handleDelete(p);
                  }}
                  className="ml-auto opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive text-xs ui-interactive"
                  aria-label={t("deleteAria")}
                >
                  ✕
                </button>
              </div>
              {(p.display_name_locs?.zh || p.display_name_locs?.en) && (
                <p className="text-xs text-muted-foreground truncate mt-0.5">
                  {p.display_name_locs?.zh || p.display_name_locs?.en}
                </p>
              )}
            </div>
          ))}
        </div>
      </aside>

      <main className="flex-1 min-w-0 overflow-y-auto flex flex-col">
        {!selected || !draft ? (
          <div className="flex-1 grid place-items-center">
            {list.length === 0 ? (
              <EditorEmptyState
                variant="hero"
                className="max-w-md"
                icon={
                  <PanelsTopLeft
                    className="h-10 w-10 text-muted-foreground/60"
                    aria-hidden
                  />
                }
                title={t("emptyHeroTitle")}
                description={t("emptyHeroDesc")}
                actions={
                  <Button
                    size="sm"
                    onClick={() => {
                      void handleCreate(newToken.trim() || "my_panel");
                    }}
                  >
                    {tCommon("emptyCtaCreate")}
                  </Button>
                }
              />
            ) : (
              <div className="ui-panel p-8 text-center space-y-2 max-w-sm">
                <PanelsTopLeft
                  className="w-6 h-6 mx-auto text-muted-foreground"
                  aria-hidden
                />
                <h3 className="text-base font-semibold tracking-tight">
                  {t("emptyTitle")}
                </h3>
                <p className="text-xs text-muted-foreground">
                  {t("emptyBody")}
                </p>
              </div>
            )}
          </div>
        ) : (
          <>
            <div className="flex items-start gap-4 px-4 py-3 border-b border-border flex-wrap">
              <div className="space-y-1">
                <Label className="text-micro uppercase tracking-widest text-muted-foreground">
                  token
                </Label>
                <Input
                  value={draft.token}
                  onChange={(e) => patchDraft({ token: e.target.value })}
                  className="h-8 text-xs font-mono w-44"
                />
              </div>
              <div className="space-y-1">
                <Label className="text-micro uppercase tracking-widest text-muted-foreground">
                  {t("contextType")}
                </Label>
                <Select
                  className="h-8 px-2 text-xs font-mono"
                  value={draft.context_type}
                  onChange={(e) => patchDraft({ context_type: e.target.value })}
                >
                  {CONTEXT_TYPES.map((ct) => (
                    <option key={ct} value={ct}>
                      {ct}
                    </option>
                  ))}
                </Select>
              </div>
              <div className="space-y-1">
                <Label className="text-micro uppercase tracking-widest text-muted-foreground">
                  {t("windowName")}
                </Label>
                <span className="h-8 flex items-center px-2 rounded-md border border-border bg-muted/40 text-xs font-mono text-muted-foreground">
                  {selected.derived?.window_name ?? "—"}
                </span>
              </div>
              <div className="space-y-1">
                <Label className="text-micro uppercase tracking-widest text-muted-foreground">
                  {t("parentWindow")}
                </Label>
                <Input
                  value={draft.parent_window_token}
                  onChange={(e) =>
                    patchDraft({ parent_window_token: e.target.value })
                  }
                  placeholder="decision_tab"
                  className="h-8 text-xs font-mono w-40"
                />

                {draft.context_type === "decision_category" && (
                  <p className="text-micro text-muted-foreground">
                    {t("parentWindowHint")}
                  </p>
                )}
              </div>
            </div>

            <div className="flex flex-1 min-h-[320px]">
              <CanvasArea
                projectId={projectId}
                panel={selected}
                controls={draft.controls}
                onControlsChange={(controls) => patchDraft({ controls })}
              />
              <InspectorArea
                projectId={projectId}
                panel={selected}
                controls={draft.controls}
                onControlsChange={(controls) => patchDraft({ controls })}
              />
            </div>

            <div className="p-4 space-y-4 border-t border-border">
              <div className="space-y-1">
                <Label>{t("displayName")}</Label>
                <DisplayNameLocsTabs
                  entityKey={`ui-panel:${selectedId}:display-name`}
                  value={draft.display_name_locs}
                  exportLanguages={exportLanguages}
                  onChange={(display_name_locs) =>
                    patchDraft({ display_name_locs })
                  }
                />
              </div>
              <div className="space-y-1">
                <Label>{t("panelVisible")}</Label>
                <Textarea
                  value={draft.visible_trigger}
                  onChange={(e) =>
                    patchDraft({ visible_trigger: e.target.value })
                  }
                  placeholder={t("panelVisiblePlaceholder")}
                  className="min-h-16 text-xs font-mono"
                />
              </div>
              <WeldCard
                projectId={projectId}
                panel={selected}
                controls={draft.controls}
                onControlsChange={(controls) => patchDraft({ controls })}
              />

              {(saving || saveStatus !== "idle" || isUnsaved) && (
                <div
                  className={`-mx-4 px-4 py-1.5 text-xs font-medium flex items-center gap-2 transition-[opacity,background-color,border-color] duration-300 ${
                    saving
                      ? "bg-muted/80 text-muted-foreground border-y border-border"
                      : saveStatus === "success"
                        ? "bg-success/10 text-success border-y border-success/20"
                        : saveStatus === "error"
                          ? "bg-destructive/10 text-destructive border-y border-destructive/20"
                          : isUnsaved
                            ? "bg-warning/10 text-warning border-y border-warning/20"
                            : ""
                  }`}
                >
                  <span>
                    {saving
                      ? t("saving")
                      : saveStatus === "success"
                        ? t("saved")
                        : saveStatus === "error"
                          ? (error ?? t("saveFailed"))
                          : isUnsaved
                            ? t("unsavedChanges")
                            : ""}
                  </span>
                </div>
              )}
              <div className="flex items-center gap-2 pt-2 border-t border-border">
                <Button
                  onClick={() => void handleSave()}
                  disabled={saving}
                  size="sm"
                >
                  {t("saveButton")}
                </Button>
              </div>
            </div>
          </>
        )}
      </main>
    </div>
  );
}
