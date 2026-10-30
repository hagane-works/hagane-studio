"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { projectsApi } from "@/lib/api";
import {
  StatesMap,
  useStatesGeometry,
} from "@/components/workbench/states-map";
import type { State } from "@/types";

const EMPTY_STATES: State[] = [];
const NOOP_SELECT = () => {};

export interface StageScopePickerProps {
  projectId: number;
  open: boolean;
  onOpenChange: (v: boolean) => void;

  value: number[] | null;

  onSaved?: (ids: number[] | null) => void;
}

export function StageScopePicker({
  projectId,
  open,
  onOpenChange,
  value,
  onSaved,
}: StageScopePickerProps) {
  const t = useTranslations("stageScope");

  const { geometry, available, loading } = useStatesGeometry(
    open ? projectId : 0,
  );

  const [selected, setSelected] = useState<Set<number>>(
    () => new Set(value ?? []),
  );
  const [saving, setSaving] = useState(false);

  const wasOpenRef = useRef(false);
  useEffect(() => {
    if (open && !wasOpenRef.current) {
      setSelected(new Set(value ?? []));
    }
    wasOpenRef.current = open;
  }, [open, value]);

  const toggle = useCallback((geoId: number) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(geoId)) next.delete(geoId);
      else next.add(geoId);
      return next;
    });
  }, []);

  const multiSelect = useMemo(
    () => ({ selectedGeoIds: selected, onToggleGeoId: toggle }),
    [selected, toggle],
  );

  const handleSave = async () => {
    if (saving) return;
    setSaving(true);
    try {
      const ids = Array.from(selected).sort((a, b) => a - b);
      await projectsApi.update(projectId, { stage_state_ids: ids });
      toast.success(t("saved"));
      onSaved?.(ids);
      onOpenChange(false);
    } catch (e) {
      toast.error(t("saveFailed", { msg: (e as Error).message }));
    } finally {
      setSaving(false);
    }
  };

  const showMap = !loading && available && geometry != null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="!max-w-5xl !w-[92vw] h-[82vh] flex flex-col p-0 gap-0">
        <DialogHeader className="shrink-0 px-4 pt-4 pb-3">
          <DialogTitle>{t("title")}</DialogTitle>
          <DialogDescription>{t("subtitle")}</DialogDescription>
          {showMap && (
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 pt-1">
              <span className="text-xs font-medium text-foreground">
                {t("selectedCount", { count: selected.size })}
              </span>
              <button
                type="button"
                onClick={() => setSelected(new Set())}
                disabled={selected.size === 0}
                className="text-xs text-accent hover:underline ui-interactive ui-focus-ring rounded disabled:opacity-50 disabled:no-underline"
              >
                {t("clear")}
              </button>
            </div>
          )}
        </DialogHeader>

        <div className="flex-1 min-h-0 flex flex-col border-t border-border">
          {loading ? (
            <div className="flex-1 grid place-items-center p-8">
              <p className="text-xs text-muted-foreground">{t("loading")}</p>
            </div>
          ) : showMap ? (
            <StatesMap
              projectId={projectId}
              states={EMPTY_STATES}
              selectedId={null}
              onSelectState={NOOP_SELECT}
              multiSelect={multiSelect}
            />
          ) : (
            <div className="flex-1 grid place-items-center p-8">
              <p className="max-w-sm text-center text-sm text-muted-foreground">
                {t("unavailable")}
              </p>
            </div>
          )}
        </div>

        <DialogFooter className="shrink-0 mx-0 mb-0">
          {showMap ? (
            <>
              <Button
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={saving}
              >
                {t("cancel")}
              </Button>
              <Button onClick={handleSave} disabled={saving}>
                {saving ? t("saving") : t("save")}
              </Button>
            </>
          ) : (
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              {t("close")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
