"use client";

import { useTranslations } from "use-intl";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { politicsApi } from "@/lib/api/politics";
import { IdeologyLibraryWorkbench } from "./politics/ideologies/IdeologyLibrary";

interface PoliticsSettingsDialogProps {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onChanged?: () => void | Promise<void>;
}

export const politicsSettingsApiContract = [
  politicsApi.references,
  politicsApi.createSubtype,
] as const;

export function PoliticsSettingsDialog({
  projectId,
  open,
  onOpenChange,
  onChanged,
}: PoliticsSettingsDialogProps) {
  const t = useTranslations("politicsPanel");

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-6xl">
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
          <DialogDescription>{t("description")}</DialogDescription>
        </DialogHeader>
        {open && (
          <IdeologyLibraryWorkbench
            projectId={projectId}
            onChanged={onChanged}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}
