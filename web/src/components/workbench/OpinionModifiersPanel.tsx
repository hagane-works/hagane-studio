"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";

import { useConfirm } from "@/components/shared/ConfirmDialogProvider";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { opinionModifiersApi } from "@/lib/api/opinionModifiers";
import { useActiveCountry } from "@/lib/store/projectStore";
import type { OpinionModifier } from "@/types/opinionModifier";

import {
  EMPTY_OPINION_MODIFIER_DRAFT,
  opinionDraftToPayload,
  opinionModifierToDraft,
  type OpinionModifierDraft,
} from "./opinionModifierForm";

interface OpinionModifiersPanelProps {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onChanged?: () => void;
}

const NUMBER_FIELDS = [
  { key: "value", label: "opinionFieldValue", integer: false },
  { key: "decay", label: "opinionFieldDecay", integer: false },
  { key: "days", label: "opinionFieldDays", integer: true },
  { key: "months", label: "opinionFieldMonths", integer: true },
  { key: "years", label: "opinionFieldYears", integer: true },
  { key: "min_trust", label: "opinionFieldMinTrust", integer: false },
  { key: "max_trust", label: "opinionFieldMaxTrust", integer: false },
] as const;

const DOCUMENTED_BODY_FIELDS = new Set([
  "value",
  "decay",
  "days",
  "months",
  "years",
  "min_trust",
  "max_trust",
  "trade",
]);

