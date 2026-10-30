"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import useSWR from "swr";
import { useParams } from "@/platform/navigation";
import { useTranslations } from "use-intl";
import { SearchX, Shield } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select } from "@/components/ui/select";
import { divisionTemplatesApi, subUnitsApi, vanillaDataApi } from "@/lib/api";
import { subunitOptionLabel } from "@/lib/i18n/subunitGloss";
import { useVanillaUnitNames } from "@/lib/vanillaUnitNames";
import { useLocaleStore } from "@/lib/store/localeStore";
import { useActiveCountry, useProjectStore } from "@/lib/store/projectStore";
import { pickDefaultOobFile } from "@/lib/oob/oobDefaults";
import type {
  DivisionTemplate,
  DivisionRegiment,
  SubUnit,
  VanillaSubunit,
} from "@/types";
import {
  confirmDiscardIfDirty,
  useRegisterDirty,
} from "@/lib/hooks/useUnsavedGuard";
import { useListboxNav } from "@/lib/hooks/useListboxNav";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import {
  buildDivisionSubunitCatalog,
  optionsPreservingCurrentToken,
} from "./divisionSubunitOptions";
import { buildRegimentGrid, buildSupportColumn } from "./divisionGridModel";
import { DivisionDesignerGrid } from "./DivisionDesignerGrid";

type Tri = "unset" | "yes" | "no";

const ABSOLUTE_AGG_FIELDS = [
  ["combat_width", "aggCombatWidth"],
  ["max_strength", "aggMaxStrength"],
  ["max_organisation", "aggMaxOrg"],
  ["default_morale", "aggRecovery"],
  ["supply_consumption", "aggSupply"],
  ["manpower", "aggManpower"],
] as const;

const AVERAGED_AGG_KEYS = new Set(["max_organisation", "default_morale"]);

const EQUIPMENT_MODIFIER_FIELDS = [
  ["soft_attack", "aggSoftAttack"],
  ["hard_attack", "aggHardAttack"],
  ["defense", "aggDefense"],
  ["breakthrough", "aggBreakthrough"],
] as const;

const AGG_FIELDS = [
  ...ABSOLUTE_AGG_FIELDS,
  ...EQUIPMENT_MODIFIER_FIELDS,
] as const;

function formatEquipmentModifier(value: number): string {
  const percent = Math.round(value * 10000) / 100;
  return `${percent > 0 ? "+" : ""}${percent}%`;
}

interface DivDraft {
  template_name: string;
  oob_file: string;
  division_names_group: string;
  priority: string;
  is_locked: Tri;
  regiments: DivisionRegiment[];
  support: DivisionRegiment[];
}

const triFromBool = (b: boolean | null): Tri =>
  b === null ? "unset" : b ? "yes" : "no";
const boolFromTri = (t: Tri): boolean | null =>
  t === "unset" ? null : t === "yes";

function tplToDraft(tpl: DivisionTemplate): DivDraft {
  return {
    template_name: tpl.template_name || "",
    oob_file: tpl.oob_file || "",
    division_names_group: tpl.division_names_group || "",
    priority:
      tpl.priority === null || tpl.priority === undefined
        ? ""
        : String(tpl.priority),
    is_locked: triFromBool(tpl.is_locked),
    regiments: (tpl.regiments || []).map((r) => ({
      token: r.token,
      x: r.x,
      y: r.y,
    })),
    support: (tpl.support || []).map((r) => ({
      token: r.token,
      x: r.x,
      y: r.y,
    })),
  };
}

function draftToPayload(d: DivDraft): Partial<DivisionTemplate> {
  return {
    template_name: d.template_name.trim(),
    oob_file: d.oob_file.trim(),
    division_names_group: d.division_names_group.trim(),
    priority: d.priority.trim() === "" ? null : Math.trunc(Number(d.priority)),
    is_locked: boolFromTri(d.is_locked),
    regiments: d.regiments.filter((r) => r.token.trim()),
    support: d.support.filter((r) => r.token.trim()),
  };
}

