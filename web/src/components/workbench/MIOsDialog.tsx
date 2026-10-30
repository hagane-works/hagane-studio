"use client";

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useLinkedRowSelection } from "@/lib/workbench-state/useLinkedRowSelection";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import { useTranslations } from "use-intl";
import { ArrowLeft, Factory, SearchX } from "lucide-react";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { ImageUpload } from "@/shared/upload/ImageUpload";
import { IDEA_PICTURE_SIZE } from "@/shared/upload/sizes";
import { miosApi, type MIORow, type MIOTraitEntry } from "@/lib/api";
import {
  useActiveCountry,
  useEffectiveCountryFilter,
  useProjectStore,
} from "@/lib/store/projectStore";
import { matchesCountryScope, partitionByScope } from "@/lib/utils/scope";
import {
  ScopeBadge,
  ScopeSectionHeader,
} from "@/components/workbench/ScopeBadge";
import { CountryTagChip } from "@/components/workbench/CountryTagChip";
import { EditorEmptyState } from "@/components/workbench/EditorEmptyState";
import { WorkbenchLoadError } from "@/components/workbench/WorkbenchLoadError";
import { WorkbenchShell } from "@/components/workbench/WorkbenchShell";
import { gameViewHref } from "@/lib/gameViewLinks";
import { WorkbenchListHeader } from "@/components/workbench/WorkbenchListHeader";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { DialogSaveStatusBar } from "@/components/workbench/DialogSaveStatusBar";
import { MIOTraitsEditor } from "@/components/workbench/MIOTraitsEditor";
import {
  beginDraftSave,
  confirmDraftSelection,
  editDraftSession,
  emptyDraftSession,
  openDraftSession,
  requestDraftSelection,
  resolveDraftSave,
  stableDraftFingerprint,
  type DraftSession,
} from "@/lib/workbench-state/draftSession";

interface MIOsDialogProps {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

interface MIOsContentProps {
  projectId: number;
}

type Draft = Omit<
  MIORow,
  "id" | "project_id" | "created_at" | "updated_at" | "source"
>;

const blankDraft = (): Draft => ({
  mio_id: "",
  include: "",
  icon: "",
  equipment_type: [],
  research_categories: [],
  allowed: "",
  available: "",
  visible: "",
  traits: [],
  raw_extras: {},
  country_tag: "",
  scope_kind: "own",
  shared_country_tags: null,
  name_en: "",
  name_zh: "",
  desc_en: "",
  desc_zh: "",

  display_name_locs: {},
  description_locs: {},
});

function rowToDraft(row: MIORow): Draft {
  return {
    mio_id: row.mio_id,
    include: row.include,
    icon: row.icon,
    equipment_type: Array.isArray(row.equipment_type) ? row.equipment_type : [],
    research_categories: Array.isArray(row.research_categories)
      ? row.research_categories
      : [],
    allowed: row.allowed,
    available: row.available,
    visible: row.visible,
    traits: Array.isArray(row.traits) ? row.traits : [],
    raw_extras: row.raw_extras || {},
    country_tag: row.country_tag,
    scope_kind: row.scope_kind,
    shared_country_tags: row.shared_country_tags,
    name_en: row.name_en,
    name_zh: row.name_zh,
    desc_en: row.desc_en,
    desc_zh: row.desc_zh,
    display_name_locs: { ...(row.display_name_locs ?? {}) },
    description_locs: { ...(row.description_locs ?? {}) },
  };
}

function traitsToJson(traits: MIOTraitEntry[]): string {
  if (!traits || traits.length === 0) return "";
  return JSON.stringify(traits, null, 2);
}

function jsonToTraits(text: string): MIOTraitEntry[] | null {
  const trimmed = text.trim();
  if (!trimmed) return [];
  try {
    const parsed = JSON.parse(trimmed);
    if (!Array.isArray(parsed)) return null;
    return parsed as MIOTraitEntry[];
  } catch {
    return null;
  }
}

function rawExtrasToJson(raw: Record<string, unknown>): string {
  if (!raw || Object.keys(raw).length === 0) return "";
  return JSON.stringify(raw, null, 2);
}

function jsonToRawExtras(text: string): Record<string, unknown> | null {
  const trimmed = text.trim();
  if (!trimmed) return {};
  try {
    const parsed = JSON.parse(trimmed);
    if (
      typeof parsed !== "object" ||
      parsed === null ||
      Array.isArray(parsed)
    ) {
      return null;
    }
    return parsed as Record<string, unknown>;
  } catch {
    return null;
  }
}

function tokensToText(tokens: string[] | undefined): string {
  return Array.isArray(tokens) ? tokens.join(" ") : "";
}
function textToTokens(text: string): string[] {
  return text
    .split(/[\s,]+/)
    .map((s) => s.trim())
    .filter(Boolean);
}

const RAW_EXTRAS_PLACEHOLDER = `{
  "tree_header_text": "text = mio_header_tank_construction x = 3",
  "background": "GFX_idea_unknown"
}`;

const CONDITION_LABEL_KEYS = {
  allowed: "fieldAllowed",
  available: "fieldAvailable",
  visible: "fieldVisible",
} as const;

const TRAITS_PLACEHOLDER = `[
  {
    "kind": "initial",
    "token": "USA_mio_trait_strict_design_specifications",
    "icon": "GFX_generic_mio_trait_icon_reliability",
    "equipment_bonus": { "reliability": 0.1 }
  },
  {
    "kind": "add",
    "token": "USA_mio_trait_expanded_production",
    "position": { "x": 0, "y": 1 },
    "all_parents": ["USA_mio_trait_strict_design_specifications"],
    "production_bonus": { "production_cost_factor": -0.1 }
  }
]`;

export function MIOsContent({ projectId }: MIOsContentProps) {
  const t = useTranslations("mIOsDialog");
  const tCommon = useTranslations("common");
  const tShell = useTranslations("workbenchShell");

  const baseModDepKeys = useBaseModDepKeys();
  const confirmDialog = useConfirm();

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );

