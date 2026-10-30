"use client";

import { readProjectEditRevisions } from "./save-diagnostics/revisionRequest";
import type { EditRevisions } from "./save-diagnostics/editRevisions";
import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { Library } from "lucide-react";
import { useTranslations } from "use-intl";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import type { FocusNode } from "@/types";
import { withoutSaveDiagnostics } from "@/lib/saveDiagnostics";
import { SaveDiagnostics } from "./save-diagnostics/SaveDiagnostics";
import { useSaveDiagnostics } from "./save-diagnostics/useSaveDiagnostics";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { IdeaReferenceDropdown } from "./IdeaReferenceDropdown";
import { EventReferenceDropdown } from "./EventReferenceDropdown";
import { DecisionReferenceDropdown } from "./DecisionReferenceDropdown";
import { CharacterReferenceDropdown } from "./CharacterReferenceDropdown";
import { FocusReferenceDropdown } from "./FocusReferenceDropdown";
import {
  ScriptBlockEditor,
  EFFECT_TEMPLATES,
  CONDITION_TEMPLATES,
  extractSaveErrorMessage,
  validateRequiredId,
} from "@/components/shared/editors";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import {
  nextDraftOnEntityChange,
  stableFingerprint,
} from "@/components/shared/editors/draftReseed";
import { useProjectStore } from "@/lib/store/projectStore";
import { TooltipProvider } from "@/components/ui/tooltip";
import {
  ImageUpload,
  type ImageUploadResult,
} from "@/shared/upload/ImageUpload";
import { FOCUS_ICON_SIZE } from "@/shared/upload/sizes";
import { focusTreesApi, API_BASE } from "@/lib/api";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import { FocusIconLibraryDialog } from "@/components/workbench/focus-graph/FocusIconLibraryDialog";
import { SourceEditorDialog } from "@/shared/dialogs/SourceEditorDialog";
import { useImportedSourceGate } from "@/shared/dialogs/useImportedSourceGate";
import {
  useBaseModIdCollision,
  BaseModIdCollisionNotice,
} from "./baseModIdCollision";

const HINT_CLASS =
  "border-l-2 border-border/60 pl-2 text-xs leading-snug text-muted-foreground";

const SEARCH_FILTER_OPTIONS = [
  "FOCUS_FILTER_POLITICAL",
  "FOCUS_FILTER_RESEARCH",
  "FOCUS_FILTER_INDUSTRY",
  "FOCUS_FILTER_STABILITY",
  "FOCUS_FILTER_WAR_SUPPORT",
  "FOCUS_FILTER_MANPOWER",
  "FOCUS_FILTER_ANNEXATION",
];

interface VisualPolishSectionProps {
  textIcon: string;
  offset: { trigger: string; x: number; y: number }[];
  dynamicIcon: { icon: string; trigger: string }[];
  onChangeTextIcon: (v: string) => void;
  onChangeOffset: (v: { trigger: string; x: number; y: number }[]) => void;
  onChangeDynamicIcon: (v: { icon: string; trigger: string }[]) => void;

  projectId: number;
}

