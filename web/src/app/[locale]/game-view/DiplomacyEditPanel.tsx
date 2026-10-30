"use client";

import { useTranslations } from "use-intl";
import { HistoryFileEditor } from "@/components/workbench/HistoryFileEditor";
import type { HistoryFileRow } from "@/lib/api/scenario";
import { GamePanel } from "./GamePanel";
import { RELATION_TYPES, diplomacyProblem } from "./diplomacyEdit";
import { useHistoryFileDraft } from "./historyFileEdit";
import { useHistoryRow } from "./startStateEdit";

export function DiplomacyEditPanel({
  projectId,
  tag,
  autonomyLevels,
  onSaved,
  onClose,
}: {
  projectId: number;
  tag: string;

  autonomyLevels: readonly { id: string; label?: string }[];
  onSaved: () => void;
  onClose: () => void;
}) {
  const tPanel = useTranslations("gameViewEdit.panel");
  const tSpirit = useTranslations("gameViewEdit.spirit");
  const t = useTranslations("gameViewWorld.diplomacy");
  const row = useHistoryRow(projectId, tag, true);
  return (
    <GamePanel
      title={t("title")}
      kind={tSpirit("historyKind", { tag })}
      source={row.isLoading ? undefined : row.data ? "项目" : "原版"}
      workbenchHref={
        row.data
          ? `/project/${projectId}/history-files?hf=${row.data.id}`
          : null
      }

      wide={row.data ? "workbench" : true}
      onClose={onClose}
    >
      {row.isLoading && (
        <p className="mt-4 text-micro text-muted-foreground">
          {tSpirit("loadingHistory")}
        </p>
      )}
      {!row.isLoading && !row.data && (
        <p className="mt-4 text-micro text-muted-foreground">
          {tPanel("notInProjectHistory")}
        </p>
      )}
      {row.data && (
        <div className="-mx-4 mt-2">
          <ProjectDiplomacy
            key={row.data.id}
            projectId={projectId}
            tag={tag}
            row={row.data}
            autonomyLevels={autonomyLevels}
            reload={() => row.mutate()}
            onSaved={onSaved}
          />
        </div>
      )}
    </GamePanel>
  );
}

function ProjectDiplomacy({
  projectId,
  tag,
  row,
  autonomyLevels,
  reload,
  onSaved,
}: {
  projectId: number;
  tag: string;
  row: HistoryFileRow;
  autonomyLevels: readonly { id: string; label?: string }[];

  reload: () => Promise<unknown>;
  onSaved: () => void;
}) {
  const t = useTranslations("gameViewWorld");
  const tSpirit = useTranslations("gameViewEdit.spirit");
  const history = useHistoryFileDraft({
    projectId,
    row,
    title: t("diplomacy.title"),
    kind: tSpirit("historyKind", { tag }),
    workbenchHref: `/project/${projectId}/history-files?hf=${row.id}`,
    check: (draft) => {
      const problem = diplomacyProblem(draft, tag);
      if (!problem) return null;
      return problem.key === "notTags"
        ? t("common.notTags", problem.values)
        : t(`diplomacy.problem.${problem.key}`);
    },
    afterSave: async () => {
      await reload().catch(() => undefined);
      onSaved();
    },
  });
  return (
    <HistoryFileEditor
      projectId={projectId}
      activeId={row.id}
      section="diplomacy"
      draft={history.draft}
      onChange={history.onChange}
      rows={[history.base]}
      diplomacyOptions={{ autonomyLevels, relationTypes: RELATION_TYPES }}
      saving={history.saving}
      saved={history.saved}
      dirty={history.dirty}
      error={history.error}
      onSave={() => void history.save()}
    />
  );
}
