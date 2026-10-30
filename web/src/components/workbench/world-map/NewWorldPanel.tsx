"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { RadioCardGroup } from "@/components/ui/RadioCardGroup";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { FilePick } from "@/shared/upload/FilePick";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { gameViewHref, gameViewTargetOf } from "@/lib/gameViewLinks";
import type { NewLand } from "./NewLandPanel";
import {
  addTag,
  cleanTag,
  unloadedLabel,
  UPLOAD_FIELDS,
  uploadReady,
  WORLD_MAX_COUNTRIES,
  worldForm,
  worldKey,
  type UploadField,
  type UploadFiles,
  type WorldPreview,
  type WorldSize,
  type WorldSource,
} from "./newWorldModel";

const UPLOAD_INPUTS: Record<UploadField, { label: string; accept: string }> = {
  provinces: {
    label: "worldUploadProvinces",
    accept: "image/png,image/bmp,.bmp",
  },
  heightmap: {
    label: "worldUploadHeightmap",
    accept: "image/png,image/bmp,.bmp",
  },
  terrain: { label: "worldUploadTerrain", accept: "image/png,image/bmp,.bmp" },
  rivers: { label: "worldUploadRivers", accept: "image/png,image/bmp,.bmp" },
  definition: { label: "worldUploadDefinition", accept: ".csv,text/csv" },
};
import {
  downloadCanvas,
  LegendSwatches,
  sampleWorld,
  useSketchLegend,
} from "./sketchLegend";
import { mapResponseProblem } from "@/lib/mapToolRequest";
import { MapProblemError } from "@/app/[locale]/game-view/map-studio/mapProblems";
import { useMapErrorText } from "@/app/[locale]/game-view/map-studio/mapErrorText";