function VisualPolishSection({
  textIcon,
  offset,
  dynamicIcon,
  onChangeTextIcon,
  onChangeOffset,
  onChangeDynamicIcon,
  projectId,
}: VisualPolishSectionProps) {
  const t = useTranslations("focusNodeEditor");
  const [open, setOpen] = useState(false);

  const [pickerTarget, setPickerTarget] = useState<"text" | number | null>(
    null,
  );
  const filled =
    (textIcon ? 1 : 0) + (offset.length ? 1 : 0) + (dynamicIcon.length ? 1 : 0);

  const addOffset = () =>
    onChangeOffset([...offset, { trigger: "tag = USA", x: 0, y: 0 }]);
  const removeOffset = (idx: number) =>
    onChangeOffset(offset.filter((_, i) => i !== idx));
  const updateOffset = (
    idx: number,
    field: "trigger" | "x" | "y",
    value: string,
  ) =>
    onChangeOffset(
      offset.map((entry, i) =>
        i === idx
          ? field === "trigger"
            ? { ...entry, trigger: value }
            : { ...entry, [field]: Number(value) || 0 }
          : entry,
      ),
    );

  const addDynamicIcon = () =>
    onChangeDynamicIcon([
      ...dynamicIcon,
      { icon: "GFX_goal_unknown", trigger: "always" },
    ]);
  const removeDynamicIcon = (idx: number) =>
    onChangeDynamicIcon(dynamicIcon.filter((_, i) => i !== idx));
  const updateDynamicIcon = (
    idx: number,
    field: "icon" | "trigger",
    value: string,
  ) =>
    onChangeDynamicIcon(
      dynamicIcon.map((entry, i) =>
        i === idx ? { ...entry, [field]: value } : entry,
      ),
    );

  return (
    <div className="space-y-2">
      <button
        type="button"
        className="ui-interactive flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground"
        onClick={() => setOpen((o) => !o)}
      >
        <span>{open ? "▼" : "▶"}</span>
        <span className="uppercase tracking-wider font-semibold">
          {t("visualPolish")}
        </span>
        {filled > 0 && (
          <span className="rounded-full bg-accent/20 px-2 py-0.5 text-micro font-semibold">
            {t("filledCount", { n: filled })}
          </span>
        )}
      </button>
      {open && (
        <div className="ui-panel space-y-4 p-4">
          <p className="text-xs text-muted-foreground">
            {t("visualPolishDesc")}
          </p>

          <div className="space-y-1.5">
            <Label>text_icon</Label>
            <div className="flex gap-1">
              <Input
                value={textIcon}
                onChange={(e) => onChangeTextIcon(e.target.value)}
                placeholder="GFX_focus_decoration_alt"
                className="text-xs h-8 font-mono flex-1 min-w-0"
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setPickerTarget("text")}
                title={t("iconLibraryTitle")}
                className="shrink-0"
              >
                <Library className="h-3.5 w-3.5" aria-hidden />
                {t("iconLibraryButton")}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">{t("textIconDesc")}</p>
            <p className="text-xs text-muted-foreground">
              {t("customIconShipNote")}
            </p>
          </div>

          <Separator />

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>{t("offsetLabel")}</Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={addOffset}
                className="text-xs"
              >
                {t("addOffset")}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              {t.rich("offsetDesc", {
                mono: (chunks) => <span className="font-mono">{chunks}</span>,
              })}
            </p>
            {offset.length === 0 && (
              <p className="text-xs italic text-muted-foreground">
                {t("empty")}
              </p>
            )}
            {offset.map((entry, idx) => (
              <div
                key={idx}
                className="grid grid-cols-[1fr_auto_auto_auto] items-center gap-2 rounded-md border border-border p-2"
              >
                <Input
                  value={entry.trigger}
                  onChange={(e) => updateOffset(idx, "trigger", e.target.value)}
                  placeholder="tag = USA"
                  className="text-xs h-8 font-mono"
                />
                <Input
                  type="number"
                  value={entry.x}
                  onChange={(e) => updateOffset(idx, "x", e.target.value)}
                  className="text-xs h-8 w-16"
                />
                <Input
                  type="number"
                  value={entry.y}
                  onChange={(e) => updateOffset(idx, "y", e.target.value)}
                  className="text-xs h-8 w-16"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => removeOffset(idx)}
                  className="h-7 text-xs text-destructive"
                >
                  ×
                </Button>
              </div>
            ))}
          </div>

          <Separator />

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>{t("dynamicIconLabel")}</Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={addDynamicIcon}
                className="text-xs"
              >
                {t("addEntry")}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              {t.rich("dynamicIconDesc", {
                mono: (chunks) => <span className="font-mono">{chunks}</span>,
              })}
            </p>
            {dynamicIcon.length === 0 && (
              <p className="text-xs italic text-muted-foreground">
                {t("empty")}
              </p>
            )}
            {dynamicIcon.map((entry, idx) => (
              <div
                key={idx}
                className="space-y-1.5 rounded-md border border-border p-2"
              >
                <div className="flex items-center gap-2">
                  <Input
                    value={entry.icon}
                    onChange={(e) =>
                      updateDynamicIcon(idx, "icon", e.target.value)
                    }
                    placeholder="GFX_focus_alt"
                    className="text-xs h-8 font-mono flex-1"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setPickerTarget(idx)}
                    title={t("iconLibraryTitle")}
                    className="h-7 shrink-0"
                  >
                    {t("iconLibraryButton")}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => removeDynamicIcon(idx)}
                    className="h-7 text-xs text-destructive"
                  >
                    ×
                  </Button>
                </div>
                <Input
                  value={entry.trigger}
                  onChange={(e) =>
                    updateDynamicIcon(idx, "trigger", e.target.value)
                  }
                  placeholder={t("dynamicIconTriggerPlaceholder")}
                  className="text-xs h-8 font-mono"
                />
              </div>
            ))}
            {dynamicIcon.length > 0 && (
              <p className="text-xs text-muted-foreground">
                {t("customIconShipNote")}
              </p>
            )}
          </div>

          <FocusIconLibraryDialog
            open={pickerTarget !== null}
            onOpenChange={(o) => {
              if (!o) setPickerTarget(null);
            }}
            projectId={projectId}
            currentIcon={
              pickerTarget === "text"
                ? textIcon
                : typeof pickerTarget === "number"
                  ? dynamicIcon[pickerTarget]?.icon
                  : undefined
            }
            onSelect={(gfxName) => {
              if (pickerTarget === "text") {
                onChangeTextIcon(gfxName);
              } else if (typeof pickerTarget === "number") {
                updateDynamicIcon(pickerTarget, "icon", gfxName);
              }
              setPickerTarget(null);
            }}
          />
        </div>
      )}
    </div>
  );
}

interface FocusNodeEditorProps {
  node: FocusNode;
  allNodes: FocusNode[];
  onSave: (data: Partial<FocusNode>) => Promise<void>;
  projectId: number;
  countryTag: string;
  onDraftChange?: (draft: FocusNode) => void;
  compact?: boolean;

