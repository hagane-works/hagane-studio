"use client";

import { useMemo, useState } from "react";
import { toast } from "sonner";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Link } from "@/i18n/navigation";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { OpenInGameLink } from "@/components/workbench/OpenInGameLink";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { MapNameEditor } from "@/components/workbench/world-map/MapNameEditor";
import {
  NewLandPanel,
  useNewLand,
} from "@/components/workbench/world-map/NewLandPanel";
import { NewWorldPanel } from "@/components/workbench/world-map/NewWorldPanel";
import {
  draftOf,
  ProvinceEditor,
  provinceChanges,
  saveProvince,
  type PickTarget,
  type ProvinceDraft,
} from "@/components/workbench/world-map/ProvinceEditor";
import { useMapErrorText } from "@/app/[locale]/game-view/map-studio/mapErrorText";
import { SketchTrialDialog } from "@/components/workbench/world-map/SketchTrialDialog";
import {
  clearRewrite,
  loadRewrites,
  nameKey,
  nameLanguages,
  setRewrite,
  type NameKind,
} from "@/components/workbench/world-map/mapNames";
import {
  WorldMapCanvas,
  type MapFocus,
} from "@/components/workbench/world-map/WorldMapCanvas";
import {
  parseWorldMap,
  searchNames,
  type Layer,
} from "@/components/workbench/world-map/worldMapModel";
import { loadWorldMap } from "@/components/workbench/world-map/worldMapApi";
import { statesApi } from "@/lib/api/scenario";
import { gameViewHref } from "@/lib/gameViewLinks";
import { useProjectStore } from "@/lib/store/projectStore";

const DEFAULT_DATE = "1936.1.1";

