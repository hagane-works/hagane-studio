"use client";

import {
  useState,
  useEffect,
  useCallback,
  useMemo,
  useRef,
  type ReactNode,
} from "react";
import { useTranslations } from "use-intl";
import { ArrowLeft, Folder, Route, SearchX, Swords } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Select } from "@/components/ui/select";
import {
  doctrinesApi,
  DOCTRINE_ENTITY_TYPES,
  type DoctrineRow,
  type DoctrineEntityType,
} from "@/lib/api";
import { useProjectStore } from "@/lib/store/projectStore";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { useActiveCountry } from "@/lib/store/projectStore";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { FocusIconLibraryDialog } from "@/components/workbench/focus-graph/FocusIconLibraryDialog";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import {
  parseNamedBlocks,
  serializeNamedBlocks,
  tokenBlockToText,
  textToTokenBlock,
  sectionToText,
  type NamedBlockEntry,
} from "./doctrineBlocks";
import { normalizeInnerList } from "./rawInnerList";
import {
  beginDraftSave,
  confirmDraftSelection,
  editDraftSession,
  emptyDraftSession,
  openDraftSession,
  requestDraftSelection,
  resolveDraftSave,
  stableDraftFingerprint,
  type DraftSession,
} from "@/lib/workbench-state/draftSession";

const STRUCTURED_SECTION_KEYS = [
  "mastery",
  "tracks",
  "milestones",
  "rewards",
] as const;

const ENTITY_TYPE_LABEL_KEY: Record<DoctrineEntityType, string> = {
  folder: "typeFolder",
  grand_doctrine: "typeGrandDoctrine",
  track: "typeTrack",
  subdoctrine: "typeSubdoctrine",
};

const ENTITY_TYPE_ICON: Record<DoctrineEntityType, ReactNode> = {
  folder: <Folder className="h-3.5 w-3.5 shrink-0" aria-hidden />,
  grand_doctrine: "⭐",
  track: <Route className="h-3.5 w-3.5 shrink-0" aria-hidden />,
  subdoctrine: "⚔️",
};

interface DraftState {
  entity_type: DoctrineEntityType;
  entity_key: string;
  folder_key: string;
  track_key: string;
  name: string;
  description: string;
  icon: string;
  xp_cost: number;
  xp_type: string;

  xor: string;

  allow_in_multiple_tracks: boolean | null;
  source_file: string;
  display_name_locs: Record<string, string>;
  description_locs: Record<string, string>;
  raw_extras: Record<string, unknown>;
}

interface DoctrineEditorExtras {
  rawExtrasText: string;
  masteryText: string;
  tracksText: string;
  milestones: NamedBlockEntry[];
  rewards: NamedBlockEntry[];
}

interface DoctrineEditorDraft {
  form: DraftState;
  extras: DoctrineEditorExtras;
}

function extrasFromRaw(
  rawExtras: Record<string, unknown>,
): DoctrineEditorExtras {
  const raw = rawExtras || {};
  const passthrough: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(raw)) {
    if (!(STRUCTURED_SECTION_KEYS as readonly string[]).includes(key)) {
      passthrough[key] = value;
    }
  }
  return {
    rawExtrasText: JSON.stringify(passthrough, null, 2),
    masteryText: sectionToText(raw["mastery"]),
    tracksText: tokenBlockToText(raw["tracks"]),
    milestones: parseNamedBlocks(sectionToText(raw["milestones"])),
    rewards: parseNamedBlocks(sectionToText(raw["rewards"])),
  };
}

function editorDraft(
  form: DraftState,
  extras: DoctrineEditorExtras,
): DoctrineEditorDraft {
  return { form, extras };
}

function blankDraft(): DraftState {
  return {
    entity_type: "grand_doctrine",
    entity_key: "",
    folder_key: "",
    track_key: "",
    name: "",
    description: "",
    icon: "",
    xp_cost: 0,
    xp_type: "army",
    xor: "",
    allow_in_multiple_tracks: null,
    source_file: "",
    display_name_locs: {},
    description_locs: {},
    raw_extras: {},
  };
}

