"use client";

import { useTranslations } from "use-intl";
import useSWR from "swr";
import { statesApi } from "@/lib/api/scenario";
import type { State } from "@/types";
import type { GuiStates } from "./StateLayer";
import { FIELD, GamePanel } from "./GamePanel";
import { StateWorkbenchForm } from "./StateWorkbenchForm";

export function useStateRows(projectId: number) {
  return useSWR(
    ["game-view-states-db", projectId],
    () => statesApi.list(projectId),
    { revalidateOnFocus: false },
  );
}

export function StateEditPanel({
  projectId,
  states,
  onPickState,
  onSaved,
  onClose,
}: {
  projectId: number;

  states: GuiStates | undefined;
  onPickState: (id: number) => void;

  onSaved: (id: number) => void;
  onClose: () => void;
}) {
  const tw = useTranslations("gameViewWorld.state");
  const shown = states?.state;
  const rows = useStateRows(projectId);
  const row: State | null =
    (shown && rows.data?.find((r) => r.state_id === shown.id)) || null;
  return (
    <GamePanel
      title={shown?.name || tw("fallbackTitle")}
      kind={tw("kind", { id: shown?.id ?? "…" })}
      source={
        rows.isLoading ? undefined : row ? "项目" : shown ? "原版" : undefined
      }
      workbenchHref={
        shown ? `/project/${projectId}/states?state=${shown.id}` : null
      }
      wide="workbench"
      onClose={onClose}
    >
      {states && states.states.length > 0 && (
        <label className="mt-3 block">
          <span className="text-micro font-medium text-muted-foreground">
            {tw("countryStates")}
          </span>
          <select
            aria-label={tw("pick")}
            value={shown?.id ?? ""}
            onChange={(e) => onPickState(Number(e.target.value))}
            className={FIELD}
          >
            {states.states.map((s) => (
              <option key={s.id} value={s.id}>
                {s.id} · {s.name || s.id}
              </option>
            ))}
          </select>
        </label>
      )}
      {shown && (
        <div className="mt-3">
          <StateWorkbenchForm
            key={shown.id}
            projectId={projectId}
            stateId={shown.id}
            shownName={shown.name ?? ""}
            onSaved={(id) => {
              void rows.mutate();
              onSaved(id);
            }}
          />
        </div>
      )}
    </GamePanel>
  );
}