export default function WorldMapPage() {
  const params = useParams();
  const projectId = Number(params.id);
  const t = useTranslations("worldMap");
  const errorText = useMapErrorText();
  const startDate = useProjectStore((s) => s.currentProject?.start_date);
  const date = /^\d{1,4}\.\d{1,2}\.\d{1,2}(\.\d{1,2})?$/.test(startDate ?? "")
    ? (startDate as string)
    : DEFAULT_DATE;
  const data = useSWR(
    ["world-map", projectId, date],
    () => loadWorldMap(projectId, date),
    { revalidateOnFocus: false },
  );
  const map = useMemo(
    () => (data.data ? parseWorldMap(data.data) : null),
    [data.data],
  );
  const [layer, setLayer] = useState<Layer>("political");
  const [selected, setSelected] = useState(0);
  const [hovered, setHovered] = useState(0);
  const [focus, setFocus] = useState<MapFocus | null>(null);
  const [query, setQuery] = useState("");
  const [sketchOpen, setSketchOpen] = useState(false);
  const [landMode, setLandMode] = useState(false);
  const [worldMode, setWorldMode] = useState(false);

  const [mapVersion, setMapVersion] = useState(0);
  const land = useNewLand(projectId, landMode);

  const [draft, setDraft] = useState<ProvinceDraft | null>(null);
  const [picking, setPicking] = useState<PickTarget | null>(null);
  const [editBusy, setEditBusy] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);
  const hits = useMemo(
    () => (map ? searchNames(map, query) : []),
    [map, query],
  );

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages,
  );
  const languages = useMemo(
    () => nameLanguages(exportLanguages),
    [exportLanguages],
  );
  const rewrites = useSWR(
    ["world-map-rewrites", projectId, languages.join(",")],
    async () => {
      const lanes = await Promise.all(
        languages.map((lang) => loadRewrites(projectId, lang)),
      );
      return Object.fromEntries(
        languages.map((lang, i) => [lang, lanes[i]]),
      ) as Record<string, Record<string, string>>;
    },
  );
  const stateRows = useSWR(["world-map-state-rows", projectId], async () => {
    const rows = await statesApi.list(projectId);
    return new Map(rows.map((row) => [row.state_id, row]));
  });
  const savedNames = (
    kind: NameKind,
    id: number,
  ): Record<string, string | undefined> => {
    const row = kind === "state" ? stateRows.data?.get(id) : undefined;
    if (row)
      return Object.fromEntries(
        languages.map((lang) => [
          lang,
          row.display_name_locs?.[lang] || undefined,
        ]),
      );
    const key = nameKey(kind, id);
    return Object.fromEntries(
      languages.map((lang) => [lang, rewrites.data?.[lang]?.[key]]),
    );
  };
  const saveName = async (
    kind: NameKind,
    id: number,
    changes: Record<string, string | null>,
  ) => {
    const row = kind === "state" ? stateRows.data?.get(id) : undefined;
    if (row) {
      const locs = { ...(row.display_name_locs ?? {}) };
      for (const [lang, value] of Object.entries(changes)) {
        if (value === null) delete locs[lang];
        else locs[lang] = value;
      }
      await statesApi.update(projectId, row.id, { display_name_locs: locs });
    } else {
      const key = nameKey(kind, id);
      for (const [lang, value] of Object.entries(changes)) {
        if (value === null) await clearRewrite(projectId, lang, key);
        else await setRewrite(projectId, lang, key, value);
      }
    }
    await Promise.all([rewrites.mutate(), stateRows.mutate()]);
    toast.success(t("renamed"), { description: t("namesUpdateSoon") });

    void data.mutate();
  };

  const records = land.additions.data ?? [];
  const landCount = records.filter((r) => r.kind === "land").length;
  const editCount = records.filter((r) => r.kind === "edit").length;
  const subtitle = records.some((r) => r.kind === "world")
    ? t("subtitleWorld", { date })
    : landCount > 0 && editCount > 0
      ? t("subtitleWithLandAndEdits", {
          date,
          count: landCount,
          edits: editCount,
        })
      : landCount > 0
        ? t("subtitleWithLand", { date, count: landCount })
        : editCount > 0
          ? t("subtitleWithEdits", { date, edits: editCount })
          : t("subtitle", { date });
  const owners = useMemo(
    () =>
      [
        ...new Set(
          [...(map?.states.values() ?? [])].map((s) => s.owner).filter(Boolean),
        ),
      ].sort(),
    [map],
  );

  const projectTags = useProjectStore((s) => s.currentProject?.country_tags);
  const countryTags = useMemo(
    () => [...new Set([...owners, ...(projectTags ?? [])])].sort(),
    [owners, projectTags],
  );

  const mapChanged = async () => {
    setSelected(0);
    await Promise.all([data.mutate(), stateRows.mutate()]);
    setMapVersion((v) => v + 1);
  };

  const province = map && selected ? map.provinces[selected] : null;
  const editing =
    province && draft && draft.province === province.id ? draft : null;
  const pick = (index: number) => {
    const at = map?.provinces[index];
    if (picking && editing && at) {
      if (picking === "state" && at.state)
        setDraft({ ...editing, state: at.state });
      if (picking === "region" && at.region)
        setDraft({ ...editing, region: at.region });
      setPicking(null);
      return;
    }
    setSelected(index);
  };
  const saveEdit = async () => {
    if (!province || !editing) return;
    setEditBusy(true);
    setEditError(null);
    try {
      await saveProvince(
        projectId,
        province.id,
        provinceChanges(province, editing),
      );
      setDraft(null);
      setPicking(null);
      toast.success(t("editSaved", { id: province.id }), {
        description: t("namesUpdateSoon"),
      });
      await Promise.all([
        data.mutate(),
        stateRows.mutate(),
        land.additions.mutate(),
      ]);
    } catch (err) {
      setEditError(errorText(err));
    } finally {
      setEditBusy(false);
    }
  };
  const state = province?.state ? map?.states.get(province.state) : undefined;
  const hoveredProvince = map && hovered ? map.provinces[hovered] : null;
  const hoveredName = hoveredProvince
    ? (hoveredProvince.state && map?.states.get(hoveredProvince.state)?.name) ||
      map?.regions.get(hoveredProvince.region) ||
      t("province", { id: hoveredProvince.id })
    : "";
  const kindText = (kind: string) =>
    kind === "sea"
      ? t("kindSea")
      : kind === "lake"
        ? t("kindLake")
        : t("kindLand");

  return (
    <div className="flex h-full min-h-0 flex-col">
      <header className="flex flex-wrap items-center gap-3 border-b border-border px-4 py-2">
        <div className="min-w-0">
          <h1 className="truncate text-base font-semibold tracking-tight">
            {t("title")}
          </h1>
          <p className="truncate text-xs text-muted-foreground">
            {subtitle}
            {map
              ? ` · ${t("counts", { provinces: map.provinces.length - 1, states: map.states.size, regions: map.regions.size })}`
              : ""}
            {map && data.isValidating ? ` · ${t("refreshing")}` : ""}
          </p>
        </div>
        <span className="flex-1" />
        <Button
          size="sm"
          variant={landMode ? "default" : "outline"}
          aria-pressed={landMode}
          onClick={() => {
            setLandMode((on) => !on);
            setWorldMode(false);
          }}
        >
          {t("landMode")}
        </Button>
        <Button
          size="sm"
          variant={worldMode ? "default" : "outline"}
          aria-pressed={worldMode}
          onClick={() => {
            setWorldMode((on) => !on);
            setLandMode(false);
          }}
        >
          {t("worldMode")}
        </Button>
        <Button size="sm" variant="outline" onClick={() => setSketchOpen(true)}>
          {t("sketchTrial")}
        </Button>
        <SketchTrialDialog
          projectId={projectId}
          open={sketchOpen}
          onOpenChange={setSketchOpen}
        />
        <SegmentedControl
          label={t("layers")}
          size="sm"
          className="w-80"
          value={layer}
          onChange={setLayer}
          options={[
            { value: "political", label: t("layerPolitical") },
            { value: "state", label: t("layerState") },
            { value: "region", label: t("layerRegion") },
            { value: "names", label: t("layerNames") },
          ]}
        />
        <div className="relative w-64">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t("searchPlaceholder")}
            aria-label={t("searchPlaceholder")}
            className="h-8"
          />
          {query.trim() && (
            <ul className="absolute right-0 top-full z-20 mt-1 max-h-80 w-full overflow-y-auto rounded-md border border-border bg-popover p-1 text-sm shadow-lg">
              {hits.length === 0 && (
                <li className="px-2 py-1 text-xs text-muted-foreground">
                  {t("searchNone")}
                </li>
              )}
              {hits.map((hit) => (
                <li key={`${hit.kind}-${hit.id}`}>
                  <button
                    type="button"
                    className="flex w-full items-center justify-between gap-2 rounded px-2 py-1 text-left hover:bg-muted"
                    onClick={() => {
                      setSelected(hit.province);
                      setFocus({
                        province: hit.province,
                        scale: hit.kind === "city" ? 3 : 1.5,
                        at: Date.now(),
                      });
                      setQuery("");
                    }}
                  >
                    <span className="truncate">{hit.name}</span>
                    <span className="shrink-0 text-xs text-muted-foreground">
                      {hit.kind === "state"
                        ? t("state")
                        : hit.kind === "city"
                          ? t("city")
                          : t("region")}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </header>
      <div className="flex min-h-0 flex-1">
        <div className="relative min-w-0 flex-1">
          {map ? (
            <WorldMapCanvas
              map={map}
              imageUrl={`/api/go/projects/${projectId}/gui-render/world-map/provinces.png?v=${mapVersion}`}
              layer={layer}
              selected={selected}
              focus={focus}
              onPick={pick}
              onHover={setHovered}
              loadingText={t("loading")}
              land={land.layer}
            />
          ) : (
            <div className="flex h-full items-center justify-center p-6">
              {data.error ? (
                <div className="max-w-md rounded-md border border-destructive/40 bg-card px-4 py-3 text-sm text-destructive">
                  {t("loadFailed", {
                    reason:
                      data.error instanceof Error
                        ? data.error.message
                        : String(data.error),
                  })}
                </div>
              ) : (
                <div className="animate-pulse-ring rounded-md border border-accent bg-card px-4 py-3 text-sm text-muted-foreground">
                  {t("loading")}
                </div>
              )}
            </div>
          )}
          {hoveredName && (
            <div className="pointer-events-none absolute bottom-3 left-3 rounded-md border border-border bg-card/90 px-2 py-1 text-xs text-card-foreground shadow-sm">
              {t("hovering", { name: hoveredName })}
            </div>
          )}
        </div>
        <aside className="w-80 shrink-0 overflow-y-auto border-l border-border bg-card p-4 text-sm">
          {worldMode ? (
            <NewWorldPanel
              projectId={projectId}
              land={land}
              languages={languages}
              suggestions={countryTags}
              onSaved={async (states) => {
                await mapChanged();
                toast.success(t("worldSaved"), {
                  description: t("worldSavedBody", { states }),
                });
              }}
              onRestored={async (latest) => {
                await mapChanged();
                toast.success(latest ? t("landUndone") : t("landRestored"));
              }}
            />
          ) : landMode ? (
            <NewLandPanel
              projectId={projectId}
              land={land}
              owners={owners}
              languages={languages}
              onSaved={async (name) => {
                await mapChanged();
                toast.success(t("landSaved", { name }), {
                  description: t("landSavedBody"),
                });
              }}
              onRestored={async (latest) => {
                await mapChanged();
                toast.success(latest ? t("landUndone") : t("landRestored"));
              }}
            />
          ) : !province ? (
            <p className="text-xs text-muted-foreground">{t("hint")}</p>
          ) : (
            <div className="space-y-4">
              <section>
                <div className="font-medium">
                  {t("province", { id: province.id })}
                </div>
                <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
                  <dt className="text-muted-foreground">
                    {kindText(province.kind)}
                  </dt>
                  <dd>{province.coastal ? t("coastal") : ""}</dd>
                  <dt className="text-muted-foreground">{t("terrain")}</dt>
                  <dd className="font-mono">{province.terrain}</dd>
                  {province.continent > 0 && map && (
                    <>
                      <dt className="text-muted-foreground">
                        {t("continent")}
                      </dt>
                      <dd>
                        {map.continents[province.continent - 1] ??
                          province.continent}
                      </dd>
                    </>
                  )}
                </dl>
                {editing && map ? (
                  <div className="mt-3">
                    <ProvinceEditor
                      map={map}
                      province={province}
                      draft={editing}
                      onDraft={setDraft}
                      picking={picking}
                      onPick={setPicking}
                      busy={editBusy}
                      error={editError}
                      onSave={() => void saveEdit()}
                      onCancel={() => {
                        setDraft(null);
                        setPicking(null);
                        setEditError(null);
                      }}
                    />
                  </div>
                ) : (
                  <Button
                    className="mt-2"
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      setDraft(draftOf(province));
                      setPicking(null);
                      setEditError(null);
                    }}
                  >
                    {t("editOpen")}
                  </Button>
                )}
              </section>
              {map && map.points.has(province.id) && (
                <section>
                  <MapNameEditor
                    key={`city-${province.id}`}
                    label={`${t("nameCity")} · ${t("points", { value: map.points.get(province.id)!.value })}`}
                    shown={map.points.get(province.id)!.name}
                    languages={languages}
                    saved={savedNames("city", province.id)}
                    where={t("whereRewrite")}
                    onSave={(changes) => saveName("city", province.id, changes)}
                  />
                </section>
              )}
              <section>
                {state && province.state ? (
                  <>
                    <MapNameEditor
                      key={`state-${province.state}`}
                      label={`${t("nameState")} · #${province.state}`}
                      shown={state.name}
                      languages={languages}
                      saved={savedNames("state", province.state)}
                      where={
                        stateRows.data?.has(province.state)
                          ? t("whereRow")
                          : t("whereRewrite")
                      }
                      onSave={(changes) =>
                        saveName("state", province.state, changes)
                      }
                    />
                    <div className="mt-1 text-xs">
                      <span className="text-muted-foreground">
                        {t("owner")}{" "}
                      </span>
                      <span className="font-mono">
                        {state.owner || t("noOwner")}
                      </span>
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-2">
                      <Link
                        href={`/project/${projectId}/states?state=${province.state}`}
                        className="text-xs text-accent hover:underline"
                      >
                        {t("openInStates")} ↗
                      </Link>
                      <OpenInGameLink
                        href={gameViewHref(projectId, {
                          panel: "countrystateview",
                          state: province.state,
                          tag: state.owner || undefined,
                        })}
                      />
                    </div>
                  </>
                ) : (
                  <>
                    <div className="text-xs text-muted-foreground">
                      {t("state")}
                    </div>
                    <div className="mt-0.5 text-xs text-muted-foreground">
                      {t("noState")}
                    </div>
                  </>
                )}
              </section>
              {province.region > 0 && map && (
                <section>
                  <MapNameEditor
                    key={`region-${province.region}`}
                    label={`${t("nameRegion")} · #${province.region}`}
                    shown={map.regions.get(province.region) ?? ""}
                    languages={languages}
                    saved={savedNames("region", province.region)}
                    where={t("whereRewrite")}
                    onSave={(changes) =>
                      saveName("region", province.region, changes)
                    }
                  />
                </section>
              )}
            </div>
          )}
        </aside>
      </div>
    </div>
  );
}
