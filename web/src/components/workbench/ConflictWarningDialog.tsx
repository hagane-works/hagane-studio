"use client";

import { useTranslations } from "use-intl";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useVanillaTagName } from "@/lib/vanillaCountryTags";

interface ConflictWarningDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tag: string;
  onConfirm: () => void;
}

export function ConflictWarningDialog({
  open,
  onOpenChange,
  tag,
  onConfirm,
}: ConflictWarningDialogProps) {
  const t = useTranslations("conflictWarningDialog");

  const vanillaName = useVanillaTagName()(tag) ?? tag;

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="text-base flex items-center gap-2">
            <span aria-hidden>⚠</span>
            <span>
              <span className="font-mono">{tag}</span> {t("titleSuffix")}
            </span>
          </DialogTitle>
          <DialogDescription className="text-sm">
            {t("description", { tag, vanillaName })}
          </DialogDescription>
        </DialogHeader>
        <div className="rounded-md border border-warning/40 bg-warning/5 px-3 py-2 text-xs text-foreground/90 space-y-1">
          <p>
            <span className="font-semibold">{t("ifGoalLabel")}</span>{" "}
            {t("ifGoalText", { tag })}
          </p>
          <p>
            <span className="font-semibold">{t("ifCustomLabel")}</span>{" "}
            {t("ifCustomText")}
          </p>
        </div>
        <DialogFooter className="flex flex-row justify-end gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("cancel")}
          </Button>
          <Button onClick={onConfirm}>{t("confirm")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
