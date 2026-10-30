"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export type SourceEditorChoice =
  "keep_imported" | "override_as_creator" | "create_new_copy";

export type SourceEntityType =
  "idea" | "character" | "event" | "decision" | "focus_node" | "tech";

interface SourceEditorDialogProps {
  open: boolean;
  entityType: SourceEntityType;
  entityId: string;
  entityLabel: string;

  defaultNewId?: string;
  onChoice: (choice: SourceEditorChoice, newId?: string) => void;
  onCancel: () => void;

  hideForkOption?: boolean;
}

export function SourceEditorDialog({
  open,
  entityType,
  entityId,
  entityLabel,
  defaultNewId,
  onChoice,
  onCancel,
  hideForkOption = false,
}: SourceEditorDialogProps) {
  const t = useTranslations("sourceEditorDialog");
  const [newId, setNewId] = useState(defaultNewId || `${entityId}_copy`);
  const [idError, setIdError] = useState("");

  const handleCreateCopy = () => {
    const trimmed = newId.trim();
    if (!trimmed) {
      setIdError("New id cannot be empty");
      return;
    }
    if (trimmed === entityId) {
      setIdError("New id must differ from the original");
      return;
    }
    setIdError("");
    onChoice("create_new_copy", trimmed);
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) onCancel();
      }}
      disablePointerDismissal
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {t("title", { noun: t(`noun.${entityType}`) })}
          </DialogTitle>
          <DialogDescription>
            {t.rich("desc", {
              label: entityLabel,
              mono: (chunks) => <span className="font-mono">{chunks}</span>,
            })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <button
            type="button"
            onClick={() => onChoice("keep_imported")}
            className="ui-interactive w-full text-left rounded-md border border-border p-3 hover:bg-muted/50 hover:border-border/80"
          >
            <div className="font-medium text-sm">{t("keepImportedTitle")}</div>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t("keepImportedDesc")}
            </p>
          </button>

          <button
            type="button"
            onClick={() => onChoice("override_as_creator")}
            className="ui-interactive w-full text-left rounded-md border border-border p-3 hover:bg-muted/50 hover:border-border/80"
          >
            <div className="font-medium text-sm">{t("overrideTitle")}</div>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t("overrideDesc")}
            </p>
          </button>

          {!hideForkOption && (
            <div className="rounded-md border border-border p-3 space-y-2">
              <div>
                <div className="font-medium text-sm">
                  {t("createCopyTitle")}
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {t("createCopyDesc", { entityType })}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Label htmlFor="source-dialog-new-id" className="sr-only">
                  New id
                </Label>
                <Input
                  id="source-dialog-new-id"
                  value={newId}
                  onChange={(e) => {
                    setNewId(e.target.value);
                    if (idError) setIdError("");
                  }}
                  placeholder="new_idea_id"
                  className="text-xs h-8 font-mono"
                />
                <Button
                  type="button"
                  size="sm"
                  onClick={handleCreateCopy}
                  className="shrink-0"
                >
                  Fork
                </Button>
              </div>
              {idError && <p className="text-xs text-destructive">{idError}</p>}
            </div>
          )}
        </div>

        <div className="flex justify-end">
          <Button type="button" variant="ghost" size="sm" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