  const countryTag = useActiveCountry();
  const filterCountry = useEffectiveCountryFilter();
  const viewMode = useProjectStore((s) => s.viewMode);
  const [rows, setRows] = useState<MIORow[]>([]);

  const [search, setSearch] = useState("");
  const [activeId, setActiveId] = useState<number | "new" | null>(null);
  const [draft, setDraft] = useState<Draft>(() => blankDraft());
  const [traitsBuffer, setTraitsBuffer] = useState<string>("");
  const [traitsError, setTraitsError] = useState<string | null>(null);

  const [traitsView, setTraitsView] = useState<"rich" | "raw">("rich");
  const [rawExtrasBuffer, setRawExtrasBuffer] = useState<string>("");
  const [rawExtrasError, setRawExtrasError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [listLoadFailed, setListLoadFailed] = useState(false);
  const [status, setStatus] = useState<"" | "saved" | "deleted">("");
  const ownership = useRef<DraftSession<Draft>>(emptyDraftSession());
  const latestListRequest = useRef(0);
  const fingerprint = useCallback(
    (value: Draft) => stableDraftFingerprint(value),
    [],
  );

  const refresh = useCallback(async (): Promise<MIORow[] | null> => {
    const request = ++latestListRequest.current;
    try {
      const res = await miosApi.list(projectId);
      if (request !== latestListRequest.current) return null;
      setRows(res.mios);
      setListLoadFailed(false);
      return res.mios;
    } catch {
      if (request === latestListRequest.current) setListLoadFailed(true);
      return null;
    }
  }, [projectId]);

  const applyRow = useCallback((row: MIORow) => {
    const next = rowToDraft(row);
    setDraft(next);
    setTraitsBuffer(traitsToJson(row.traits || []));
    setTraitsError(null);
    setTraitsView("rich");
    setRawExtrasBuffer(rawExtrasToJson(row.raw_extras || {}));
    setRawExtrasError(null);
    return next;
  }, []);

  const selectRow = useCallback(
    async (row: MIORow) => {
      const next = rowToDraft(row);
      const requested = requestDraftSelection(
        ownership.current,
        String(row.id),
        fingerprint(next),
        next,
      );
      if (
        requested.kind === "confirm-discard" &&
        !(await confirmDialog({ body: tCommon("discardUnsavedConfirm") }))
      )
        return;
      ownership.current =
        requested.kind === "selected"
          ? requested.session
          : confirmDraftSelection(
              ownership.current,
              String(row.id),
              fingerprint(next),
              next,
            );
      setActiveId(row.id);
      applyRow(row);
      setError(null);
    },
    [applyRow, confirmDialog, fingerprint, tCommon],
  );

  useLinkedRowSelection("mio", rows, selectRow);

  const startNew = useCallback(async () => {
    const next = { ...blankDraft(), country_tag: countryTag };
    const requested = requestDraftSelection(
      ownership.current,
      "new",
      fingerprint(next),
      next,
    );
    if (
      requested.kind === "confirm-discard" &&
      !(await confirmDialog({ body: tCommon("discardUnsavedConfirm") }))
    )
      return;
    ownership.current =
      requested.kind === "selected"
        ? requested.session
        : confirmDraftSelection(
            ownership.current,
            "new",
            fingerprint(next),
            next,
          );
    setActiveId("new");
    setDraft(next);
    setTraitsBuffer("");
    setTraitsError(null);
    setTraitsView("rich");
    setRawExtrasBuffer("");
    setRawExtrasError(null);
    setError(null);
  }, [confirmDialog, countryTag, fingerprint, tCommon]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (activeId !== null && ownership.current.identity === String(activeId))
      return;
    if (activeId === "new") {
      const blank = { ...blankDraft(), country_tag: countryTag };
      setDraft(blank);
      setTraitsBuffer("");
      setTraitsError(null);
      setTraitsView("rich");
      setRawExtrasBuffer("");
      setRawExtrasError(null);
    } else if (typeof activeId === "number") {
      const row = rows.find((r) => r.id === activeId);
      if (row) {
        setDraft({
          mio_id: row.mio_id,
          include: row.include,
          icon: row.icon,
          equipment_type: Array.isArray(row.equipment_type)
            ? row.equipment_type
            : [],
          research_categories: Array.isArray(row.research_categories)
            ? row.research_categories
            : [],
          allowed: row.allowed,
          available: row.available,
          visible: row.visible,
          traits: Array.isArray(row.traits) ? row.traits : [],
          raw_extras: row.raw_extras || {},
          country_tag: row.country_tag,
          scope_kind: row.scope_kind,
          shared_country_tags: row.shared_country_tags,
          name_en: row.name_en,
          name_zh: row.name_zh,
          desc_en: row.desc_en,
          desc_zh: row.desc_zh,

          display_name_locs: { ...(row.display_name_locs ?? {}) },
          description_locs: { ...(row.description_locs ?? {}) },
        });
        setTraitsBuffer(traitsToJson(row.traits || []));
        setTraitsError(null);
        setTraitsView("rich");
        setRawExtrasBuffer(rawExtrasToJson(row.raw_extras || {}));
        setRawExtrasError(null);
      }
    }
  }, [activeId, rows, countryTag]);

