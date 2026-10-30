"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { IntelAgencyUpgrade } from "@/lib/api";
import { useRegisterDirty } from "@/lib/hooks/useUnsavedGuard";
import { IntelSaveBar, useIntelSave } from "./IntelSaveBar";

export function levelsFromText(raw: string): string[] {
  return raw
    .split(/\n\s*\n/)
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

export function IntelUpgradeFields({
  upgrade,
  onChange,
  onRemove,
}: {
  upgrade: IntelAgencyUpgrade;
  onChange: (next: IntelAgencyUpgrade) => void;

  onRemove?: () => void;
}) {
  const t = useTranslations("intelligenceAgencies");
  return (
    <div className="rounded border border-border/60 bg-muted/30 p-2 space-y-1.5">
      <div
        className={`grid ${onRemove ? "grid-cols-[1fr_1fr_auto]" : "grid-cols-2"} gap-2`}
      >
        <Input
          placeholder="upgrade_economy_civilian"
          className="font-mono text-xs"
          value={upgrade.token || ""}
          onChange={(e) => onChange({ ...upgrade, token: e.target.value })}
        />
        <Input
          placeholder="GFX_agency_economy_department"
          className="font-mono text-xs"
          value={upgrade.picture || ""}
          onChange={(e) => onChange({ ...upgrade, picture: e.target.value })}
        />
        {onRemove && (
          <Button
            size="sm"
            variant="ghost"
            className="px-2 text-muted-foreground hover:text-destructive"
            onClick={onRemove}
            aria-label={t("removeUpgradeAria")}
            title={t("removeUpgradeAria")}
          >
            ×
          </Button>
        )}
      </div>
      <Textarea
        className="font-mono text-xs min-h-16"
        placeholder={"modifier = {\n\tcivilian_intel_factor = 0.25\n}"}
        value={(upgrade.levels || []).join("\n\n")}
        onChange={(e) =>
          onChange({ ...upgrade, levels: levelsFromText(e.target.value) })
        }
      />
      <p className="text-micro text-muted-foreground">{t("levelsHint")}</p>
    </div>
  );
}

export function IntelUpgradeEditor({
  upgrade,
  onSave,
}: {
  upgrade: IntelAgencyUpgrade;

  onSave: (next: IntelAgencyUpgrade) => Promise<void>;
}) {
  const [draft, setDraft] = useState<IntelAgencyUpgrade>(upgrade);
  const [savedDraft, setSavedDraft] = useState<IntelAgencyUpgrade>(upgrade);
  const { saving, status, error, run } = useIntelSave();
  const unsaved = JSON.stringify(draft) !== JSON.stringify(savedDraft);
  useRegisterDirty("intelligence", unsaved);
  const save = () =>
    run(async () => {
      await onSave(draft);
      setSavedDraft(draft);
    });
  return (
    <div className="p-4 max-w-3xl space-y-4">
      <IntelUpgradeFields upgrade={draft} onChange={setDraft} />
      <IntelSaveBar
        saving={saving}
        status={status}
        error={error}
        unsaved={unsaved}
        onSave={() => void save()}
      />
    </div>
  );
}
