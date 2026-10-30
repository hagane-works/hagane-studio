"use client";

import { useTranslations } from "use-intl";

import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import type { UiPanelControl } from "@/types";

const FONTS = [
  { value: "hoi_16mbs", label: "fontSmall" },
  { value: "hoi_18mbs", label: "fontBody" },
  { value: "hoi_20b", label: "fontBold" },
  { value: "hoi_24header", label: "fontHeader" },
  { value: "hoi_30header", label: "fontHeaderLarge" },
  { value: "hoi_36header", label: "fontHeaderXL" },
] as const;
const DEFAULT_FONT = "hoi_18mbs";
const FORMATS = ["left", "center", "right"] as const;

export function FontField({
  ctrl,
  onPatch,
}: {
  ctrl: UiPanelControl;
  onPatch: (partial: Partial<UiPanelControl>) => void;
}) {
  const t = useTranslations("uiPanelsInspector");
  const font = ctrl.font || DEFAULT_FONT;

  const known = FONTS.some((option) => option.value === font);
  return (
    <div className="flex items-end gap-2">
      <div className="flex-1 space-y-1">
        <Label>{t("fontLabel")}</Label>
        <Select
          aria-label={t("fontLabel")}
          className="h-8 px-2 text-xs"
          value={font}
          onChange={(e) =>
            onPatch({
              font:
                e.target.value === DEFAULT_FONT ? undefined : e.target.value,
            })
          }
        >
          {!known && <option value={font}>{font}</option>}
          {FONTS.map((option) => (
            <option key={option.value} value={option.value}>
              {`${t(option.label)} · ${option.value}`}
            </option>
          ))}
        </Select>
      </div>
      {ctrl.kind === "text" && (
        <div className="space-y-1">
          <Label>{t("formatLabel")}</Label>
          <Select
            aria-label={t("formatLabel")}
            className="h-8 px-2 text-xs"
            value={ctrl.format ?? "left"}
            onChange={(e) =>
              onPatch({ format: e.target.value as (typeof FORMATS)[number] })
            }
          >
            {FORMATS.map((format) => (
              <option key={format} value={format}>
                {t(`format_${format}`)}
              </option>
            ))}
          </Select>
        </div>
      )}
    </div>
  );
}
