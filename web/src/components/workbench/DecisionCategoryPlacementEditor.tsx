"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { decisionsApi, workbenchIconsApi } from "@/lib/api";
import type { DecisionCategory } from "@/types";
import { useContentLocaleStore } from "@/lib/store/contentLocaleStore";
import { resolveDisplayName } from "@/lib/i18n/localizedName";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: number;
  categories: DecisionCategory[];

  onSaved: () => void;
}

export function DecisionCategoryPlacementEditor({
  open,
  onOpenChange,
  projectId,
  categories,
  onSaved,
}: Props) {
  const t = useTranslations("decisionCategoryPlacementEditor");
  const locale = useContentLocaleStore((s) => s.locale);
  const [order, setOrder] = useState<DecisionCategory[]>([]);
  const [dragSrc, setDragSrc] = useState<number | null>(null);
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (!open) return;
    const sorted = [...categories].sort((a, b) => a.priority - b.priority);
    setOrder(sorted);
    setDirty(false);
  }, [open, categories]);

  const handleDragStart = (i: number) => {
    setDragSrc(i);
  };
  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };
  const handleDrop = (dropIdx: number) => {
    if (dragSrc === null || dragSrc === dropIdx) return;
    setOrder((prev) => {
      const next = [...prev];
      const [moved] = next.splice(dragSrc, 1);
      next.splice(dropIdx, 0, moved);
      return next;
    });
    setDragSrc(null);
    setDirty(true);
  };

  const handleReset = () => {
    const sorted = [...categories].sort((a, b) =>
      resolveDisplayName(
        locale,
        a.display_name_locs,
        a.name_en,
        a.name_zh,
        a.category_id,
      ).localeCompare(
        resolveDisplayName(
          locale,
          b.display_name_locs,
          b.name_en,
          b.name_zh,
          b.category_id,
        ),
        locale,
      ),
    );
    setOrder(sorted);
    setDirty(true);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const updates = order.map((c, i) => ({ id: c.id, priority: i * 10 }));

      let failed = 0;
      for (const u of updates) {
        const current = categories.find((c) => c.id === u.id);
        if (current && current.priority === u.priority) continue;
        try {
          await decisionsApi.updateCategory(projectId, u.id, {
            priority: u.priority,
          });
        } catch (err) {
          failed++;
          console.warn(
            `[placement editor] update failed for cat ${u.id}:`,
            err,
          );
        }
      }
      onSaved();
      if (failed > 0) {
        toast.error(t("saveDone", { failed }));
      }
      onOpenChange(false);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      disablePointerDismissal={dirty}
    >
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Visual Placement · Decision Categories</DialogTitle>
          <DialogDescription>{t("dragHint")}</DialogDescription>
        </DialogHeader>

        <div className="my-4 rounded-md border border-border bg-muted/30 p-3 overflow-x-auto">
          <p className="text-xs text-muted-foreground mb-2 font-mono">
            HOI4 Decisions Panel · sample preview · {order.length} categor
            {order.length === 1 ? "y" : "ies"}
          </p>
          {order.length === 0 ? (
            <p className="text-sm text-muted-foreground italic">
              No categories to arrange · create one in the sidebar first.
            </p>
          ) : (
            <div className="flex flex-row gap-2 items-stretch min-w-fit">
              {order.map((cat, i) => (
                <div
                  key={cat.id}
                  draggable
                  onDragStart={() => handleDragStart(i)}
                  onDragOver={handleDragOver}
                  onDrop={() => handleDrop(i)}
                  className={`shrink-0 w-32 rounded-md border bg-card px-2 py-2 cursor-grab active:cursor-grabbing transition-shadow ${
                    dragSrc === i
                      ? "opacity-50 border-accent"
                      : "border-border hover:border-accent/60"
                  }`}
                  title={`Drag to reorder · current priority ${i * 10} (preview)`}
                >
                  <div className="flex items-center justify-center h-12 mb-1.5">
                    {cat.icon ? (
                      <img
                        src={workbenchIconsApi.decisionIconUrl(
                          projectId,
                          cat.icon,
                        )}
                        alt=""
                        className="max-h-12 max-w-full object-contain"
                        onError={(e) => {
                          (e.target as HTMLImageElement).style.display = "none";
                        }}
                      />
                    ) : (
                      <span className="text-xs text-muted-foreground">⚪</span>
                    )}
                  </div>
                  <p className="text-xs font-mono truncate text-center">
                    {resolveDisplayName(
                      locale,
                      cat.display_name_locs,
                      cat.name_en,
                      cat.name_zh,
                      cat.category_id,
                    )}
                  </p>
                  <p className="text-micro text-muted-foreground text-center font-mono">
                    priority {i * 10}
                  </p>
                </div>
              ))}
            </div>
          )}
        </div>

        <DialogFooter className="gap-2">
          <Button
            variant="outline"
            type="button"
            onClick={handleReset}
            disabled={saving || order.length === 0}
          >
            Reset to Alphabetical
          </Button>
          <Button
            variant="outline"
            type="button"
            onClick={() => onOpenChange(false)}
            disabled={saving}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSave}
            disabled={saving || !dirty || order.length === 0}
          >
            {saving ? "Saving..." : "Save Layout"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