export function NewWorldPanel({
  projectId,
  land,
  languages,
  suggestions,
  onSaved,
  onRestored,
}: {
  projectId: number;

  land: NewLand;

  languages: string[];

  suggestions: string[];
  onSaved: (states: number) => Promise<void> | void;
  onRestored: (latest: boolean) => Promise<void> | void;
}) {
  const t = useTranslations("worldMap");
  const errorText = useMapErrorText();
  const confirm = useConfirm();
  const legend = useSketchLegend(projectId, true);
  const sea = legend.data?.find((entry) => entry.key === "ocean")?.color;
  const [source, setSource] = useState<WorldSource>("sketch");
  const [sketch, setSketch] = useState<File | null>(null);
  const [upload, setUpload] = useState<UploadFiles>({});
  const [size, setSize] = useState<WorldSize>("medium");
  const [seed, setSeed] = useState("1");
  const [tags, setTags] = useState<string[]>([]);
  const [tagInput, setTagInput] = useState("");
  const [preview, setPreview] = useState<{
    key: string;
    data: WorldPreview;
  } | null>(null);
  const [view, setView] = useState<"political" | "terrain">("political");
  const [busy, setBusy] = useState<
    "preview" | "save" | "restore" | "undo" | null
  >(null);
  const [error, setError] = useState<string | null>(null);
  const [confirmed, setConfirmed] = useState(false);

  const request = { source, sketch, upload, size, seed, tags, languages };
  const mapChosen =
    source === "sketch"
      ? sketch !== null
      : source === "upload"
        ? uploadReady(upload)
        : true;
  const fresh =
    preview && preview.key === worldKey(request) ? preview.data : null;
  const records = land.additions.data ?? [];
  const world = records.find((record) => record.kind === "world");
  const blocked = (fresh?.findings ?? []).some(
    (finding) => finding.severity === "error",
  );
  const waitingFor =
    source === "sketch" && !sketch
      ? t("worldNeedSketch")
      : source === "upload" && !mapChosen
        ? t("worldNeedUpload")
        : tags.length === 0
          ? t("worldNeedCountries")
          : busy === "preview"
            ? t("worldPreviewing")
            : !fresh
              ? t("worldNeedPreview")
              : blocked
                ? t("worldBlocked")
                : !confirmed
                  ? t("worldNeedConfirm")
                  : null;

  const run = async (save: boolean) => {
    setBusy(save ? "save" : "preview");
    setError(null);
    try {
      const response = await fetch(
        `/api/go/projects/${projectId}/gui-render/world-map/${save ? "world" : "world-preview"}`,
        { method: "POST", body: worldForm(request) },
      );

      if (!response.ok)
        throw new MapProblemError(await mapResponseProblem(response));
      if (save) {
        const out = (await response.json()) as {
          addition: { state_ids: number[] };
        };
        await land.additions.mutate();
        setPreview(null);
        setConfirmed(false);
        await onSaved(out.addition.state_ids.length);
      } else {
        setPreview({
          key: worldKey(request),
          data: (await response.json()) as WorldPreview,
        });
      }
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(null);
    }
  };

  const restore = async (latest: boolean) => {
    if (
      !(await confirm({
        title: latest ? t("landUndoTitle") : t("landRestoreTitle"),
        body: t("worldUndoWorld"),
        danger: true,
        confirmLabel: latest ? t("landUndo") : t("landRestore"),
      }))
    )
      return;
    setBusy(latest ? "undo" : "restore");
    try {
      if (await land.restore(latest)) await onRestored(latest);
    } finally {
      setBusy(null);
    }
  };

  const step = (n: number, title: string) => (
    <div className="flex items-center gap-2 text-xs">
      <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-border tabular-nums">
        {n}
      </span>
      <span>{title}</span>
    </div>
  );

  const header = (
    <section>
      <h2 className="font-medium">{t("worldTitle")}</h2>
      <p className="mt-1 text-xs text-muted-foreground">{t("worldIntro")}</p>

      <a
        href={gameViewHref(projectId, { mapStudio: true })}
        target={gameViewTargetOf(gameViewHref(projectId, { mapStudio: true }))}
        className="ui-focus-ring mt-2 inline-block rounded text-xs text-accent hover:underline"
      >
        {t("worldStudioLink")}
      </a>
    </section>
  );

  if (world) {
    return (
      <div className="space-y-5">
        {header}
        <section className="space-y-3">
          <p className="text-xs">
            {t("worldIsWorld", {
              states: world.state_ids.length,
              tags: world.owner.split(",").join("、"),
            })}
          </p>
          <div className="flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={busy !== null || land.busy !== null}
              onClick={() => void restore(true)}
            >
              {busy === "undo" ? t("landRestoring") : t("landUndo")}
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={busy !== null || land.busy !== null}
              onClick={() => void restore(false)}
            >
              {busy === "restore" ? t("landRestoring") : t("landRestore")}
            </Button>
          </div>
          {land.error && (
            <p className="text-xs text-destructive">{land.error}</p>
          )}
        </section>
      </div>
    );
  }

  if (records.length > 0) {
    return (
      <div className="space-y-5">
        {header}
        <p className="text-xs text-warning">{t("worldNotBase")}</p>
      </div>
    );
  }

  const summary = fresh?.summary;
  const picture = fresh
    ? view === "terrain" && fresh.previews.terrain
      ? fresh.previews.terrain
      : fresh.previews.political
    : null;

  return (
    <div className="space-y-5">
      {header}

      <section className="space-y-2">
        {step(1, t("worldStepSource"))}
        <RadioCardGroup
          label={t("worldStepSource")}
          value={source}
          onChange={setSource}
          options={[
            {
              value: "sketch",
              title: t("worldSourceSketch"),
              consequence: t("worldSourceSketchNote"),
            },
            {
              value: "game",
              title: t("worldSourceGame"),
              consequence: t("worldSourceGameNote"),
            },
            {
              value: "upload",
              title: t("worldSourceUpload"),
              consequence: t("worldSourceUploadNote"),
            },
          ]}
        />
        {source === "game" ? null : source === "upload" ? (
          <>
            <p className="text-xs text-muted-foreground">
              {t("worldUploadHint")}
            </p>
            {UPLOAD_FIELDS.map((field) => (
              <label key={field} className="block text-xs">
                <span className="text-muted-foreground">
                  {t(UPLOAD_INPUTS[field].label)}
                </span>
                <FilePick
                  className="mt-1"
                  accept={UPLOAD_INPUTS[field].accept}
                  fileName={upload[field]?.name ?? null}
                  onPick={(file) =>
                    setUpload((files) => ({ ...files, [field]: file }))
                  }
                />
              </label>
            ))}
          </>
        ) : (
          <>
            <LegendSwatches legend={legend.data ?? []} />
            <div className="flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="outline"
                disabled={!legend.data}
                onClick={() =>
                  void sampleWorld(legend.data ?? []).then(
                    (file) => file && setSketch(file),
                  )
                }
              >
                {t("worldSample")}
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={!sea}
                onClick={() =>
                  sea && downloadCanvas(sea, 1408, 512, "world-sketch.png")
                }
              >
                {t("downloadCanvas")}
              </Button>
            </div>
            <label className="block text-xs">
              <span className="text-muted-foreground">{t("uploadSketch")}</span>
              <FilePick
                className="mt-1"
                accept="image/png,image/jpeg"
                onPick={setSketch}
              />
            </label>
            {sketch && (
              <p className="text-xs text-muted-foreground">
                {t("landSketchChosen", { name: sketch.name })}
              </p>
            )}
            <div className="text-xs text-muted-foreground">
              {t("worldSize")}
            </div>
            <SegmentedControl
              label={t("worldSize")}
              size="sm"
              value={size}
              onChange={setSize}
              options={[
                { value: "fine", label: t("worldSizeFine") },
                { value: "medium", label: t("worldSizeMedium") },
                { value: "coarse", label: t("worldSizeCoarse") },
              ]}
            />
          </>
        )}
      </section>

      <section className="space-y-2">
        {step(2, t("worldStepCountries"))}
        <p className="text-xs text-muted-foreground">
          {t("worldCountriesHint")}
        </p>
        {tags.length > 0 && (
          <ul className="flex flex-wrap gap-1.5">
            {tags.map((tag) => (
              <li
                key={tag}
                className="flex h-7 items-center gap-1 rounded-md border border-border px-2 font-mono text-xs"
              >
                {tag}
                <button
                  type="button"
                  className="ui-interactive ui-focus-ring rounded-sm text-muted-foreground hover:text-foreground"
                  aria-label={t("worldCountryRemove", { tag })}
                  onClick={() => setTags(tags.filter((other) => other !== tag))}
                >
                  <X className="h-3.5 w-3.5" aria-hidden />
                </button>
              </li>
            ))}
          </ul>
        )}
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            setTags(addTag(tags, tagInput));
            setTagInput("");
          }}
        >
          <Input
            className="h-8 font-mono uppercase"
            value={tagInput}
            maxLength={3}
            list="world-map-countries"
            placeholder={t("worldCountryPlaceholder")}
            aria-label={t("worldCountryPlaceholder")}
            disabled={tags.length >= WORLD_MAX_COUNTRIES}
            onChange={(e) => setTagInput(cleanTag(e.target.value))}
          />
          <datalist id="world-map-countries">
            {suggestions.map((tag) => (
              <option key={tag} value={tag} />
            ))}
          </datalist>
          <Button
            size="sm"
            variant="outline"
            type="submit"
            disabled={
              tagInput.length !== 3 || tags.length >= WORLD_MAX_COUNTRIES
            }
          >
            {t("worldCountryAdd")}
          </Button>
        </form>
      </section>

      <section className="space-y-2">
        {step(3, t("worldStepPreview"))}
        <div className="flex items-end gap-2">
          <label className="block flex-1 text-xs">
            <span className="text-muted-foreground">{t("seed")}</span>
            <Input
              className="mt-1"
              inputMode="numeric"
              value={seed}
              onChange={(e) => setSeed(e.target.value.replace(/[^\d-]/g, ""))}
            />
          </label>
          <Button
            size="sm"
            variant="outline"
            disabled={!mapChosen || tags.length === 0 || busy !== null}
            onClick={() => void run(false)}
          >
            {fresh ? t("worldRepreview") : t("worldPreview")}
          </Button>
        </div>
        {busy === "preview" ? (
          <div className="animate-pulse-ring rounded-md border border-accent px-3 py-2 text-xs text-muted-foreground">
            {t("worldPreviewing")}
          </div>
        ) : fresh && summary && picture ? (
          <>
            {fresh.previews.terrain && (
              <SegmentedControl
                label={t("layers")}
                size="sm"
                value={view}
                onChange={setView}
                options={[
                  { value: "political", label: t("worldViewPolitical") },
                  { value: "terrain", label: t("viewTerrain") },
                ]}
              />
            )}
            <img
              src={picture}
              alt=""
              className="w-full rounded-md border border-border"
            />
            <p className="text-xs">
              {t("worldSummary", {
                land: summary.provinces.land,
                sea: summary.provinces.sea,
                states: summary.states,
                regions: summary.regions,
              })}
            </p>
            {(summary.made ?? []).map((made) => (
              <p key={made} className="text-xs text-muted-foreground">
                {made === "definitions"
                  ? t("worldMadeDefinitions")
                  : t("worldMadeRivers")}
              </p>
            ))}
            <ul className="space-y-0.5 text-xs text-muted-foreground">
              {summary.countries.map((country) => (
                <li key={country.tag}>
                  {t("worldCountryStates", {
                    tag: country.tag,
                    states: country.states,
                  })}
                </li>
              ))}
            </ul>
            {(fresh.findings ?? []).length > 0 && (
              <ul className="space-y-1 text-xs">
                {fresh.findings.map((finding) => (
                  <li
                    key={finding.code}
                    className={
                      finding.severity === "error"
                        ? "text-destructive"
                        : "text-warning"
                    }
                  >
                    {finding.message} ×{finding.count}
                  </li>
                ))}
              </ul>
            )}
          </>
        ) : (
          <p className="text-xs text-muted-foreground">
            {t("worldPreviewHint")}
          </p>
        )}
        {error && <p className="text-xs text-destructive">{error}</p>}
      </section>

      {summary && (
        <section className="space-y-2">
          {step(4, t("worldStepUnload"))}
          <p className="text-xs text-muted-foreground">
            {t("worldUnloadIntro")}
          </p>
          <ul className="grid grid-cols-2 gap-x-3 gap-y-0.5 text-xs">
            {summary.unloaded.map((folder) => (
              <li key={folder} title={folder}>
                {t(unloadedLabel(folder))}
              </li>
            ))}
          </ul>
          {summary.replaced_states > 0 && (
            <p className="text-xs text-warning">
              {t("worldReplacedStates", { count: summary.replaced_states })}
            </p>
          )}
          {summary.countries
            .filter((country) => country.own_start)
            .map((country) => (
              <p key={country.tag} className="text-xs text-muted-foreground">
                {t("worldOwnStart", { tag: country.tag })}
              </p>
            ))}
          <label className="flex items-start gap-2 text-xs">
            <input
              type="checkbox"
              className="ui-focus-ring mt-0.5"
              checked={confirmed}
              onChange={(e) => setConfirmed(e.target.checked)}
            />
            <span>{t("worldConfirm")}</span>
          </label>
        </section>
      )}

      <div className="sticky bottom-0 -mx-4 -mb-4 space-y-1 border-t border-border bg-card px-4 py-3">
        <Button
          size="sm"
          className="w-full"
          disabled={waitingFor !== null || busy !== null}
          onClick={() => void run(true)}
        >
          {busy === "save" ? t("worldSaving") : t("worldSave")}
        </Button>
        <p className="text-xs text-muted-foreground">
          {busy === "save"
            ? t("worldSaving")
            : (waitingFor ?? t("worldSaveNote"))}
        </p>
      </div>
    </div>
  );
}