export default function DivisionTemplatesPage() {
  const params = useParams();
  const projectId = Number(params.id);
  const t = useTranslations("divisionTemplate");
  const tCommon = useTranslations("common");
  const tShell = useTranslations("workbenchShell");

  const locale = useLocaleStore((s) => s.locale);
  const unitNames = useVanillaUnitNames();

  const activeCountry = useActiveCountry();
  const projectStartDate = useProjectStore(
    (s) => s.currentProject?.start_date ?? null,
  );

  const listNav = useListboxNav();

  const confirmDialog = useConfirm();

  const [list, setList] = useState<DivisionTemplate[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [draft, setDraft] = useState<DivDraft | null>(null);

  const [search, setSearch] = useState("");
  const [savedDraft, setSavedDraft] = useState<DivDraft | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<"idle" | "success" | "error">(
    "idle",
  );
  const [error, setError] = useState<string | null>(null);
  const [vanillaSubunits, setVanillaSubunits] = useState<
    Record<string, VanillaSubunit>
  >({});
  const [projectSubunits, setProjectSubunits] = useState<SubUnit[]>([]);
  const [showImport, setShowImport] = useState(false);
  const [importText, setImportText] = useState("");
  const [importOob, setImportOob] = useState("");
  const [importMsg, setImportMsg] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    const res = await divisionTemplatesApi.list(projectId);
    setList(res);
  }, [projectId]);

  const { error: listLoadError, mutate: retryList } = useSWR(
    projectId ? ["division-templates/list", projectId] : null,
    () => refresh(),
  );
  useSWR("vanilla/subunits", () =>
    vanillaDataApi.subunits().then((r) => setVanillaSubunits(r.subunits || {})),
  );
  useSWR(
    projectId ? ["sub-units/for-division-templates", projectId] : null,
    () => subUnitsApi.list(projectId).then((rows) => setProjectSubunits(rows)),
  );

  const subunits = useMemo(
    () => buildDivisionSubunitCatalog(vanillaSubunits, projectSubunits),
    [vanillaSubunits, projectSubunits],
  );

  useEffect(() => {
    if (selectedId == null) {
      setDraft(null);
      setSavedDraft(null);
      return;
    }
    const tpl = list.find((x) => x.id === selectedId);
    if (tpl) {
      const d = tplToDraft(tpl);
      setDraft(d);
      setSavedDraft(d);
    }
    setSaveStatus("idle");
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedId]);

  const { regimentTokens, supportTokens } = useMemo(() => {
    const vals = Object.values(subunits);
    return {
      regimentTokens: vals
        .filter((s) => !s.is_support)
        .map((s) => s.token)
        .sort(),
      supportTokens: vals
        .filter((s) => s.is_support)
        .map((s) => s.token)
        .sort(),
    };
  }, [subunits]);

  const regimentOptions = useMemo(
    () =>
      regimentTokens.map((tok) => (
        <option key={tok} value={tok}>
          {subunitOptionLabel(tok, locale, unitNames.subunits)}
          {subunits[tok]?.source === "project"
            ? ` · ${t("projectSubunit")}`
            : ""}
        </option>
      )),
    [regimentTokens, locale, unitNames, subunits, t],
  );
  const supportOptions = useMemo(
    () =>
      supportTokens.map((tok) => (
        <option key={tok} value={tok}>
          {subunitOptionLabel(tok, locale, unitNames.subunits)}
          {subunits[tok]?.source === "project"
            ? ` · ${t("projectSubunit")}`
            : ""}
        </option>
      )),
    [supportTokens, locale, unitNames, subunits, t],
  );

  const aggregate = useMemo(() => {
    const totals: Record<string, number> = {};
    AGG_FIELDS.forEach(([k]) => {
      totals[k] = 0;
    });
    const unknown: string[] = [];
    let nKnown = 0;
    const rows = [...(draft?.regiments || []), ...(draft?.support || [])];
    for (const r of rows) {
      const su = subunits[r.token];
      if (!su) {
        if (r.token) unknown.push(r.token);
        continue;
      }
      nKnown += 1;
      AGG_FIELDS.forEach(([k]) => {
        totals[k] += Number((su as unknown as Record<string, number>)[k] || 0);
      });
    }
    if (nKnown > 0)
      AVERAGED_AGG_KEYS.forEach((k) => {
        totals[k] /= nKnown;
      });
    return { totals, unknown: Array.from(new Set(unknown)) };
  }, [draft?.regiments, draft?.support, subunits]);

  const defaultOobFile = useMemo(
    () =>
      pickDefaultOobFile({
        countryTag: activeCountry,
        startDate: projectStartDate,
        existing: list.map((tp) => tp.oob_file),
      }),
    [activeCountry, projectStartDate, list],
  );

  const handleCreate = async () => {
    setError(null);
    try {
      const defaultOob = defaultOobFile;
      const tpl = await divisionTemplatesApi.create(projectId, {
        template_name: "New Division",
        oob_file: defaultOob,
      });
      await refresh();
      setSelectedId(tpl.id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const handleSave = async () => {
    if (!draft || selectedId == null) return;
    setSaving(true);
    setSaveStatus("idle");
    setError(null);
    try {
      await divisionTemplatesApi.update(
        projectId,
        selectedId,
        draftToPayload(draft),
      );
      await refresh();
      setSavedDraft(draft);
      setSaveStatus("success");
      setTimeout(() => setSaveStatus("idle"), 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setSaveStatus("error");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (!(await confirmDialog({ body: t("deleteConfirm"), danger: true })))
      return;
    await divisionTemplatesApi.delete(projectId, id);
    await refresh();
    if (selectedId === id) setSelectedId(null);
  };

  const handleImport = async () => {
    setImportMsg(null);
    if (!importOob.trim()) {
      setImportMsg(t("importOobRequired"));
      return;
    }
    try {
      const r = await divisionTemplatesApi.importText(
        projectId,
        importOob.trim(),
        importText,
      );
      await refresh();
      setImportMsg(
        t("importResult", {
          created: r.created,
          updated: r.updated,
          skipped: r.skipped,
        }),
      );
      setImportText("");
    } catch (e) {
      setImportMsg(e instanceof Error ? e.message : String(e));
    }
  };

  const selected = list.find((x) => x.id === selectedId);
  const isUnsaved =
    draft != null && JSON.stringify(draft) !== JSON.stringify(savedDraft);
  useRegisterDirty("division-templates", isUnsaved);

  const renderOverflowRows = (
    kind: "regiments" | "support",
    indices: number[],
  ) => {
    if (!draft || indices.length === 0) return null;
    const rows = draft[kind];
    const tokens = kind === "regiments" ? regimentTokens : supportTokens;
    const baseOptions = kind === "regiments" ? regimentOptions : supportOptions;
    const placeholder =
      kind === "regiments" ? t("tokenPlaceholder") : t("supportPlaceholder");
    const update = (i: number, patch: Partial<DivisionRegiment>) => {
      const next = rows.map((r, j) => (j === i ? { ...r, ...patch } : r));
      setDraft({ ...draft, [kind]: next });
    };
    const remove = (i: number) =>
      setDraft({ ...draft, [kind]: rows.filter((_, j) => j !== i) });
    return (
      <section className="space-y-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-warning">
          {kind === "regiments"
            ? t("overflowTitle")
            : t("overflowSupportTitle")}{" "}
          <span className="text-warning/70">({indices.length})</span>
        </h3>
        {indices.map((i) => {
          const r = rows[i];

          const visibleTokens = optionsPreservingCurrentToken(tokens, r.token);
          const unknownTokens = visibleTokens.slice(
            0,
            visibleTokens.length - tokens.length,
          );
          return (
            <div
              key={i}
              className="grid grid-cols-[1fr_70px_70px_auto] gap-2 items-center"
            >
              <Select
                className="px-2 text-xs font-mono"
                value={r.token}
                onChange={(e) => update(i, { token: e.target.value })}
              >
                <option value="">{placeholder}</option>
                {unknownTokens.map((token) => (
                  <option key={token} value={token}>
                    {token} · {t("unknownPreserved")}
                  </option>
                ))}
                {baseOptions}
              </Select>
              <Input
                type="number"
                className="font-mono text-xs"
                title={t("colX")}
                value={r.x}
                onChange={(e) =>
                  update(i, { x: Math.trunc(Number(e.target.value)) || 0 })
                }
              />
              <Input
                type="number"
                className="font-mono text-xs"
                title={t("colY")}
                value={r.y}
                onChange={(e) =>
                  update(i, { y: Math.trunc(Number(e.target.value)) || 0 })
                }
              />
              <Button
                size="sm"
                variant="ghost"
                className="px-2 text-muted-foreground hover:text-destructive"
                aria-label={t("removeRowAria")}
                title={t("removeRowAria")}
                onClick={() => remove(i)}
              >
                ×
              </Button>
            </div>
          );
        })}
      </section>
    );
  };

  const filteredList = list.filter((tpl) => {
    const needle = search.trim().toLowerCase();
    if (!needle) return true;
    return (
      (tpl.template_name || "").toLowerCase().includes(needle) ||
      (tpl.oob_file || "").toLowerCase().includes(needle)
    );
  });

  const zeroHero = (
    <EditorEmptyState
      variant="hero"
      icon={<Shield className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyAll")}
      description={t("emptyHint")}
      actions={
        <Button
          size="sm"
          onClick={() => {
            void handleCreate();
          }}
        >
          {tCommon("emptyCtaCreate")}
        </Button>
      }
    />
  );

  const listPanel = (
    <div>
      <WorkbenchListHeader
        title={t("title")}
        count={filteredList.length}
        totalCount={list.length}
        searchValue={search}
        onSearchChange={setSearch}
        searchPlaceholder={tShell("searchPlaceholderGeneric")}
        renderSearch={list.length > 0}
        action={
          <Button size="sm" onClick={handleCreate}>
            {t("newButton")}
          </Button>
        }
        filters={
          <button
            type="button"
            onClick={() => setShowImport((v) => !v)}
            className="text-xs underline decoration-dotted text-muted-foreground hover:text-foreground ui-interactive"
          >
            {t("importTitle")}
          </button>
        }
      />
      <div
        className="px-3 pb-3 space-y-0.5 outline-none"
        {...listNav.containerProps}
        aria-label={t("title")}
      >
        {list.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("emptyAll")}
          </p>
        ) : filteredList.length === 0 ? (
          <div className="p-4 flex flex-col items-center gap-2 text-center border border-dashed rounded-lg bg-muted/20">
            <SearchX className="h-5 w-5 text-muted-foreground" />
            <p className="text-xs text-muted-foreground">
              {tShell("noMatchesGeneric")}
            </p>
          </div>
        ) : null}
        {filteredList.map((tpl) => (
          <div
            key={tpl.id}
            {...listNav.optionProps(tpl.id === selectedId)}
            onClick={() => {
              if (tpl.id === selectedId || confirmDiscardIfDirty())
                setSelectedId(tpl.id);
            }}
            className={
              "w-full text-left px-2 py-1.5 rounded text-xs group flex items-center justify-between cursor-pointer hover:bg-muted focus:outline-none focus-visible:ring-1 focus-visible:ring-ring " +
              (tpl.id === selectedId
                ? "bg-primary/10 text-primary font-semibold"
                : "")
            }
          >
            <div className="min-w-0 flex-1">
              <p className="truncate font-medium">{tpl.template_name}</p>
              <p className="text-muted-foreground truncate text-micro font-mono">
                {tpl.oob_file}
              </p>
            </div>
            <button
              type="button"
              aria-label={t("deleteAria")}
              title={t("deleteAria")}
              onClick={(e) => {
                e.stopPropagation();
                handleDelete(tpl.id);
              }}
              className="opacity-0 group-hover:opacity-100 focus-visible:opacity-100 ml-2 text-muted-foreground hover:text-destructive shrink-0 cursor-pointer rounded"
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </div>
  );

  const importBlock = showImport && (
    <div className="p-4 border-b border-border bg-muted/30 space-y-2">
      <p className="text-xs text-muted-foreground">{t("importHint")}</p>
      <div className="space-y-1 max-w-xs">
        <Label>{t("importOobLabel")}</Label>
        <Input
          className="font-mono text-xs"
          placeholder={t("oobFilePlaceholder")}
          value={importOob}
          onChange={(e) => setImportOob(e.target.value)}
        />
      </div>
      <Textarea
        className="font-mono text-xs min-h-28"
        placeholder={t("importPlaceholder")}
        value={importText}
        onChange={(e) => setImportText(e.target.value)}
      />
      <div className="flex items-center gap-2">
        <Button
          size="sm"
          onClick={handleImport}
          disabled={!importText.trim() || !importOob.trim()}
        >
          {t("importButton")}
        </Button>
        {importMsg && (
          <span className="text-xs text-muted-foreground">{importMsg}</span>
        )}
      </div>
    </div>
  );

  const renderEditor = (draft: DivDraft) => (
    <div className="flex min-w-0">
      <div className="flex-1 min-w-0 p-4 max-w-3xl space-y-4">
        <p className="text-xs text-muted-foreground">
          {t("landing")}{" "}
          <span className="font-mono">
            history/units/{draft.oob_file || "<oob>"}.txt
          </span>
        </p>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label>{t("nameLabel")}</Label>
            <Input
              value={draft.template_name}
              onChange={(e) =>
                setDraft({ ...draft, template_name: e.target.value })
              }
            />
          </div>
          <div className="space-y-1">
            <Label>{t("oobFileLabel")}</Label>
            <Input
              className="font-mono text-xs"
              placeholder={t("oobFilePlaceholder")}
              value={draft.oob_file}
              onChange={(e) => setDraft({ ...draft, oob_file: e.target.value })}
            />
          </div>
          <div className="space-y-1">
            <Label>{t("divisionNamesGroupLabel")}</Label>
            <Input
              className="font-mono text-xs"
              value={draft.division_names_group}
              onChange={(e) =>
                setDraft({ ...draft, division_names_group: e.target.value })
              }
            />
          </div>
          <div className="space-y-1">
            <Label>{t("priorityLabel")}</Label>
            <Select
              className="px-2 text-xs"
              value={draft.priority}
              onChange={(e) => setDraft({ ...draft, priority: e.target.value })}
            >
              <option value="">{t("priorityUnset")}</option>
              <option value="0">{t("priorityReserves")}</option>
              <option value="1">{t("priorityDefault")}</option>
              <option value="2">{t("priorityElite")}</option>
            </Select>
          </div>
          <div className="space-y-1">
            <Label>{t("isLockedLabel")}</Label>
            <div className="flex gap-1">
              {(["unset", "yes", "no"] as Tri[]).map((opt) => (
                <button
                  key={opt}
                  type="button"
                  onClick={() => setDraft({ ...draft, is_locked: opt })}
                  className={`px-2 py-1 rounded text-micro border ui-interactive ${draft.is_locked === opt ? "bg-primary/10 text-primary border-primary/40 font-semibold" : "border-border text-muted-foreground hover:bg-muted"}`}
                >
                  {opt === "unset"
                    ? t("triUnset")
                    : opt === "yes"
                      ? t("triYes")
                      : t("triNo")}
                </button>
              ))}
            </div>
          </div>
        </div>

        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {t("designerTitle")}
            </h3>
            <span className="text-micro text-muted-foreground">
              {t("nRegiments", { count: draft.regiments.length })} ·{" "}
              {t("nSupport", { count: draft.support.length })}
            </span>
          </div>
          <DivisionDesignerGrid
            regiments={draft.regiments}
            support={draft.support}
            subunits={subunits}
            regimentTokens={regimentTokens}
            supportTokens={supportTokens}
            locale={locale}
            onChangeRegiments={(rows) =>
              setDraft({ ...draft, regiments: rows })
            }
            onChangeSupport={(rows) => setDraft({ ...draft, support: rows })}
          />
        </div>
        {renderOverflowRows(
          "regiments",
          buildRegimentGrid(draft.regiments).overflow,
        )}
        {renderOverflowRows(
          "support",
          buildSupportColumn(draft.support).overflow,
        )}

        {(saving || saveStatus !== "idle" || isUnsaved) && (
          <div
            className={`-mx-4 px-4 py-1.5 text-xs font-medium flex items-center gap-2 transition-[opacity,background-color,border-color] duration-300 ${
              saving
                ? "bg-muted/80 text-muted-foreground border-y border-border"
                : saveStatus === "success"
                  ? "bg-success/10 text-success border-y border-success/20"
                  : saveStatus === "error"
                    ? "bg-destructive/10 text-destructive border-y border-destructive/20"
                    : isUnsaved
                      ? "bg-warning/10 text-warning border-y border-warning/20"
                      : ""
            }`}
          >
            <span>
              {saving
                ? t("saving")
                : saveStatus === "success"
                  ? t("saved")
                  : saveStatus === "error"
                    ? (error ?? t("saveFailed"))
                    : isUnsaved
                      ? t("unsavedChanges")
                      : ""}
            </span>
          </div>
        )}
        <div className="flex items-center gap-2 pt-2 border-t border-border">
          <Button onClick={handleSave} disabled={saving} size="sm">
            {t("save")}
          </Button>
        </div>
      </div>
    </div>
  );

  const statRow = (labelKey: string, value: string) => (
    <div
      key={labelKey}
      className="flex items-center justify-between gap-2 text-xs"
    >
      <span className="text-ingame-text/80">{t(labelKey)}</span>
      <span className="tech-tabular shrink-0 font-mono font-semibold text-ingame-title">
        {value}
      </span>
    </div>
  );
  const round2 = (k: string) =>
    String(Math.round((aggregate.totals[k] || 0) * 100) / 100);
  const previewStats = draft && (
    <div className="space-y-3">
      <div className="rounded-md border border-ingame-edge/60 bg-gradient-to-b from-ingame-panel/90 to-ingame-panel-deep/95 p-3 shadow-sm">
        <p className="text-micro font-semibold uppercase tracking-widest text-ingame-gold/90">
          {t("statsBase")}
        </p>
        <div className="mt-1.5 space-y-1">
          {statRow("aggCombatWidth", round2("combat_width"))}
          {statRow("aggMaxStrength", round2("max_strength"))}
          {statRow("aggMaxOrg", round2("max_organisation"))}
          {statRow("aggRecovery", round2("default_morale"))}
          {statRow("aggSupply", round2("supply_consumption"))}
        </div>
        <p className="mt-3 text-micro font-semibold uppercase tracking-widest text-ingame-gold/90">
          {t("statsCombat")}
        </p>
        <div className="mt-1.5 space-y-1">
          {EQUIPMENT_MODIFIER_FIELDS.map(([k, labelKey]) =>
            statRow(
              labelKey,
              formatEquipmentModifier(aggregate.totals[k] || 0),
            ),
          )}
        </div>
        <p className="mt-3 text-micro font-semibold uppercase tracking-widest text-ingame-gold/90">
          {t("statsCost")}
        </p>
        <div className="mt-1.5 space-y-1">
          {statRow("aggManpower", round2("manpower"))}
        </div>
        <div className="mt-3 flex items-center justify-between border-t border-ingame-edge/40 pt-2 text-micro text-ingame-text/70">
          <span>{t("nRegiments", { count: draft.regiments.length })}</span>
          <span>{t("nSupport", { count: draft.support.length })}</span>
        </div>
      </div>
      <p className="text-micro text-muted-foreground">{t("aggHint")}</p>
      {aggregate.unknown.length > 0 && (
        <p className="text-micro text-warning">
          {t("unknownTokens", { tokens: aggregate.unknown.join(", ") })}
        </p>
      )}
    </div>
  );

  if (listLoadError && list.length === 0) {
    return (
      <WorkbenchLoadError
        onRetry={() => retryList(undefined, { throwOnError: false })}
      />
    );
  }

  return (
    <WorkbenchShell
      gameViewHref={gameViewHref(projectId, {
        panel: "countrydivisiondesignerview",
        template: selected?.template_name,
        tag: activeCountry,
      })}
      listPanel={listPanel}
      editorPanel={
        <div className="flex min-h-full flex-col">
          {importBlock}
          {!selected || !draft ? (
            <div className="flex flex-1 items-center justify-center p-6">
              {list.length === 0 ? (
                zeroHero
              ) : (
                <EditorEmptyState
                  className="max-w-xs"
                  icon={<Shield className="h-6 w-6 text-muted-foreground/60" />}
                  title={tShell("noSelectionGeneric")}
                  description={t("selectHint")}
                />
              )}
            </div>
          ) : (
            renderEditor(draft)
          )}
        </div>
      }
      previewPanel={
        previewStats ? (
          <div className="h-full space-y-3 overflow-y-auto p-3">
            {previewStats}
          </div>
        ) : (
          <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
            {tShell("previewUnavailableGeneric")}
          </div>
        )
      }
    />
  );
}