  useEffect(() => {
    if (typeof activeId !== "number") return;
    if (!filterCountry) return;
    const sel = rows.find((r) => r.id === activeId);
    if (sel && !matchesCountryScope(sel, filterCountry)) {
      ownership.current = emptyDraftSession();
      setActiveId(null);
      setDraft(blankDraft());
    }
  }, [filterCountry, activeId, rows]);

  const update = <K extends keyof Draft>(key: K, value: Draft[K]) => {
    setDraft((current) => {
      const next = { ...current, [key]: value };
      ownership.current = editDraftSession(
        ownership.current,
        next,
        fingerprint(next),
      );
      return next;
    });
  };

  const handleTraitsChange = (text: string) => {
    setTraitsBuffer(text);
    const parsed = jsonToTraits(text);
    if (parsed === null) {
      setTraitsError(t("invalidJson"));
    } else {
      setTraitsError(null);
      update("traits", parsed);
    }
  };

  const toggleTraitsView = () => {
    setTraitsView((view) => {
      if (view === "raw") {
        const parsed = jsonToTraits(traitsBuffer);
        if (parsed === null) {
          setTraitsError(t("invalidJson"));
        } else {
          setTraitsError(null);
          update("traits", parsed);
        }
        return "rich";
      }

      setTraitsBuffer(traitsToJson(draft.traits));
      setTraitsError(null);
      return "raw";
    });
  };

  const handleRawExtrasChange = (text: string) => {
    setRawExtrasBuffer(text);
    const parsed = jsonToRawExtras(text);
    if (parsed === null) {
      setRawExtrasError(t("invalidJson"));
    } else {
      setRawExtrasError(null);
      update("raw_extras", parsed);
    }
  };

