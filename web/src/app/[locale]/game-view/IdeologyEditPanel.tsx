"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";

import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import {
  CountryPoliticsEditor,
  type CountryPoliticsSaving,
} from "@/components/workbench/politics/country-politics/CountryPoliticsEditor";
import {
  countryLeaders,
  historyToPoliticsDraft,
  leaderSubtypeDrafts,
  politicsHistoryPayload,
  politicsReadbackMatches,
  validatePopularities,
  type CountryPoliticsDraft,
} from "@/components/workbench/politics/country-politics/model";
import { IdeologyEditor } from "@/components/workbench/politics/ideologies/IdeologyEditor";
import {
  definitionDraft,
  definitionDraftDirty,
  definitionPatch,
  ideologyDisplayName,
  projectDefinition,
  targetKey,
  type IdeologyDefinitionDraft,
  type ProjectDefinition,
} from "@/components/workbench/politics/ideologies/ideologyDefinition";
import { charactersApi } from "@/lib/api";
import { politicsApi, type PoliticsIdeology } from "@/lib/api/politics";
import { projectsApi } from "@/lib/api/project";
import { historyFilesApi, type HistoryFileRow } from "@/lib/api/scenario";
import { ideologyIconsApi } from "@/lib/api/tooling";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { useIdeologyOptions } from "@/lib/ideology/useIdeologyOptions";
import { normalizeRegistry } from "@/lib/ideology/registry";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { useProjectStore } from "@/lib/store/projectStore";
import type { Character } from "@/types";
import { GamePanel } from "./GamePanel";
import { pick, recordChange, rowUndo } from "./changeLog";
import { useHistoryRow } from "./startStateEdit";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

const IDEOLOGY_SECTION_ID = "game-view-ideology-definition";

