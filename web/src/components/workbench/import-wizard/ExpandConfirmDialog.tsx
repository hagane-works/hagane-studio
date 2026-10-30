import type { Dispatch, SetStateAction } from "react";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";

export interface ExpandConfirmDialogProps {
  pendingAddTag: string | null;
  setPendingAddTag: Dispatch<SetStateAction<string | null>>;
  addAcknowledged: boolean;
  setAddAcknowledged: Dispatch<SetStateAction<boolean>>;
  t: ReturnType<typeof useTranslations>;
  performAddCountryAndReImport: () => Promise<void>;
}

export function ExpandConfirmDialog({
  pendingAddTag,
  setPendingAddTag,
  addAcknowledged,
  setAddAcknowledged,
  t,
  performAddCountryAndReImport,
}: ExpandConfirmDialogProps) {
  return (
    <Dialog
      open={pendingAddTag !== null}
      disablePointerDismissal
      onOpenChange={(isOpen) => {
        if (!isOpen) {
          setPendingAddTag(null);
          setAddAcknowledged(false);
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("expandConfirmTitle")}</DialogTitle>
          <DialogDescription>
            {t.rich("expandIntro", {
              tag: pendingAddTag ?? "",
              mono: (chunks) => (
                <span className="font-mono font-semibold">{chunks}</span>
              ),
            })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="rounded-md border border-border bg-muted/40 p-3 text-xs text-muted-foreground">
            <p className="font-semibold mb-1">{t("primaryNotSwitched")}</p>
            <p>
              {t.rich("expandBody", {
                tag: pendingAddTag ?? "",
                mono: (chunks) => <span className="font-mono">{chunks}</span>,
              })}
            </p>
          </div>

          <div className="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-xs">
            <p className="font-semibold mb-1">{t("timeHintTitle")}</p>
            <p>{t("timeHintBody")}</p>
          </div>

          <label className="flex items-start gap-2 rounded-md border border-border bg-muted/40 p-3 text-xs cursor-pointer">
            <input
              type="checkbox"
              checked={addAcknowledged}
              onChange={(e) => setAddAcknowledged(e.target.checked)}
              className="mt-0.5 h-4 w-4 shrink-0"
            />
            <span>
              {t.rich("ackExpand", {
                b: (chunks) => <strong>{chunks}</strong>,
              })}
            </span>
          </label>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => {
              setPendingAddTag(null);
              setAddAcknowledged(false);
            }}
          >
            {t("cancel")}
          </Button>
          <Button
            onClick={() => {
              void performAddCountryAndReImport();
            }}
            disabled={!addAcknowledged}
          >
            {addAcknowledged ? t("continueExpand") : t("needAck")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
