"use client";

import { useCallback, useState } from "react";

export interface BatchGfxApplyState {
  selectedIds: Set<number>;

  pickerOpen: boolean;

  progress: {
    done: number;
    total: number;
    failed: number;
    gfxName: string;
  } | null;

  toggle: (
    id: number,
    opts?: { range?: boolean; sortedIds?: number[] },
  ) => void;

  clear: () => void;

  openPicker: () => void;

  closePicker: () => void;

  applyGfx: (gfxName: string) => Promise<void>;
}

export interface UseBatchGfxApplyArgs {
  applyOne: (id: number, gfxName: string) => Promise<void>;

  onComplete?: (succeeded: number, failed: number) => void;
}

export function useBatchGfxApply({
  applyOne,
  onComplete,
}: UseBatchGfxApplyArgs): BatchGfxApplyState {
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [pickerOpen, setPickerOpen] = useState(false);
  const [progress, setProgress] =
    useState<BatchGfxApplyState["progress"]>(null);
  const [lastClickedId, setLastClickedId] = useState<number | null>(null);

  const toggle: BatchGfxApplyState["toggle"] = useCallback(
    (id, opts) => {
      setSelectedIds((prev) => {
        const next = new Set(prev);
        if (opts?.range && lastClickedId !== null && opts.sortedIds) {
          const a = opts.sortedIds.indexOf(lastClickedId);
          const b = opts.sortedIds.indexOf(id);
          if (a >= 0 && b >= 0) {
            const [lo, hi] = a < b ? [a, b] : [b, a];
            for (let i = lo; i <= hi; i++) next.add(opts.sortedIds[i]);
            return next;
          }
        }

        if (next.has(id)) next.delete(id);
        else next.add(id);
        return next;
      });
      setLastClickedId(id);
    },
    [lastClickedId],
  );

  const clear = useCallback(() => {
    setSelectedIds(new Set());
    setLastClickedId(null);
  }, []);

  const openPicker = useCallback(() => setPickerOpen(true), []);
  const closePicker = useCallback(() => setPickerOpen(false), []);

  const applyGfx = useCallback(
    async (gfxName: string) => {
      const ids = Array.from(selectedIds);
      setPickerOpen(false);
      if (ids.length === 0) return;
      setProgress({ done: 0, total: ids.length, failed: 0, gfxName });
      let succeeded = 0;
      let failed = 0;
      for (const id of ids) {
        try {
          await applyOne(id, gfxName);
          succeeded++;
        } catch (err) {
          failed++;
          console.warn(`[useBatchGfxApply] applyOne failed for id=${id}:`, err);
        }
        setProgress({
          done: succeeded + failed,
          total: ids.length,
          failed,
          gfxName,
        });
      }
      onComplete?.(succeeded, failed);

      setTimeout(() => {
        setProgress(null);
        setSelectedIds(new Set());
        setLastClickedId(null);
      }, 2000);
    },
    [selectedIds, applyOne, onComplete],
  );

  return {
    selectedIds,
    pickerOpen,
    progress,
    toggle,
    clear,
    openPicker,
    closePicker,
    applyGfx,
  };
}
