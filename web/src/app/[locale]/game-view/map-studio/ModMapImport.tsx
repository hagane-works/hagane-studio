"use client";

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { createPortal } from "react-dom";
import { useLocale, useTranslations } from "use-intl";
import useSWR from "swr";
import { Link } from "@/i18n/navigation";
import { workbenchTargetOf } from "@/lib/gameViewLinks";
import { API_BASE } from "@/lib/api";
import { projectsApi } from "@/lib/api/project";
import { withGameLanguage } from "@/lib/guiRenderApi";
import { hasDirectoryPicker } from "@/lib/fsAccess";
import { yieldToMainThread } from "@/lib/fs-scanner";
import {
  mapToolFetch,
  mapToolUpload,
  mapResponseProblem,
} from "@/lib/mapToolRequest";
import { nameLanguages } from "@/components/workbench/world-map/mapNames";
import {
  unloadedLabel,
  type WorldPreview,
} from "@/components/workbench/world-map/newWorldModel";
import { BUTTON, GamePanel, PRIMARY } from "../GamePanel";
import {
  mapFindingFix,
  mapFindingMessage,
  mapProblemMessage,
  ONE_CLICK_FIXES,
} from "./mapProblems";
import { TAG, useStudioCountries } from "./studioCountries";
import {
  bundleModFiles,
  namingLocalisation,
  scanModFolder,
  tagMapField,
  type ModFile,
  type ModMapFolder,
} from "./modMapFolder";
import { readModMapDirectory, readModMapFileList } from "./modMapScan";

const subscribeNothing = () => () => {};

const fixField = (fixes: ReadonlySet<string>) => [...fixes].sort().join(",");

interface UnknownTag {
  tag: string;
  states: number;
}

const MAKE = "__make__";

type Scan =
  | { phase: "browser"; startedAt: number }
  | {
      phase: "files";
      startedAt: number;
      files: number;
      directories: number;
      path: string;
      total?: number;
    }
  | {
      phase: "names";
      startedAt: number;
      done: number;
      total: number;
      files: number;
    };

interface ScanRun {
  id: number;
  startedAt: number;
  controller: AbortController;
}

function ScanElapsed({ startedAt }: { startedAt: number }) {
  const t = useTranslations("mapStudio");
  const [now, setNow] = useState(startedAt);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);
  return (
    <p className="tabular-nums text-muted-foreground" aria-live="off">
      {t("modScanElapsed", {
        seconds: Math.max(0, Math.floor((now - startedAt) / 1000)),
      })}
    </p>
  );
}

