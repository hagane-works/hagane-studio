"use client";

import { mapResponseProblem } from "@/lib/mapToolRequest";
import { MapProblemError } from "@/app/[locale]/game-view/map-studio/mapProblems";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import type { Province, ProvinceKind, WorldMap } from "./worldMapModel";

export const LAND_TERRAINS = [
  "plains",
  "forest",
  "hills",
  "mountain",
  "desert",
  "jungle",
  "urban",
  "marsh",
] as const;

export interface ProvinceDraft {
  province: number;
  kind: ProvinceKind;
  terrain: string;
  continent: number;
  state: number;
  region: number;
}

export type PickTarget = "state" | "region";

export function draftOf(province: Province): ProvinceDraft {
  return {
    province: province.id,
    kind: province.kind,
    terrain: province.terrain,
    continent: province.continent,
    state: province.state,
    region: province.region,
  };
}

export function provinceChanges(
  province: Province,
  draft: ProvinceDraft,
): Record<string, string | number> {
  const out: Record<string, string | number> = {};
  if (draft.kind !== province.kind) out.kind = draft.kind;
  if (draft.terrain !== province.terrain) out.terrain = draft.terrain;
  if (draft.continent !== province.continent) out.continent = draft.continent;
  if (draft.state !== province.state && draft.state > 0)
    out.state = draft.state;
  if (draft.region !== province.region && draft.region > 0)
    out.region = draft.region;
  return out;
}

export async function saveProvince(
  projectId: number,
  province: number,
  changes: Record<string, string | number>,
): Promise<string[]> {
  const response = await fetch(
    `/api/go/projects/${projectId}/gui-render/world-map/province`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ province, ...changes }),
    },
  );

  if (!response.ok)
    throw new MapProblemError(await mapResponseProblem(response));
  const out = (await response.json().catch(() => ({}))) as {
    changed?: string[];
  };
  return out.changed ?? [];
}

const SELECT =
  "ui-focus-ring h-8 w-full rounded-md border border-input bg-card px-2 text-xs";

export function ProvinceEditor({
  map,
  province,
  draft,
  onDraft,
  picking,
  onPick,
  busy,
  error,
  onSave,
  onCancel,
}: {
  map: WorldMap;
  province: Province;
  draft: ProvinceDraft;
  onDraft: (draft: ProvinceDraft) => void;

  picking: PickTarget | null;
  onPick: (target: PickTarget | null) => void;
  busy: boolean;
  error: string | null;
  onSave: () => void;
  onCancel: () => void;
}) {
  const t = useTranslations("worldMap");
  const land = province.kind === "land";
  const changes = provinceChanges(province, draft);
  const stateName = (id: number) =>
    id ? `#${id} ${map.states.get(id)?.name ?? ""}` : t("noState");
  const regionName = (id: number) =>
    id ? `#${id} ${map.regions.get(id) ?? ""}` : "—";
  const pickButton = (target: PickTarget) => (
    <Button
      size="sm"
      variant={picking === target ? "default" : "outline"}
      aria-pressed={picking === target}
      onClick={() => onPick(picking === target ? null : target)}
    >
      {picking === target ? t("editPicking") : t("editPick")}
    </Button>
  );

  return (
    <div className="space-y-3 rounded-md border border-border p-3">
      <div className="text-xs text-muted-foreground">{t("editTitle")}</div>
      <div className="space-y-1">
        <div className="text-xs text-muted-foreground">{t("editKind")}</div>
        {land ? (
          <p className="text-xs">{t("kindLand")}</p>
        ) : (
          <SegmentedControl
            label={t("editKind")}
            size="sm"
            value={draft.kind}
            onChange={(kind) => onDraft({ ...draft, kind })}
            options={[
              { value: "sea", label: t("kindSea") },
              { value: "lake", label: t("kindLake") },
            ]}
          />
        )}
        <p className="text-xs text-muted-foreground">{t("editKindNote")}</p>
      </div>
      {land && (
        <>
          <label className="block text-xs">
            <span className="text-muted-foreground">{t("terrain")}</span>
            <select
              className={`${SELECT} mt-1`}
              value={draft.terrain}
              onChange={(e) => onDraft({ ...draft, terrain: e.target.value })}
            >
              {LAND_TERRAINS.map((terrain) => (
                <option key={terrain} value={terrain}>
                  {t(`terrain_${terrain}`)}
                </option>
              ))}
            </select>
          </label>
          <label className="block text-xs">
            <span className="text-muted-foreground">{t("continent")}</span>
            <select
              className={`${SELECT} mt-1`}
              value={draft.continent}
              onChange={(e) =>
                onDraft({ ...draft, continent: Number(e.target.value) })
              }
            >
              {map.continents.map((name, i) => (
                <option key={name} value={i + 1}>
                  {name}
                </option>
              ))}
            </select>
          </label>
          <div className="space-y-1 text-xs">
            <div className="text-muted-foreground">{t("state")}</div>
            <div className="flex items-center justify-between gap-2">
              <span className="truncate">{stateName(draft.state)}</span>
              {pickButton("state")}
            </div>
          </div>
        </>
      )}
      <div className="space-y-1 text-xs">
        <div className="text-muted-foreground">{t("region")}</div>
        <div className="flex items-center justify-between gap-2">
          <span className="truncate">{regionName(draft.region)}</span>
          {pickButton("region")}
        </div>
      </div>
      {picking && <p className="text-xs text-warning">{t("editPickHint")}</p>}
      {error && <p className="text-xs text-destructive">{error}</p>}
      <div className="flex gap-2">
        <Button
          size="sm"
          disabled={busy || Object.keys(changes).length === 0}
          onClick={onSave}
        >
          {busy ? t("editSaving") : t("editSave")}
        </Button>
        <Button size="sm" variant="ghost" disabled={busy} onClick={onCancel}>
          {t("editCancel")}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">{t("editNote")}</p>
    </div>
  );
}
