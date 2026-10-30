"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { WorkbenchRightPane } from "@/components/workbench/WorkbenchRightPane";
import { firstBaseModDepKey } from "@/components/workbench/BaseModFlagPreview";
import { CountryFlagPreview } from "@/components/workbench/country-profile/CountryFlagPreview";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { ApiError } from "@/lib/api/_client";
import {
  countriesApi,
  historyFilesApi,
  type CountryRow,
  type HistoryFileRow,
} from "@/lib/api";
import {
  politicsApi,
  type PoliticsDefinition,
  type PoliticsDeletePreview,
  type PoliticsIdeology,
  type PoliticsIdeologyConflict,
  type PoliticsReference,
  type PoliticsReferenceSnapshot,
  type PoliticsSourceOwnership,
} from "@/lib/api/politics";
import { projectsApi } from "@/lib/api/project";
import { ideologyIconsApi } from "@/lib/api/tooling";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import {
  normalizeRegistry,
  type IdeologyLocalisationMap,
  type IdeologyRegistry,
} from "@/lib/ideology/registry";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { useActiveCountry, useProjectStore } from "@/lib/store/projectStore";
import {
  defaultRootBody,
  rootBodyColor,
  rootBodyDraftError,
  rootBodyWithColor,
} from "./rootBody";
import { IdeologyEditor } from "./IdeologyEditor";
import {
  definitionDraft,
  definitionDraftDirty,
  definitionPatch,
  ideologyDisplayName,
  localizedValue,
  normalizedLocaleMap,
  projectDefinition,
  referenceKey,
  rootSwatchColor,
  suggestedPoliticsScriptId,
  targetKey,
  type CreateIntent,
  type IdeologyDefinitionDraft,
  type RootRuleMode,
} from "./ideologyDefinition";
import {
  PoliticsInGamePreview,
  type PoliticsPreviewCountry,
  type PoliticsPreviewGroup,
  type PoliticsPreviewTarget,
} from "./PoliticsInGamePreview";

export interface IdeologyLibraryWorkbenchProps {
  projectId: number;
  initialItemId?: string;
  onItemChange?: (scriptId: string) => void;
  onChanged?: () => void | Promise<void>;
  onOpenCountryPolitics?: () => void;
  onAssignLeader?: (subtypeId: string) => void;
  className?: string;
}

function historyPopularities(
  history: HistoryFileRow | null,
): Record<string, number> {
  if (!history) return {};
  if (
    history.set_popularities &&
    Object.keys(history.set_popularities).length > 0
  ) {
    return history.set_popularities;
  }
  return {
    democratic: history.set_popularities_democratic ?? 0,
    communism: history.set_popularities_communism ?? 0,
    fascism: history.set_popularities_fascism ?? 0,
    neutrality: history.set_popularities_neutrality ?? 0,
  };
}

function errorText(value: unknown): string {
  return value instanceof Error ? value.message : String(value);
}

function politicsErrorPayload(value: unknown): Record<string, unknown> | null {
  if (
    !(value instanceof ApiError) ||
    !value.payload ||
    typeof value.payload !== "object"
  ) {
    return null;
  }
  return value.payload as Record<string, unknown>;
}

