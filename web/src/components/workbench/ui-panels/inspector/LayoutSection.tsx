"use client";

import { useTranslations } from "use-intl";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { UiPanelControl } from "@/types";

function parseIntOrZero(raw: string): number {
  const n = Number.parseInt(raw, 10);
  return Number.isFinite(n) ? n : 0;
}

function parseIntOrNull(raw: string): number | null {
  if (raw.trim() === "") return null;
  return parseIntOrZero(raw);
}

export function LayoutSection({
  ctrl,
  onPatch,
  finalizeName,
}: {
  ctrl: UiPanelControl;
  onPatch: (partial: Partial<UiPanelControl>) => void;

  finalizeName: (raw: string) => string;
}) {
  const t = useTranslations("uiPanelsInspector");
  return (
    <div className="p-4 border-b border-border space-y-3">
      <span className="block text-micro uppercase tracking-widest font-semibold text-muted-foreground">
        {t("layoutTitle")}
      </span>
      <div className="grid grid-cols-4 gap-2">
        {(["x", "y", "w", "h"] as const).map((k) => (
          <div key={k} className="space-y-1">
            <Label className="text-micro font-mono text-muted-foreground">
              {k}
            </Label>
            <Input
              type="number"
              value={String(ctrl[k] ?? "")}
              placeholder={k}
              className="h-8 text-xs font-mono"
              onChange={(e) =>
                onPatch(
                  k === "x" || k === "y"
                    ? { [k]: parseIntOrZero(e.target.value) }
                    : { [k]: parseIntOrNull(e.target.value) },
                )
              }
            />
          </div>
        ))}
      </div>
      <div className="space-y-1">
        <Label>{t("nameLabel")}</Label>
        <Input
          value={ctrl.name}
          className="h-8 text-xs font-mono"
          onChange={(e) => onPatch({ name: e.target.value })}
          onBlur={() => {
            const fixed = finalizeName(ctrl.name);
            if (fixed !== ctrl.name) onPatch({ name: fixed });
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter") e.currentTarget.blur();
          }}
        />
        <p className="text-micro text-muted-foreground">{t("nameHint")}</p>
      </div>
      {(ctrl.kind === "icon" ||
        ctrl.kind === "progressbar" ||
        ctrl.kind === "button" ||
        ctrl.kind === "container") && (
        <div className="space-y-1">
          <Label>{t("spriteLabel")}</Label>
          <Input
            value={ctrl.sprite ?? ""}
            placeholder={ctrl.kind === "button" ? "GFX_button_261x34" : "GFX_*"}
            className="h-8 text-xs font-mono"
            onChange={(e) => onPatch({ sprite: e.target.value })}
          />
          <p className="text-micro text-muted-foreground">
            {ctrl.kind === "button"
              ? t("spriteHintButton")
              : ctrl.kind === "container"
                ? t("spriteHintContainer")
                : t("spriteHint")}
          </p>
        </div>
      )}

      {ctrl.kind === "container" && (
        <label className="flex items-start gap-2 ui-interactive">
          <input
            type="checkbox"
            checked={ctrl.scroll === true}
            onChange={(e) => onPatch({ scroll: e.target.checked || undefined })}
            className="mt-0.5 h-4 w-4 rounded border-border accent-accent ui-focus-ring"
          />
          <span className="min-w-0 space-y-0.5">
            <span className="block text-xs text-foreground">
              {t("scrollLabel")}
            </span>
            <span className="block text-micro text-muted-foreground">
              {t("scrollHint")}
            </span>
          </span>
        </label>
      )}

      {ctrl.kind === "container" && (
        <div className="space-y-2">
          <div className="space-y-1">
            <Label>{t("listArrayLabel")}</Label>
            <Input
              value={ctrl.array ?? ""}
              placeholder="my_items"
              className="h-8 text-xs font-mono"
              onChange={(e) =>
                onPatch({ array: e.target.value.trim() || undefined })
              }
            />
            <p className="text-micro text-muted-foreground">
              {t("listArrayHint")}
            </p>
          </div>
          {ctrl.array && (
            <>
              <div className="grid grid-cols-3 gap-2">
                {(
                  [
                    ["columns", "listColumns"],
                    ["slot_w", "listSlotW"],
                    ["slot_h", "listSlotH"],
                  ] as const
                ).map(([field, label]) => (
                  <div key={field} className="space-y-1">
                    <Label className="text-micro text-muted-foreground">
                      {t(label)}
                    </Label>
                    <Input
                      type="number"
                      min={1}
                      value={String(ctrl[field] ?? "")}
                      className="h-8 text-xs font-mono"
                      onChange={(e) =>
                        onPatch({
                          [field]: parseIntOrNull(e.target.value) ?? undefined,
                        })
                      }
                    />
                  </div>
                ))}
              </div>
              <label className="flex items-start gap-2 ui-interactive">
                <input
                  type="checkbox"
                  checked={ctrl.change_scope === true}
                  onChange={(e) =>
                    onPatch({ change_scope: e.target.checked || undefined })
                  }
                  className="mt-0.5 h-4 w-4 rounded border-border accent-accent ui-focus-ring"
                />
                <span className="min-w-0 space-y-0.5">
                  <span className="block text-xs text-foreground">
                    {t("listScopeLabel")}
                  </span>
                  <span className="block text-micro text-muted-foreground">
                    {t("listScopeHint")}
                  </span>
                </span>
              </label>
            </>
          )}
        </div>
      )}
    </div>
  );
}
