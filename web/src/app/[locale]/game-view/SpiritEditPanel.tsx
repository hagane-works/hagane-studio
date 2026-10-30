"use client";

import useSWR from "swr";
import { useTranslations } from "use-intl";
import { DynamicModifierEditor } from "@/components/workbench/DynamicModifierEditor";
import IdeaEditor from "@/components/workbench/IdeaEditor";
import { ideasApi } from "@/lib/api/content";
import { dynamicModifiersApi } from "@/lib/api/tooling";
import type { DynamicModifier, Idea } from "@/types";
import { GamePanel } from "./GamePanel";
import { spiritName, type SpiritRow } from "./spiritEdit";
import { pick, recordChange, rowUndo } from "./changeLog";
import { BaseTextEdit } from "./BaseTextEdit";
import { changedKeys, useWorkbenchProject } from "./workbenchProject";

export function SpiritEditPanel({
  projectId,
  spirit,
  onSaved,
  onClose,
  onManageStart,
}: {
  projectId: number;

  spirit: { token: string; label?: string };
  onSaved: (label: string) => void;
  onClose: () => void;

  onManageStart?: () => void;
}) {
  const t = useTranslations("gameViewEdit");
  const ideas = useSWR(
    ["game-view-ideas-db", projectId],
    () => ideasApi.list(projectId),
    { revalidateOnFocus: false },
  );
  const dynamics = useSWR(
    ["game-view-dynamic-modifiers-db", projectId],
    () => dynamicModifiersApi.list(projectId),
    {
      revalidateOnFocus: false,
    },
  );
  const idea = ideas.data?.find((row: Idea) => row.idea_id === spirit.token);
  const dynamic = dynamics.data?.dynamic_modifiers.find(
    (row: DynamicModifier) => row.dm_id === spirit.token,
  );
  const found: SpiritRow | null = idea
    ? { kind: "idea", row: idea }
    : dynamic
      ? { kind: "dynamic", row: dynamic }
      : null;
  const loading = ideas.isLoading || dynamics.isLoading;
  const shown = spirit.label || spirit.token;

  const kind = t(
    found?.kind === "dynamic" ? "spirit.kindDynamic" : "spirit.kindSpirit",
    { token: spirit.token },
  );
  const workbenchHref =
    found?.kind === "idea"
      ? `/project/${projectId}/ideas?idea=${found.row.id}`
      : found?.kind === "dynamic"
        ? `/project/${projectId}/dynamic-modifiers`
        : null;

  const projectReady = useWorkbenchProject(projectId);
  const saveIdea = async (data: Partial<Idea>) => {
    if (!idea) return;
    const id = idea.id;
    const keys = changedKeys(idea, data);
    const before = pick(idea, keys);
    const saved = await ideasApi.update(projectId, id, data);
    const name = spiritName(
      { kind: "idea", row: saved },
      spirit.label ?? spirit.token,
    );
    recordChange({
      projectId,
      title: name || shown,
      kind,
      workbenchHref,
      undo: keys.length
        ? rowUndo<Idea>({
            before,
            after: pick(saved, keys),
            load: async () =>
              (await ideasApi.list(projectId)).find((row) => row.id === id),
            save: (back) => ideasApi.update(projectId, id, back),
          })
        : undefined,
    });
    await ideas.mutate();
    onSaved(name || shown);
  };
  const saveDynamic = async (data: Partial<DynamicModifier>) => {
    if (found?.kind !== "dynamic") return;
    const row = found.row;
    const id = row.id;
    const keys = changedKeys(row, data);
    const before = pick(row, keys);
    const saved = await dynamicModifiersApi.update(projectId, id, data);
    const name = spiritName(
      { kind: "dynamic", row: saved },
      spirit.label ?? spirit.token,
    );
    recordChange({
      projectId,
      title: name || shown,
      kind,
      workbenchHref,
      undo: keys.length
        ? rowUndo<DynamicModifier>({
            before,
            after: pick(saved, keys),
            load: () => dynamicModifiersApi.get(projectId, id),
            save: (back) => dynamicModifiersApi.update(projectId, id, back),
          })
        : undefined,
    });
    await dynamics.mutate();
    onSaved(name || shown);
  };

  return (
    <GamePanel
      title={shown}
      kind={kind}
      source={loading ? undefined : found ? "项目" : "原版"}
      workbenchHref={workbenchHref}

      wide={found ? "workbench" : true}
      onClose={onClose}
    >
      {loading && (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("spirit.loading")}
        </p>
      )}
      {!loading && !found && (
        <BaseTextEdit
          projectId={projectId}
          title={shown}
          fields={[
            { field: "name", key: spirit.token },
            { field: "description", key: `${spirit.token}_desc` },
          ]}
          onSaved={(key, value) =>
            onSaved(key === spirit.token && value !== null ? value : shown)
          }
        />
      )}
      {found && !projectReady && (
        <p className="mt-4 text-micro text-muted-foreground">
          {t("panel.loadingProject")}
        </p>
      )}
      {found?.kind === "idea" && projectReady && (
        <div className="-mx-4 mt-2">
          <IdeaEditor
            key={found.row.id}
            idea={found.row}
            projectId={projectId}
            allIdeas={ideas.data}
            onSave={saveIdea}
            onPictureUploaded={() => {
              void ideas.mutate();
              onSaved(shown);
            }}
            embedded
          />
        </div>
      )}
      {found?.kind === "dynamic" && projectReady && (
        <div className="-mx-4 mt-2">
          <DynamicModifierEditor
            key={found.row.id}
            dm={found.row}
            onSave={saveDynamic}
          />
        </div>
      )}
      {onManageStart && (
        <button
          type="button"
          onClick={onManageStart}
          className="mt-4 rounded border border-border px-2 py-0.5 text-micro hover:bg-muted"
        >
          {t("spirit.manageStart")}
        </button>
      )}
    </GamePanel>
  );
}
