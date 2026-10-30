"use client";

import { useCallback, memo } from "react";
import { useBaseModDepKeys, gfxUrl } from "@/lib/api/basemod";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { ImageUpload } from "@/shared/upload/ImageUpload";
import { miosApi, type MIOTraitEntry } from "@/lib/api";

const TRAIT_ICON_SIZE = { width: 64, height: 64 } as const;

const SCRIPT_FIELDS = [
  "available",
  "visible",
  "on_complete",
  "ai_will_do",
] as const;
type ScriptField = (typeof SCRIPT_FIELDS)[number];

const MODIFIER_MAPS = [
  "equipment_bonus",
  "production_bonus",
  "organization_modifier",
] as const;
type ModifierMap = (typeof MODIFIER_MAPS)[number];

const SCRIPT_FIELD_LABEL_KEYS: Record<ScriptField, string> = {
  available: "fieldAvailable",
  visible: "fieldVisible",
  on_complete: "fieldOnComplete",
  ai_will_do: "fieldAiWillDo",
};

const MODIFIER_MAP_LABEL_KEYS: Record<ModifierMap, string> = {
  equipment_bonus: "modMapEquipmentBonus",
  production_bonus: "modMapProductionBonus",
  organization_modifier: "modMapOrganizationModifier",
};

const MANAGED_ICON_PREFIX = "GFX_idea_mio_trait_";

export function omitKey<T extends Record<string, unknown>>(
  obj: T,
  key: string,
): T {
  if (!(key in obj)) return obj;
  const next = { ...obj };
  delete next[key];
  return next;
}

function readPos(trait: MIOTraitEntry, axis: "x" | "y"): string {
  const p = trait.position;
  if (!p) return "";
  const v = p[axis];
  return typeof v === "number" && !Number.isNaN(v) ? String(v) : "";
}

function parentsToText(trait: MIOTraitEntry): string {
  const list = trait.all_parents;
  return Array.isArray(list) ? list.join(", ") : "";
}

export function textToParents(text: string): string[] {
  return text
    .split(/[\s,]+/)
    .map((s) => s.trim())
    .filter(Boolean);
}

export interface MIOTraitsEditorProps {
  traits: MIOTraitEntry[];
  onChange: (next: MIOTraitEntry[]) => void;
  projectId: number;

  mioDbId: number | null;
}

export function MIOTraitsEditor({
  traits,
  onChange,
  projectId,
  mioDbId,
}: MIOTraitsEditorProps) {
  const t = useTranslations("mIOsDialog");

  const patchAt = useCallback(
    (index: number, patch: (prev: MIOTraitEntry) => MIOTraitEntry) => {
      onChange(traits.map((tr, i) => (i === index ? patch({ ...tr }) : tr)));
    },
    [traits, onChange],
  );

  const removeAt = useCallback(
    (index: number) => {
      onChange(traits.filter((_, i) => i !== index));
    },
    [traits, onChange],
  );

  const moveAt = useCallback(
    (index: number, dir: -1 | 1) => {
      const target = index + dir;
      if (target < 0 || target >= traits.length) return;
      const next = [...traits];
      const [moved] = next.splice(index, 1);
      next.splice(target, 0, moved);
      onChange(next);
    },
    [traits, onChange],
  );

  const addTrait = useCallback(() => {
    const kind: MIOTraitEntry["kind"] = traits.length === 0 ? "initial" : "add";
    onChange([...traits, { kind, token: "" }]);
  }, [traits, onChange]);

  return (
    <div className="space-y-3">
      <div className="space-y-2">
        {traits.length === 0 && (
          <p className="text-xs text-muted-foreground">{t("traitsEmpty")}</p>
        )}
        {traits.map((trait, index) => (
          <MIOTraitCard
            key={`${trait.token || ""}-${index}`}
            index={index}
            trait={trait}
            isFirst={index === 0}
            isLast={index === traits.length - 1}
            projectId={projectId}
            mioDbId={mioDbId}
            patchAt={patchAt}
            removeAt={removeAt}
            moveAt={moveAt}
          />
        ))}
      </div>
      <Button size="sm" variant="outline" onClick={addTrait}>
        {t("addTrait")}
      </Button>
    </div>
  );
}

export interface MIOTraitCardProps {
  index: number;
  trait: MIOTraitEntry;
  isFirst?: boolean;
  isLast?: boolean;
  projectId: number;
  mioDbId: number | null;
  patchAt: (
    index: number,
    patch: (prev: MIOTraitEntry) => MIOTraitEntry,
  ) => void;

  removeAt?: (index: number) => void;

  moveAt?: (index: number, dir: -1 | 1) => void;
}

