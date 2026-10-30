"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DEFAULT_EXPORT_LANGUAGES } from "@/components/shared/editors/displayNameLocsHelpers";
import { FontField } from "./FontField";
import { useProjectStore } from "@/lib/store/projectStore";
import type { UiPanelControl, UiPanelDerivedControl } from "@/types";

const EMPTY_LOCS: Record<string, string> = {};
const TOKEN_FORMATS = ["0", "1", "2", "%"] as const;

export function TextSection({
  entityKey,
  ctrl,
  derived,
  onPatch,
}: {
  entityKey: string;
  ctrl: UiPanelControl;
  derived: UiPanelDerivedControl | undefined;
  onPatch: (partial: Partial<UiPanelControl>) => void;
}) {
  const t = useTranslations("uiPanelsInspector");

  const exportLanguages = useProjectStore(
    (s) => s.currentProject?.export_languages ?? DEFAULT_EXPORT_LANGUAGES,
  );
  const [varName, setVarName] = useState("");
  const [fmt, setFmt] = useState<(typeof TOKEN_FORMATS)[number]>("0");

  const handleLocsChange = (next: Record<string, string>) => {
    onPatch({
      text_locs: next,

      text: next.en !== undefined ? next.en : (ctrl.text ?? ""),
    });
  };

  const insertToken = () => {
    const name = varName.trim();
    if (!name) return;
    const token = `[?${name}|${fmt}]`;

    const base = ctrl.text_locs?.en || ctrl.text || "";
    const nextText = `${base}${token}`;
    onPatch({
      text: nextText,
      text_locs: { ...(ctrl.text_locs ?? {}), en: nextText },
    });
  };

  return (
    <div className="p-4 border-b border-border space-y-3">
      <span className="block text-micro uppercase tracking-widest font-semibold text-muted-foreground">
        {t("textTitle")}
      </span>
      <DescriptionLocsTabs
        entityKey={`${entityKey}:text`}
        value={ctrl.text_locs ?? EMPTY_LOCS}
        exportLanguages={exportLanguages}
        onChange={handleLocsChange}
        legacyFallback={{ en: ctrl.text }}
        inputComponent="textarea"
        rows={3}
        placeholder={t("textPlaceholder")}
        ariaLabel={t("textTitle")}
      />
      {derived && (
        <p className="text-micro text-muted-foreground font-mono break-all">
          {t("textLocKeyHint")} {derived.text_loc_key}
        </p>
      )}
      <FontField ctrl={ctrl} onPatch={onPatch} />

      <div className="space-y-1.5">
        <Label>{t("wizardTitle")}</Label>
        <div className="flex items-end gap-1.5">
          <div className="flex-1 space-y-1">
            <span className="block text-micro text-muted-foreground">
              {t("wizardVarLabel")}
            </span>
            <Input
              value={varName}
              placeholder="legitimacy"
              className="h-8 text-xs font-mono"
              onChange={(e) => setVarName(e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <span className="block text-micro text-muted-foreground">
              {t("wizardFormatLabel")}
            </span>
            <Select
              className="h-8 px-2 text-xs font-mono"
              value={fmt}
              onChange={(e) =>
                setFmt(e.target.value as (typeof TOKEN_FORMATS)[number])
              }
            >
              {TOKEN_FORMATS.map((f) => (
                <option key={f} value={f}>
                  {`|${f}`}
                </option>
              ))}
            </Select>
          </div>
          <Button
            size="sm"
            variant="outline"
            className="text-xs"
            disabled={varName.trim() === ""}
            onClick={insertToken}
          >
            {t("wizardInsert")}
          </Button>
        </div>
        {varName.trim() !== "" && (
          <p className="text-micro text-muted-foreground font-mono break-all">
            {`[?${varName.trim()}|${fmt}]`}
          </p>
        )}
        <p className="text-micro text-muted-foreground">{t("wizardHint")}</p>
      </div>
    </div>
  );
}