  const handleSave = async () => {
    if (traitsError) {
      setError(t("fixTraitsFirst", { error: traitsError }));
      return;
    }
    if (rawExtrasError) {
      setError(t("fixTraitsFirst", { error: rawExtrasError }));
      return;
    }
    setSaving(true);
    setError(null);
    const save = beginDraftSave(ownership.current);
    if (!save) {
      setSaving(false);
      return;
    }
    const savingDraft = draft;
    try {
      if (activeId === "new") {
        const created = await miosApi.create(projectId, savingDraft);
        await refresh();
        const saved = rowToDraft(created);
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(saved),
          saved,
        );
        if (ownership.current.identity === "new") {
          ownership.current = openDraftSession(
            ownership.current,
            String(created.id),
            fingerprint(saved),
            saved,
          );
          setActiveId(created.id);
          applyRow(created);
        }
      } else if (typeof activeId === "number") {
        const updated = await miosApi.update(projectId, activeId, savingDraft);
        await refresh();
        const saved = rowToDraft(updated);
        ownership.current = resolveDraftSave(
          ownership.current,
          save,
          fingerprint(saved),
          saved,
        );
        if (ownership.current.identity === String(updated.id)) {
          setDraft(ownership.current.draft!);
          setTraitsBuffer(traitsToJson(ownership.current.draft!.traits));
          setRawExtrasBuffer(
            rawExtrasToJson(ownership.current.draft!.raw_extras),
          );
        }
      }
      setStatus("saved");
      setTimeout(() => setStatus(""), 1500);
    } catch (e) {
      const msg = (e as Error).message || "";
      const isCollision =
        msg.includes("already exists") && msg.includes("Pick a unique");
      if (!isCollision) {
        setError(msg);
      }
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (typeof activeId !== "number") return;
    if (!(await confirmDialog({ body: t("confirmDelete"), danger: true })))
      return;
    setSaving(true);
    setError(null);
    try {
      await miosApi.remove(projectId, activeId);
      await refresh();
      if (ownership.current.identity === String(activeId)) {
        ownership.current = emptyDraftSession();
        setActiveId(null);
        setDraft(blankDraft());
      }
      setStatus("deleted");
      setTimeout(() => setStatus(""), 1500);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const isExisting = typeof activeId === "number";

  const sortedRows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return rows
      .filter((r) => matchesCountryScope(r, filterCountry))
      .filter(
        (r) =>
          !needle ||
          (r.mio_id || "").toLowerCase().includes(needle) ||
          (r.name_en || "").toLowerCase().includes(needle) ||
          (r.name_zh || "").toLowerCase().includes(needle),
      )
      .sort((a, b) => a.mio_id.localeCompare(b.mio_id));
  }, [rows, filterCountry, search]);

  const { own: ownRows, shared: sharedRows } = useMemo(
    () => partitionByScope(sortedRows, filterCountry),
    [sortedRows, filterCountry],
  );