export const MIOTraitCard = memo(function MIOTraitCard({
  index,
  trait,
  isFirst = false,
  isLast = false,
  projectId,
  mioDbId,
  patchAt,
  removeAt,
  moveAt,
}: MIOTraitCardProps) {
  const t = useTranslations("mIOsDialog");

  const baseModDepKeys = useBaseModDepKeys();

  const setField = useCallback(
    <K extends keyof MIOTraitEntry>(key: K, value: MIOTraitEntry[K]) => {
      patchAt(index, (prev) => ({ ...prev, [key]: value }));
    },
    [patchAt, index],
  );

  const setKind = useCallback(
    (kind: "initial" | "add" | "override") => setField("kind", kind),
    [setField],
  );

  const setToken = useCallback(
    (token: string) => setField("token", token),
    [setField],
  );

  const setName = useCallback(
    (name: string) => setField("name", name),
    [setField],
  );

  const setPos = useCallback(
    (axis: "x" | "y", raw: string) => {
      patchAt(index, (prev) => {
        const other = axis === "x" ? "y" : "x";
        const otherVal = prev.position ? prev.position[other] : 0;
        if (raw.trim() === "") {
          if (
            !prev.position ||
            (typeof otherVal === "number" && otherVal === 0)
          ) {
            return omitKey(prev, "position");
          }
        }
        const num = Number(raw);
        const value = Number.isNaN(num) ? 0 : num;
        const nextPos = {
          x:
            axis === "x"
              ? value
              : typeof prev.position?.x === "number"
                ? prev.position.x
                : 0,
          y:
            axis === "y"
              ? value
              : typeof prev.position?.y === "number"
                ? prev.position.y
                : 0,
        };
        return { ...prev, position: nextPos };
      });
    },
    [patchAt, index],
  );

  const setParents = useCallback(
    (text: string) => {
      patchAt(index, (prev) => {
        const list = textToParents(text);
        if (list.length === 0) {
          return omitKey(prev, "all_parents");
        }
        return { ...prev, all_parents: list };
      });
    },
    [patchAt, index],
  );

  const setScript = useCallback(
    (field: ScriptField, value: string) => {
      patchAt(index, (prev) => {
        if (value.trim() === "") {
          const next = { ...prev };
          delete next[field];
          return next;
        }
        return { ...prev, [field]: value };
      });
    },
    [patchAt, index],
  );

  const setSpecialBackground = useCallback(
    (checked: boolean) => {
      patchAt(index, (prev) => {
        if (!checked) {
          return omitKey(prev, "special_trait_background");
        }
        return { ...prev, special_trait_background: true };
      });
    },
    [patchAt, index],
  );

  const setModifierEntry = useCallback(
    (mapKey: ModifierMap, oldKey: string, newKey: string, value: number) => {
      patchAt(index, (prev) => {
        const map = { ...(prev[mapKey] ?? {}) } as Record<string, number>;
        if (oldKey !== newKey) delete map[oldKey];
        map[newKey] = value;
        return { ...prev, [mapKey]: map };
      });
    },
    [patchAt, index],
  );

  const removeModifierEntry = useCallback(
    (mapKey: ModifierMap, key: string) => {
      patchAt(index, (prev) => {
        const map = { ...(prev[mapKey] ?? {}) } as Record<string, number>;
        delete map[key];
        if (Object.keys(map).length === 0) {
          return omitKey(prev, mapKey);
        }
        return { ...prev, [mapKey]: map };
      });
    },
    [patchAt, index],
  );

  const addModifierEntry = useCallback(
    (mapKey: ModifierMap) => {
      patchAt(index, (prev) => {
        const map = { ...(prev[mapKey] ?? {}) } as Record<string, number>;
        if ("" in map) return prev;
        map[""] = 0;
        return { ...prev, [mapKey]: map };
      });
    },
    [patchAt, index],
  );

  const setRawExtras = useCallback(
    (value: Record<string, unknown> | undefined) => {
      patchAt(index, (prev) => {
        if (!value || Object.keys(value).length === 0) {
          return omitKey(prev, "raw_extras");
        }
        return { ...prev, raw_extras: value };
      });
    },
    [patchAt, index],
  );

  const onRemove = useCallback(() => removeAt?.(index), [removeAt, index]);
  const onMoveUp = useCallback(() => moveAt?.(index, -1), [moveAt, index]);
  const onMoveDown = useCallback(() => moveAt?.(index, 1), [moveAt, index]);

  const token = (trait.token ?? "").trim();
  const iconManaged = (trait.icon ?? "").startsWith(MANAGED_ICON_PREFIX);
  const canUploadIcon = mioDbId != null && token !== "";

  return (
    <div className="rounded-md border border-border bg-card p-3 space-y-3">
      <div className="flex items-center gap-2">
        <span className="text-micro font-mono text-muted-foreground">
          #{index + 1}
        </span>

        <Select
          value={trait.kind ?? "add"}
          onChange={(e) =>
            setKind(e.target.value as "initial" | "add" | "override")
          }
          aria-label={t("traitKindLabel")}
          className="h-8 w-auto rounded px-2 text-xs font-mono"
          title={t("traitKindHint")}
        >
          <option value="initial">{t("traitKindInitial")}</option>
          <option value="add">{t("traitKindAdd")}</option>
          <option value="override">{t("traitKindOverride")}</option>
        </Select>
        <div className="flex-1" />
        {moveAt && (
          <>
            <Button
              size="sm"
              variant="ghost"
              onClick={onMoveUp}
              disabled={isFirst}
              aria-label={t("moveUp")}
              className="h-7 px-2 text-xs"
            >
              ↑
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={onMoveDown}
              disabled={isLast}
              aria-label={t("moveDown")}
              className="h-7 px-2 text-xs"
            >
              ↓
            </Button>
          </>
        )}
        {removeAt && (
          <Button
            size="sm"
            variant="ghost"
            onClick={onRemove}
            aria-label={t("removeTrait")}
            className="h-7 px-2 text-xs text-destructive hover:text-destructive"
          >
            {t("removeTrait")}
          </Button>
        )}
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-1">
          <Label>{t("traitTokenLabel")} ★</Label>
          <Input
            value={trait.token ?? ""}
            onChange={(e) => setToken(e.target.value)}
            placeholder="USA_mio_trait_strict_design"
            className="font-mono text-xs"
          />
        </div>
        <div className="space-y-1">
          <Label>{t("traitNameLabel")}</Label>
          <Input
            value={trait.name ?? ""}
            onChange={(e) => setName(e.target.value)}
            placeholder={t("traitNamePlaceholder")}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-1">
          <Label>{t("traitPosX")}</Label>
          <Input
            type="number"
            value={readPos(trait, "x")}
            onChange={(e) => setPos("x", e.target.value)}
            placeholder="0"
            className="font-mono text-xs"
          />
        </div>
        <div className="space-y-1">
          <Label>{t("traitPosY")}</Label>
          <Input
            type="number"
            value={readPos(trait, "y")}
            onChange={(e) => setPos("y", e.target.value)}
            placeholder="0"
            className="font-mono text-xs"
          />
        </div>
      </div>

      <div className="space-y-1">
        <Label>{t("traitAllParents")}</Label>
        <Input
          value={parentsToText(trait)}
          onChange={(e) => setParents(e.target.value)}
          placeholder="token_a, token_b"
          className="font-mono text-xs"
        />
        <p className="text-micro text-muted-foreground mt-0.5">
          {t("traitAllParentsHint")}
        </p>
      </div>

      <div className="space-y-1">
        <Label>{t("traitIconLabel")}</Label>
        {canUploadIcon ? (
          <ImageUpload
            label={t("traitIconLabel")}
            targetSize={TRAIT_ICON_SIZE}
            previewUrl={
              iconManaged
                ? miosApi.traitIconUrl(projectId, mioDbId as number, token)
                : null
            }
            hint={t.raw("traitIconHint")}
            gfxName={trait.icon || undefined}
            baseModFallbackUrls={
              trait.icon && baseModDepKeys.length
                ? baseModDepKeys.map((dk) =>
                    gfxUrl(projectId, dk, trait.icon as string),
                  )
                : []
            }
            onUpload={(file, crop) =>
              miosApi.uploadTraitIcon(
                projectId,
                mioDbId as number,
                token,
                file,
                crop,
              )
            }
            onUploaded={(result) => {
              const icon = result["icon"];
              if (typeof icon === "string") setField("icon", icon);
            }}
          />
        ) : (
          <div className="space-y-1">
            <Input
              value={trait.icon ?? ""}
              onChange={(e) => setField("icon", e.target.value)}
              placeholder="GFX_generic_mio_trait_icon_reliability"
              className="font-mono text-xs"
            />
            <p className="text-micro text-muted-foreground">
              {mioDbId == null
                ? t("traitIconSaveFirst")
                : t("traitIconSetToken")}
            </p>
          </div>
        )}
      </div>

      {MODIFIER_MAPS.map((mapKey) => (
        <ModifierMapEditor
          key={mapKey}
          mapKey={mapKey}
          map={
            (trait[mapKey] ?? undefined) as Record<string, number> | undefined
          }
          onSetEntry={setModifierEntry}
          onRemoveEntry={removeModifierEntry}
          onAddEntry={addModifierEntry}
        />
      ))}

      <details className="rounded border border-border bg-muted/20 px-2 py-1.5">
        <summary className="cursor-pointer text-xs font-semibold text-muted-foreground hover:text-foreground select-none">
          {t("traitScriptHeading")}
        </summary>
        <div className="pt-2 space-y-2">
          {SCRIPT_FIELDS.map((field) => (
            <div key={field} className="space-y-1">
              <Label>{t(SCRIPT_FIELD_LABEL_KEYS[field])}</Label>
              <textarea
                value={(trait[field] as string | undefined) ?? ""}
                onChange={(e) => setScript(field, e.target.value)}
                placeholder={
                  field === "ai_will_do" ? "factor = 1" : "always = yes"
                }
                className="w-full font-mono text-xs bg-muted/30 border border-border rounded p-2 min-h-[48px] resize-y"
                rows={2}
              />
            </div>
          ))}
          <label className="flex items-center gap-2 text-xs text-foreground select-none">
            <input
              type="checkbox"
              checked={trait.special_trait_background === true}
              onChange={(e) => setSpecialBackground(e.target.checked)}
            />
            {t("traitSpecialBackground")}
          </label>
        </div>
      </details>

      <TraitRawExtras
        value={
          (trait.raw_extras ?? undefined) as Record<string, unknown> | undefined
        }
        onChange={setRawExtras}
      />
    </div>
  );
});