export function ModMapImport({
  projectId,
  date,
  width,
  height,
  onClose,
  onBack,
  onSaved,
}: {
  projectId: number;

  date: string;
  width: number;
  height: number;
  onClose: () => void;

  onBack: () => void;

  onSaved: (states: number) => Promise<void> | void;
}) {
  const t = useTranslations("mapStudio");
  const tw = useTranslations("worldMap");
  const locale = useLocale();
  const mounted = useSyncExternalStore(
    subscribeNothing,
    () => true,
    () => false,
  );
  const input = useRef<HTMLInputElement>(null);

  const project = useSWR(
    ["game-view-project", projectId],
    () => projectsApi.get(projectId),
    { revalidateOnFocus: false },
  );
  const additions = useSWR(["world-map-land", projectId], async () => {
    const response = await mapToolFetch(
      withGameLanguage(
        `/api/go/projects/${projectId}/gui-render/world-map/land`,
      ),
    );
    if (!response.ok)
      throw new Error(
        mapProblemMessage(await mapResponseProblem(response), t, locale),
      );
    return ((await response.json()) as { additions: unknown[] }).additions;
  });
  const countryInfo = useStudioCountries(projectId, undefined);
  const { rows, nameOf } = countryInfo;
  const languages = useMemo(
    () => nameLanguages(project.data?.export_languages),
    [project.data],
  );

  const [folders, setFolders] = useState<ModMapFolder[] | null>(null);
  const [picked, setPicked] = useState(0);
  const [chosen, setChosen] = useState(0);
  const [unknown, setUnknown] = useState<UnknownTag[]>([]);
  const [choices, setChoices] = useState<ReadonlyMap<string, string>>(
    new Map(),
  );
  const [preview, setPreview] = useState<{
    key: string;
    data: WorldPreview;
  } | null>(null);
  const [busy, setBusy] = useState<"preview" | "save" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [confirmed, setConfirmed] = useState(false);

  const [scan, setScan] = useState<Scan | null>(null);
  const [scanCancelled, setScanCancelled] = useState(false);
  const [scanned, setScanned] = useState(0);
  const [upload, setUpload] = useState<number | null>(null);
  const scanRun = useRef(0);
  const currentScan = useRef<ScanRun | null>(null);
  const pickerCleanup = useRef<(() => void) | null>(null);

  useEffect(
    () => () => {
      scanRun.current += 1;
      currentScan.current?.controller.abort();
      pickerCleanup.current?.();
    },
    [],
  );

  const [fixes, setFixes] = useState<ReadonlySet<string>>(new Set());
  const [errorFix, setErrorFix] = useState<string | null>(null);

  const folder = folders?.[chosen] ?? null;
  const merges = useMemo(
    () => new Map([...choices].filter(([, to]) => to && to !== MAKE)),
    [choices],
  );
  const keyOf = (use: ReadonlySet<string>) =>
    `${picked}|${chosen}|${tagMapField(merges)}|${fixField(use)}`;
  const key = keyOf(fixes);
  const fresh = preview && preview.key === key ? preview : null;
  const findings = fresh?.data.findings ?? [];
  const blocked = findings.some((finding) => finding.severity === "error");
  const notBase = (additions.data ?? []).length > 0;
  const undecided = unknown.filter((u) => !choices.get(u.tag));
  const projectTags = (rows.data?.countries ?? [])
    .map((row) => row.tag)
    .filter((tag) => TAG.test(tag));

  const activeScan = (run: ScanRun) =>
    currentScan.current === run &&
    run.id === scanRun.current &&
    !run.controller.signal.aborted;
  const beginScan = (): ScanRun => {
    currentScan.current?.controller.abort();
    pickerCleanup.current?.();
    pickerCleanup.current = null;
    const run = {
      id: ++scanRun.current,
      startedAt: Date.now(),
      controller: new AbortController(),
    };
    currentScan.current = run;
    setScanCancelled(false);
    setError(null);
    setScan({ phase: "browser", startedAt: run.startedAt });
    return run;
  };
  const cancelScan = () => {
    scanRun.current += 1;
    currentScan.current?.controller.abort();
    currentScan.current = null;
    pickerCleanup.current?.();
    pickerCleanup.current = null;
    if (input.current) input.current.value = "";
    setScan(null);
    setScanCancelled(true);
  };
  const resetSelection = () => {
    setFolders(null);
    setScanned(0);
    setPicked((n) => n + 1);
    setChosen(0);
    setUnknown([]);
    setChoices(new Map());
    setPreview(null);
    setConfirmed(false);
    setError(null);
    setFixes(new Set());
    setErrorFix(null);
  };

  const nameFiles = async (files: ModFile[], count: number, run: ScanRun) => {
    await yieldToMainThread();
    if (!activeScan(run)) return;
    const found = scanModFolder(files);
    const total = found.reduce((n, mod) => n + mod.localisation.length, 0);
    setScanned(count);
    if (total === 0) {
      setFolders(found);
      return;
    }
    let done = 0;
    let shownAt = 0;
    setScan({
      phase: "names",
      startedAt: run.startedAt,
      done,
      total,
      files: count,
    });
    await yieldToMainThread();
    const named: ModMapFolder[] = [];
    for (const mod of found) {
      if (!activeScan(run)) return;
      const localisation = await namingLocalisation(
        mod.localisation,
        () => {
          done += 1;
          const now = performance.now();
          if (activeScan(run) && (now - shownAt >= 100 || done === total)) {
            shownAt = now;
            setScan({
              phase: "names",
              startedAt: run.startedAt,
              done,
              total,
              files: count,
            });
          }
        },
        run.controller.signal,
      );
      named.push({ ...mod, localisation });
    }
    if (activeScan(run)) setFolders(named);
  };

  const readFolder = async (
    source: FileSystemDirectoryHandle | FileList,
    run: ScanRun,
  ) => {
    resetSelection();
    setScan({
      phase: "files",
      startedAt: run.startedAt,
      files: 0,
      directories: 0,
      path: "",
    });
    try {
      const progress = (value: {
        files: number;
        directories: number;
        path: string;
        total?: number;
      }) => {
        if (activeScan(run))
          setScan({ phase: "files", startedAt: run.startedAt, ...value });
      };
      const result =
        "kind" in source
          ? await readModMapDirectory(source, progress, run.controller.signal)
          : await readModMapFileList(source, progress, run.controller.signal);
      if (activeScan(run)) await nameFiles(result.files, result.count, run);
    } catch (err) {
      if (activeScan(run)) {
        if (err instanceof DOMException && err.name === "AbortError")
          setScanCancelled(true);
        else setError(t("modReadFailed"));
      }
    } finally {
      if (activeScan(run)) {
        currentScan.current = null;
        setScan(null);
        if (input.current) input.current.value = "";
      }
    }
  };

  const choose = async () => {
    const run = beginScan();
    if (hasDirectoryPicker()) {
      try {
        const handle = await window.showDirectoryPicker({
          mode: "read",
          id: "mod-map-import",
        });
        if (activeScan(run)) await readFolder(handle, run);
      } catch (err) {
        if (activeScan(run)) {
          currentScan.current = null;
          setScan(null);
          if (err instanceof DOMException && err.name === "AbortError")
            setScanCancelled(true);
          else setError(t("modReadFailed"));
        }
      }
      return;
    }
    const element = input.current;
    if (!element) {
      cancelScan();
      return;
    }
    const cancelled = () => {
      if (activeScan(run)) cancelScan();
    };
    element.addEventListener("cancel", cancelled, { once: true });
    pickerCleanup.current = () =>
      element.removeEventListener("cancel", cancelled);
    element.click();
  };

  const pick = (list: FileList | null) => {
    pickerCleanup.current?.();
    pickerCleanup.current = null;
    if (!list?.length) {
      cancelScan();
      return;
    }
    const run = currentScan.current;
    if (!run || !activeScan(run)) return;
    void readFolder(list, run);
  };

  const form = (mod: ModMapFolder, use: ReadonlySet<string>): FormData => {
    const body = new FormData();
    body.append("source", "mod");
    if (use.size > 0) body.append("fix", fixField(use));
    for (const [field, file] of Object.entries(mod.pictures))
      if (file) body.append(field, file, file.name);
    if (mod.definition)
      body.append("definition", mod.definition, "definition.csv");
    const { index, bundle } = bundleModFiles([
      ...mod.regions,
      ...mod.states,
      ...mod.localisation,
    ]);
    body.append("mod_index", index);
    body.append("mod_bundle", bundle, "mod_bundle.bin");
    body.append("tag_map", tagMapField(merges));
    body.append("languages", languages.join(","));
    body.append("date", date);
    body.append("seed", "1");
    return body;
  };

  const makeCountries = async () => {
    for (const [tag, to] of choices) {
      if (to !== MAKE || projectTags.includes(tag)) continue;
      const response = await fetch(
        `${API_BASE}/go/projects/${projectId}/countries`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ tag, name: tag, ideology: "neutrality" }),
        },
      );

      if (!response.ok) throw new Error(t("modMakeFailed", { tag }));
    }
    await rows.mutate();
  };

  const run = async (save: boolean, use: ReadonlySet<string> = fixes) => {
    if (!folder) return;
    setBusy(save ? "save" : "preview");
    setError(null);
    setErrorFix(null);
    try {
      await makeCountries();
      const madeFor = keyOf(use);
      setUpload(0);
      const response = await mapToolUpload(
        `/api/go/projects/${projectId}/gui-render/world-map/${save ? "world" : "world-preview"}`,
        form(folder, use),
        setUpload,
      );
      if (!response.ok) {
        const copy = response.clone();
        const out = (await response.json().catch(() => ({}))) as {
          code?: string;
          values?: { tags?: UnknownTag[] };
        };
        if (
          out.code === "mod_unknown_tags" &&
          Array.isArray(out.values?.tags)
        ) {
          setUnknown(out.values.tags);
          setPreview(null);
          return;
        }
        const problem = await mapResponseProblem(copy);

        if (problem.code === "mod_state_no_land" && !use.has("state_no_land"))
          setErrorFix("state_no_land");
        throw new Error(mapProblemMessage(problem, t, locale));
      }
      if (save) {
        const out = (await response.json()) as {
          addition: { state_ids: number[] };
        };
        await additions.mutate();
        await onSaved(out.addition.state_ids.length);
        return;
      }
      setUnknown([]);
      setPreview({
        key: madeFor,
        data: (await response.json()) as WorldPreview,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(null);
      setUpload(null);
    }
  };

  const fix = (code: string) => {
    const next = new Set(fixes).add(code);
    setFixes(next);
    setConfirmed(false);
    void run(false, next);
  };
  const fixButton = (code: string) => (
    <button
      type="button"
      className={`${BUTTON} mt-1 px-2 py-0.5`}
      disabled={busy !== null}
      onClick={() => fix(code)}
    >
      {t("fixOneClick")}
    </button>
  );

  const working =
    upload !== null && upload < 100
      ? t("modUploading", { percent: upload })
      : busy === "save"
        ? t("saving")
        : t("generating");

  const waitingFor = !folder
    ? t("modPickFirst")
    : folder.missing.length > 0
      ? t("modMissingFirst")
      : notBase
        ? t("waitNotBase")
        : undecided.length > 0
          ? t("modChooseCountries")
          : !fresh
            ? t("waitPreview")
            : blocked
              ? t("waitBlocked")
              : !confirmed
                ? t("waitConfirm")
                : null;

  const panel = (
    <GamePanel
      title={t("modTitle")}
      kind={t("editKind")}
      workbenchHref={`/project/${projectId}/world-map`}
      onClose={onClose}
    >
      <div className="mt-3 space-y-4 text-sm">
        <button
          type="button"
          className="text-xs text-accent hover:underline"
          onClick={onBack}
        >
          {t("modBack")}
        </button>
        <p className="text-xs text-muted-foreground">{t("modIntro")}</p>
        <section className="space-y-2">
          <input
            ref={input}
            type="file"
            className="hidden"

            {...({ webkitdirectory: "", directory: "" } as Record<
              string,
              string
            >)}
            multiple
            onChange={(e) => pick(e.target.files)}
          />
          <button
            type="button"
            className={`${BUTTON} ui-focus-ring w-full py-1.5`}
            disabled={busy !== null || scan !== null}
            onClick={() => void choose()}
          >
            {folders ? t("modPickAgain") : t("modPick")}
          </button>
          {scan !== null && (
            <section
              className="space-y-2 rounded-md border border-border bg-muted/30 p-3 text-xs"
              data-mod-map-scan
            >
              <div role="status" className="space-y-1" aria-live="polite">
                <p className="flex items-center gap-2 font-medium">
                  <span
                    className="animate-pulse-ring h-2 w-2 shrink-0 rounded-full bg-accent"
                    aria-hidden
                  />
                  {scan.phase === "browser"
                    ? t("modReading")
                    : scan.phase === "names"
                      ? t("modCheckingNames", {
                          done: scan.done,
                          total: scan.total,
                        })
                      : t("modScanning")}
                </p>
                {scan.phase === "files" && (
                  <p className="tabular-nums text-muted-foreground">
                    {t("modDiscovered", {
                      count: scan.files,
                      directories: scan.directories,
                    })}
                  </p>
                )}
                {scan.phase === "names" && (
                  <p className="text-muted-foreground">
                    {t("modScanned", { count: scan.files })}
                  </p>
                )}
                {scan.phase === "files" && scan.path && (
                  <p className="break-all text-muted-foreground">
                    {t("modCurrentFolder", { path: scan.path })}
                  </p>
                )}
              </div>
              <progress
                className="block h-2 w-full accent-accent"
                aria-label={
                  scan.phase === "names"
                    ? t("modCheckingNames", {
                        done: scan.done,
                        total: scan.total,
                      })
                    : t("modScanning")
                }
                max={
                  scan.phase !== "browser" && scan.total
                    ? scan.total
                    : undefined
                }
                value={
                  scan.phase === "names" && scan.total > 0
                    ? scan.done
                    : scan.phase === "files" && scan.total
                      ? scan.files
                      : undefined
                }
              />
              <p className="text-muted-foreground">
                {scan.phase === "browser"
                  ? t("modScanBrowser")
                  : t("modScanLocal")}
              </p>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <ScanElapsed key={scan.startedAt} startedAt={scan.startedAt} />
                <button
                  type="button"
                  className={`${BUTTON} ui-focus-ring shrink-0`}
                  onClick={cancelScan}
                >
                  {t("modScanCancel")}
                </button>
              </div>
            </section>
          )}
          {scan === null && scanCancelled && (
            <p role="status" className="text-xs text-muted-foreground">
              {t("modScanCancelled")}
            </p>
          )}
          {scan === null && folders && (
            <p className="text-xs text-muted-foreground">
              {t("modScanned", { count: scanned })}
            </p>
          )}
          {folders && folders.length === 0 && (
            <p role="alert" className="text-xs text-warning">
              {t("modNoMap")}
            </p>
          )}
          {folders && folders.length > 1 && (
            <label className="block space-y-1 text-xs">
              <span className="text-muted-foreground">{t("modWhich")}</span>
              <select
                className="h-7 w-full rounded border border-input bg-background px-1 text-xs"
                value={chosen}
                onChange={(e) => {
                  setChosen(Number(e.target.value));
                  setUnknown([]);
                  setChoices(new Map());
                  setConfirmed(false);
                  setFixes(new Set());
                  setErrorFix(null);
                }}
              >
                {folders.map((mod, i) => (
                  <option key={mod.root} value={i}>
                    {mod.name}
                  </option>
                ))}
              </select>
            </label>
          )}
          {folder && (
            <div className="space-y-1 rounded-md border border-border p-2 text-xs">
              <p className="font-medium">{folder.name}</p>
              <ul className="space-y-0.5 text-muted-foreground">
                <li>
                  {t("modFoundPictures", {
                    count: Object.keys(folder.pictures).length,
                  })}
                </li>
                <li>{t("modFoundStates", { count: folder.states.length })}</li>
                <li>
                  {t("modFoundRegions", { count: folder.regions.length })}
                </li>
                <li>
                  {t("modFoundNames", { count: folder.localisation.length })}
                </li>
              </ul>
              {folder.missing.length > 0 && (
                <p className="text-warning">
                  {t("modMissing", {
                    files: new Intl.ListFormat(locale, {
                      type: "conjunction",
                    }).format(
                      folder.missing.map((need) => t(`modNeed_${need}`)),
                    ),
                  })}
                </p>
              )}
            </div>
          )}
        </section>

        {unknown.length > 0 && (
          <section className="space-y-2">
            <h3 className="text-xs font-medium">{t("modCountriesTitle")}</h3>
            <p className="text-xs text-muted-foreground">
              {t("modCountriesHint")}
            </p>
            <ul className="space-y-1.5">
              {unknown.map(({ tag, states }) => (
                <li key={tag} className="flex items-center gap-2 text-xs">
                  <span className="w-24 shrink-0">
                    {t("modCountryStates", { tag, states })}
                  </span>
                  <select
                    aria-label={t("modCountryChoice", { tag })}
                    className="h-7 min-w-0 flex-1 rounded border border-input bg-background px-1 text-xs"
                    value={choices.get(tag) ?? ""}
                    onChange={(e) =>
                      setChoices((old) => new Map(old).set(tag, e.target.value))
                    }
                  >
                    <option value="">{t("modChoose")}</option>
                    <option value={MAKE}>{t("modMake", { tag })}</option>
                    <optgroup label={t("modMergeInto")}>
                      {projectTags.map((to) => (
                        <option key={to} value={to}>
                          {nameOf(to)}
                        </option>
                      ))}
                    </optgroup>
                  </select>
                </li>
              ))}
            </ul>
          </section>
        )}

        {notBase && (
          <div className="space-y-1 rounded-md border border-warning/50 bg-warning/10 p-2 text-xs">
            <p>{t("notBase")}</p>
            <Link
              href={`/project/${projectId}/world-map`}
              target={workbenchTargetOf(`/project/${projectId}/world-map`)}
              className="text-accent hover:underline"
            >
              {t("openWorkbench")}
            </Link>
          </div>
        )}

        <button
          type="button"
          className={`${BUTTON} w-full py-1.5`}
          disabled={
            !folder ||
            folder.missing.length > 0 ||
            undecided.length > 0 ||
            busy !== null ||
            scan !== null
          }
          onClick={() => void run(false)}
        >
          {busy === "preview" ? working : t("modPreview")}
        </button>

        {fresh && (
          <section className="space-y-2">
            <p className="text-xs">
              {t("modSummary", {
                states: fresh.data.summary.states,
                regions: fresh.data.summary.regions,
                land: fresh.data.summary.provinces.land,
              })}
            </p>
            {(fresh.data.summary.fixed?.pixels ?? 0) > 0 && (
              <p className="text-xs text-success">
                {t("fixedPixels", {
                  count: fresh.data.summary.fixed?.pixels ?? 0,
                })}
              </p>
            )}
            {(fresh.data.summary.fixed?.lake_states ?? 0) > 0 && (
              <p className="text-xs text-success">
                {t("fixedLakeStates", {
                  count: fresh.data.summary.fixed?.lake_states ?? 0,
                })}
              </p>
            )}
            <ul className="space-y-0.5 text-xs text-muted-foreground">
              {fresh.data.summary.countries.map((c) => (
                <li key={c.tag}>
                  {t("countryStates", {
                    name: nameOf(c.tag),
                    states: c.states,
                  })}
                </li>
              ))}
            </ul>
            {findings.length > 0 && (
              <ul className="space-y-1 text-xs">
                {findings.map((finding) => {
                  const howTo =
                    finding.severity === "error"
                      ? mapFindingFix(finding.code, t)
                      : null;
                  return (
                    <li
                      key={finding.code}
                      className={
                        finding.severity === "error"
                          ? "text-destructive"
                          : "text-warning"
                      }
                    >
                      {mapFindingMessage(finding, t, locale)} ×{finding.count}
                      {howTo && (
                        <p className="text-muted-foreground">{howTo}</p>
                      )}
                      {finding.severity === "error" &&
                        ONE_CLICK_FIXES.has(finding.code) &&
                        !fixes.has(finding.code) &&
                        fixButton(finding.code)}
                    </li>
                  );
                })}
              </ul>
            )}
            <div className="space-y-1">
              <h3 className="text-xs font-medium">{t("unloadTitle")}</h3>
              <ul className="grid grid-cols-2 gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
                {fresh.data.summary.unloaded.map((path) => (
                  <li key={path} title={path}>
                    {tw(unloadedLabel(path))}
                  </li>
                ))}
              </ul>
            </div>
            <label className="flex items-start gap-2 text-xs">
              <input
                type="checkbox"
                className="ui-focus-ring mt-0.5"
                checked={confirmed}
                onChange={(e) => setConfirmed(e.target.checked)}
              />
              <span>{t("modConfirm")}</span>
            </label>
          </section>
        )}
        {error && (
          <div role="alert" className="text-xs text-destructive">
            <p>{error}</p>
            {errorFix && (
              <>
                <p className="text-muted-foreground">
                  {mapFindingFix(errorFix, t)}
                </p>
                {fixButton(errorFix)}
              </>
            )}
          </div>
        )}
        <div className="space-y-1 border-t border-border pt-3">
          <button
            type="button"
            className={`${PRIMARY} w-full py-1.5`}
            disabled={waitingFor !== null || busy !== null || scan !== null}
            onClick={() => void run(true)}
          >
            {busy === "save" ? working : t("modSave")}
          </button>
          <p className="text-xs text-muted-foreground">
            {busy === "save" ? working : (waitingFor ?? t("modSaveNote"))}
          </p>
        </div>
      </div>
    </GamePanel>
  );

  return (
    <>
      <div
        data-map-studio-canvas=""
        className="flex items-center justify-center bg-background"
        style={{ position: "absolute", left: 0, top: 0, width, height }}
      >
        {fresh ? (
          <img
            src={fresh.data.previews.political}
            alt={t("modPreviewAlt")}
            className="max-h-full max-w-full object-contain"
          />
        ) : (
          <p
            className={`text-sm text-muted-foreground ${busy === "preview" ? "animate-pulse-ring rounded-md p-3" : ""}`}
          >
            {busy === "preview" ? working : t("modStageHint")}
          </p>
        )}
      </div>
      {mounted &&
        createPortal(
          <div
            data-map-studio=""
            className="pointer-events-none fixed inset-0 z-[65]"
          >
            {panel}
          </div>,
          document.body,
        )}
    </>
  );
}