function rowToDraft(r: DoctrineRow): DraftState {
  return {
    entity_type: r.entity_type,
    entity_key: r.entity_key,
    folder_key: r.folder_key || "",
    track_key: r.track_key || "",
    name: r.name || "",
    description: r.description || "",
    icon: r.icon || "",
    xp_cost: r.xp_cost || 0,
    xp_type: r.xp_type || "",
    xor: r.xor ?? "",
    allow_in_multiple_tracks: r.allow_in_multiple_tracks ?? null,
    source_file: r.source_file || "",
    display_name_locs: r.display_name_locs || {},
    description_locs: r.description_locs || {},
    raw_extras: r.raw_extras || {},
  };
}

export function DoctrinesContent({ projectId }: { projectId: number }) {
  const t = useTranslations("doctrines");
  const tCommon = useTranslations("common");
  const tShell = useTranslations("workbenchShell");
  const confirmDialog = useConfirm();

  const currentProject = useProjectStore((s) => s.currentProject);
  const exportLanguages = useMemo(
    () =>
      currentProject?.export_languages?.length
        ? currentProject.export_languages
        : DEFAULT_EXPORT_LANGUAGES,
    [currentProject?.export_languages],
  );

  const [rows, setRows] = useState<DoctrineRow[]>([]);

  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const activeCountry = useActiveCountry();
  const [activeId, setActiveId] = useState<number | "new" | null>(null);
  const [draft, setDraft] = useState<DraftState>(blankDraft());

  const [rawExtrasText, setRawExtrasText] = useState<string>("{}");

  const [masteryText, setMasteryText] = useState<string>("");
  const [tracksText, setTracksText] = useState<string>("");
  const [milestones, setMilestones] = useState<NamedBlockEntry[]>([]);
  const [rewards, setRewards] = useState<NamedBlockEntry[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [listLoadFailed, setListLoadFailed] = useState(false);
  const [saving, setSaving] = useState(false);
  const [iconLibraryOpen, setIconLibraryOpen] = useState(false);
  const extrasRef = useRef<DoctrineEditorExtras>(extrasFromRaw({}));
  const ownership =
    useRef<DraftSession<DoctrineEditorDraft>>(emptyDraftSession());
  const latestListRequest = useRef(0);
  const fingerprint = useCallback(
    (value: DoctrineEditorDraft) => stableDraftFingerprint(value),
    [],
  );

  const reload = useCallback(async () => {
    const requestId = ++latestListRequest.current;
    setLoading(true);
    try {
      const res = await doctrinesApi.list(projectId);
      if (requestId !== latestListRequest.current) return;
      setRows(res.doctrines);
      setListLoadFailed(false);
    } catch {
      if (requestId === latestListRequest.current) setListLoadFailed(true);
    } finally {
      if (requestId === latestListRequest.current) setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    reload();
  }, [reload]);

  const update = useCallback(
    <K extends keyof DraftState>(key: K, value: DraftState[K]) => {
      setDraft((current) => {
        const next = { ...current, [key]: value };
        const nextEditor = editorDraft(next, extrasRef.current);
        ownership.current = editDraftSession(
          ownership.current,
          nextEditor,
          fingerprint(nextEditor),
        );
        return next;
      });
    },
    [fingerprint],
  );

  const seedSections = useCallback((rawExtras: Record<string, unknown>) => {
    const next = extrasFromRaw(rawExtras);
    extrasRef.current = next;
    setMasteryText(next.masteryText);
    setTracksText(next.tracksText);
    setMilestones(next.milestones);
    setRewards(next.rewards);
    setRawExtrasText(next.rawExtrasText);
  }, []);

  const setEditorExtras = useCallback(
    (next: DoctrineEditorExtras) => {
      extrasRef.current = next;
      setMasteryText(next.masteryText);
      setTracksText(next.tracksText);
      setMilestones(next.milestones);
      setRewards(next.rewards);
      setRawExtrasText(next.rawExtrasText);
      const form = ownership.current.draft?.form;
      if (form) {
        const nextEditor = editorDraft(form, next);
        ownership.current = editDraftSession(
          ownership.current,
          nextEditor,
          fingerprint(nextEditor),
        );
      }
    },
    [fingerprint],
  );

  const selectRow = useCallback(
    async (r: DoctrineRow) => {
      const next = rowToDraft(r);
      const nextEditor = editorDraft(next, extrasFromRaw(r.raw_extras || {}));
      const requested = requestDraftSelection(
        ownership.current,
        String(r.id),
        fingerprint(nextEditor),
        nextEditor,
      );
      if (
        requested.kind === "confirm-discard" &&
        !(await confirmDialog({ body: tCommon("discardUnsavedConfirm") }))
      )
        return;
      ownership.current =
        requested.kind === "selected"
          ? requested.session
          : confirmDraftSelection(
              ownership.current,
              String(r.id),
              fingerprint(nextEditor),
              nextEditor,
            );
      setActiveId(r.id);
      setDraft(next);
      seedSections(r.raw_extras || {});
      setError(null);
    },
    [confirmDialog, fingerprint, seedSections, tCommon],
  );

  const startNew = useCallback(async () => {
    const next = blankDraft();
    const nextEditor = editorDraft(next, extrasFromRaw({}));
    const requested = requestDraftSelection(
      ownership.current,
      "new",
      fingerprint(nextEditor),
      nextEditor,
    );
    if (
      requested.kind === "confirm-discard" &&
      !(await confirmDialog({ body: tCommon("discardUnsavedConfirm") }))
    )
      return;
    ownership.current =
      requested.kind === "selected"
        ? requested.session
        : confirmDraftSelection(
            ownership.current,
            "new",
            fingerprint(nextEditor),
            nextEditor,
          );
    setActiveId("new");
    setDraft(next);
    seedSections({});
    setError(null);
  }, [confirmDialog, fingerprint, seedSections, tCommon]);

  const activeRow = useMemo(
    () =>
      typeof activeId === "number"
        ? rows.find((r) => r.id === activeId)
        : undefined,
    [rows, activeId],
  );

  const handleSave = useCallback(async () => {
    setError(null);
    const key = draft.entity_key.trim();
    if (!key) {
      setError(t("entityKeyRequired"));
      return;
    }
    let parsedRaw: Record<string, unknown> = {};
    try {
      const parsed = JSON.parse(rawExtrasText || "{}");
      if (
        parsed === null ||
        typeof parsed !== "object" ||
        Array.isArray(parsed)
      ) {
        throw new Error(t("rawExtrasMustBeObject"));
      }
      parsedRaw = parsed as Record<string, unknown>;
    } catch (e) {
      setError(
        t("rawExtrasInvalid", {
          error: e instanceof Error ? e.message : String(e),
        }),
      );
      return;
    }

    {
      const mastery = masteryText.trim();
      if (mastery) parsedRaw["mastery"] = mastery;
      else delete parsedRaw["mastery"];

      const tracks = textToTokenBlock(tracksText);
      if (tracks) parsedRaw["tracks"] = tracks;
      else delete parsedRaw["tracks"];

      const milestonesText = serializeNamedBlocks(milestones);
      if (milestonesText) parsedRaw["milestones"] = milestonesText;
      else delete parsedRaw["milestones"];

      const rewardsText = serializeNamedBlocks(rewards);
      if (rewardsText) parsedRaw["rewards"] = rewardsText;
      else delete parsedRaw["rewards"];
    }
    const xorList = normalizeInnerList(draft.xor);
    setSaving(true);
    const save = beginDraftSave(ownership.current);
    if (!save) {
      setSaving(false);
      return;
    }
    try {
      const payload = {
        entity_type: draft.entity_type,
        entity_key: key,
        folder_key: draft.folder_key.trim(),
        track_key: draft.track_key.trim(),
        name: draft.name.trim(),
        description: draft.description.trim(),
        icon: draft.icon.trim(),
        xp_cost: Number(draft.xp_cost) || 0,
        xp_type: draft.xp_type.trim(),
        source_file: draft.source_file.trim(),
        display_name_locs: draft.display_name_locs,
        description_locs: draft.description_locs,
        raw_extras: parsedRaw,

        ...(xorList
          ? { xor: xorList }
          : activeRow?.xor != null
            ? { xor: null }
            : {}),
        ...(draft.allow_in_multiple_tracks != null
          ? { allow_in_multiple_tracks: draft.allow_in_multiple_tracks }
          : {}),
      };
      if (activeId === "new") {
        const created = await doctrinesApi.create(projectId, payload);
        await reload();
        const saved = rowToDraft(created);
        const savedEditor = editorDraft(
          saved,
          extrasFromRaw(created.raw_extras || {}),
        );
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(savedEditor),
          savedEditor,
        );
        if (ownership.current.identity === "new") {
          const current = ownership.current.draft!;
          const hasPostSaveEdit = ownership.current.dirty;
          ownership.current = openDraftSession(
            ownership.current,
            String(created.id),
            fingerprint(savedEditor),
            current,
          );
          if (hasPostSaveEdit) {
            ownership.current = editDraftSession(
              ownership.current,
              current,
              fingerprint(current),
            );
          }
          setActiveId(created.id);
          if (!hasPostSaveEdit) {
            setDraft(saved);
            seedSections(created.raw_extras || {});
          }
        }
      } else if (typeof activeId === "number") {
        const updated = await doctrinesApi.update(projectId, activeId, payload);
        await reload();
        const saved = rowToDraft(updated);
        const savedEditor = editorDraft(
          saved,
          extrasFromRaw(updated.raw_extras || {}),
        );
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(savedEditor),
          savedEditor,
        );
        if (
          ownership.current.identity === String(updated.id) &&
          !ownership.current.dirty
        ) {
          setDraft(saved);
          seedSections(updated.raw_extras || {});
        }
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }, [
    activeId,
    activeRow,
    draft,
    projectId,
    rawExtrasText,
    reload,
    t,
    seedSections,
    fingerprint,
    masteryText,
    tracksText,
    milestones,
    rewards,
  ]);

  const handleDelete = useCallback(async () => {
    if (typeof activeId !== "number") return;
    if (!(await confirmDialog({ body: t("deleteConfirm"), danger: true })))
      return;
    try {
      await doctrinesApi.remove(projectId, activeId);
      if (ownership.current.identity === String(activeId)) {
        ownership.current = emptyDraftSession();
        setActiveId(null);
        setDraft(blankDraft());
        seedSections({});
      }
      await reload();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [activeId, confirmDialog, projectId, reload, t, seedSections]);

  const filteredRows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) return rows;
    return rows.filter((r) => r.entity_key.toLowerCase().includes(needle));
  }, [rows, search]);

  const grouped = useMemo(() => {
    const byType: Record<string, DoctrineRow[]> = {};
    for (const r of filteredRows) {
      (byType[r.entity_type] ||= []).push(r);
    }
    for (const k of Object.keys(byType)) {
      byType[k].sort((a, b) => a.entity_key.localeCompare(b.entity_key));
    }
    return byType;
  }, [filteredRows]);

  const showFolderLink =
    draft.entity_type === "grand_doctrine" ||
    draft.entity_type === "track" ||
    draft.entity_type === "subdoctrine";
  const showTrackLink = draft.entity_type === "subdoctrine";
  const showXp =
    draft.entity_type === "grand_doctrine" ||
    draft.entity_type === "subdoctrine";

  const showMastery = draft.entity_type === "track";
  const showTracks = draft.entity_type === "grand_doctrine";
  const showMilestones = draft.entity_type === "grand_doctrine";
  const showRewards = draft.entity_type === "subdoctrine";

  const updateMastery = useCallback(
    (value: string) => {
      setEditorExtras({ ...extrasRef.current, masteryText: value });
    },
    [setEditorExtras],
  );
  const updateTracks = useCallback(
    (value: string) => {
      setEditorExtras({ ...extrasRef.current, tracksText: value });
    },
    [setEditorExtras],
  );
  const updateRawExtras = useCallback(
    (value: string) => {
      setEditorExtras({ ...extrasRef.current, rawExtrasText: value });
    },
    [setEditorExtras],
  );
  const addMilestone = useCallback(() => {
    const entries = extrasRef.current.milestones;
    setEditorExtras({
      ...extrasRef.current,
      milestones: [
        ...entries,
        { name: `milestone_${entries.length + 1}`, body: "" },
      ],
    });
  }, [setEditorExtras]);
  const patchMilestone = useCallback(
    (index: number, patch: Partial<NamedBlockEntry>) => {
      setEditorExtras({
        ...extrasRef.current,
        milestones: extrasRef.current.milestones.map((entry, i) =>
          i === index ? { ...entry, ...patch } : entry,
        ),
      });
    },
    [setEditorExtras],
  );
  const removeMilestone = useCallback(
    (index: number) => {
      setEditorExtras({
        ...extrasRef.current,
        milestones: extrasRef.current.milestones.filter((_, i) => i !== index),
      });
    },
    [setEditorExtras],
  );
  const addReward = useCallback(() => {
    const entries = extrasRef.current.rewards;
    setEditorExtras({
      ...extrasRef.current,
      rewards: [...entries, { name: `reward_${entries.length + 1}`, body: "" }],
    });
  }, [setEditorExtras]);
  const patchReward = useCallback(
    (index: number, patch: Partial<NamedBlockEntry>) => {
      setEditorExtras({
        ...extrasRef.current,
        rewards: extrasRef.current.rewards.map((entry, i) =>
          i === index ? { ...entry, ...patch } : entry,
        ),
      });
    },
    [setEditorExtras],
  );
  const removeReward = useCallback(
    (index: number) => {
      setEditorExtras({
        ...extrasRef.current,
        rewards: extrasRef.current.rewards.filter((_, i) => i !== index),
      });
    },
    [setEditorExtras],
  );

  const zeroHero = (
    <EditorEmptyState
      variant="hero"
      icon={<Swords className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyHeroTitle")}
      description={t("emptyHeroDesc")}
      actions={
        <Button
          size="sm"
          onClick={() => {
            void startNew();
          }}
        >
          {tCommon("emptyCtaCreate")}
        </Button>
      }
    />
  );

  const listPanel = (
    <div>
      <WorkbenchListHeader
        title={t("header")}
        count={filteredRows.length}
        totalCount={rows.length}
        searchValue={search}
        onSearchChange={setSearch}
        searchPlaceholder={tShell("searchPlaceholderGeneric")}
        renderSearch={rows.length > 0}
        action={
          <Button size="sm" onClick={() => void startNew()}>
            {t("newButton")}
          </Button>
        }
      />
      <div className="px-3 pb-3 space-y-3">
        {loading ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("loading")}
          </p>
        ) : listLoadFailed && rows.length === 0 ? null : rows.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("emptyHeroTitle")}
          </p>
        ) : filteredRows.length === 0 ? (
          <div className="p-4 flex flex-col items-center gap-2 text-center border border-dashed rounded-lg bg-muted/20">
            <SearchX className="h-5 w-5 text-muted-foreground" />
            <p className="text-xs text-muted-foreground">
              {tShell("noMatchesGeneric")}
            </p>
          </div>
        ) : (
          (DOCTRINE_ENTITY_TYPES as readonly DoctrineEntityType[]).map(
            (etype) =>
              grouped[etype]?.length ? (
                <div key={etype} className="space-y-0.5">
                  <p className="text-micro font-semibold uppercase text-muted-foreground tracking-wide">
                    {ENTITY_TYPE_ICON[etype]} {t(ENTITY_TYPE_LABEL_KEY[etype])}
                  </p>
                  {grouped[etype].map((r) => (
                    <button
                      key={r.id}
                      type="button"
                      onClick={() => void selectRow(r)}

                      className={`w-full text-left px-2 py-1.5 rounded text-xs truncate transition-colors focus-visible:ring-1 focus-visible:ring-ring ${
                        activeId === r.id
                          ? "bg-accent/10 text-accent font-medium shadow-[inset_3px_0_0_0] shadow-accent"
                          : "hover:bg-muted text-muted-foreground"
                      }`}
                      title={r.entity_key}
                    >
                      {r.entity_key}
                      {r.source === "imported" ? (
                        <span className="ml-1 text-micro opacity-60">
                          ({t("imported")})
                        </span>
                      ) : null}
                    </button>
                  ))}
                </div>
              ) : null,
          )
        )}
      </div>
    </div>
  );

  const editorForm = (
    <div className="p-4">
      {activeId === "new" && (
        <button
          type="button"
          onClick={() => {
            ownership.current = emptyDraftSession();
            setActiveId(null);
            setDraft(blankDraft());
            seedSections({});
          }}
          className="ui-interactive mb-3 inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          {tShell("backFromNew")}
        </button>
      )}
      <div className="max-w-3xl space-y-4">
        {error ? (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-xs text-destructive">
            {error}
          </div>
        ) : null}

        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label>{t("entityTypeLabel")}</Label>
            <Select
              className="px-2"
              value={draft.entity_type}
              onChange={(e) =>
                update("entity_type", e.target.value as DoctrineEntityType)
              }
            >
              {(DOCTRINE_ENTITY_TYPES as readonly DoctrineEntityType[]).map(
                (et) => (
                  <option key={et} value={et}>
                    {t(ENTITY_TYPE_LABEL_KEY[et])}
                  </option>
                ),
              )}
            </Select>
          </div>
          <div className="space-y-1">
            <Label>{t("entityKeyLabel")}</Label>
            <Input
              value={draft.entity_key}
              onChange={(e) => update("entity_key", e.target.value)}
              placeholder="new_mobile_warfare"
              className="text-sm"
            />
          </div>
        </div>

        {(showFolderLink || showTrackLink) && (
          <div className="grid grid-cols-2 gap-3">
            {showFolderLink && (
              <div className="space-y-1">
                <Label>
                  {draft.entity_type === "grand_doctrine"
                    ? t("folderVanilla")
                    : t("folderGrouping")}
                </Label>
                <Input
                  value={draft.folder_key}
                  onChange={(e) => update("folder_key", e.target.value)}
                  placeholder="land"
                  className="text-sm"
                />
              </div>
            )}
            {showTrackLink && (
              <div className="space-y-1">
                <Label>{t("trackVanilla")}</Label>
                <Input
                  value={draft.track_key}
                  onChange={(e) => update("track_key", e.target.value)}
                  placeholder="infantry"
                  className="text-sm"
                />
              </div>
            )}
          </div>
        )}

        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label>{t("nameKeyLabel")}</Label>
            <Input
              value={draft.name}
              onChange={(e) => update("name", e.target.value)}
              placeholder="GRAND_DOCTRINE_MOBILE_WARFARE"
              className="text-sm"
            />
          </div>
          <div className="space-y-1">
            <Label>{t("iconRefLabel")}</Label>
            <div className="flex gap-1">
              <Input
                value={draft.icon}
                onChange={(e) => update("icon", e.target.value)}
                placeholder="GFX_doctrine_mobile_warfare_medium"
                className="min-w-0 flex-1 font-mono text-xs"
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
            <p className="text-micro text-muted-foreground">
              {t("iconLibraryHint")}
            </p>
          </div>
        </div>

        <FocusIconLibraryDialog
          open={iconLibraryOpen}
          onOpenChange={setIconLibraryOpen}
          projectId={projectId}
          currentIcon={draft.icon}
          onSelect={(gfxName) => update("icon", gfxName)}
          title={t("iconDialogTitle")}
          description={t("iconDialogDescription")}
          uploadDescription={t("iconDialogUploadDescription")}
          uploadNamePlaceholder={t("iconDialogUploadPlaceholder")}
          projectOnly
        />

        <div className="space-y-1">
          <Label>{t("descriptionKeyLabel")}</Label>
          <Input
            value={draft.description}
            onChange={(e) => update("description", e.target.value)}
            placeholder="GRAND_DOCTRINE_MOBILE_WARFARE_DESC"
            className="text-sm"
          />
        </div>

        {showXp && (
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label>{t("xpCostLabel")}</Label>
              <Input
                type="number"
                value={draft.xp_cost}
                onChange={(e) => update("xp_cost", Number(e.target.value))}
                className="text-sm"
              />
            </div>
            <div className="space-y-1">
              <Label>{t("xpTypeLabel")}</Label>
              <Select
                className="px-2"
                value={draft.xp_type}
                onChange={(e) => update("xp_type", e.target.value)}
              >
                <option value="">{t("xpTypeNone")}</option>
                <option value="army">{t("xpTypeArmy")}</option>
                <option value="naval">{t("xpTypeNaval")}</option>
                <option value="air">{t("xpTypeAir")}</option>
              </Select>
            </div>
          </div>
        )}

        {draft.entity_type === "subdoctrine" && (
          <div className="space-y-2">
            <div className="space-y-1">
              <Label>{t("xorLabel")}</Label>
              <Textarea
                value={draft.xor}
                onChange={(e) => update("xor", e.target.value)}
                placeholder="mass_assault_deep_battle mass_assault_mass_mobilization"
                spellCheck={false}
                rows={2}
                className="font-mono text-xs"
              />
              <p className="text-micro text-muted-foreground">{t("xorHint")}</p>
            </div>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={draft.allow_in_multiple_tracks === true}
                onChange={(e) =>
                  update(
                    "allow_in_multiple_tracks",
                    e.target.checked
                      ? true
                      : typeof activeRow?.allow_in_multiple_tracks === "boolean"
                        ? false
                        : null,
                  )
                }
                className="rounded"
              />
              <span className="text-sm">{t("allowMultipleTracksLabel")}</span>
            </label>
            <p className="text-micro text-muted-foreground ml-6">
              {t("allowMultipleTracksHint")}
            </p>
          </div>
        )}

        <Separator />

        <div className="space-y-2">
          <div className="space-y-1">
            <Label>{t("displayNameLabel")}</Label>
            <DisplayNameLocsTabs
              entityKey={`doctrine:${activeId}:display-name`}
              value={draft.display_name_locs}
              exportLanguages={exportLanguages}
              onChange={(next) => update("display_name_locs", next)}
              placeholder={t("namePlaceholder")}
              ariaLabel={t("displayNameLabel")}
            />
          </div>
          <div className="space-y-1">
            <Label>{t("descriptionTextLabel")}</Label>
            <DescriptionLocsTabs
              entityKey={`doctrine:${activeId}:description`}
              value={draft.description_locs}
              exportLanguages={exportLanguages}
              onChange={(next) => update("description_locs", next)}
              placeholder={t("descPlaceholder")}
              ariaLabel={t("descriptionTextLabel")}
              rows={2}
            />
          </div>
        </div>

        <Separator />

        {(showMastery || showTracks || showMilestones || showRewards) && (
          <div className="space-y-3">
            {showMastery && (
              <div className="space-y-1">
                <Label>{t("masteryLabel")}</Label>
                <textarea
                  value={masteryText}
                  onChange={(e) => updateMastery(e.target.value)}
                  rows={5}
                  spellCheck={false}
                  className="w-full rounded-md border border-input bg-background px-2 py-1.5 font-mono text-xs"
                  placeholder={
                    "multiplier = 8.0\nequipment = {\n\tfighter\n\tinterceptor\n}"
                  }
                />
                <p className="text-micro text-muted-foreground">
                  {t("masteryHint")}
                </p>
              </div>
            )}

            {showTracks && (
              <div className="space-y-1">
                <Label>{t("tracksLabel")}</Label>
                <Input
                  value={tracksText}
                  onChange={(e) => updateTracks(e.target.value)}
                  placeholder="infantry combat_support armor operations"
                  className="font-mono text-xs"
                />
                <p className="text-micro text-muted-foreground">
                  {t("tracksHint")}
                </p>
              </div>
            )}

            {showMilestones && (
              <NamedBlockSection
                title={t("milestonesLabel")}
                hint={t("milestonesHint")}
                entries={milestones}
                namePlaceholder="milestone_1"
                bodyPlaceholder={"name = ms1_loc_key\narmy_org_factor = 0.02"}
                addLabel={t("addMilestone")}
                removeLabel={t("removeEntry")}
                nameLabel={t("entryNameLabel")}
                bodyLabel={t("entryBodyLabel")}
                emptyLabel={t("noMilestones")}
                onAdd={addMilestone}
                onPatch={patchMilestone}
                onRemove={removeMilestone}
              />
            )}

            {showRewards && (
              <NamedBlockSection
                title={t("rewardsLabel")}
                hint={t("rewardsHint")}
                entries={rewards}
                namePlaceholder="interception_priority"
                bodyPlaceholder={"air_intercept_efficiency = 0.15"}
                addLabel={t("addReward")}
                removeLabel={t("removeEntry")}
                nameLabel={t("entryNameLabel")}
                bodyLabel={t("entryBodyLabel")}
                emptyLabel={t("noRewards")}
                onAdd={addReward}
                onPatch={patchReward}
                onRemove={removeReward}
              />
            )}
          </div>
        )}

        <Separator />

        <details className="rounded-md border border-border bg-muted/30 px-3 py-2">
          <summary className="cursor-pointer text-xs font-semibold text-muted-foreground hover:text-foreground select-none">
            {t("rawExtrasSummary")}
            <span className="ml-2 text-xs font-normal">
              {t("rawExtrasHint")}
            </span>
          </summary>
          <div className="pt-3 space-y-1">
            <textarea
              value={rawExtrasText}
              onChange={(e) => updateRawExtras(e.target.value)}
              rows={14}
              spellCheck={false}
              className="w-full rounded-md border border-input bg-background px-2 py-1.5 font-mono text-xs"
              placeholder={
                '{\n  "planning_speed": "0.20",\n  "ai_will_do": "base = 1"\n}'
              }
            />
            <p className="text-micro text-muted-foreground">
              {t("rawExtrasNote")} <code>{`key = { … }`}</code>
              {t("rawExtrasNoteTail")} <code>planning_speed</code>
              {t("rawExtrasNoteEnd")}
            </p>
          </div>
        </details>

        {draft.source_file ? (
          <p className="text-micro text-muted-foreground">
            {t("sourceFilePrefix")} <code>{draft.source_file}</code>{" "}
            {t("sourceFileSuffix")}
          </p>
        ) : null}

        <div className="flex items-center gap-2 pt-2">
          <Button onClick={handleSave} disabled={saving}>
            {saving
              ? t("saving")
              : activeId === "new"
                ? t("create")
                : t("save")}
          </Button>
          {typeof activeId === "number" ? (
            <Button variant="destructive" onClick={handleDelete}>
              {t("delete")}
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  );

  return (
    <WorkbenchShell
      gameViewHref={gameViewHref(projectId, {
        panel: "country_doctrine_view",
        tag: activeCountry,
      })}
      listPanel={listPanel}
      editorPanel={
        activeId === null ? (
          <div className="flex h-full items-center justify-center p-6">
            {listLoadFailed && rows.length === 0 ? (
              <WorkbenchLoadError onRetry={() => void reload()} />
            ) : rows.length === 0 ? (
              zeroHero
            ) : (
              <EditorEmptyState
                className="max-w-xs"
                icon={<Swords className="h-6 w-6 text-muted-foreground/60" />}
                title={tShell("noSelectionGeneric")}
                description={t("selectHint")}
              />
            )}
          </div>
        ) : (
          editorForm
        )
      }
      previewPanel={
        <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
          {tShell("previewUnavailableGeneric")}
        </div>
      }
    />
  );
}

interface NamedBlockSectionProps {
  title: string;
  hint: string;
  entries: NamedBlockEntry[];
  namePlaceholder: string;
  bodyPlaceholder: string;
  addLabel: string;
  removeLabel: string;
  nameLabel: string;
  bodyLabel: string;
  emptyLabel: string;
  onAdd: () => void;
  onPatch: (index: number, patch: Partial<NamedBlockEntry>) => void;
  onRemove: (index: number) => void;
}

function NamedBlockSection({
  title,
  hint,
  entries,
  namePlaceholder,
  bodyPlaceholder,
  addLabel,
  removeLabel,
  nameLabel,
  bodyLabel,
  emptyLabel,
  onAdd,
  onPatch,
  onRemove,
}: NamedBlockSectionProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <div>
          <Label>{title}</Label>
          <p className="text-micro text-muted-foreground">{hint}</p>
        </div>
        <Button
          size="sm"
          variant="outline"
          onClick={onAdd}
          className="h-7 px-2 text-xs"
        >
          {addLabel}
        </Button>
      </div>
      {entries.length === 0 ? (
        <p className="text-micro text-muted-foreground">{emptyLabel}</p>
      ) : (
        <div className="space-y-2">
          {entries.map((entry, index) => (
            <div
              key={index}
              className="rounded-md border border-border bg-card p-2 space-y-1.5"
            >
              <div className="flex items-center gap-2">
                <Input
                  value={entry.name}
                  onChange={(e) => onPatch(index, { name: e.target.value })}
                  placeholder={namePlaceholder}
                  aria-label={nameLabel}
                  className="font-mono text-xs flex-1"
                />
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => onRemove(index)}
                  aria-label={removeLabel}
                  className="h-7 px-2 text-xs text-destructive hover:text-destructive shrink-0"
                >
                  ✕
                </Button>
              </div>
              <textarea
                value={entry.body}
                onChange={(e) => onPatch(index, { body: e.target.value })}
                placeholder={bodyPlaceholder}
                aria-label={bodyLabel}
                rows={3}
                spellCheck={false}
                className="w-full rounded-md border border-input bg-background px-2 py-1.5 font-mono text-xs resize-y"
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