interface ModifierMapEditorProps {
  mapKey: ModifierMap;
  map: Record<string, number> | undefined;
  onSetEntry: (
    mapKey: ModifierMap,
    oldKey: string,
    newKey: string,
    value: number,
  ) => void;
  onRemoveEntry: (mapKey: ModifierMap, key: string) => void;
  onAddEntry: (mapKey: ModifierMap) => void;
}

const ModifierMapEditor = memo(function ModifierMapEditor({
  mapKey,
  map,
  onSetEntry,
  onRemoveEntry,
  onAddEntry,
}: ModifierMapEditorProps) {
  const t = useTranslations("mIOsDialog");
  const entries = map ? Object.entries(map) : [];

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between">
        <Label className="font-mono">
          {t(MODIFIER_MAP_LABEL_KEYS[mapKey])}
        </Label>
        <Button
          size="sm"
          variant="ghost"
          onClick={() => onAddEntry(mapKey)}
          className="h-6 px-2 text-micro"
        >
          {t("addModifier")}
        </Button>
      </div>
      {entries.length === 0 ? (
        <p className="text-micro text-muted-foreground">{t("noModifiers")}</p>
      ) : (
        <div className="space-y-1">
          {entries.map(([k, v], i) => (
            <div key={`${mapKey}-${i}`} className="flex items-center gap-1.5">
              <Input
                value={k}
                onChange={(e) => onSetEntry(mapKey, k, e.target.value, v)}
                placeholder="reliability"
                className="font-mono text-xs flex-1"
              />
              <Input
                type="number"
                step="any"
                value={String(v)}
                onChange={(e) => {
                  const num = Number(e.target.value);
                  onSetEntry(mapKey, k, k, Number.isNaN(num) ? 0 : num);
                }}
                placeholder="0.1"
                className="font-mono text-xs w-24"
              />
              <Button
                size="sm"
                variant="ghost"
                onClick={() => onRemoveEntry(mapKey, k)}
                aria-label={t("removeModifier")}
                className="h-7 px-2 text-xs text-destructive hover:text-destructive shrink-0"
              >
                ✕
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
});

interface TraitRawExtrasProps {
  value: Record<string, unknown> | undefined;
  onChange: (next: Record<string, unknown> | undefined) => void;
}

const TraitRawExtras = memo(function TraitRawExtras({
  value,
  onChange,
}: TraitRawExtrasProps) {
  const t = useTranslations("mIOsDialog");

  const text =
    value && Object.keys(value).length > 0
      ? JSON.stringify(value, null, 2)
      : "";

  const handleChange = useCallback(
    (raw: string) => {
      const trimmed = raw.trim();
      if (!trimmed) {
        onChange(undefined);
        return;
      }
      try {
        const parsed = JSON.parse(trimmed);
        if (
          typeof parsed === "object" &&
          parsed !== null &&
          !Array.isArray(parsed)
        ) {
          onChange(parsed as Record<string, unknown>);
        }
      } catch {}
    },
    [onChange],
  );

  return (
    <details className="rounded border border-border bg-muted/20 px-2 py-1.5">
      <summary className="cursor-pointer text-xs font-semibold text-muted-foreground hover:text-foreground select-none">
        {t("traitRawExtrasHeading")}
        <span className="ml-2 text-micro font-normal">
          {t("traitRawExtrasNote")}
        </span>
      </summary>
      <div className="pt-2">
        <textarea
          key={text}
          defaultValue={text}
          onChange={(e) => handleChange(e.target.value)}
          placeholder={'{ "scaling_factor": "..." }'}
          className="w-full font-mono text-xs bg-muted/30 border border-border rounded p-2 min-h-[64px] resize-y"
          rows={3}
        />
      </div>
    </details>
  );
});
