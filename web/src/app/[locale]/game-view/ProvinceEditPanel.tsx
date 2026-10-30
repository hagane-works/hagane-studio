"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type MutableRefObject,
} from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Link } from "@/i18n/navigation";
import { projectsApi } from "@/lib/api/project";
import { workbenchTargetOf } from "@/lib/gameViewLinks";
import { statesApi } from "@/lib/api/scenario";
import { MapNameEditor } from "@/components/workbench/world-map/MapNameEditor";
import {
  clearRewrite,
  loadRewrites,
  nameKey,
  nameLanguages,
  setRewrite,
  type NameKind,
} from "@/components/workbench/world-map/mapNames";
import {
  draftOf,
  ProvinceEditor,
  provinceChanges,
  saveProvince,
  type PickTarget,
  type ProvinceDraft,
} from "@/components/workbench/world-map/ProvinceEditor";
import { useMapErrorText } from "./map-studio/mapErrorText";
import type {
  PickScope,
  Province,
  WorldMap,
} from "@/components/workbench/world-map/worldMapModel";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { BUTTON, GamePanel, type PanelSource } from "./GamePanel";
import { recordChange } from "./changeLog";
import { useStateRows } from "./StateEditPanel";
import { StateWorkbenchForm } from "./StateWorkbenchForm";