export function IdeologyEditPanel({
  projectId,
  tag,
  focus,
  onSaved,
  onClose,
}: {
  projectId: number;
  tag: string;

  focus: string | null;

  colors: Readonly<Record<string, [number, number, number]>>;
  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("politicsPanel");
  const tPanel = useTranslations("gameViewEdit.panel");
  const tw = useTranslations("gameViewWorld.ideology");
  const contentLocale = useContentLocaleStore((state) => state.locale);
  const exportLanguages = useProjectStore(
    (state) =>
      state.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );
  const projectReady = useWorkbenchProject(projectId);
  const { groupOptions, subOptions } = useIdeologyOptions();
  const row = useHistoryRow(projectId, tag, true);
  const people = useSWR(
    ["game-view-country-politics-characters", projectId, tag],
    async () => countryLeaders(await charactersApi.list(projectId), tag),
    { revalidateOnFocus: false },
  );
  const definitions = useSWR(
    ["game-view-politics-definitions", projectId],
    async () => {
      const [snapshot, project] = await Promise.all([
        politicsApi.references(projectId),
        projectsApi.get(projectId),
      ]);
      return {
        snapshot,
        registry: normalizeRegistry(project.ideology_registry),
        icons: project.ideology_icon_paths ?? {},
      };
    },
    { revalidateOnFocus: false },
  );

  const ideologies = useMemo(
    () => definitions.data?.snapshot.ideologies ?? [],
    [definitions.data?.snapshot.ideologies],
  );
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [seenFocus, setSeenFocus] = useState(focus);
  if (seenFocus !== focus) {
    setSeenFocus(focus);
    setSelectedKey(null);
  }
  const focused = focus
    ? ideologies.find(
        (ideology) => ideology.kind === "root" && ideology.script_id === focus,
      )
    : undefined;
  const selected =
    ideologies.find((ideology) => targetKey(ideology) === selectedKey) ??
    focused ??
    ideologies.find((ideology) => ideology.kind === "root") ??
    ideologies[0] ??
    null;
  const registry = definitions.data?.registry ?? normalizeRegistry(undefined);

  if (!projectReady) {
    return (
      <GamePanel
        title={tw("title")}
        kind={tw("kind", { tag })}
        workbenchHref={`/project/${projectId}/country-politics`}
        wide="workbench"
        onClose={onClose}
      >
        <p className="mt-4 text-micro text-muted-foreground">
          {tPanel("loadingProject")}
        </p>
      </GamePanel>
    );
  }

  return (
    <GamePanel
      title={tw("title")}
      kind={tw("kind", { tag })}
      source={row.isLoading ? undefined : row.data ? "项目" : "原版"}
      workbenchHref={`/project/${projectId}/country-politics`}
      wide="workbench"
      onClose={onClose}
    >
      <div className="-mx-4 mt-2 space-y-5">
        {row.isLoading || people.isLoading ? (
          <p className="px-4 py-3 text-micro text-muted-foreground">
            {tw("loadingPolitics")}
          </p>
        ) : !row.data ? (
          <p className="px-4 py-3 text-micro text-muted-foreground">
            {tPanel("notInProjectHistory")}
          </p>
        ) : (
          <ProjectCountryPolitics
            key={`${row.data.id}:${groupOptions.map((option) => option.value).join(",")}`}
            projectId={projectId}
            tag={tag}
            row={row.data}
            roots={groupOptions}
            subtypes={subOptions}
            leaders={people.data ?? []}
            reloadHistory={() => row.mutate()}
            reloadLeaders={() => people.mutate()}
            onSaved={onSaved}
            onOpenIdeologies={() =>
              document
                .getElementById(IDEOLOGY_SECTION_ID)
                ?.scrollIntoView({ behavior: "smooth" })
            }
          />
        )}

        <section
          id={IDEOLOGY_SECTION_ID}
          className="space-y-3 border-t border-border px-4 pt-4"
          aria-label={tw("definitions")}
        >
          <div className="flex items-end justify-between gap-3">
            <div>
              <h2 className="text-sm font-semibold">{tw("definitions")}</h2>
              <p className="text-xs text-muted-foreground">
                {tw("definitionsHint")}
              </p>
            </div>
            {ideologies.length > 0 && (
              <label className="min-w-56 text-xs text-muted-foreground">
                {tw("target")}
                <select
                  aria-label={tw("targetLabel")}
                  value={selected ? targetKey(selected) : ""}
                  onChange={(event) => setSelectedKey(event.target.value)}
                  className="mt-1 h-9 w-full rounded-md border border-input bg-card px-2 text-xs"
                >
                  {ideologies.map((ideology) => (
                    <option
                      key={targetKey(ideology)}
                      value={targetKey(ideology)}
                    >
                      {ideologyDisplayName(registry, ideology, contentLocale)} ·{" "}
                      {ideology.script_id}
                    </option>
                  ))}
                </select>
              </label>
            )}
          </div>

          {definitions.isLoading ? (
            <p className="py-3 text-micro text-muted-foreground">
              {tw("loadingDefinitions")}
            </p>
          ) : definitions.error ? (
            <p role="alert" className="py-3 text-xs text-destructive">
              {definitions.error instanceof Error
                ? definitions.error.message
                : String(definitions.error)}
            </p>
          ) : selected ? (
            <ProjectIdeology
              key={`${targetKey(selected)}:${JSON.stringify(projectDefinition(registry, selected))}`}
              projectId={projectId}
              ideology={selected}
              definition={projectDefinition(registry, selected)}
              displayName={ideologyDisplayName(
                registry,
                selected,
                contentLocale,
              )}
              sourceLabel={t(`source.${selected.source}`)}
              exportLanguages={exportLanguages}
              iconAvailable={Boolean(
                definitions.data?.icons[selected.script_id],
              )}
              reload={() => definitions.mutate()}
              onSaved={onSaved}
            />
          ) : (
            <p className="py-3 text-micro text-muted-foreground">
              {tw("none")}
            </p>
          )}
          <p className="pb-2 text-micro text-muted-foreground">
            {tw("workbenchNote")}
          </p>
        </section>
      </div>
    </GamePanel>
  );
}

function ProjectCountryPolitics({
  projectId,
  tag,
  row,
  roots,
  subtypes,
  leaders,
  reloadHistory,
  reloadLeaders,
  onSaved,
  onOpenIdeologies,
}: {
  projectId: number;
  tag: string;
  row: HistoryFileRow;
  roots: ReturnType<typeof useIdeologyOptions>["groupOptions"];
  subtypes: ReturnType<typeof useIdeologyOptions>["subOptions"];
  leaders: Character[];
  reloadHistory: () => Promise<unknown>;
  reloadLeaders: () => Promise<unknown>;
  onSaved: (message: string) => void;
  onOpenIdeologies: () => void;
}) {
  const t = useTranslations("countrySettingsDialog.workbench.politics");
  const tw = useTranslations("gameViewWorld.ideology");
  const tSpirit = useTranslations("gameViewEdit.spirit");
  const [base, setBase] = useState(row);
  const [draft, setDraft] = useState<CountryPoliticsDraft>(() =>
    historyToPoliticsDraft(row, roots),
  );
  const [savedDraft, setSavedDraft] = useState(draft);
  const [leaderRows, setLeaderRows] = useState(leaders);
  const [leaderDrafts, setLeaderDrafts] = useState<Record<number, string>>(() =>
    leaderSubtypeDrafts(leaders),
  );
  const [savedLeaderDrafts, setSavedLeaderDrafts] = useState<
    Record<number, string>
  >(() => leaderSubtypeDrafts(leaders));
  const [saving, setSaving] = useState<CountryPoliticsSaving>(null);
  const [status, setStatus] = useState<"" | "saved">("");
  const [error, setError] = useState<string | null>(null);
  const dirty =
    JSON.stringify(draft) !== JSON.stringify(savedDraft) ||
    JSON.stringify(leaderDrafts) !== JSON.stringify(savedLeaderDrafts);
  useRegisterDirty(
    `game-view-country-politics:${projectId}:${row.id}`,
    dirty || saving !== null,
  );
  const workbenchHref = `/project/${projectId}/country-politics`;

  const editPolitics = (
    update: (current: CountryPoliticsDraft) => CountryPoliticsDraft,
  ) => {
    setDraft(update);
    setStatus("");
    setError(null);
  };
  const editLeaders = (
    update: (current: Record<number, string>) => Record<number, string>,
  ) => {
    setLeaderDrafts(update);
    setStatus("");
    setError(null);
  };

  const savePolitics = async () => {
    if (!validatePopularities(draft.popularities).valid) return;
    const payload = politicsHistoryPayload(draft);
    const keys = changedKeys(base, payload);
    if (keys.length === 0) return;
    const before = pick(base, keys);
    setSaving("politics");
    setStatus("");
    setError(null);
    try {
      const saved = await historyFilesApi.update(projectId, base.id, payload);
      const readback = await historyFilesApi.get(projectId, saved.id);
      if (!politicsReadbackMatches(readback, draft))
        throw new Error(t("readbackMismatch"));
      recordChange({
        projectId,
        title: tw("politicsLog", { tag }),
        kind: tSpirit("historyKind", { tag }),
        workbenchHref,
        undo: rowUndo<HistoryFileRow>({
          before,
          after: pick(readback, keys),
          load: () => historyFilesApi.get(projectId, readback.id),
          save: (back) => historyFilesApi.update(projectId, readback.id, back),
        }),
      });
      const next = historyToPoliticsDraft(readback, roots);
      setBase(readback);
      setDraft(next);
      setSavedDraft(next);
      setStatus("saved");
      await reloadHistory().catch(() => undefined);
      onSaved(tw("politicsSaved", { tag }));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setSaving(null);
    }
  };

  const saveLeader = async (leader: Character) => {
    const ideology = leaderDrafts[leader.id] ?? "";
    if (!subtypes.some((option) => option.value === ideology)) {
      setError(t("unknownSubtype"));
      return;
    }
    setSaving(`leader-${leader.id}`);
    setStatus("");
    setError(null);
    try {
      await charactersApi.update(projectId, leader.id, { ideology });
      const readback = await charactersApi.get(projectId, leader.id);
      if (readback.ideology !== ideology)
        throw new Error(t("readbackMismatch"));
      recordChange({
        projectId,
        title: readback.name_en || readback.name || readback.tag,
        kind: tw("leaderKind", { token: readback.tag }),
        workbenchHref,
        undo: rowUndo<Character>({
          before: { ideology: leader.ideology },
          after: { ideology: readback.ideology },
          load: () => charactersApi.get(projectId, leader.id),
          save: (back) => charactersApi.update(projectId, leader.id, back),
        }),
      });
      setLeaderRows((current) =>
        current.map((item) => (item.id === readback.id ? readback : item)),
      );
      setLeaderDrafts((current) => ({
        ...current,
        [readback.id]: readback.ideology,
      }));
      setSavedLeaderDrafts((current) => ({
        ...current,
        [readback.id]: readback.ideology,
      }));
      setStatus("saved");
      await reloadLeaders().catch(() => undefined);
      onSaved(
        tw("leaderSaved", {
          name: readback.name_en || readback.name || readback.tag,
        }),
      );
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setSaving(null);
    }
  };

  return (
    <CountryPoliticsEditor
      country={{ tag, name: tag, name_en: tag }}
      roots={roots}
      subtypes={subtypes}
      savedRulingParty={base.set_politics_ruling_party}
      draft={draft}
      onEdit={editPolitics}
      leaders={leaderRows}
      leaderDrafts={leaderDrafts}
      onEditLeaders={editLeaders}
      saving={saving}
      onSavePolitics={() => void savePolitics()}
      onSaveLeader={(leader) => void saveLeader(leader)}
      onOpenIdeologies={onOpenIdeologies}
      status={status}
      error={error}
    />
  );
}

function ProjectIdeology({
  projectId,
  ideology,
  definition,
  displayName,
  sourceLabel,
  exportLanguages,
  iconAvailable,
  reload,
  onSaved,
}: {
  projectId: number;
  ideology: PoliticsIdeology;
  definition: ProjectDefinition;
  displayName: string;
  sourceLabel: string;
  exportLanguages: readonly string[];
  iconAvailable: boolean;
  reload: () => Promise<unknown>;
  onSaved: (message: string) => void;
}) {
  const tw = useTranslations("gameViewWorld.ideology");
  const [draft, setDraft] = useState<IdeologyDefinitionDraft>(() =>
    definitionDraft(definition),
  );
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [iconBust, setIconBust] = useState(0);
  const [status, setStatus] = useState<"" | "saved">("");
  const [error, setError] = useState<string | null>(null);
  const dirty = definitionDraftDirty(ideology, definition, draft);
  useRegisterDirty(
    `game-view-ideology:${projectId}:${targetKey(ideology)}`,
    dirty || saving || uploading,
  );
  const workbenchHref = `/project/${projectId}/country-politics`;

  const change = (next: IdeologyDefinitionDraft) => {
    setDraft(next);
    setStatus("");
    setError(null);
  };
  const save = async () => {
    if (!dirty || ideology.source !== "project_managed") return;
    setSaving(true);
    setStatus("");
    setError(null);
    try {
      await politicsApi.updateDefinition(
        projectId,
        { kind: ideology.kind, script_id: ideology.script_id },
        definitionPatch(ideology.kind, draft),
      );
      recordChange({
        projectId,
        title: tw("definitionLog", { name: displayName }),
        kind: tw("definitionKind", { id: ideology.script_id }),
        workbenchHref,
      });
      setStatus("saved");
      await reload();
      onSaved(tw("definitionSaved", { name: displayName }));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setSaving(false);
    }
  };
  const upload = async (file: File) => {
    if (ideology.kind !== "root") return;
    setUploading(true);
    setStatus("");
    setError(null);
    try {
      await ideologyIconsApi.upload(projectId, ideology.script_id, file);
      recordChange({
        projectId,
        title: tw("iconLog", { name: displayName }),
        kind: tw("definitionKind", { id: ideology.script_id }),
        workbenchHref,
      });
      setIconBust((value) => value + 1);
      setStatus("saved");
      await reload();
      onSaved(tw("iconSaved", { name: displayName }));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="space-y-2">
      <IdeologyEditor
        projectId={projectId}
        ideology={ideology}
        displayName={displayName}
        sourceLabel={sourceLabel}
        draft={draft}
        exportLanguages={exportLanguages}
        onChange={change}
        saving={saving}
        onSave={() => void save()}
        iconAvailable={iconAvailable}
        iconBust={iconBust}
        uploadingIcon={uploading}
        onUploadIcon={(file) => void upload(file)}
      />
      {(status || error) && (
        <p
          role="status"
          className={`rounded-md border p-3 text-sm ${error ? "border-destructive/30 bg-destructive/10 text-destructive" : "border-success/30 bg-success/10 text-success"}`}
        >
          {error ?? tw("saved")}
        </p>
      )}
    </div>
  );
}
