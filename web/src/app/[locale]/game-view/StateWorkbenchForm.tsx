"use client";

import { useCallback, useMemo } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { StateEditor } from "@/components/workbench/StateEditor";
import { statesApi } from "@/lib/api/scenario";
import { resolveDisplayName } from "@/lib/i18n/localizedName";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { useStateStore } from "@/lib/store/stateStore";
import type { State, StateUpdate } from "@/types";
import { BaseTextEdit } from "./BaseTextEdit";
import { pick, recordChange, rowUndo } from "./changeLog";

export function StateWorkbenchForm({
  projectId,
  stateId,
  shownName,
  countryTags,
  onSaved,
}: {
  projectId: number;

  stateId: number;

  shownName: string;

  countryTags?: string[];
  onSaved: (id: number) => void;
}) {
  const t = useTranslations("states");
  const tw = useTranslations("gameViewWorld.state");
  const locale = useContentLocaleStore((s) => s.locale);
  const allStates = useStateStore((s) => s.states);
  const knownTags = useMemo(
    () =>
      [
        ...new Set(
          allStates.flatMap((s) => [s.owner, ...s.add_core_of]).filter(Boolean),
        ),
      ].sort(),
    [allStates],
  );

  const loaded = useSWR(
    ["game-view-state-store", projectId],
    async () => {
      await useStateStore.getState().fetchStates(projectId);
      return true;
    },
    { revalidateOnFocus: false },
  );
  const row = useStateStore(
    useCallback(
      (s) => s.states.find((st) => st.state_id === stateId) ?? null,
      [stateId],
    ),
  );

  const adopt = useSWR(
    loaded.data && !row
      ? ["game-view-state-store-adopt", projectId, stateId]
      : null,
    async () => {
      const res = await statesApi.seedFromVanilla(projectId, {
        state_ids: [stateId],
      });
      await useStateStore.getState().fetchStates(projectId);
      return res;
    },
    {
      revalidateOnFocus: false,
      revalidateIfStale: false,
      shouldRetryOnError: false,
    },
  );
  const save = useCallback(
    async (id: number, data: StateUpdate): Promise<State> => {
      const before = useStateStore.getState().states.find((st) => st.id === id);
      const saved = await useStateStore
        .getState()
        .updateState(projectId, id, data);
      const keys = Object.keys(data);
      recordChange({
        projectId,
        title:
          resolveDisplayName(
            locale,
            saved.display_name ?? saved.display_name_locs,
            null,
            null,
            "",
          ) || `STATE_${saved.state_id}`,
        kind: tw("logKind", { id: saved.state_id }),
        workbenchHref: `/project/${projectId}/states?state=${saved.state_id}`,

        undo:
          before && !keys.includes("provinces")
            ? rowUndo({
                before: pick(before, keys),
                after: pick(saved, keys),
                load: () => statesApi.get(projectId, id),
                save: (back) => statesApi.update(projectId, id, back),
              })
            : undefined,
      });
      onSaved(saved.state_id);
      return saved;
    },
    [projectId, onSaved, locale, tw],
  );

  if (!row) {
    if (loaded.error || adopt.error) {
      const error = loaded.error ?? adopt.error;
      return (
        <div className="space-y-2">
          <p className="text-micro text-destructive">
            {tw("loadFailed", {
              error: error instanceof Error ? error.message : String(error),
            })}
          </p>

          <BaseTextEdit
            projectId={projectId}
            title={shownName || `STATE_${stateId}`}
            fields={[{ field: "name", key: `STATE_${stateId}` }]}
            onSaved={() => onSaved(stateId)}
          />
        </div>
      );
    }
    if (adopt.data && adopt.data.missing.includes(stateId)) {
      return (
        <p className="text-micro text-muted-foreground">
          {tw("missing", { id: stateId })}
        </p>
      );
    }
    return (
      <p className="text-micro text-muted-foreground">
        {adopt.isLoading ? tw("adopting") : tw("loading")}
      </p>
    );
  }

  const display =
    resolveDisplayName(
      locale,
      row.display_name ?? row.display_name_locs,
      null,
      null,
      "",
    ) || shownName;
  const membership =
    row.membership_provenance?.status !== "known" ||
    row.membership_provenance.membership_changed == null
      ? t("membershipFlagUnknown")
      : row.membership_provenance.membership_changed
        ? t("membershipFlagChanged")
        : t("membershipFlagUnchanged");
  return (
    <div className="space-y-3">
      <section className="space-y-1 rounded-md border border-border/60 bg-muted/30 p-3 text-xs">
        <h4 className="text-micro font-semibold uppercase tracking-widest text-muted-foreground">
          {t("summaryTitle")}
        </h4>
        <SummaryRow label={t("stateId")} value={String(row.state_id)} />
        <SummaryRow label={t("displayName")} value={display || "—"} />
        <SummaryRow label={t("owner")} value={row.owner || "—"} />
        <SummaryRow
          label={t("controller")}
          value={row.controller || t("controllerSameAsOwner")}
        />
        <SummaryRow
          label={t("manpower")}
          value={row.manpower.toLocaleString()}
        />
        <SummaryRow
          label={t("provinceCount")}
          value={String(row.provinces.length)}
        />
        <SummaryRow label={t("membershipFlag")} value={membership} />
        <SummaryRow
          label={t("cores")}
          value={row.add_core_of.length ? row.add_core_of.join(", ") : "—"}
        />
      </section>

      <div className="-mx-4">
        <StateEditor
          state={row}
          countryTags={countryTags ?? knownTags}
          onSave={save}
        />
      </div>
    </div>
  );
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline gap-2">
      <span className="w-24 shrink-0 text-muted-foreground">{label}</span>
      <span className="break-all font-mono">{value}</span>
    </div>
  );
}