export function IdeologyLibraryWorkbench({
  projectId,
  initialItemId,
  onItemChange,
  onChanged,
  onOpenCountryPolitics,
  onAssignLeader,
  className = "",
}: IdeologyLibraryWorkbenchProps) {
  const t = useTranslations("politicsPanel");
  const contentLocale = useContentLocaleStore((state) => state.locale);
  const activeCountryTag = useActiveCountry();
  const exportLanguages = useProjectStore(
    (state) =>
      state.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );
  const projectDependencies = useProjectStore(
    (state) => state.currentProject?.dependencies,
  );
  const baseModDepKey = useMemo(
    () => firstBaseModDepKey(projectDependencies),
    [projectDependencies],
  );
  const [snapshot, setSnapshot] = useState<PoliticsReferenceSnapshot | null>(
    null,
  );
  const [registry, setRegistry] = useState<IdeologyRegistry>(() =>
    normalizeRegistry(undefined),
  );
  const [iconPaths, setIconPaths] = useState<Record<string, string>>({});
  const [previewCountrySource, setPreviewCountrySource] = useState<{
    country: CountryRow;
    history: HistoryFileRow | null;
  } | null>(null);
  const [iconBust, setIconBust] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [loadErrorCode, setLoadErrorCode] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [editDraft, setEditDraft] = useState<IdeologyDefinitionDraft>(() =>
    definitionDraft({ name: {}, desc: {}, randomDisabled: false, body: "" }),
  );
  const [uploadingIcon, setUploadingIcon] = useState(false);
  const [previewing, setPreviewing] = useState(false);
  const [applyingDelete, setApplyingDelete] = useState(false);
  const [deletePreview, setDeletePreview] =
    useState<PoliticsDeletePreview | null>(null);
  const [creatorOpen, setCreatorOpen] = useState(false);
  const [createIntent, setCreateIntent] = useState<CreateIntent | null>(null);
  const [createName, setCreateName] = useState<IdeologyLocalisationMap>({});
  const [createDesc, setCreateDesc] = useState<IdeologyLocalisationMap>({});
  const [createScriptId, setCreateScriptId] = useState("new_subtype");
  const [createScriptIdTouched, setCreateScriptIdTouched] = useState(false);
  const [createParent, setCreateParent] = useState("");
  const [createRandomDisabled, setCreateRandomDisabled] = useState(false);
  const [createRootBody, setCreateRootBody] = useState(() => defaultRootBody());
  const [createRootColor, setCreateRootColor] = useState("#808080");
  const [createRuleMode, setCreateRuleMode] = useState<RootRuleMode>("safe");
  const [creating, setCreating] = useState(false);
  const [createdDefinition, setCreatedDefinition] =
    useState<PoliticsDefinition | null>(null);
  const [activePaneTab, setActivePaneTab] = useState("live");
  const [activePreviewTarget, setActivePreviewTarget] =
    useState<PoliticsPreviewTarget | null>(null);
  const [scriptEditing, setScriptEditing] = useState(false);
  const previousInitialItemIdRef = useRef<string | undefined>(undefined);
  const addIdeologyButtonRef = useRef<HTMLButtonElement>(null);
  const firstIntentButtonRef = useRef<HTMLButtonElement>(null);
  const createNameFieldRef = useRef<HTMLDivElement>(null);
  const createdTitleRef = useRef<HTMLHeadingElement>(null);
  const pendingCreatorFocusRef = useRef<
    "trigger" | "intent" | "form" | "success" | null
  >(null);
  const deletePreviewButtonRef = useRef<HTMLButtonElement>(null);
  const deleteCancelButtonRef = useRef<HTMLButtonElement>(null);
  const selectedIdeologyButtonRef = useRef<HTMLButtonElement>(null);
  const scriptTextareaRef = useRef<HTMLTextAreaElement>(null);
  const pendingDeleteFocusRef = useRef<
    "trigger" | "preview" | "selection" | null
  >(null);
  const deletedSelectionKeyRef = useRef<string | null>(null);

  const refresh = useCallback(
    async (signal?: AbortSignal) => {
      setLoading(true);
      setLoadError(null);
      setLoadErrorCode(null);
      try {
        const [nextSnapshot, project] = await Promise.all([
          politicsApi.references(projectId, signal),
          projectsApi.get(projectId),
        ]);
        if (signal?.aborted) return;
        setSnapshot(nextSnapshot);
        setRegistry(normalizeRegistry(project.ideology_registry));
        setIconPaths(project.ideology_icon_paths ?? {});

        const countryTag = (
          activeCountryTag ||
          project.country_tag ||
          ""
        ).trim();
        if (!countryTag) {
          setPreviewCountrySource(null);
        } else {
          const [countriesResult, historiesResult] = await Promise.allSettled([
            countriesApi.list(projectId),
            historyFilesApi.list(projectId),
          ]);
          if (signal?.aborted) return;
          const country =
            countriesResult.status === "fulfilled"
              ? (countriesResult.value.countries.find(
                  (item) => item.tag.toUpperCase() === countryTag.toUpperCase(),
                ) ?? null)
              : null;
          const history =
            historiesResult.status === "fulfilled"
              ? (historiesResult.value.find(
                  (item) =>
                    item.country_tag.toUpperCase() === countryTag.toUpperCase(),
                ) ?? null)
              : null;
          setPreviewCountrySource(country ? { country, history } : null);
        }
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError")
          return;
        setLoadError(errorText(error));
        const payload = politicsErrorPayload(error);
        setLoadErrorCode(
          typeof payload?.code === "string" ? payload.code : null,
        );
      } finally {
        if (!signal?.aborted) setLoading(false);
      }
    },
    [activeCountryTag, projectId],
  );

  useEffect(() => {
    const controller = new AbortController();
    void refresh(controller.signal);
    return () => controller.abort();
  }, [refresh]);

  const ideologies = useMemo(
    () => snapshot?.ideologies ?? [],
    [snapshot?.ideologies],
  );

  const conflicts = useMemo(
    () => snapshot?.conflicts ?? [],
    [snapshot?.conflicts],
  );
  const [discardingConflict, setDiscardingConflict] = useState<string | null>(
    null,
  );

  const discardConflict = useCallback(
    async (scriptId: string) => {
      setDiscardingConflict(scriptId);
      setActionError(null);
      setStatusMessage(null);
      try {
        const usageCount =
          conflicts.find((item) => item.script_id === scriptId)?.usage_count ??
          0;
        await politicsApi.discardConflict(projectId, scriptId);
        await refresh();

        setStatusMessage(
          usageCount > 0
            ? t("conflictDiscardedWithUse", { id: scriptId, count: usageCount })
            : t("conflictDiscarded", { id: scriptId }),
        );
        await onChanged?.();
      } catch (error) {
        setActionError(t("conflictDiscardFailed", { error: errorText(error) }));
      } finally {
        setDiscardingConflict(null);
      }
    },
    [projectId, refresh, t, onChanged, conflicts],
  );

  const describeConflict = useCallback(
    (conflict: PoliticsIdeologyConflict) => {
      switch (conflict.reason) {
        case "duplicate_script_id":
          return t("conflictReasonDuplicateId");
        case "missing_root":
          return t("conflictReasonMissingRoot");
        case "unknown_root":
          return t("conflictReasonUnknownRoot", {
            root: conflict.root_id ?? "",
          });
        case "malformed_definition":
          return t("conflictReasonMalformed");
        default:
          return conflict.detail;
      }
    },
    [t],
  );
  const roots = useMemo(
    () => ideologies.filter((item) => item.kind === "root"),
    [ideologies],
  );
  const previewRootIds = useMemo(
    () => roots.map((root) => root.script_id),
    [roots],
  );
  const subtypesByRoot = useMemo(() => {
    const result = new Map<string, PoliticsIdeology[]>();
    for (const subtype of ideologies.filter(
      (item) => item.kind === "subtype",
    )) {
      const root = subtype.root_id ?? "";
      result.set(root, [...(result.get(root) ?? []), subtype]);
    }
    return result;
  }, [ideologies]);

  useEffect(() => {
    if (ideologies.length === 0) {
      setSelectedKey(null);
      return;
    }
    const initialItemChanged =
      previousInitialItemIdRef.current !== initialItemId;
    previousInitialItemIdRef.current = initialItemId;
    const requested = initialItemId
      ? ideologies.find((item) => item.script_id === initialItemId)
      : null;
    const existing = selectedKey
      ? ideologies.find((item) => targetKey(item) === selectedKey)
      : null;
    const next = initialItemChanged
      ? (requested ?? existing ?? roots[0] ?? ideologies[0])
      : (existing ?? requested ?? roots[0] ?? ideologies[0]);
    if (next && targetKey(next) !== selectedKey) {
      setSelectedKey(targetKey(next));
      onItemChange?.(next.script_id);
    }
  }, [ideologies, initialItemId, onItemChange, roots, selectedKey]);

  const selected = useMemo(
    () => ideologies.find((item) => targetKey(item) === selectedKey) ?? null,
    [ideologies, selectedKey],
  );

  const selectedDefinition = useMemo(
    () => (selected ? projectDefinition(registry, selected) : null),
    [registry, selected],
  );
  const visibleNameFor = useCallback(
    (ideology: PoliticsIdeology) =>
      ideologyDisplayName(registry, ideology, contentLocale),
    [contentLocale, registry],
  );
  const editDirty = definitionDraftDirty(
    selected,
    selectedDefinition,
    editDraft,
  );
  const creatorDirty =
    creatorOpen && createIntent !== null && !createdDefinition;
  useRegisterDirty("ideologies", editDirty || creatorDirty);

  useEffect(() => {
    const pending = pendingCreatorFocusRef.current;
    if (!pending) return;

    const target =
      pending === "trigger"
        ? addIdeologyButtonRef.current
        : pending === "intent"
          ? firstIntentButtonRef.current
          : pending === "form"
            ? (createNameFieldRef.current?.querySelector<HTMLInputElement>(
                "input",
              ) ?? null)
            : createdTitleRef.current;
    if (!target) return;

    pendingCreatorFocusRef.current = null;
    target.focus();
  }, [createIntent, createdDefinition, creatorOpen]);

  useEffect(() => {
    const pending = pendingDeleteFocusRef.current;
    if (!pending) return;
    if (
      pending === "selection" &&
      selectedKey === deletedSelectionKeyRef.current
    )
      return;

    const target =
      pending === "trigger"
        ? deletePreviewButtonRef.current
        : pending === "preview"
          ? deleteCancelButtonRef.current
          : selectedIdeologyButtonRef.current;
    if (!target) return;

    pendingDeleteFocusRef.current = null;
    deletedSelectionKeyRef.current = null;
    target.focus();
  }, [deletePreview, selectedKey]);

  useEffect(() => {
    if (!selected) return;
    const definition = projectDefinition(registry, selected);
    setEditDraft(definitionDraft(definition));
    setActivePreviewTarget(null);
    setScriptEditing(false);
    setDeletePreview(null);
    setActionError(null);
    setStatusMessage(null);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [registry, selected?.kind, selected?.script_id]);

  useEffect(() => {
    if (!createParent && roots.length > 0)
      setCreateParent(roots[0]?.script_id ?? "");
  }, [createParent, roots]);

  const referencesByTarget = useMemo(() => {
    const result = new Map<string, PoliticsReference[]>();
    for (const reference of snapshot?.references ?? []) {
      const key = targetKey(reference.target);
      result.set(key, [...(result.get(key) ?? []), reference]);
    }
    return result;
  }, [snapshot?.references]);

  const selectedReferences = selected
    ? (referencesByTarget.get(targetKey(selected)) ?? [])
    : [];
  const previewCountry = useMemo<PoliticsPreviewCountry | null>(() => {
    if (!previewCountrySource) return null;
    const { country, history } = previewCountrySource;
    return {
      tag: country.tag,
      name: localizedValue(
        country.display_name_locs,
        contentLocale,
        country.name || country.tag,
      ),
      hasPoliticsState: Boolean(history),
      rulingParty: history?.set_politics_ruling_party ?? "",
      popularities: historyPopularities(history),
      electionsAllowed: history?.set_politics_elections_allowed ?? null,
      lastElection: history?.set_politics_last_election ?? "",
    };
  }, [contentLocale, previewCountrySource]);
  const selectedRoot = useMemo(() => {
    if (!selected) return null;
    return selected.kind === "root"
      ? selected
      : (roots.find((root) => root.script_id === selected.root_id) ?? null);
  }, [roots, selected]);
  const selectedRootChildren = useMemo(() => {
    if (!selectedRoot) return [];
    const children = subtypesByRoot.get(selectedRoot.script_id) ?? [];
    const query = search.trim().toLocaleLowerCase();
    if (
      !query ||
      `${visibleNameFor(selectedRoot)} ${selectedRoot.script_id}`
        .toLocaleLowerCase()
        .includes(query)
    ) {
      return children;
    }
    return children.filter((subtype) =>
      `${visibleNameFor(subtype)} ${subtype.script_id}`
        .toLocaleLowerCase()
        .includes(query),
    );
  }, [search, selectedRoot, subtypesByRoot, visibleNameFor]);
  const selectedGroupId = selectedRoot?.script_id ?? "";
  const selectedVisibleName = selected
    ? editDraft.name[contentLocale]?.trim() || visibleNameFor(selected)
    : "";
  const selectedDescription = editDraft.desc[contentLocale]?.trim() || "";
  const selectedGroupLabel =
    selected?.kind === "root"
      ? selectedVisibleName
      : selectedRoot
        ? visibleNameFor(selectedRoot)
        : selectedGroupId;
  const previewGroups = useMemo<PoliticsPreviewGroup[]>(
    () =>
      roots.map((root, index) => {
        const savedColor = rootSwatchColor(registry, root.script_id, index);
        const isEditedRoot =
          selected?.kind === "root" && root.script_id === selected.script_id;
        return {
          id: root.script_id,
          label: isEditedRoot ? selectedVisibleName : visibleNameFor(root),
          color: isEditedRoot ? editDraft.rootColor : savedColor,
          value: previewCountry?.popularities[root.script_id] ?? 0,
        };
      }),
    [
      editDraft.rootColor,
      previewCountry?.popularities,
      registry,
      roots,
      selected,
      selectedVisibleName,
      visibleNameFor,
    ],
  );
  const selectedPreviewGroup = previewGroups.find(
    (group) => group.id === selectedGroupId,
  );
  const selectedIconUrl =
    selectedGroupId && iconPaths[selectedGroupId]
      ? `${ideologyIconsApi.previewUrl(projectId, selectedGroupId)}?bust=${iconBust}`
      : null;
  const editablePreviewTargets = useMemo<PoliticsPreviewTarget[]>(() => {
    if (selected?.source !== "project_managed") return [];
    return selected.kind === "root"
      ? ["name", "description", "color", "icon"]
      : ["name", "description"];
  }, [selected]);
  const createRootBodyError =
    createIntent === "root" ? rootBodyDraftError(createRootBody) : null;
  const editRootBodyError =
    selected?.kind === "root" ? rootBodyDraftError(editDraft.rootBody) : null;

  const selectPreviewTarget = useCallback((target: PoliticsPreviewTarget) => {
    setActivePaneTab("live");
    setActivePreviewTarget(target);
  }, []);

  const filteredRoots = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    if (!query) return roots;
    return roots.filter((root) => {
      const rootMatches = `${visibleNameFor(root)} ${root.script_id}`
        .toLocaleLowerCase()
        .includes(query);
      const childMatches = (subtypesByRoot.get(root.script_id) ?? []).some(
        (subtype) =>
          `${visibleNameFor(subtype)} ${subtype.script_id}`
            .toLocaleLowerCase()
            .includes(query),
      );
      return rootMatches || childMatches;
    });
  }, [roots, search, subtypesByRoot, visibleNameFor]);

  const selectIdeology = useCallback(
    (ideology: PoliticsIdeology) => {
      setSelectedKey(targetKey(ideology));
      onItemChange?.(ideology.script_id);
    },
    [onItemChange],
  );

  const sourceLabel = useCallback(
    (source: PoliticsSourceOwnership) => t(`source.${source}`),
    [t],
  );

  const saveDefinition = useCallback(async () => {
    if (!selected || selected.source !== "project_managed") return;
    const names = normalizedLocaleMap(editDraft.name);
    if (Object.keys(names).length === 0) return;
    if (selected.kind === "root" && editRootBodyError) {
      setActionError(t(`workbench.rootBodyError.${editRootBodyError}`));
      return;
    }
    setSaving(true);
    setActionError(null);
    setStatusMessage(null);
    try {
      await politicsApi.updateDefinition(
        projectId,
        {
          kind: selected.kind,
          script_id: selected.script_id,
        },
        definitionPatch(selected.kind, editDraft),
      );
      await refresh();
      setStatusMessage(t("workbench.saved"));
      await onChanged?.();
    } catch (error) {
      setActionError(t("workbench.updateFailed", { error: errorText(error) }));
    } finally {
      setSaving(false);
    }
  }, [
    editDraft,
    editRootBodyError,
    onChanged,
    projectId,
    refresh,
    selected,
    t,
  ]);

  const uploadIcon = useCallback(
    async (file: File) => {
      if (!selected || selected.kind !== "root") return;
      setUploadingIcon(true);
      setActionError(null);
      try {
        const project = await ideologyIconsApi.upload(
          projectId,
          selected.script_id,
          file,
        );
        setIconPaths(project.ideology_icon_paths ?? {});
        setIconBust((value) => value + 1);
        setStatusMessage(t("workbench.iconSaved"));
        await onChanged?.();
      } catch (error) {
        setActionError(t("iconUploadFailed", { error: errorText(error) }));
      } finally {
        setUploadingIcon(false);
      }
    },
    [onChanged, projectId, selected, t],
  );

  const previewDelete = useCallback(async () => {
    if (!selected || selected.source !== "project_managed") return;
    setPreviewing(true);
    setActionError(null);
    setDeletePreview(null);
    try {
      const preview = await politicsApi.deletePreview(projectId, {
        kind: selected.kind,
        script_id: selected.script_id,
      });
      pendingDeleteFocusRef.current = "preview";
      setDeletePreview(preview);
    } catch (error) {
      setActionError(t("deletePreviewFailed", { error: errorText(error) }));
    } finally {
      setPreviewing(false);
    }
  }, [projectId, selected, t]);

  const cancelDeletePreview = () => {
    pendingDeleteFocusRef.current = "trigger";
    setDeletePreview(null);
  };

  const applyDelete = useCallback(async () => {
    if (!deletePreview?.allowed || !deletePreview.token) return;
    setApplyingDelete(true);
    setActionError(null);
    try {
      await politicsApi.deleteApply(projectId, {
        preview_token: deletePreview.token,
      });
      pendingDeleteFocusRef.current = "selection";
      deletedSelectionKeyRef.current = selectedKey;
      setDeletePreview(null);
      setSelectedKey(null);
      await refresh();
      setStatusMessage(t("workbench.deleted"));
      await onChanged?.();
    } catch (error) {
      const payload = politicsErrorPayload(error);
      const authorization = payload?.authorization;
      if (
        payload?.code === "politics_delete_blocked" &&
        authorization &&
        typeof authorization === "object" &&
        Array.isArray((authorization as { impacts?: unknown }).impacts)
      ) {
        setDeletePreview((current) =>
          current
            ? {
                ...current,
                allowed: false,
                impacts: (
                  authorization as { impacts: PoliticsDeletePreview["impacts"] }
                ).impacts,
              }
            : current,
        );
      }
      setActionError(t("deleteApplyFailed", { error: errorText(error) }));
    } finally {
      setApplyingDelete(false);
    }
  }, [deletePreview, onChanged, projectId, refresh, selectedKey, t]);

  const createDefinition = useCallback(async () => {
    if (!createIntent) return;
    const names = normalizedLocaleMap(createName);
    const scriptId = createScriptId.trim();
    if (Object.keys(names).length === 0 || !scriptId) return;
    if (createIntent === "root" && createRootBodyError) {
      setActionError(t(`workbench.rootBodyError.${createRootBodyError}`));
      return;
    }
    setCreating(true);
    setActionError(null);
    setCreatedDefinition(null);
    try {
      const created =
        createIntent === "subtype"
          ? await politicsApi.createSubtype(projectId, {
              script_id: scriptId,
              root_id: createParent,
              name: names,
              desc: normalizedLocaleMap(createDesc),
              random_disabled: createRandomDisabled,
            })
          : await politicsApi.createRoot(projectId, {
              script_id: scriptId,
              body: createRootBody.trim(),
              name: names,
              desc: normalizedLocaleMap(createDesc),
            });
      pendingCreatorFocusRef.current = "success";
      setCreatedDefinition(created);
      await refresh();
      setSelectedKey(targetKey(created));
      onItemChange?.(created.script_id);
      await onChanged?.();
    } catch (error) {
      const payload = politicsErrorPayload(error);
      if (payload?.code === "politics_source_lock_required")
        setActionError(t("sourceLockRequired"));
      else if (payload?.code === "politics_root_takeover_out_of_scope")
        setActionError(t("createRootStandaloneOnly"));
      else
        setActionError(
          t(
            createIntent === "subtype"
              ? "createSubtypeFailed"
              : "createRootFailed",
            { error: errorText(error) },
          ),
        );
    } finally {
      setCreating(false);
    }
  }, [
    createDesc,
    createIntent,
    createName,
    createParent,
    createRandomDisabled,
    createRootBody,
    createRootBodyError,
    createScriptId,
    onChanged,
    onItemChange,
    projectId,
    refresh,
    t,
  ]);

  const openCreator = () => {
    pendingCreatorFocusRef.current = "intent";
    setCreatorOpen(true);
    setCreateIntent(null);
    setCreatedDefinition(null);
    setActionError(null);
    setCreateRootColor("#808080");
    setCreateRuleMode("safe");
    setCreateRootBody(defaultRootBody());
  };

  const closeCreator = () => {
    pendingCreatorFocusRef.current = "trigger";
    setCreatorOpen(false);
  };

  const chooseCreateIntent = (intent: CreateIntent) => {
    pendingCreatorFocusRef.current = "form";
    setCreateIntent(intent);
    if (intent === "root" && !createScriptIdTouched) {
      setCreateScriptId("new_political_system");
    }
  };

  const returnToCreateIntent = () => {
    pendingCreatorFocusRef.current = "intent";
    setCreateIntent(null);
  };

  const updateCreateName = (next: IdeologyLocalisationMap) => {
    const changedValue =
      Object.entries(next).find(
        ([locale, value]) =>
          createName[locale as keyof IdeologyLocalisationMap] !== value,
      )?.[1] ??
      Object.values(next).find((value) => value.trim()) ??
      "";
    setCreateName(next);
    if (!createScriptIdTouched) {
      setCreateScriptId(suggestedPoliticsScriptId(changedValue));
    }
  };

  const updateCreateRootColor = (color: string) => {
    setCreateRootColor(color);
    setCreateRootBody((current) =>
      createRuleMode === "safe"
        ? defaultRootBody(color)
        : rootBodyWithColor(current, color),
    );
  };

  return (
    <div className={`space-y-4 ${className}`} data-politics-ideology-library>
      <section className="ui-panel p-4" aria-labelledby="politics-source-title">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            <h2
              id="politics-source-title"
              className="text-lg font-semibold tracking-tight"
            >
              {t("workbench.projectSourceTitle")}
            </h2>
            <p className="max-w-3xl text-sm text-muted-foreground">
              {t("workbench.projectSourceSummary")}
            </p>
          </div>
          <span className="rounded-full border border-border bg-muted px-3 py-1 text-xs text-muted-foreground">
            {t("workbench.sourceCount", {
              project: ideologies.filter(
                (item) => item.source === "project_managed",
              ).length,
              inherited: ideologies.filter(
                (item) => item.source !== "project_managed",
              ).length,
            })}
          </span>
        </div>
        <p className="mt-3 text-xs text-muted-foreground">
          {t("workbench.evidenceBoundary")}
        </p>
      </section>

      {loadError && (
        <section
          role="alert"
          className="rounded-md border border-destructive/40 bg-destructive/5 p-3 space-y-2"
        >
          <p className="text-sm font-medium text-destructive">
            {t("coverageBlockedTitle")}
          </p>
          <p className="text-xs text-destructive">
            {loadErrorCode === "politics_reference_coverage_incomplete"
              ? t("coverageIncomplete", { error: loadError })
              : t("coverageBlocked", { error: loadError })}
          </p>
          <Button
            type="button"
            size="xs"
            variant="outline"
            onClick={() => void refresh()}
          >
            {t("retry")}
          </Button>
        </section>
      )}
      {conflicts.length > 0 && (
        <section
          role="alert"
          data-politics-ideology-conflicts
          className="rounded-md border border-warning/40 bg-warning/5 p-3 space-y-2"
        >
          <p className="text-sm font-medium text-warning-foreground">
            ⚠ {t("conflictsTitle", { count: conflicts.length })}
          </p>
          <p className="text-xs text-muted-foreground">{t("conflictsBody")}</p>
          <ul className="space-y-1 text-xs">
            {conflicts.map((conflict) => (
              <li
                key={`${conflict.reason}:${conflict.script_id}`}
                className="flex flex-wrap items-baseline gap-x-2"
              >
                <code className="rounded bg-muted px-1 py-0.5 font-mono">
                  {conflict.script_id}
                </code>
                <span className="text-muted-foreground">
                  {describeConflict(conflict)}
                </span>
                {conflict.usage_count > 0 && (
                  <span className="rounded-full border border-warning/40 px-2 py-0.5 text-micro text-warning-foreground">
                    {t("conflictInUse", { count: conflict.usage_count })}
                  </span>
                )}
                <Button
                  type="button"
                  size="xs"
                  variant="outline"
                  disabled={discardingConflict !== null}
                  onClick={() => void discardConflict(conflict.script_id)}
                >
                  {discardingConflict === conflict.script_id
                    ? t("conflictDiscarding")
                    : conflict.usage_count > 0
                      ? t("conflictDiscardWithUse", {
                          count: conflict.usage_count,
                        })
                      : t("conflictDiscard")}
                </Button>
              </li>
            ))}
          </ul>
          <p className="text-xs text-muted-foreground">{t("conflictsHint")}</p>
        </section>
      )}
      {actionError && (
        <p
          role="alert"
          className="rounded-md border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive"
        >
          {actionError} {t("workbench.draftPreserved")}
        </p>
      )}
      {statusMessage && (
        <p
          role="status"
          className="rounded-md border border-success/40 bg-success/5 p-3 text-sm"
        >
          {statusMessage}
        </p>
      )}

      {loading && !snapshot ? (
        <div className="animate-pulse-ring rounded-md border border-border p-4 text-sm text-muted-foreground">
          {t("loading")}
        </div>
      ) : snapshot ? (
        <div className="grid gap-4 xl:grid-cols-[20rem_minmax(0,1fr)] 2xl:grid-cols-[20rem_minmax(0,1fr)_24rem]">
          <aside className="ui-panel p-3" aria-label={t("workbench.treeLabel")}>
            <div className="space-y-3">
              <Label htmlFor="politics-ideology-search">
                {t("workbench.searchLabel")}
              </Label>
              <Input
                id="politics-ideology-search"
                type="search"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t("workbench.searchPlaceholder")}
              />
              <Button
                ref={addIdeologyButtonRef}
                type="button"
                className="w-full"
                onClick={openCreator}
              >
                {t("workbench.addIdeology")}
              </Button>
            </div>
            <div className="mt-4 space-y-1">
              {filteredRoots.map((root) => {
                const active = selectedRoot?.script_id === root.script_id;
                const previewGroup = previewGroups.find(
                  (group) => group.id === root.script_id,
                );
                return (
                  <button
                    key={root.script_id}
                    ref={
                      selected?.kind === "root" &&
                      selectedKey === targetKey(root)
                        ? selectedIdeologyButtonRef
                        : undefined
                    }
                    type="button"
                    className={`ui-interactive ui-focus-ring grid w-full grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-2 rounded-md border border-transparent border-l-2 px-3 py-2.5 text-left ${active ? "border-l-primary bg-muted text-foreground" : "hover:bg-muted"}`}
                    onClick={() => selectIdeology(root)}
                  >
                    <span
                      aria-hidden="true"
                      className="mt-1 size-2.5 rounded-sm border border-border"
                      style={{ backgroundColor: previewGroup?.color }}
                    />
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-semibold">
                        {visibleNameFor(root)}
                      </span>
                      <span className="block truncate text-xs text-muted-foreground">
                        {t("workbench.rootMeta", {
                          count:
                            subtypesByRoot.get(root.script_id)?.length ?? 0,
                          id: root.script_id,
                        })}
                      </span>
                    </span>
                    <span className="mt-0.5 whitespace-nowrap rounded-full border border-border bg-card px-2 py-0.5 text-micro text-muted-foreground">
                      {sourceLabel(root.source)}
                    </span>
                  </button>
                );
              })}
              {filteredRoots.length === 0 && (
                <p className="rounded-md border border-border bg-muted p-3 text-sm text-muted-foreground">
                  {t("workbench.noSearchResults")}
                </p>
              )}
            </div>
          </aside>

          <main className="space-y-4">
            {creatorOpen ? (
              <section
                className="ui-panel p-5"
                aria-labelledby="politics-create-title"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h2
                      id="politics-create-title"
                      className="text-lg font-semibold tracking-tight"
                    >
                      {t("workbench.createTitle")}
                    </h2>
                    <p className="text-sm text-muted-foreground">
                      {t("workbench.createHint")}
                    </p>
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={closeCreator}
                  >
                    {t("cancel")}
                  </Button>
                </div>

                {createdDefinition ? (
                  <div className="mt-5 space-y-4">
                    <div className="rounded-md border border-success/40 bg-success/5 p-4">
                      <h3
                        ref={createdTitleRef}
                        tabIndex={-1}
                        className="ui-focus-ring rounded-sm text-sm font-semibold"
                      >
                        {t("workbench.createdTitle")}
                      </h3>
                      <p className="mt-1 text-sm text-muted-foreground">
                        {t("workbench.createdHint", {
                          id: createdDefinition.script_id,
                        })}
                      </p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button type="button" size="sm" onClick={closeCreator}>
                        {t("workbench.continueEditing")}
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        disabled={!onOpenCountryPolitics}
                        onClick={onOpenCountryPolitics}
                      >
                        {t("workbench.applyCountry")}
                      </Button>
                      {createdDefinition.kind === "subtype" && (
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          disabled={!onAssignLeader}
                          onClick={() =>
                            onAssignLeader?.(createdDefinition.script_id)
                          }
                        >
                          {t("workbench.assignLeader")}
                        </Button>
                      )}
                    </div>
                    {!onOpenCountryPolitics && (
                      <p className="text-xs text-muted-foreground">
                        {t("workbench.followupNeedsContext")}
                      </p>
                    )}
                  </div>
                ) : !createIntent ? (
                  <div className="mt-5 grid gap-3 lg:grid-cols-2">
                    <button
                      ref={firstIntentButtonRef}
                      type="button"
                      className="ui-card ui-interactive ui-focus-ring p-4 text-left"
                      onClick={() => chooseCreateIntent("subtype")}
                    >
                      <span className="rounded-full border border-accent px-2 py-0.5 text-micro text-accent">
                        {t("workbench.recommended")}
                      </span>
                      <span className="mt-3 block text-base font-semibold tracking-tight">
                        {t("workbench.intentSubtypeTitle")}
                      </span>
                      <span className="mt-1 block text-sm text-muted-foreground">
                        {t("workbench.intentSubtypeDescription")}
                      </span>
                    </button>
                    <button
                      type="button"
                      className="ui-card ui-interactive ui-focus-ring p-4 text-left"
                      onClick={() => chooseCreateIntent("root")}
                    >
                      <span className="rounded-full border border-border px-2 py-0.5 text-micro text-muted-foreground">
                        {t("workbench.advanced")}
                      </span>
                      <span className="mt-3 block text-base font-semibold tracking-tight">
                        {t("workbench.intentRootTitle")}
                      </span>
                      <span className="mt-1 block text-sm text-muted-foreground">
                        {t("workbench.intentRootDescription")}
                      </span>
                    </button>
                  </div>
                ) : (
                  <div className="mt-5 space-y-4">
                    <Button
                      type="button"
                      size="xs"
                      variant="ghost"
                      onClick={returnToCreateIntent}
                    >
                      {t("workbench.backToIntent")}
                    </Button>
                    <div className="grid gap-3 md:grid-cols-2">
                      <div ref={createNameFieldRef} className="space-y-1">
                        <Label>{t("displayNameLabel")}</Label>
                        <DisplayNameLocsTabs
                          entityKey={`politics-create:${createIntent}:display-name`}
                          value={createName}
                          exportLanguages={exportLanguages}
                          onChange={updateCreateName}
                          placeholder={
                            createIntent === "subtype"
                              ? t("displayNamePlaceholder")
                              : t("rootDisplayNamePlaceholder")
                          }
                          ariaLabel={t("displayNameLabel")}
                        />
                      </div>
                      <div className="space-y-1">
                        <Label htmlFor="politics-create-id">
                          {createIntent === "subtype"
                            ? t("scriptIdLabel")
                            : t("rootScriptIdLabel")}
                        </Label>
                        <Input
                          id="politics-create-id"
                          className="font-mono"
                          value={createScriptId}
                          onChange={(event) => {
                            setCreateScriptIdTouched(true);
                            setCreateScriptId(event.target.value);
                          }}
                        />
                        <p className="text-micro text-muted-foreground">
                          {t("scriptIdHint")}
                        </p>
                      </div>
                    </div>
                    {createIntent === "subtype" && (
                      <div className="grid gap-3 md:grid-cols-2">
                        <div className="space-y-1">
                          <Label htmlFor="politics-create-parent">
                            {t("parentRootLabel")}
                          </Label>
                          <Select
                            id="politics-create-parent"
                            value={createParent}
                            onChange={(event) =>
                              setCreateParent(event.target.value)
                            }
                          >
                            {roots.map((root) => (
                              <option
                                key={root.script_id}
                                value={root.script_id}
                              >
                                {visibleNameFor(root)}
                              </option>
                            ))}
                          </Select>
                        </div>
                        <label className="flex items-center gap-2 rounded-md border border-border p-3 text-sm">
                          <input
                            type="checkbox"
                            checked={createRandomDisabled}
                            onChange={(event) =>
                              setCreateRandomDisabled(event.target.checked)
                            }
                            className="ui-focus-ring size-4"
                          />
                          {t("workbench.randomDisabled")}
                        </label>
                      </div>
                    )}
                    <div className="space-y-1">
                      <Label>{t("workbench.descriptionLabel")}</Label>
                      <DescriptionLocsTabs
                        entityKey={`politics-create:${createIntent}:description`}
                        value={createDesc}
                        exportLanguages={exportLanguages}
                        onChange={setCreateDesc}
                        ariaLabel={t("workbench.descriptionLabel")}
                        rows={3}
                      />
                    </div>
                    {createIntent === "root" && (
                      <details className="rounded-md border border-border bg-muted/20 p-3">
                        <summary className="ui-focus-ring cursor-pointer text-sm font-medium">
                          {t("workbench.advancedSettings")}
                        </summary>
                        <div className="mt-3 grid gap-3 md:grid-cols-2">
                          <div className="space-y-1">
                            <Label htmlFor="politics-create-root-color">
                              {t("workbench.groupColor")}
                            </Label>
                            <div className="flex items-center gap-3">
                              <Input
                                id="politics-create-root-color"
                                type="color"
                                value={createRootColor}
                                onChange={(event) =>
                                  updateCreateRootColor(event.target.value)
                                }
                                className="h-10 w-20 cursor-pointer p-1"
                              />
                              <code className="text-xs text-muted-foreground">
                                {createRootColor.toUpperCase()}
                              </code>
                            </div>
                            <p className="text-micro text-muted-foreground">
                              {t("workbench.groupColorHint")}
                            </p>
                          </div>
                          <div className="space-y-1">
                            <Label htmlFor="politics-create-rule-mode">
                              {t("workbench.ruleSource")}
                            </Label>
                            <Select
                              id="politics-create-rule-mode"
                              value={createRuleMode}
                              onChange={(event) => {
                                const mode = event.target.value as RootRuleMode;
                                setCreateRuleMode(mode);
                                if (mode === "safe")
                                  setCreateRootBody(
                                    defaultRootBody(createRootColor),
                                  );
                              }}
                            >
                              <option value="safe">
                                {t("workbench.ruleModeSafe")}
                              </option>
                              <option value="custom">
                                {t("workbench.ruleModeCustom")}
                              </option>
                            </Select>
                            <p className="text-micro text-muted-foreground">
                              {t("workbench.ruleSourceHint")}
                            </p>
                          </div>
                        </div>
                        {createRuleMode === "custom" && (
                          <details
                            open
                            className="mt-3 rounded-md border border-border bg-card p-3"
                          >
                            <summary className="ui-focus-ring cursor-pointer text-sm font-medium">
                              {t("workbench.editAdvancedScript")}
                            </summary>
                            <div className="mt-3 space-y-1">
                              <Label htmlFor="politics-create-root-body">
                                {t("workbench.advancedScript")}
                              </Label>
                              <Textarea
                                id="politics-create-root-body"
                                value={createRootBody}
                                onChange={(event) => {
                                  setCreateRootBody(event.target.value);
                                  setCreateRootColor(
                                    rootBodyColor(event.target.value),
                                  );
                                }}
                                className="min-h-28 font-mono text-xs"
                              />
                              <p className="text-micro text-muted-foreground">
                                {t("workbench.advancedScriptHint")}
                              </p>
                              {createRootBodyError && (
                                <p
                                  role="alert"
                                  className="text-xs text-destructive"
                                >
                                  {t(
                                    `workbench.rootBodyError.${createRootBodyError}`,
                                  )}
                                </p>
                              )}
                            </div>
                          </details>
                        )}
                      </details>
                    )}
                    <div className="rounded-md border border-accent/40 bg-accent/5 p-3">
                      <h3 className="text-sm font-semibold">
                        {t("workbench.impactTitle")}
                      </h3>
                      <p className="mt-1 text-sm text-muted-foreground">
                        {createIntent === "subtype"
                          ? t("workbench.subtypeImpact", {
                              root: createParent || "—",
                            })
                          : t("workbench.rootImpact")}
                      </p>
                    </div>
                    <Button
                      type="button"
                      size="sm"
                      disabled={
                        creating ||
                        Object.keys(normalizedLocaleMap(createName)).length ===
                          0 ||
                        !createScriptId.trim() ||
                        (createIntent === "subtype" && !createParent) ||
                        (createIntent === "root" &&
                          (!createRootBody.trim() ||
                            Boolean(createRootBodyError)))
                      }
                      onClick={() => void createDefinition()}
                    >
                      {creating
                        ? t(
                            createIntent === "subtype"
                              ? "creatingSubtype"
                              : "creatingRoot",
                          )
                        : t(
                            createIntent === "subtype"
                              ? "createSubtype"
                              : "createRoot",
                          )}
                    </Button>
                  </div>
                )}
              </section>
            ) : selected ? (
              <>
                {selectedRootChildren.length > 0 && (
                  <section
                    className="ui-panel p-4"
                    aria-labelledby="politics-stances-title"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <h2
                        id="politics-stances-title"
                        className="text-sm font-semibold"
                      >
                        {t("workbench.livePreviewStances")}
                      </h2>
                      <span className="text-xs text-muted-foreground">
                        {t("workbench.livePreviewStanceCount", {
                          count: selectedRootChildren.length,
                        })}
                      </span>
                    </div>
                    <ul className="mt-3 space-y-2">
                      {selectedRootChildren.map((subtype) => {
                        const active =
                          selected.kind === "subtype" &&
                          selected.script_id === subtype.script_id;
                        return (
                          <li key={subtype.script_id}>
                            <button
                              ref={
                                active ? selectedIdeologyButtonRef : undefined
                              }
                              type="button"
                              className={`ui-interactive ui-focus-ring flex w-full items-center justify-between gap-3 rounded-md border px-3 py-2.5 text-left ${active ? "border-primary bg-muted text-foreground" : "border-border bg-muted/20 hover:bg-muted"}`}
                              onClick={() => selectIdeology(subtype)}
                            >
                              <span className="min-w-0">
                                <span className="block truncate text-sm font-medium">
                                  {visibleNameFor(subtype)}
                                </span>
                                <span className="block truncate font-mono text-xs text-muted-foreground">
                                  {subtype.script_id}
                                </span>
                              </span>
                              <span className="shrink-0 rounded-full border border-border bg-card px-2 py-0.5 text-micro text-muted-foreground">
                                {t("workbench.availableForLeader")}
                              </span>
                            </button>
                          </li>
                        );
                      })}
                    </ul>
                  </section>
                )}
                <IdeologyEditor
                  projectId={projectId}
                  ideology={selected}
                  displayName={selectedVisibleName}
                  sourceLabel={sourceLabel(selected.source)}
                  draft={editDraft}
                  exportLanguages={exportLanguages}
                  onChange={setEditDraft}
                  saving={saving}
                  onSave={() => void saveDefinition()}
                  iconAvailable={Boolean(iconPaths[selected.script_id])}
                  iconBust={iconBust}
                  uploadingIcon={uploadingIcon}
                  onUploadIcon={(file) => void uploadIcon(file)}
                  onOpenScript={() => {
                    setActivePaneTab("script");
                    setActivePreviewTarget(null);
                  }}
                  focusTarget={activePreviewTarget}
                  onFocusTarget={setActivePreviewTarget}
                />

                <section
                  className="ui-panel p-4"
                  aria-labelledby="politics-references-title"
                >
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <h3
                        id="politics-references-title"
                        className="text-sm font-semibold"
                      >
                        {t("workbench.referencesTitle")}
                      </h3>
                      <p className="text-xs text-muted-foreground">
                        {t("referenceCount", {
                          count: selectedReferences.length,
                        })}
                      </p>
                    </div>
                    {selected.source === "project_managed" && (
                      <Button
                        ref={deletePreviewButtonRef}
                        type="button"
                        size="xs"
                        variant="outline"
                        disabled={previewing || applyingDelete}
                        onClick={() => void previewDelete()}
                      >
                        {previewing ? t("previewing") : t("previewDelete")}
                      </Button>
                    )}
                  </div>
                  {selectedReferences.length > 0 ? (
                    <ul className="mt-3 space-y-2">
                      {selectedReferences.map((reference) => (
                        <li
                          key={referenceKey(reference)}
                          className="rounded-md border border-border bg-muted/20 p-3"
                        >
                          <p className="text-sm font-medium">
                            {t("impactEntity", {
                              kind: reference.entity.kind,
                              id: reference.entity.id,
                            })}
                          </p>
                          <p className="font-mono text-xs text-muted-foreground break-all">
                            {reference.entity.path}
                          </p>
                          {reference.context && (
                            <p className="mt-1 text-xs text-muted-foreground">
                              {reference.context}
                            </p>
                          )}
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="mt-3 rounded-md border border-border bg-muted/20 p-3 text-sm text-muted-foreground">
                      {t("workbench.noReferences")}
                    </p>
                  )}
                  {deletePreview && (
                    <div
                      className={`mt-4 rounded-md border p-4 ${deletePreview.allowed ? "border-destructive/40 bg-destructive/5" : "border-warning/40 bg-warning/5"}`}
                    >
                      <h4 className="text-sm font-semibold">
                        {t("deletePreviewTitle")}
                      </h4>
                      <p className="mt-1 text-sm">
                        {deletePreview.allowed
                          ? t("deleteAllowed")
                          : t("deleteBlocked")}
                      </p>
                      {!deletePreview.allowed && (
                        <ul className="mt-2 space-y-1 text-xs">
                          {deletePreview.impacts.map((impact) => (
                            <li key={referenceKey(impact)}>
                              {impact.entity.path}
                            </li>
                          ))}
                        </ul>
                      )}
                      <p className="mt-2 text-xs text-muted-foreground">
                        {t("noAutomaticRewrite")}
                      </p>
                      <div className="mt-3 flex justify-end gap-2">
                        <Button
                          ref={deleteCancelButtonRef}
                          type="button"
                          size="xs"
                          variant="outline"
                          onClick={cancelDeletePreview}
                        >
                          {t("cancel")}
                        </Button>
                        <Button
                          type="button"
                          size="xs"
                          variant="destructive"
                          disabled={!deletePreview.allowed || applyingDelete}
                          onClick={() => void applyDelete()}
                        >
                          {applyingDelete ? t("deleting") : t("confirmDelete")}
                        </Button>
                      </div>
                    </div>
                  )}
                </section>
              </>
            ) : (
              <section className="ui-panel p-6 text-sm text-muted-foreground">
                {t("workbench.selectObject")}
              </section>
            )}
          </main>

          <aside
            className="ui-panel min-h-[34rem] overflow-hidden xl:col-span-2 xl:col-start-1 2xl:col-span-1 2xl:col-start-auto 2xl:sticky 2xl:top-4 2xl:h-[calc(100vh-11rem)]"
            aria-label={t("workbench.previewPaneLabel")}
          >
            <WorkbenchRightPane
              activeTabId={activePaneTab}
              onTabChange={(tabId) => {
                setActivePaneTab(tabId);
                setActivePreviewTarget(null);
              }}
              tabs={[
                {
                  id: "live",
                  label: t("workbench.previewTabLive"),
                  content:
                    selected && selectedRoot ? (
                      <PoliticsInGamePreview
                        scriptId={selected.script_id}
                        displayName={selectedVisibleName}
                        description={selectedDescription}
                        groupId={selectedGroupId}
                        groupLabel={selectedGroupLabel}
                        groupColor={selectedPreviewGroup?.color ?? "#808080"}
                        stanceCount={
                          subtypesByRoot.get(selectedGroupId)?.length ?? 0
                        }
                        iconUrl={selectedIconUrl}
                        countryFlag={
                          previewCountrySource ? (
                            <CountryFlagPreview
                              projectId={projectId}
                              country={previewCountrySource.country}
                              root={selectedGroupId}
                              roots={previewRootIds}
                              baseModDepKey={baseModDepKey}
                              className="h-auto w-10 rounded-sm object-cover"
                            />
                          ) : null
                        }
                        groups={previewGroups}
                        country={previewCountry}
                        isDraft={editDirty}
                        activeTarget={activePreviewTarget}
                        editableTargets={editablePreviewTargets}
                        onSelectTarget={selectPreviewTarget}
                      />
                    ) : (
                      <p className="p-4 text-sm text-muted-foreground">
                        {t("workbench.selectObject")}
                      </p>
                    ),
                },
                {
                  id: "script",
                  label: t("workbench.previewTabScript"),
                  hint: t("workbench.scriptTabHint"),
                  content: selected ? (
                    <div className="space-y-4 p-3">
                      <dl className="grid gap-2 rounded-md border border-border bg-muted/20 p-3 text-xs">
                        <div>
                          <dt className="text-muted-foreground">
                            {t("stableIdLabel")}
                          </dt>
                          <dd className="mt-0.5 break-all font-mono">
                            {selected.script_id}
                          </dd>
                        </div>
                        <div>
                          <dt className="text-muted-foreground">
                            {t("workbench.sourceLabel")}
                          </dt>
                          <dd className="mt-0.5">
                            {sourceLabel(selected.source)}
                          </dd>
                        </div>
                        {selected.root_id && (
                          <div>
                            <dt className="text-muted-foreground">
                              {t("parentRootLabel")}
                            </dt>
                            <dd className="mt-0.5 font-mono">
                              {selected.root_id}
                            </dd>
                          </div>
                        )}
                      </dl>

                      {selected.kind === "root" &&
                      selected.source === "project_managed" ? (
                        <div className="space-y-3">
                          <div className="flex items-center justify-between gap-2">
                            <h3 className="text-sm font-semibold">
                              {t("workbench.advancedScript")}
                            </h3>
                            <Button
                              type="button"
                              size="xs"
                              variant="outline"
                              onClick={() => {
                                setScriptEditing((value) => !value);
                                setActivePreviewTarget(null);
                                setTimeout(
                                  () => scriptTextareaRef.current?.focus(),
                                  0,
                                );
                              }}
                            >
                              {scriptEditing
                                ? t("workbench.previewGeneratedScript")
                                : t("workbench.editSourceScript")}
                            </Button>
                          </div>
                          {scriptEditing ? (
                            <Textarea
                              ref={scriptTextareaRef}
                              id="politics-edit-root-body"
                              aria-label={t("workbench.advancedScript")}
                              value={editDraft.rootBody}
                              onChange={(event) =>
                                setEditDraft({
                                  ...editDraft,
                                  rootBody: event.target.value,
                                  rootColor: rootBodyColor(event.target.value),
                                })
                              }
                              className="min-h-64 font-mono text-xs"
                            />
                          ) : (
                            <pre className="max-h-[28rem] overflow-auto whitespace-pre-wrap rounded-md border border-border bg-muted/20 p-3 font-mono text-xs">{`${selected.script_id} = {\n${editDraft.rootBody}\n}`}</pre>
                          )}
                          <p className="text-xs leading-relaxed text-muted-foreground">
                            {t("workbench.advancedScriptHint")}
                          </p>
                          {editRootBodyError && (
                            <p
                              role="alert"
                              className="text-xs text-destructive"
                            >
                              {t(
                                `workbench.rootBodyError.${editRootBodyError}`,
                              )}
                            </p>
                          )}
                          {scriptEditing && (
                            <Button
                              type="button"
                              size="sm"
                              className="w-full"
                              disabled={saving || Boolean(editRootBodyError)}
                              onClick={() => void saveDefinition()}
                            >
                              {saving
                                ? t("workbench.saving")
                                : t("workbench.saveChanges")}
                            </Button>
                          )}
                        </div>
                      ) : selected.kind === "subtype" &&
                        selected.source === "project_managed" ? (
                        <div className="space-y-2">
                          <pre className="whitespace-pre-wrap rounded-md border border-border bg-muted/20 p-3 font-mono text-xs">{`${selected.script_id} = {${editDraft.randomDisabled ? "\n  can_be_randomly_selected = no\n" : "\n"}}`}</pre>
                          <p className="text-xs leading-relaxed text-muted-foreground">
                            {t("workbench.subtypeScriptGeneratedHint", {
                              root: selected.root_id || "—",
                            })}
                          </p>
                        </div>
                      ) : (
                        <p className="rounded-md border border-border bg-muted/20 p-3 text-sm text-muted-foreground">
                          {t("workbench.sourceScriptReadOnly")}
                        </p>
                      )}
                    </div>
                  ) : (
                    <p className="p-4 text-sm text-muted-foreground">
                      {t("workbench.selectObject")}
                    </p>
                  ),
                },
              ]}
            />
          </aside>
        </div>
      ) : null}
    </div>
  );
}