  const zeroHero = (
    <EditorEmptyState
      variant="hero"
      icon={<Factory className="h-10 w-10 text-muted-foreground/60" />}
      title={t("emptyList")}
      actions={
        <Button
          size="sm"
          onClick={() => {
            void startNew();
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
        title={t("listTitle")}
        count={sortedRows.length}
        totalCount={rows.length}
        searchValue={search}
        onSearchChange={setSearch}
        searchPlaceholder={tShell("searchPlaceholderGeneric")}
        renderSearch={rows.length > 0}
        action={
          <Button
            size="sm"
            variant={activeId === "new" ? "default" : "outline"}
            onClick={() => void startNew()}
          >
            {t("newMio")}
          </Button>
        }
      />
      <div className="px-3 pb-3 space-y-1">
        {listLoadFailed && rows.length === 0 ? null : rows.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            {t("emptyList")}
          </p>
        ) : sortedRows.length === 0 ? (
          search.trim() ? (
            <div className="p-4 flex flex-col items-center gap-2 text-center border border-dashed rounded-lg bg-muted/20">
              <SearchX className="h-5 w-5 text-muted-foreground" />
              <p className="text-xs text-muted-foreground">
                {tShell("noMatchesGeneric")}
              </p>
            </div>
          ) : (
            <div className="rounded-md border border-border bg-card p-4 flex flex-col items-center gap-2 text-center">
              <Factory className="w-6 h-6 text-muted-foreground/60" />
              <p className="text-xs font-medium text-foreground">
                {t("emptyList")}
              </p>
            </div>
          )
        ) : (
          (() => {
            const renderRow = (r: MIORow) => (
              <button
                key={r.id}
                onClick={() => void selectRow(r)}
                className={`w-full text-left px-2 py-1.5 rounded text-xs font-mono
                        border transition-colors
                        ${
                          activeId === r.id
                            ? "bg-primary/10 border-primary text-foreground"
                            : "border-transparent hover:bg-muted/50 text-muted-foreground hover:text-foreground"
                        }`}
                title={`${r.mio_id}${r.country_tag ? ` · ${r.country_tag}` : ""}`}
              >
                <div className="truncate flex items-center gap-1.5">
                  {viewMode === "all" ? (
                    <CountryTagChip tag={r.country_tag} />
                  ) : (
                    <ScopeBadge entity={r} activeCountry={filterCountry} />
                  )}
                  <span className="truncate">{r.mio_id}</span>
                </div>
                {r.country_tag && (
                  <div className="text-micro text-muted-foreground mt-0.5">
                    {r.country_tag}
                    {r.include && ` · ${r.include}`}
                  </div>
                )}
              </button>
            );
            if (!filterCountry) {
              return (
                <>
                  <ScopeSectionHeader variant="all" count={sortedRows.length} />
                  {sortedRows.map(renderRow)}
                </>
              );
            }
            return (
              <>
                {ownRows.length > 0 && (
                  <>
                    <ScopeSectionHeader variant="own" count={ownRows.length} />
                    {ownRows.map(renderRow)}
                  </>
                )}
                {sharedRows.length > 0 && (
                  <>
                    <ScopeSectionHeader
                      variant="shared"
                      count={sharedRows.length}
                    />
                    {sharedRows.map(renderRow)}
                  </>
                )}
              </>
            );
          })()
        )}
      </div>
    </div>
  );

  const editorForm = (
    <div className="space-y-3 p-4">
      {activeId === "new" && (
        <button
          type="button"
          onClick={() => {
            ownership.current = emptyDraftSession();
            setActiveId(null);
            setDraft(blankDraft());
          }}
          className="ui-interactive inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          {tShell("backFromNew")}
        </button>
      )}

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-2">
        <div>
          <Label>{t("mioIdLabel")}</Label>
          <Input
            value={draft.mio_id}
            onChange={(e) => update("mio_id", e.target.value)}
            placeholder="USA_marmon_herrington_organization"
            className="font-mono text-xs"
          />
        </div>
        <div>
          <Label>{t("countryTagLabel")}</Label>
          <Input
            value={draft.country_tag}
            onChange={(e) =>
              update("country_tag", e.target.value.toUpperCase())
            }
            placeholder="USA"
            className="font-mono text-xs"
          />
        </div>
        <div>
          <Label>{t("includeLabel")}</Label>
          <Input
            value={draft.include}
            onChange={(e) => update("include", e.target.value)}
            placeholder="generic_mobile_tank_organization"
            className="font-mono text-xs"
          />
        </div>
        <div>
          <Label>{t("iconLabel")}</Label>
          <Input
            value={draft.icon}
            onChange={(e) => update("icon", e.target.value)}
            placeholder="GFX_idea_marmon_herrington"
            className="font-mono text-xs"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-2">
        <div>
          <Label>{t("equipmentTypeLabel")}</Label>
          <Input
            value={tokensToText(draft.equipment_type)}
            onChange={(e) =>
              update("equipment_type", textToTokens(e.target.value))
            }
            placeholder="armor light_tank medium_tank"
            className="font-mono text-xs"
          />
          <p className="text-micro text-muted-foreground mt-0.5">
            {t("equipmentTypeHint")}
          </p>
        </div>
        <div>
          <Label>{t("researchCategoriesLabel")}</Label>
          <Input
            value={tokensToText(draft.research_categories)}
            onChange={(e) =>
              update("research_categories", textToTokens(e.target.value))
            }
            placeholder="mio_cat_tech_light_armor_and_modules"
            className="font-mono text-xs"
          />
          <p className="text-micro text-muted-foreground mt-0.5">
            {t("researchCategoriesHint")}
          </p>
        </div>
      </div>

      {typeof activeId === "number" ? (
        <ImageUpload
          label={t("iconUploadLabel")}
          targetSize={IDEA_PICTURE_SIZE}
          previewUrl={
            draft.icon.startsWith("GFX_idea_mio_")
              ? miosApi.iconUrl(projectId, activeId)
              : null
          }
          hint={t.raw("iconUploadHint")}
          gfxName={draft.icon || undefined}
          baseModFallbackUrls={
            draft.icon && baseModDepKeys.length
              ? baseModDepKeys.map((dk) => gfxUrl(projectId, dk, draft.icon))
              : []
          }
          onUpload={(file, crop) =>
            miosApi.uploadIcon(projectId, activeId, file, crop)
          }
          onUploaded={(result) => {
            const icon = result["icon"];
            if (typeof icon === "string") update("icon", icon);
          }}
        />
      ) : (
        <p className="text-xs text-muted-foreground">
          {t("iconUploadSaveFirst")}
        </p>
      )}

      <Separator />

      <div className="space-y-2">
        <div className="space-y-1">
          <Label>{t("nameLabel")}</Label>
          <DisplayNameLocsTabs
            entityKey={`mio:${activeId}:display-name`}
            value={draft.display_name_locs}
            exportLanguages={exportLanguages}
            legacyFallback={{ en: draft.name_en, zh: draft.name_zh }}
            onChange={(next) => update("display_name_locs", next)}
            placeholder={t("namePlaceholder")}
            ariaLabel={t("nameLabel")}
          />
          <p className="text-xs text-muted-foreground">{t("nameHint")}</p>
        </div>
        <div className="space-y-1">
          <Label>{t("descLabel")}</Label>
          <DescriptionLocsTabs
            entityKey={`mio:${activeId}:description`}
            value={draft.description_locs}
            exportLanguages={exportLanguages}
            legacyFallback={{ en: draft.desc_en, zh: draft.desc_zh }}
            onChange={(next) => update("description_locs", next)}
            placeholder={t("descPlaceholder")}
            ariaLabel={t("descLabel")}
            rows={2}
          />
        </div>
      </div>

      <Separator />

      <details className="rounded-md border border-border bg-muted/30 px-3 py-2">
        <summary className="cursor-pointer text-xs font-semibold text-muted-foreground hover:text-foreground select-none">
          {t("conditionsHeading")}
          <span className="ml-2 text-xs font-normal">
            {t("conditionsNote")}
          </span>
        </summary>
        <div className="pt-3 space-y-3">
          {(["allowed", "available", "visible"] as const).map((field) => (
            <div key={field}>
              <Label>{t(CONDITION_LABEL_KEYS[field])}</Label>
              <textarea
                value={draft[field]}
                onChange={(e) => update(field, e.target.value)}
                placeholder={
                  field === "allowed"
                    ? "tag = USA"
                    : field === "available"
                      ? "FROM = { tag = USA }"
                      : "always = yes"
                }
                className="w-full font-mono text-xs bg-muted/30 border border-border rounded p-2 min-h-[60px] resize-y"
                rows={3}
              />
            </div>
          ))}
        </div>
      </details>

      <details className="rounded-md border border-border bg-muted/30 px-3 py-2">
        <summary className="cursor-pointer text-xs font-semibold text-muted-foreground hover:text-foreground select-none">
          {t("traitsHeading")}
          <span className="ml-2 text-xs font-normal">
            {t("traitsHeadingNote")}
          </span>
          {!traitsError && draft.traits.length > 0 && (
            <span className="ml-2 text-xs font-normal text-success">
              {t("traitsConfigured", { count: draft.traits.length })}
            </span>
          )}
          {traitsError && (
            <span className="ml-2 text-xs font-normal text-destructive">
              {t("jsonError")}
            </span>
          )}
        </summary>
        <div className="pt-3 space-y-2">
          <div className="flex items-center justify-end">
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={toggleTraitsView}
              className="h-7 px-2 text-xs"
            >
              {traitsView === "rich" ? t("switchToRaw") : t("switchToRich")}
            </Button>
          </div>
          {traitsView === "rich" ? (
            <MIOTraitsEditor
              traits={draft.traits}
              onChange={(next) => update("traits", next)}
              projectId={projectId}
              mioDbId={typeof activeId === "number" ? activeId : null}
            />
          ) : (
            <>
              <textarea
                value={traitsBuffer}
                onChange={(e) => handleTraitsChange(e.target.value)}
                placeholder={TRAITS_PLACEHOLDER}
                className={`w-full font-mono text-xs bg-muted/30 border rounded p-2 min-h-[180px] resize-y ${
                  traitsError ? "border-destructive" : "border-border"
                }`}
                rows={10}
              />
              {traitsError && (
                <p className="text-xs text-destructive mt-1">{traitsError}</p>
              )}
              {!traitsError && draft.traits.length > 0 && (
                <p className="text-xs text-muted-foreground mt-1">
                  {t("traitsSummary", {
                    total: draft.traits.length,
                    initial: draft.traits.filter((t) => t.kind === "initial")
                      .length,
                    add: draft.traits.filter((t) => t.kind === "add").length,
                  })}
                </p>
              )}
            </>
          )}
        </div>
      </details>

      <details className="rounded-md border border-border bg-muted/30 px-3 py-2">
        <summary className="cursor-pointer text-xs font-semibold text-muted-foreground hover:text-foreground select-none">
          {t("advancedFields")}
          <span className="ml-2 text-xs font-normal">
            {t("advancedFieldsNote")}
          </span>
          {rawExtrasError && (
            <span className="ml-2 text-xs font-normal text-destructive">
              {t("jsonError")}
            </span>
          )}
        </summary>
        <div className="pt-3">
          <textarea
            value={rawExtrasBuffer}
            onChange={(e) => handleRawExtrasChange(e.target.value)}
            placeholder={RAW_EXTRAS_PLACEHOLDER}
            className={`w-full font-mono text-xs bg-muted/30 border rounded p-2 min-h-[120px] resize-y ${
              rawExtrasError ? "border-destructive" : "border-border"
            }`}
            rows={6}
          />
          {rawExtrasError && (
            <p className="text-xs text-destructive mt-1">{rawExtrasError}</p>
          )}
        </div>
      </details>

      <Separator />

      <div className="sticky bottom-0 bg-background pt-1 pb-2">
        <div className="mb-2 empty:mb-0">
          <DialogSaveStatusBar
            saving={saving}
            saved={status === "saved"}
            error={error}
            savingLabel={t("saving")}
            savedLabel={t("saved")}
          />
        </div>
        <div className="flex items-center justify-end gap-2">
          {isExisting && (
            <Button
              size="sm"
              variant="ghost"
              onClick={handleDelete}
              disabled={saving}
              className="text-destructive hover:text-destructive"
            >
              {t("delete")}
            </Button>
          )}
          <Button
            size="sm"
            onClick={handleSave}
            disabled={
              saving ||
              !draft.mio_id.trim() ||
              !!traitsError ||
              !!rawExtrasError
            }
          >
            {saving ? t("saving") : isExisting ? t("save") : t("create")}
          </Button>
        </div>
      </div>
    </div>
  );

  return (
    <WorkbenchShell
      gameViewHref={gameViewHref(projectId, {
        panel: "industrial_organisation_list_window",
        tag: countryTag,
      })}
      listPanel={listPanel}
      editorPanel={
        activeId === null ? (
          <div className="flex h-full items-center justify-center p-6">
            {listLoadFailed && rows.length === 0 ? (
              <WorkbenchLoadError onRetry={() => void refresh()} />
            ) : rows.length === 0 ? (
              zeroHero
            ) : (
              <EditorEmptyState
                className="max-w-xs"
                icon={<Factory className="h-6 w-6 text-muted-foreground/60" />}
                title={tShell("noSelectionGeneric")}
                description={t("selectHint")}
              />
            )}
          </div>
        ) : (
          editorForm
        )
      }
      previewPanel={
        <div className="flex h-full items-center justify-center p-6 text-center text-xs text-muted-foreground">
          {tShell("previewUnavailableGeneric")}
        </div>
      }
    />
  );
}

export function MIOsDialog({ projectId, open, onOpenChange }: MIOsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent className="max-w-[1280px] w-[95vw] max-h-[88vh] flex flex-col">
        {open && <MIOsContent projectId={projectId} />}
      </DialogContent>
    </Dialog>
  );
}