function useMapNames(projectId: number) {
  const project = useSWR(
    ["game-view-project", projectId],
    () => projectsApi.get(projectId),
    { revalidateOnFocus: false },
  );
  const exportLanguages = project.data?.export_languages;
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
  const saved = (
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
  const save = async (
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
  };
  return {
    languages,
    saved,
    save,
    hasStateRow: (id: number) => !!stateRows.data?.has(id),
  };
}

export function ProvinceEditPanel({
  projectId,
  map,
  index,
  scope,
  onScope,
  pickRef,
  onSaved,
  onClose,
}: {
  projectId: number;
  map: WorldMap;

  index: number;

  scope: PickScope;
  onScope: (scope: PickScope) => void;

  pickRef: MutableRefObject<((index: number) => boolean) | null>;

  onSaved: (message: string) => void;
  onClose: () => void;
}) {
  const t = useTranslations("worldMap");
  const tw = useTranslations("gameViewWorld.province");
  const errorText = useMapErrorText();
  const province: Province | undefined = map.provinces[index];
  const names = useMapNames(projectId);
  const rows = useStateRows(projectId);

  const countryTags = useMemo(
    () =>
      [
        ...new Set(
          [...map.states.values()].map((s) => s.owner).filter(Boolean),
        ),
      ].sort(),
    [map],
  );
  const savedState = useCallback(
    (id: number) => {
      void rows.mutate();
      onSaved(tw("stateSaved", { id }));
    },
    [rows, onSaved, tw],
  );
  const [draft, setDraft] = useState<ProvinceDraft | null>(null);
  const [picking, setPicking] = useState<PickTarget | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const editing =
    province && draft && draft.province === province.id ? draft : null;

  useEffect(() => {
    if (!picking || !editing) {
      pickRef.current = null;
      return;
    }
    pickRef.current = (at) => {
      const target = map.provinces[at];
      if (!target) return false;
      if (picking === "state" && target.state)
        setDraft({ ...editing, state: target.state });
      if (picking === "region" && target.region)
        setDraft({ ...editing, region: target.region });
      setPicking(null);
      return true;
    };
    return () => {
      pickRef.current = null;
    };
  }, [picking, editing, map, pickRef]);

  if (!province) return null;
  const state = province.state ? map.states.get(province.state) : undefined;
  const point = map.points.get(province.id);
  const kindText =
    province.kind === "sea"
      ? t("kindSea")
      : province.kind === "lake"
        ? t("kindLake")
        : t("kindLand");
  const workbenchHref = `/project/${projectId}/world-map`;
  const own = province.state
    ? rows.data?.some((row) => row.state_id === province.state)
    : false;

  const saveEdit = async () => {
    if (!editing) return;
    setBusy(true);
    setError(null);
    try {
      await saveProvince(
        projectId,
        province.id,
        provinceChanges(province, editing),
      );
      setDraft(null);
      setPicking(null);
      recordChange({
        projectId,
        title: t("province", { id: province.id }),
        kind: tw("editKind"),
        workbenchHref,
      });
      onSaved(t("editSaved", { id: province.id }));
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };
  const rename = async (
    kind: NameKind,
    id: number,
    changes: Record<string, string | null>,
  ) => {
    await names.save(kind, id, changes);
    recordChange({
      projectId,
      title: tw("renameLog", { key: nameKey(kind, id) }),
      kind: tw("renameKind"),
      workbenchHref,
    });
    onSaved(t("renamed"));
  };

  const regionName = map.regions.get(province.region) ?? "";
  const stateProvinces = province.state
    ? count(map, (p) => p.state === province.state)
    : 0;
  const regionProvinces = province.region
    ? count(map, (p) => p.region === province.region)
    : 0;
  const header =
    scope === "state"
      ? {
          title:
            state && province.state
              ? tw("stateTitle", { id: province.state, name: state.name })
              : tw("noStateTitle"),
          kind:
            state && province.state
              ? tw("stateKind", {
                  id: province.state,
                  owner: state.owner || tw("noOwner"),
                  count: stateProvinces,
                })
              : tw("provinceKind", { id: province.id, kind: kindText }),
          source:
            province.state && !rows.isLoading
              ? ((own ? "项目" : "原版") as PanelSource)
              : undefined,
          href: province.state
            ? `/project/${projectId}/states?state=${province.state}`
            : workbenchHref,
        }
      : scope === "region"
        ? {
            title: province.region
              ? tw("regionTitle", { id: province.region, name: regionName })
              : tw("noRegionTitle"),
            kind: province.region
              ? tw("regionKind", {
                  id: province.region,
                  count: regionProvinces,
                })
              : tw("provinceId", { id: province.id }),
            source: undefined,
            href: workbenchHref,
          }
        : {
            title: t("province", { id: province.id }),
            kind: state
              ? tw("inState", {
                  kind: kindText,
                  id: province.state,
                  name: state.name,
                })
              : kindText,
            source: undefined,
            href: workbenchHref,
          };

  return (
    <GamePanel
      title={header.title}
      kind={header.kind}
      source={header.source}
      workbenchHref={header.href}

      wide={scope === "state" && province.state > 0 ? "workbench" : true}
      onClose={onClose}
    >
      <SegmentedControl
        label={tw("scope")}
        size="sm"
        className="mt-3"
        value={scope}
        onChange={onScope}
        options={[
          { value: "state", label: tw("scopeState") },
          { value: "province", label: tw("scopeProvince") },
          { value: "region", label: tw("scopeRegion") },
        ]}
      />
      <div className="mt-3 text-sm">
        {scope === "state" &&
          (state && province.state > 0 ? (
            <StateWorkbenchForm
              key={province.state}
              projectId={projectId}
              stateId={province.state}
              shownName={state.name}
              countryTags={countryTags}
              onSaved={savedState}
            />
          ) : (
            <p className="text-xs text-muted-foreground">
              {province.kind === "land"
                ? tw("landNoState")
                : tw("waterNoState")}
            </p>
          ))}

        {scope === "region" &&
          (province.region > 0 ? (
            <MapNameEditor
              key={`region-${province.region}`}
              label={`${t("nameRegion")} · #${province.region}`}
              shown={regionName}
              languages={names.languages}
              saved={names.saved("region", province.region)}
              where={t("whereRewrite")}
              onSave={(changes) => rename("region", province.region, changes)}
            />
          ) : (
            <p className="text-xs text-muted-foreground">{tw("noRegion")}</p>
          ))}

        {scope === "province" && (
          <section className="space-y-4">
            <div className="flex justify-end">
              <Link
                href={workbenchHref}
                target={workbenchTargetOf(workbenchHref)}
                className="text-micro text-accent hover:underline"
              >
                {tw("openWorldMap")}
              </Link>
            </div>
            <div>
              <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
                <dt className="text-muted-foreground">{kindText}</dt>
                <dd>{province.coastal ? t("coastal") : ""}</dd>
                <dt className="text-muted-foreground">{t("terrain")}</dt>
                <dd className="font-mono">{province.terrain}</dd>
                {province.continent > 0 && (
                  <>
                    <dt className="text-muted-foreground">{t("continent")}</dt>
                    <dd>
                      {map.continents[province.continent - 1] ??
                        province.continent}
                    </dd>
                  </>
                )}
              </dl>
              {editing ? (
                <div className="mt-3">
                  <ProvinceEditor
                    map={map}
                    province={province}
                    draft={editing}
                    onDraft={setDraft}
                    picking={picking}
                    onPick={setPicking}
                    busy={busy}
                    error={error}
                    onSave={() => void saveEdit()}
                    onCancel={() => {
                      setDraft(null);
                      setPicking(null);
                      setError(null);
                    }}
                  />
                </div>
              ) : (
                <button
                  type="button"
                  className={`${BUTTON} mt-2`}
                  onClick={() => {
                    setDraft(draftOf(province));
                    setPicking(null);
                    setError(null);
                  }}
                >
                  {t("editOpen")}
                </button>
              )}
            </div>
            {point && (
              <MapNameEditor
                key={`city-${province.id}`}
                label={`${t("nameCity")} · ${t("points", { value: point.value })}`}
                shown={point.name}
                languages={names.languages}
                saved={names.saved("city", province.id)}
                where={t("whereRewrite")}
                onSave={(changes) => rename("city", province.id, changes)}
              />
            )}
          </section>
        )}
      </div>
    </GamePanel>
  );
}

function count(map: WorldMap, test: (province: Province) => boolean): number {
  let n = 0;
  for (let i = 1; i < map.provinces.length; i++)
    if (test(map.provinces[i])) n++;
  return n;
}
