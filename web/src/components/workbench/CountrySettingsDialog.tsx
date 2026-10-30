"use client";

import { useTranslations } from "use-intl";

import { CountryProfilePage } from "@/components/workbench/country-profile";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useActiveCountry } from "@/lib/store/projectStore";

interface CountrySettingsDialogProps {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onManagePolitics: () => void;

  countryTag?: string;
  onChanged?: () => void | Promise<void>;
}

export function CountrySettingsDialog({
  projectId,
  open,
  onOpenChange,
  onManagePolitics,
  countryTag,
  onChanged,
}: CountrySettingsDialogProps) {
  const t = useTranslations("countrySettingsDialog");
  const activeCountryTag = useActiveCountry();
  const effectiveCountryTag = countryTag || activeCountryTag;

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal>
      <DialogContent className="max-h-[92vh] max-w-[min(1240px,calc(100vw-3rem))] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
          <p className="text-xs text-muted-foreground">{t("description")}</p>
        </DialogHeader>
        {effectiveCountryTag ? (
          <CountryProfilePage
            projectId={projectId}
            countryTag={effectiveCountryTag}
            onChanged={onChanged}
            onOpenPolitics={onManagePolitics}
          />
        ) : (
          <div className="ui-panel p-6 text-sm text-muted-foreground">
            {t("selectCountry")}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