export function OpinionModifiersPanel({
  projectId,
  open,
  onOpenChange,
  onChanged,
}: OpinionModifiersPanelProps) {
  const t = useTranslations("customTraitsPanel");
  const confirmDialog = useConfirm();
  const countryTag = useActiveCountry();
  const [rows, setRows] = useState<OpinionModifier[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [draft, setDraft] = useState<OpinionModifierDraft>({
    ...EMPTY_OPINION_MODIFIER_DRAFT,
    country_tag: countryTag,
  });
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      setRows(await opinionModifiersApi.list(projectId));
    } catch (error) {
      toast.error(
        t("opinionLoadFailed", {
          msg: error instanceof Error ? error.message : String(error),
        }),
      );
    }
  }, [projectId, t]);

  useEffect(() => {
    if (open) void load();
  }, [open, load]);

  const visibleRows = useMemo(() => {
    if (!countryTag) return rows;
    return rows.filter(
      (row) => !row.country_tag || row.country_tag === countryTag,
    );
  }, [countryTag, rows]);

  const selected = useMemo(
    () => rows.find((row) => row.id === selectedId) ?? null,
    [rows, selectedId],
  );

  const preservedFieldCount = useMemo(() => {
    if (!selected) return 0;
    const customScalars = Object.keys(selected.body ?? {}).filter(
      (key) => !DOCUMENTED_BODY_FIELDS.has(key),
    ).length;
    return customScalars + Object.keys(selected.raw_extras ?? {}).length;
  }, [selected]);

  const startNew = () => {
    setSelectedId(null);
    setDraft({
      ...EMPTY_OPINION_MODIFIER_DRAFT,
      country_tag: countryTag,
    });
  };

  const selectRow = (row: OpinionModifier) => {
    setSelectedId(row.id);
    setDraft(opinionModifierToDraft(row));
  };

  const updateDraft = <K extends keyof OpinionModifierDraft>(
    key: K,
    value: OpinionModifierDraft[K],
  ) => setDraft((previous) => ({ ...previous, [key]: value }));

  const save = async () => {
    setSaving(true);
    try {
      const payload = opinionDraftToPayload(draft);
      const saved =
        selectedId === null
          ? await opinionModifiersApi.create(projectId, payload)
          : await opinionModifiersApi.update(projectId, selectedId, payload);
      await load();
      setSelectedId(saved.id);
      setDraft(opinionModifierToDraft(saved));
      toast.success(
        selectedId === null
          ? t("opinionCreated", { modifierId: saved.modifier_id })
          : t("opinionSaved", { modifierId: saved.modifier_id }),
      );
      onChanged?.();
    } catch (error) {
      toast.error(
        t("opinionSaveFailed", {
          msg: error instanceof Error ? error.message : String(error),
        }),
      );
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (selectedId === null) return;
    if (
      !(await confirmDialog({
        body: t("opinionConfirmDelete", { modifierId: draft.modifier_id }),
        danger: true,
      }))
    )
      return;

    setSaving(true);
    try {
      await opinionModifiersApi.delete(projectId, selectedId);
      await load();
      startNew();
      onChanged?.();
    } catch (error) {
      toast.error(
        t("opinionDeleteFailed", {
          msg: error instanceof Error ? error.message : String(error),
        }),
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent className="max-w-5xl w-[95vw] sm:max-w-5xl p-0 gap-0">
        <DialogHeader className="p-4 pb-3 border-b">
          <DialogTitle>{t("opinionTitle")}</DialogTitle>
          <p className="text-xs text-muted-foreground">
            {t("opinionSubtitle")}
          </p>
        </DialogHeader>

        <div className="grid grid-cols-[260px_1fr] h-[70vh]">
          <aside className="border-r overflow-y-auto">
            <div className="p-3 border-b">
              <Button size="sm" className="w-full" onClick={startNew}>
                {t("opinionNew")}
              </Button>
            </div>
            <ul className="divide-y">
              {visibleRows.length === 0 && (
                <li className="p-3 text-xs text-muted-foreground">
                  {t("opinionEmpty")}
                </li>
              )}
              {visibleRows.map((row) => (
                <li key={row.id}>
                  <button
                    type="button"
                    onClick={() => selectRow(row)}
                    className={`w-full text-left px-3 py-2 transition-colors ${
                      selectedId === row.id
                        ? "bg-accent text-accent-foreground"
                        : "hover:bg-muted/50"
                    }`}
                  >
                    <span className="block text-xs font-mono truncate">
                      {row.modifier_id}
                    </span>
                    <span className="block text-micro text-muted-foreground">
                      {row.source === "imported"
                        ? t("opinionSourceImported")
                        : t("opinionSourceCreated")}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </aside>

          <div className="overflow-y-auto p-4 space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label htmlFor="opinion-modifier-id">
                  {t("opinionModifierId")}
                </Label>
                <Input
                  id="opinion-modifier-id"
                  value={draft.modifier_id}
                  onChange={(event) =>
                    updateDraft("modifier_id", event.target.value)
                  }
                  placeholder={t("opinionModifierIdPlaceholder")}
                  className="font-mono"
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="opinion-country-tag">
                  {t("opinionCountryTag")}
                </Label>
                <Input
                  id="opinion-country-tag"
                  value={draft.country_tag}
                  onChange={(event) =>
                    updateDraft("country_tag", event.target.value)
                  }
                  placeholder={countryTag || t("opinionCountryTagPlaceholder")}
                  className="font-mono uppercase"
                  maxLength={10}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              {NUMBER_FIELDS.map((field) => {
                const preservedValue = draft.preserved_known_fields[field.key];
                return (
                  <div className="space-y-1" key={field.key}>
                    <Label htmlFor={`opinion-${field.key}`}>
                      {t(field.label)}
                    </Label>
                    <Input
                      id={`opinion-${field.key}`}
                      type="number"
                      step={field.integer ? 1 : "any"}
                      value={draft[field.key]}
                      disabled={preservedValue !== undefined}
                      onChange={(event) =>
                        updateDraft(field.key, event.target.value)
                      }
                    />
                    {preservedValue !== undefined && (
                      <p className="text-micro text-muted-foreground">
                        {t("opinionPreservedKnownScalar", {
                          value: preservedValue,
                        })}
                      </p>
                    )}
                  </div>
                );
              })}
              <div className="space-y-1">
                <Label htmlFor="opinion-trade">{t("opinionFieldTrade")}</Label>
                <Select
                  id="opinion-trade"
                  value={draft.trade}
                  disabled={draft.preserved_known_fields.trade !== undefined}
                  onChange={(event) =>
                    updateDraft(
                      "trade",
                      event.target.value as OpinionModifierDraft["trade"],
                    )
                  }
                >
                  <option value="">{t("opinionTradeUnset")}</option>
                  <option value="yes">{t("opinionTradeYes")}</option>
                  <option value="no">{t("opinionTradeNo")}</option>
                </Select>
                {draft.preserved_known_fields.trade !== undefined && (
                  <p className="text-micro text-muted-foreground">
                    {t("opinionPreservedKnownScalar", {
                      value: draft.preserved_known_fields.trade,
                    })}
                  </p>
                )}
              </div>
            </div>

            <p className="text-xs text-muted-foreground">
              {t("opinionDurationHelp")}
            </p>

            {preservedFieldCount > 0 && (
              <p className="rounded-md border border-border bg-muted/30 p-3 text-xs text-muted-foreground">
                {t("opinionPreservedFields", { count: preservedFieldCount })}
              </p>
            )}

            <div className="flex items-center gap-2 pt-2 border-t">
              <Button onClick={save} disabled={saving}>
                {saving
                  ? t("opinionSaving")
                  : selectedId === null
                    ? t("opinionCreate")
                    : t("opinionSave")}
              </Button>
              {selectedId !== null && (
                <Button
                  variant="destructive"
                  onClick={remove}
                  disabled={saving}
                >
                  {t("opinionDelete")}
                </Button>
              )}
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
