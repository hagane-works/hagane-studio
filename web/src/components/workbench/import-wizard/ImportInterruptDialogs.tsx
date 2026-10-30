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

export interface ImportInterruptDialogsProps {
  showCloseConfirm: boolean;
  setShowCloseConfirm: Dispatch<SetStateAction<boolean>>;
  handleConfirmCancel: () => Promise<void>;
  handleKeepInBackground: () => void;
  showCancelConfirm: boolean;
  setShowCancelConfirm: Dispatch<SetStateAction<boolean>>;
  tBg: ReturnType<typeof useTranslations>;
}

export function ImportInterruptDialogs({
  showCloseConfirm,
  setShowCloseConfirm,
  handleConfirmCancel,
  handleKeepInBackground,
  showCancelConfirm,
  setShowCancelConfirm,
  tBg,
}: ImportInterruptDialogsProps) {
  return (
    <>
      <Dialog
        open={showCloseConfirm}
        onOpenChange={(o) => !o && setShowCloseConfirm(false)}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{tBg("closeConfirmTitle")}</DialogTitle>
            <DialogDescription>
              {tBg("closeConfirmDescription")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowCloseConfirm(false)}
            >
              {tBg("closeConfirmKeepOpen")}
            </Button>
            <Button
              variant="outline"
              onClick={() => void handleConfirmCancel()}
              className="border-destructive/40 text-destructive hover:bg-destructive/10"
            >
              {tBg("closeConfirmCancel")}
            </Button>
            <Button onClick={handleKeepInBackground}>
              {tBg("closeConfirmBackground")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={showCancelConfirm}
        onOpenChange={(o) => !o && setShowCancelConfirm(false)}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{tBg("cancelConfirmTitle")}</DialogTitle>
            <DialogDescription>
              {tBg("cancelConfirmDescription")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowCancelConfirm(false)}
            >
              {tBg("cancelConfirmKeepRunning")}
            </Button>
            <Button
              onClick={() => void handleConfirmCancel()}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {tBg("cancelConfirmYes")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
