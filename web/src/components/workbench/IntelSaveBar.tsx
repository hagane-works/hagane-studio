"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";

export type IntelSaveStatus = "idle" | "success" | "error";

export function useIntelSave() {
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<IntelSaveStatus>("idle");
  const [error, setError] = useState<string | null>(null);
  const run = async (work: () => Promise<void>) => {
    setSaving(true);
    setStatus("idle");
    setError(null);
    try {
      await work();
      setStatus("success");
      setTimeout(() => setStatus("idle"), 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setStatus("error");
    } finally {
      setSaving(false);
    }
  };
  return { saving, status, error, run };
}

export function IntelSaveBar({
  saving,
  status,
  error,
  unsaved,
  onSave,
}: {
  saving: boolean;
  status: IntelSaveStatus;
  error: string | null;
  unsaved: boolean;
  onSave: () => void;
}) {
  const t = useTranslations("intelligenceAgencies");
  return (
    <>
      {(saving || status !== "idle" || unsaved) && (
        <div
          className={`
            -mx-4 px-4 py-1.5 text-xs font-medium flex items-center gap-2
            transition-[opacity,background-color,border-color] duration-300
            ${
              saving
                ? "bg-muted/80 text-muted-foreground border-y border-border"
                : status === "success"
                  ? "bg-success/10 text-success border-y border-success/20"
                  : status === "error"
                    ? "bg-destructive/10 text-destructive border-y border-destructive/20"
                    : unsaved
                      ? "bg-warning/10 text-warning border-y border-warning/20"
                      : ""
            }
          `}
        >
          <span>
            {saving
              ? t("saving")
              : status === "success"
                ? t("saved")
                : status === "error"
                  ? (error ?? t("saveFailed"))
                  : unsaved
                    ? t("unsaved")
                    : ""}
          </span>
        </div>
      )}
      <div className="flex items-center gap-2 pt-2 border-t border-border">
        <Button onClick={onSave} disabled={saving} size="sm">
          {t("saveButton")}
        </Button>
      </div>
    </>
  );
}
