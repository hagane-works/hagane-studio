"use client";

import { useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import {
  useScriptDraftStore,
  draftKey,
  type ParseStatus,
  type ScriptParseError,
} from "./scriptDraftStore";
import { useScriptParse, type ParsePreviewApiResponse } from "./useScriptParse";
import type { DirtyIndicator, EditorMarker } from "./ScriptEditor";
import { useCwtoolRules } from "./cwtoolStore";
import { cwtoolValidate, type CwtoolMessageFormatter } from "./cwtoolValidate";
import { useLocaleStore } from "@/lib/store/localeStore";
import { useIdeologyOptions } from "@/lib/ideology/useIdeologyOptions";

interface UseEditorWiringOptions {
  editRevision?: string;

  previewReady?: boolean;
  workbench: string;
  entityId: number | null | undefined;
  scriptPreview: string;
  parsePreview: (script: string) => Promise<ParsePreviewApiResponse>;

  applyEntity: (
    entityId: number,
    entityDict: Record<string, unknown>,
    editRevision?: string,
  ) => Promise<unknown>;

  onAfterApply?: (entityId: number) => Promise<void> | void;

  entityNoun?: string;

  projectId?: number;

  externalMarkers?: readonly EditorMarker[];
}

function _statusLabel(
  t: (
    key:
      | "cwtoolError"
      | "parsing"
      | "parsedWithWarnings"
      | "parsedReady"
      | "parseErrorLine"
      | "parseError"
      | "diagnosticLine"
      | "diagnostics"
      | "editedWarnings"
      | "editedPending",
    values?: Record<string, number>,
  ) => string,
  status: ParseStatus | undefined,
  s2ErrCount: number,
  firstErrLine: number | undefined,
  cwtoolErrCount: number,
  cwtoolWarnCount: number,
): string {
  if (cwtoolErrCount > 0) {
    return t("cwtoolError", { count: cwtoolErrCount });
  }
  switch (status) {
    case "parsing":
      return t("parsing");
    case "ok":
      return cwtoolWarnCount > 0
        ? t("parsedWithWarnings", { count: cwtoolWarnCount })
        : t("parsedReady");
    case "error":
      return firstErrLine
        ? t("parseErrorLine", { line: firstErrLine, count: s2ErrCount })
        : t("parseError", { count: s2ErrCount });
    case "info":
      return firstErrLine
        ? t("diagnosticLine", { line: firstErrLine, count: s2ErrCount })
        : t("diagnostics", { count: s2ErrCount });
    default:
      return cwtoolWarnCount > 0
        ? t("editedWarnings", { count: cwtoolWarnCount })
        : t("editedPending");
  }
}

export interface EditorWiring {
  scriptDraft: string;
  setScriptDraft: (next: string) => void;
  triggerParse: () => Promise<void>;
  dirty: boolean;
  indicator: DirtyIndicator | undefined;
  markers: ScriptParseError[];
  canApply: boolean;
  handleResetToForm: (() => void) | undefined;
  handleApplyEditor: (() => Promise<void>) | undefined;
  clearAllDrafts: () => void;
}

export function useEditorWiring({
  editRevision,
  previewReady = true,
  workbench,
  entityId,
  scriptPreview,
  parsePreview,
  applyEntity,
  onAfterApply,
  projectId,
  externalMarkers,
}: UseEditorWiringOptions): EditorWiring {
  const tEditor = useTranslations("editorWiring");
  const dKey = draftKey(workbench, entityId);
  const draftEntry = useScriptDraftStore((s) =>
    dKey ? s.drafts.get(dKey) : null,
  );
  const setStoreDraft = useScriptDraftStore((s) => s.setDraft);
  const discardStoreDraft = useScriptDraftStore((s) => s.discardDraft);
  const isStaleFn = useScriptDraftStore((s) => s.isStale);
  const clearAllDrafts = useScriptDraftStore((s) => s.clearAll);

  const scriptDraft = draftEntry?.draft ?? scriptPreview;

  const stalePromptedFor = useRef<string | null>(null);
  const applying = useRef(new Set<string>());
  useEffect(() => {
    if (!dKey || !draftEntry || !previewReady) return;
    if (applying.current.has(dKey)) return;
    if (!isStaleFn(dKey, scriptPreview)) return;
    if (stalePromptedFor.current === dKey) return;
    stalePromptedFor.current = dKey;
    const keep = window.confirm(tEditor("stalePrompt"));
    if (!keep) discardStoreDraft(dKey);
  }, [
    dKey,
    draftEntry,
    scriptPreview,
    previewReady,
    isStaleFn,
    discardStoreDraft,
    tEditor,
  ]);

  useEffect(() => {
    stalePromptedFor.current = null;
  }, [entityId]);

  const prevProjectIdRef = useRef<number | undefined>(projectId);
  useEffect(() => {
    if (projectId === undefined) return;
    if (prevProjectIdRef.current !== projectId) {
      clearAllDrafts();
      prevProjectIdRef.current = projectId;
    }
  }, [projectId, clearAllDrafts]);

  const setScriptDraft = useCallback(
    (next: string) => {
      if (!dKey) return;
      if (!previewReady && !useScriptDraftStore.getState().getDraft(dKey))
        return;
      setStoreDraft(dKey, next, scriptPreview, editRevision);
    },
    [dKey, scriptPreview, setStoreDraft, editRevision, previewReady],
  );

  const { triggerParse } = useScriptParse({
    workbench,
    entityId,
    parsePreview,
  });

  const locale = useLocaleStore((s) => s.locale);
  const tMisc = useTranslations("misc");
  const formatCwtoolMessage = useCallback<CwtoolMessageFormatter>(
    (key, params) => tMisc(key, params),
    [tMisc],
  );
  const cwtoolRules = useCwtoolRules();

  const { knownSubIds: knownIdeologyIds } = useIdeologyOptions();
  const cwtoolMarkers = useMemo(
    () =>
      cwtoolValidate(
        draftEntry?.draft ?? scriptPreview,
        cwtoolRules,
        locale === "zh" ? "zh" : "en",
        {
          knownIdeologyIds,
          formatMessage: formatCwtoolMessage,
        },
      ),
    [
      draftEntry?.draft,
      scriptPreview,
      cwtoolRules,
      locale,
      knownIdeologyIds,
      formatCwtoolMessage,
    ],
  );
  const cwtoolErrCount = cwtoolMarkers.filter(
    (m) => m.severity === "error",
  ).length;
  const cwtoolWarnCount = cwtoolMarkers.filter(
    (m) => m.severity === "warning" || m.severity === "info",
  ).length;

  const dirty = !!draftEntry && draftEntry.draft !== draftEntry.baselineScript;

  const indicatorStatus: ParseStatus | undefined =
    cwtoolErrCount > 0 ? "error" : draftEntry?.parseStatus;

  const indicator: DirtyIndicator | undefined = dirty
    ? {
        dirty: true,
        status: indicatorStatus,
        label: _statusLabel(
          (key, values) => tEditor(key, values),
          draftEntry.parseStatus,
          draftEntry.parseErrors.length,
          draftEntry.parseErrors[0]?.line,
          cwtoolErrCount,
          cwtoolWarnCount,
        ),
      }
    : undefined;

  const markers: EditorMarker[] = useMemo(() => {
    const s2: EditorMarker[] = (draftEntry?.parseErrors ?? []).map((e) => ({
      line: e.line,
      msg: e.msg,
      severity: e.severity,
    }));

    return [...s2, ...cwtoolMarkers, ...(dirty ? [] : (externalMarkers ?? []))];
  }, [draftEntry?.parseErrors, cwtoolMarkers, externalMarkers, dirty]);

  const canApply =
    dirty &&
    draftEntry?.parseStatus === "ok" &&
    !!draftEntry?.parsedEntity &&
    cwtoolErrCount === 0;

  const handleResetToForm = useCallback(() => {
    if (!dKey) return;
    discardStoreDraft(dKey);
  }, [dKey, discardStoreDraft]);

  const handleApplyEditor = useCallback(async () => {
    if (entityId == null || !draftEntry?.parsedEntity || !dKey) return;
    if (applying.current.has(dKey)) return;
    applying.current.add(dKey);
    const submitted = draftEntry;
    try {
      if (workbench === "focus_node")
        await applyEntity(
          entityId,
          draftEntry.parsedEntity,
          draftEntry.editRevision,
        );
      else await applyEntity(entityId, draftEntry.parsedEntity);
      if (onAfterApply) await onAfterApply(entityId);

      if (useScriptDraftStore.getState().drafts.get(dKey) === submitted) {
        discardStoreDraft(dKey);
      } else {
        stalePromptedFor.current = dKey;
      }

      if (workbench === "focus_node")
        toast.success(tEditor("applyReseedTitle"));
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      const hint =
        msg === "Failed to fetch" ? `\n\n${tEditor("networkHint")}` : "";

      toast.error(tEditor("applyFailed", { message: msg, hint }));
    } finally {
      applying.current.delete(dKey);
    }
  }, [
    entityId,
    draftEntry,
    dKey,
    applyEntity,
    onAfterApply,
    discardStoreDraft,
    tEditor,
    workbench,
  ]);

  return {
    scriptDraft,
    setScriptDraft,
    triggerParse,
    dirty,
    indicator,
    markers,
    canApply,
    handleResetToForm: dirty ? handleResetToForm : undefined,
    handleApplyEditor: dirty ? handleApplyEditor : undefined,
    clearAllDrafts,
  };
}