  treeId?: number;

  onIconUploaded?: (result: ImageUploadResult) => void;

  onForked?: (forked: FocusNode) => void;
}

export function FocusNodeEditor({
  node,
  allNodes,
  onSave,
  projectId,
  countryTag,
  onDraftChange,
  compact,
  treeId,
  onIconUploaded,
  onForked,
}: FocusNodeEditorProps) {
  const t = useTranslations("focusNodeEditor");

  const tc = useTranslations("characterReferenceDropdown");

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );
  const iconPreparationIntent = useRef(0);
  useEffect(() => {
    iconPreparationIntent.current += 1;
    return () => {
      iconPreparationIntent.current += 1;
    };
  }, [projectId, treeId, node.id]);
  const iconRevisions = useRef<EditRevisions>({});
  const [form, setForm] = useState(() => withoutSaveDiagnostics(node));
  const identity = `${projectId}:focus:${node.id}`;

  const prepareIconUpload = async () => {
    const intent = ++iconPreparationIntent.current;
    const snapshot = await readProjectEditRevisions(projectId);
    if (intent !== iconPreparationIntent.current) return;
    delete (snapshot as Record<string, string>)["focus_tree:" + treeId];
    iconRevisions.current = {
      ...snapshot,
      ...(form.tree_edit_revision
        ? { ["focus_tree:" + treeId]: form.tree_edit_revision }
        : {}),
    };
  };
  const uploadIconFile = async (
    file: File,
    crop?: { x: number; y: number; width: number; height: number },
  ) => {
    if (!treeId) throw new Error("focus tree is not ready");
    return focusTreesApi.uploadNodeIcon(
      projectId,
      treeId,
      node.id,
      file,
      crop,
      iconRevisions.current["focus_tree:" + treeId],
      iconRevisions.current,
    );
  };

  const [iconVersion, setIconVersion] = useState(0);
  const afterIconUploaded = (result: ImageUploadResult) => {
    const icon = result["icon"];
    if (typeof icon === "string") {
      setForm((f) => ({ ...f, icon }));
    }
    setIconVersion((v) => v + 1);
    onIconUploaded?.(result);
  };
  const activeIdentity = useRef(identity);
  activeIdentity.current = identity;
  const { diagnostics } = useSaveDiagnostics(identity, node);

  const baseModDepKeys = useBaseModDepKeys();
  const baseModIconFallbacks = useMemo(
    () =>
      form.icon && baseModDepKeys.length
        ? baseModDepKeys.map((dk) => gfxUrl(projectId, dk, form.icon))
        : [],
    [projectId, baseModDepKeys, form.icon],
  );

  const baseModId = useBaseModIdCollision("focus", form.focus_id || "");
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );

  const [saveError, setSaveError] = useState<string | null>(null);
  const [idError, setIdError] = useState<string | null>(null);

  const isUnsaved =
    stableFingerprint(form, ["tree_edit_revision"]) !==
    stableFingerprint(withoutSaveDiagnostics(node), ["tree_edit_revision"]);

  useRegisterDirty("focus-tree", isUnsaved);
  const [showAdvanced, setShowAdvanced] = useState(false);

  const [warTagInput, setWarTagInput] = useState("");

  const [iconLibraryOpen, setIconLibraryOpen] = useState(false);

  const [iconOpen, setIconOpen] = useState(true);

  const nodeFingerprint = useCallback(
    (n: FocusNode) =>
      stableFingerprint(n, [
        "icon",
        "diagnostics",
        "tree_diagnostics",
        "tree_edit_revision",
      ]),
    [],
  );
  const seededNodeRef = useRef({
    id: node.id,
    fingerprint: nodeFingerprint(node),
  });
  const saveAttempt = useRef<{
    identity: string;
    snapshot: FocusNode;
    result?: "saved" | "failed";
  } | null>(null);
  const feedbackVersion = useRef(0);
  const seededIdentity = useRef(identity);
  const [saveAttemptVersion, setSaveAttemptVersion] = useState(0);
  useEffect(() => {
    if (seededIdentity.current !== identity) {
      seededIdentity.current = identity;
      seededNodeRef.current = {
        id: node.id,
        fingerprint: nodeFingerprint(node),
      };
      saveAttempt.current = null;
      setForm(withoutSaveDiagnostics(node));
      setSaving(false);
      setSaveStatus("idle");
      setSaveError(null);
      return;
    }
    const attempt = saveAttempt.current;
    if (attempt?.identity === identity) {
      seededNodeRef.current = {
        id: node.id,
        fingerprint: nodeFingerprint(node),
      };
      if (attempt.result) {
        if (
          attempt.result === "saved" &&
          nodeFingerprint(form) === nodeFingerprint(attempt.snapshot)
        ) {
          setForm(withoutSaveDiagnostics(node));
        } else if (attempt.result === "saved") {
          setSaveStatus("idle");
        }
        saveAttempt.current = null;
      }
      return;
    }

    if (nodeFingerprint(form) !== seededNodeRef.current.fingerprint) return;
    const decision = nextDraftOnEntityChange<FocusNode, number>({
      prevSeededId: seededNodeRef.current.id,
      prevFingerprint: seededNodeRef.current.fingerprint,
      entityId: node.id,
      entity: withoutSaveDiagnostics(node),
      currentDraft: form,
      fingerprintOf: nodeFingerprint,
      syncFieldsOnSameId: ["icon", "tree_edit_revision"],
    });
    seededNodeRef.current = {
      id: decision.seededId,
      fingerprint: decision.fingerprint,
    };
    if (decision.reseeded || decision.draft !== form) {
      setForm(decision.draft);
    }
    if (decision.reason === "switch") {
      saveAttempt.current = null;
      setSaving(false);
      setSaveStatus("idle");
      setSaveError(null);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [node, nodeFingerprint, identity, saveAttemptVersion]);

  useEffect(() => {
    onDraftChange?.(form);
  }, [form, onDraftChange]);

  const update = useCallback((field: string, value: unknown) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    setSaveStatus("idle");
  }, []);

  const persistForm = async (payload: Partial<FocusNode>, snapshot = form) => {
    const feedback = ++feedbackVersion.current;
    const attempt = { identity, snapshot } as NonNullable<
      typeof saveAttempt.current
    >;
    saveAttempt.current = attempt;
    setSaveAttemptVersion((version) => version + 1);
    setSaving(true);
    setSaveStatus("idle");
    setSaveError(null);
    try {
      await onSave({
        ...withoutSaveDiagnostics(payload),
        tree_edit_revision: snapshot.tree_edit_revision,
      });
      if (
        saveAttempt.current !== attempt ||
        activeIdentity.current !== identity
      )
        return;
      attempt.result = "saved";

      setSaveStatus("success");
      setTimeout(() => {
        if (
          activeIdentity.current === identity &&
          feedbackVersion.current === feedback
        )
          setSaveStatus("idle");
      }, 2000);
    } catch (err) {
      if (
        saveAttempt.current !== attempt ||
        activeIdentity.current !== identity
      )
        return;
      attempt.result = "failed";
      setSaveError(extractSaveErrorMessage(err));
      setSaveStatus("error");
    } finally {
      if (
        saveAttempt.current === attempt &&
        activeIdentity.current === identity
      ) {
        setSaving(false);
        setSaveAttemptVersion((version) => version + 1);
      }
    }
  };

  const canFork = !!onForked && typeof treeId === "number";
  const sourceGate = useImportedSourceGate<Partial<FocusNode>>({
    entitySource: node.source,
    performSave: (payload) => persistForm(payload, { ...form, ...payload }),
    performFork: canFork
      ? async (newFocusId) => {
          setSaving(true);
          setSaveStatus("idle");
          setSaveError(null);
          try {
            const forked = await focusTreesApi.forkNode(
              projectId,
              treeId!,
              node.id,
              newFocusId,
              form.tree_edit_revision,
            );
            setSaveStatus("success");
            setTimeout(() => setSaveStatus("idle"), 2000);
            onForked!(forked);
          } catch (err) {
            console.error("Fork failed:", err);
            setSaveError(extractSaveErrorMessage(err, "Fork failed"));
            setSaveStatus("error");
          } finally {
            setSaving(false);
          }
        }
      : undefined,
    resetForm: () => setForm(withoutSaveDiagnostics(node)),
  });

  const handleSave = async () => {
    const trimmedId = (form.focus_id || "").trim();

    const idValidation = validateRequiredId(trimmedId, "Focus ID");
    if (idValidation) {
      setIdError(idValidation);
      setSaveError(idValidation);
      setSaveStatus("error");
      return;
    }
    setIdError(null);

    if (await baseModId.verify(trimmedId)) return;
    const payload: Partial<FocusNode> = {
      ...form,
      focus_id: trimmedId,
    };
    if (sourceGate.interceptSave(payload)) return;
    await persistForm(payload);
  };

  const toggleSearchFilter = (filter: string) => {
    const current = form.search_filters || [];
    const next = current.includes(filter)
      ? current.filter((f) => f !== filter)
      : [...current, filter];
    update("search_filters", next);
  };

  const addWarTag = () => {
    const tag = warTagInput.trim().toUpperCase();

    if (!(tag.length === 3 && /^[A-Z0-9]{3}$/.test(tag))) return;
    const current = form.will_lead_to_war_with || [];
    if (!current.includes(tag)) {
      update("will_lead_to_war_with", [...current, tag]);
    }
    setWarTagInput("");
  };

  const removeWarTag = (tag: string) => {
    update(
      "will_lead_to_war_with",
      (form.will_lead_to_war_with || []).filter((tnext) => tnext !== tag),
    );
  };

  const addPrereqGroup = () => {
    const current = form.prerequisites || [];
    update("prerequisites", [...current, []]);
  };

  const updatePrereqGroup = (groupIndex: number, value: string) => {
    const current = [...(form.prerequisites || [])];
    current[groupIndex] = value
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
    update("prerequisites", current);
  };

  const removePrereqGroup = (groupIndex: number) => {
    const current = [...(form.prerequisites || [])];
    current.splice(groupIndex, 1);
    update("prerequisites", current);
  };

  const addMutuallyExclusive = (focusId: string) => {
    if (!focusId) return;
    const current = form.mutually_exclusive || [];
    if (!current.includes(focusId)) {
      update("mutually_exclusive", [...current, focusId]);
    }
  };

  const removeMutuallyExclusive = (focusId: string) => {
    const current = form.mutually_exclusive || [];
    update(
      "mutually_exclusive",
      current.filter((f) => f !== focusId),
    );
  };

  const otherNodes = allNodes.filter((n) => n.id !== node.id);

  return (
    <TooltipProvider>
      <div
        className={`${compact ? "p-4 space-y-4" : "p-6 space-y-6"} max-w-2xl mx-auto`}
      >
        {(saving || saveStatus !== "idle" || isUnsaved) && (
          <div
            className={`
            sticky top-0 z-10 ${compact ? "-mx-4 -mt-4 px-4" : "-mx-6 -mt-6 px-6"} py-1.5 text-xs font-medium flex items-center gap-2 shrink-0
            transition-[opacity,background-color,border-color] duration-300
            ${
              saving
                ? "bg-muted/80 text-muted-foreground border-b border-border"
                : saveStatus === "success"
                  ? "bg-success/10 text-success border-b border-success/20"
                  : saveStatus === "error"
                    ? "bg-destructive/10 text-destructive border-b border-destructive/20"
                    : isUnsaved
                      ? "bg-warning/10 text-warning border-b border-warning/20"
                      : ""
            }
          `}
          >
            <span>
              {saving
                ? t("saving")
                : saveStatus === "success"
                  ? t("saveSuccess")
                  : saveStatus === "error"
                    ? (saveError ?? t("saveError"))
                    : isUnsaved
                      ? t("unsavedChanges")
                      : ""}
            </span>
          </div>
        )}
        <SaveDiagnostics diagnostics={diagnostics} />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("basicInfoSection")}
          </h3>

          <div className="space-y-1">
            <Label>{t("displayNameLabel")}</Label>
            <DisplayNameLocsTabs
              entityKey={`focus-node:${node.id}:display-name`}
              value={form.display_name_locs || {}}
              exportLanguages={exportLanguages}
              legacyFallback={{ en: form.name_en, zh: form.name_zh }}
              onChange={(next) => update("display_name_locs", next)}
              placeholder={t("namePlaceholder")}
              ariaLabel="Focus node display name"
            />
            <p className={HINT_CLASS}>{t("nameHint")}</p>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label>{t("focusIdLabel")}</Label>
              <Input
                value={form.focus_id}
                onChange={(e) => {
                  if (idError) setIdError(null);
                  update("focus_id", e.target.value);
                }}
                aria-invalid={idError || baseModId.collision ? true : undefined}
                className={`font-mono text-xs ${idError || baseModId.collision ? "border-destructive" : ""}`}
                placeholder="e.g. FRA_Economic_recovery"
              />
              {idError && <p className="text-xs text-destructive">{idError}</p>}
              <BaseModIdCollisionNotice
                entityType="focus"
                collision={baseModId.collision}
              />
            </div>
            <div className="space-y-1">
              <Label>{t("iconLabel")}</Label>
              <div className="flex gap-1">
                <Input
                  value={form.icon}
                  onChange={(e) => update("icon", e.target.value)}
                  className="font-mono text-xs flex-1 min-w-0"
                  placeholder={t("iconPlaceholder")}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setIconLibraryOpen(true)}
                  title={t("iconLibraryTitle")}
                  className="shrink-0"
                >
                  {t("iconLibraryButton")}
                </Button>
              </div>
            </div>
          </div>

          <FocusIconLibraryDialog
            open={iconLibraryOpen}
            onOpenChange={setIconLibraryOpen}
            projectId={projectId}
            currentIcon={form.icon}
            onSelect={(gfxName) => {
              setForm((f) => ({ ...f, icon: gfxName }));
            }}
          />

          {treeId && form.focus_id ? (
            <details
              open={iconOpen}
              onToggle={(e) => setIconOpen(e.currentTarget.open)}
              className="group rounded-md border border-border bg-card/50"
            >
              <summary className="ui-interactive flex cursor-pointer select-none items-center gap-2 rounded-md px-3 py-2 hover:bg-card/80">
                <span className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
                  {t("iconLabel")}
                </span>
                <span className="ml-auto min-w-0 truncate font-mono text-xs text-muted-foreground">
                  {form.icon || t("iconSectionEmpty")}
                </span>
              </summary>
              <div className="space-y-3 border-t border-border p-3">
                <ImageUpload
                  key={String(treeId) + ":" + node.id}
                  label={t("uploadLabel")}
                  hideLabel
                  targetSize={FOCUS_ICON_SIZE}
                  previewUrl={
                    form.icon
                      ? focusTreesApi.nodeIconUrl(projectId, form.icon)
                      : null
                  }

                  fallbackUrl={
                    form.icon &&
                    (form.icon.startsWith("GFX_focus_") ||
                      form.icon.startsWith("GFX_goal_"))
                      ? `${API_BASE}/go/vanilla/focus-icon/${encodeURIComponent(form.icon)}`
                      : null
                  }
                  baseModFallbackUrls={baseModIconFallbacks}
                  gfxName={form.icon || undefined}
                  hint={t("uploadHint")}
                  onPrepareUpload={prepareIconUpload}
                  onUpload={uploadIconFile}
                  onUploaded={afterIconUploaded}
                  previewVersion={iconVersion}
                />
              </div>
            </details>
          ) : null}

          <div className="space-y-1">
            <Label>{t("positionSystem")}</Label>
            <Select
              value={form.relative_position_id || ""}
              onChange={(e) => update("relative_position_id", e.target.value)}
              title={
                form.relative_position_id
                  ? t("relativeToTitle", { id: form.relative_position_id })
                  : undefined
              }
            >
              <option value="">{t("noneAbsolutePosition")}</option>
              {otherNodes.map((n) => (
                <option key={n.id} value={n.focus_id}>
                  {n.focus_id}
                </option>
              ))}
            </Select>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1">
              <Label>{t("costWeeksLabel")}</Label>
              <Input
                type="number"
                min={1}
                max={50}
                value={form.cost}
                onChange={(e) => {
                  const n = Number(e.target.value);
                  if (!Number.isFinite(n) || e.target.value.trim() === "") {
                    return;
                  }
                  update("cost", Math.min(50, Math.max(1, n)));
                }}
              />
            </div>
            <div className="space-y-1">
              <Label>
                {form.relative_position_id
                  ? t("xOffsetLabel")
                  : t("xPositionLabel")}
              </Label>
              <Input
                type="number"
                value={form.x}
                onChange={(e) => update("x", Number(e.target.value))}
              />
            </div>
            <div className="space-y-1">
              <Label>
                {form.relative_position_id
                  ? t("yOffsetLabel")
                  : t("yPositionLabel")}
              </Label>
              <Input
                type="number"
                value={form.y}
                onChange={(e) => update("y", Number(e.target.value))}
              />
            </div>
          </div>

          {!form.relative_position_id &&
            (() => {
              const collisions = allNodes.filter(
                (n) =>
                  n.id !== node.id &&
                  !(n.relative_position_id || "") &&
                  n.x === form.x &&
                  n.y === form.y,
              );
              if (collisions.length === 0) return null;
              return (
                <div
                  className="inline-flex items-center gap-1 rounded-md border border-accent bg-accent/15 px-2 py-1 text-xs text-foreground"
                  title={t("positionCollisionTitle")}
                >
                  <span className="font-medium">
                    {t("positionCollisionLabel")}
                  </span>
                  <span className="font-mono">
                    {collisions.map((n) => n.focus_id).join(", ")}
                  </span>
                </div>
              );
            })()}
        </section>

        <Separator />

        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
              {t("prerequisitesSection")}
            </h3>
            <Button size="sm" variant="outline" onClick={addPrereqGroup}>
              {t("addGroup")}
            </Button>
          </div>
          <p className={HINT_CLASS}>{t("prereqLogicNote")}</p>
          {(form.prerequisites || []).length === 0 && (
            <p className="text-xs text-muted-foreground italic">
              {t("noPrerequisites")}
            </p>
          )}
          {(form.prerequisites || []).map((group, i) => (
            <div key={i} className="flex gap-2 items-center">
              <span className="text-xs text-muted-foreground shrink-0 w-16">
                {i === 0 ? t("requiresConnector") : t("andConnector")}
              </span>

              <Input
                value={group.join(", ")}
                onChange={(e) => updatePrereqGroup(i, e.target.value)}
                className="flex-1 font-mono text-xs"
                placeholder="FOCUS_A, FOCUS_B (comma = OR)"
              />
              <Button
                size="sm"
                variant="ghost"
                onClick={() => removePrereqGroup(i)}
              >
                x
              </Button>
            </div>
          ))}
        </section>

        <Separator />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("mutuallyExclusiveSection")}
          </h3>
          <div className="flex gap-2">
            <Select
              className="flex-1"
              onChange={(e) => {
                addMutuallyExclusive(e.target.value);
                e.target.value = "";
              }}
              defaultValue=""
            >
              <option value="" disabled>
                {t("addMutuallyExclusive")}
              </option>
              {otherNodes
                .filter(
                  (n) => !(form.mutually_exclusive || []).includes(n.focus_id),
                )
                .map((n) => (
                  <option key={n.id} value={n.focus_id}>
                    {n.focus_id}
                  </option>
                ))}
            </Select>
          </div>
          <div className="flex flex-wrap gap-1">
            {(form.mutually_exclusive || []).map((fid) => (
              <Badge
                key={fid}
                variant="default"
                className="cursor-pointer text-xs"
                onClick={() => removeMutuallyExclusive(fid)}
              >
                {fid} x
              </Badge>
            ))}
          </div>
        </section>

        <Separator />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("searchFiltersSection")}
          </h3>
          <div className="flex flex-wrap gap-1">
            {SEARCH_FILTER_OPTIONS.map((filter) => (
              <Badge
                key={filter}
                variant={
                  (form.search_filters || []).includes(filter)
                    ? "default"
                    : "outline"
                }
                className="cursor-pointer text-xs"
                onClick={() => toggleSearchFilter(filter)}
              >
                {filter.replace("FOCUS_FILTER_", "")}
              </Badge>
            ))}
          </div>
        </section>

        <Separator />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("completionFlagsSection")}
          </h3>

          <div className="space-y-2.5">
            {(
              [
                ["cancel_if_invalid", true, "cancelIfInvalidHint"],
                ["continue_if_invalid", false, "continueIfInvalidHint"],
                [
                  "available_if_capitulated",
                  false,
                  "availableIfCapitulatedHint",
                ],
              ] as const
            ).map(([field, fallback, hintKey]) => (
              <label
                key={field}
                className="flex cursor-pointer items-start gap-2 text-sm"
              >
                <input
                  type="checkbox"
                  className="mt-0.5 shrink-0"
                  checked={form[field] ?? fallback}
                  onChange={(e) => update(field, e.target.checked)}
                />
                <span className="min-w-0 flex-1">
                  <span className="font-mono">{field}</span>
                  <span className="mt-0.5 block text-xs leading-snug text-muted-foreground">
                    {t(hintKey)}
                  </span>
                </span>
              </label>
            ))}
          </div>
          <div className="space-y-1">
            <Label className="font-mono">will_lead_to_war_with</Label>
            <p className="text-xs text-muted-foreground">
              {t("willLeadToWarWithHint")}
            </p>
            <div className="flex gap-2">
              <Input
                value={warTagInput}
                onChange={(e) => setWarTagInput(e.target.value.toUpperCase())}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    addWarTag();
                  }
                }}
                placeholder="USA / GER / SOV"
                className="h-8 text-sm font-mono"
                maxLength={3}
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                className=""
                onClick={addWarTag}
              >
                {t("addWarTagButton")}
              </Button>
            </div>
            <div className="flex flex-wrap gap-1">
              {(form.will_lead_to_war_with || []).map((tag) => (
                <Badge
                  key={tag}
                  variant="default"
                  className="cursor-pointer text-xs font-mono"
                  onClick={() => removeWarTag(tag)}
                >
                  {tag} x
                </Badge>
              ))}
            </div>
          </div>
        </section>

        <Separator />

        <section className="space-y-4">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("scriptBlocksSection")}
          </h3>

          <ScriptBlockEditor
            label={t("completionRewardLabel")}
            value={form.completion_reward}
            onChange={(v) => update("completion_reward", v)}
            placeholder="add_political_power = 100"
            templates={EFFECT_TEMPLATES}
            minHeight="min-h-24"
          >
            {(insertCode) => (
              <>
                <IdeaReferenceDropdown
                  projectId={projectId}
                  countryTag={countryTag}
                  effectType="add_ideas"
                  buttonLabel={t("addNationalSpirit")}
                  onSelect={insertCode}
                />
                <IdeaReferenceDropdown
                  projectId={projectId}
                  countryTag={countryTag}
                  effectType="remove_ideas"
                  buttonLabel={t("removeNationalSpirit")}
                  onSelect={insertCode}
                />
                <EventReferenceDropdown
                  projectId={projectId}
                  effectType="country_event"
                  buttonLabel={t("triggerEvent")}
                  onSelect={insertCode}
                />
                <DecisionReferenceDropdown
                  projectId={projectId}
                  effectType="activate_decision"
                  buttonLabel={t("activateDecision")}
                  onSelect={insertCode}
                />
                <CharacterReferenceDropdown
                  projectId={projectId}
                  effectType="promote_character"
                  buttonLabel={tc("promote")}
                  onSelect={insertCode}
                />
                <CharacterReferenceDropdown
                  projectId={projectId}
                  effectType="recruit_character"
                  buttonLabel={tc("recruit")}
                  onSelect={insertCode}
                />
                <CharacterReferenceDropdown
                  projectId={projectId}
                  effectType="retire_character"
                  buttonLabel={tc("retire")}
                  onSelect={insertCode}
                />
              </>
            )}
          </ScriptBlockEditor>

          <div className="space-y-1">
            <Label>{t("effectTooltipLabel")}</Label>
            <DescriptionLocsTabs
              entityKey={`focus-node:${node.id}:effect-tooltip`}
              value={form.effect_tooltip_locs || {}}
              exportLanguages={exportLanguages}
              onChange={(next) => update("effect_tooltip_locs", next)}
              ariaLabel="Focus effect tooltip"
              rows={2}
            />
            <p className="text-xs text-muted-foreground">
              {t("effectTooltipHint")}
            </p>
          </div>

          <ScriptBlockEditor
            label={t("selectEffectLabel")}
            labelTooltip={t("selectEffectHint")}
            value={form.select_effect}
            onChange={(v) => update("select_effect", v)}
            placeholder="set_country_flag = branch_locked_in"
            templates={EFFECT_TEMPLATES}
            minHeight="min-h-12"
          />

          <ScriptBlockEditor
            label={t("availableLabel")}
            value={form.available}
            onChange={(v) => update("available", v)}
            placeholder="has_civil_war = no"
            templates={CONDITION_TEMPLATES}
          >
            {(insertCode) => (
              <>
                <IdeaReferenceDropdown
                  projectId={projectId}
                  countryTag={countryTag}
                  effectType="has_idea"
                  buttonLabel={t("hasNationalSpirit")}
                  onSelect={insertCode}
                />
                <FocusReferenceDropdown
                  projectId={projectId}
                  effectType="has_completed_focus"
                  buttonLabel={t("hasCompletedFocus")}
                  onSelect={insertCode}
                />
                <DecisionReferenceDropdown
                  projectId={projectId}
                  effectType="has_decision"
                  buttonLabel={t("hasDecision")}
                  onSelect={insertCode}
                />
              </>
            )}
          </ScriptBlockEditor>

          <ScriptBlockEditor
            label={t("bypassLabel")}
            value={form.bypass}
            onChange={(v) => update("bypass", v)}
            minHeight="min-h-12"
          />

          <ScriptBlockEditor
            label={t("bypassEffectLabel")}
            value={String(
              (form.raw_extras as Record<string, unknown> | undefined)
                ?.bypass_effect ?? "",
            )}
            onChange={(v) => {
              const next = {
                ...((form.raw_extras as Record<string, unknown>) || {}),
              };
              if (v && v.trim()) {
                next.bypass_effect = v;
              } else {
                delete next.bypass_effect;
              }
              update("raw_extras", next);
            }}
            placeholder={t("bypassEffectPlaceholder")}
            minHeight="min-h-12"
          />

          <ScriptBlockEditor
            label={t("cancelLabel")}
            value={form.cancel}
            onChange={(v) => update("cancel", v)}
            templates={CONDITION_TEMPLATES}
            minHeight="min-h-12"
          />

          <ScriptBlockEditor
            label="ai_will_do"
            value={form.ai_will_do}
            onChange={(v) => update("ai_will_do", v)}
            placeholder="factor = 1"
            minHeight="min-h-12"
          />

          <VisualPolishSection
            textIcon={form.text_icon || ""}
            offset={form.offset || []}
            dynamicIcon={form.dynamic_icon || []}
            onChangeTextIcon={(v) => update("text_icon", v)}
            onChangeOffset={(v) => update("offset", v)}
            onChangeDynamicIcon={(v) => update("dynamic_icon", v)}
            projectId={projectId}
          />

          {(() => {
            const extras = (form.raw_extras as Record<string, unknown>) || {};
            const promotedKeys = new Set(["bypass_effect"]);
            const visibleEntries = Object.entries(extras).filter(
              ([k]) => !promotedKeys.has(k),
            );
            if (visibleEntries.length === 0) return null;
            return (
              <>
                <button
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors mt-2"
                  onClick={() => setShowAdvanced(!showAdvanced)}
                >
                  <span>{showAdvanced ? "▼" : "▶"}</span>
                  {t("preservedFieldsHeader", { count: visibleEntries.length })}
                </button>
                {showAdvanced && (
                  <div className="space-y-3 pl-3 border-l-2 border-muted">
                    <p className="text-xs text-muted-foreground">
                      {t("preservedFieldsNote")}
                    </p>
                    {visibleEntries.map(([key, value]) => (
                      <div key={key} className="space-y-1">
                        <Label className="font-mono">{key}</Label>
                        <Textarea
                          value={String(value)}
                          onChange={(e) => {
                            const newExtras = {
                              ...extras,
                              [key]: e.target.value,
                            };
                            update("raw_extras", newExtras);
                          }}
                          className="font-mono text-xs min-h-12"
                        />
                      </div>
                    ))}
                  </div>
                )}
              </>
            );
          })()}
        </section>

        <Separator />

        <section className="space-y-3">
          <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
            {t("descriptionSection")}
          </h3>
          <DescriptionLocsTabs
            entityKey={`focus-node:${node.id}:description`}
            value={form.description_locs || {}}
            exportLanguages={exportLanguages}
            legacyFallback={{ en: form.desc_en, zh: form.desc_zh }}
            onChange={(next) => update("description_locs", next)}
            placeholder={t("descPlaceholder")}
            ariaLabel="Focus node description"
            rows={3}
          />
        </section>

        <Separator />

        <Button
          onClick={handleSave}
          disabled={saving || !!baseModId.collision}
          className="w-full"
        >
          {saving ? t("saving") : t("saveButton")}
        </Button>
      </div>
      <SourceEditorDialog
        open={sourceGate.showDialog}
        entityType="focus_node"
        entityId={node.focus_id}
        entityLabel={node.name_en || node.name || node.focus_id}
        hideForkOption={!canFork}
        onChoice={sourceGate.handleChoice}
        onCancel={sourceGate.handleCancel}
      />
    </TooltipProvider>
  );
}
