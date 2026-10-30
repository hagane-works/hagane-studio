"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
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
import { countriesApi } from "@/lib/api";
import { useProjectStore } from "@/lib/store/projectStore";

const CASCADE_KEYS = [
  "characters",
  "focus_trees",
  "ideas",
  "custom_traits",
  "event_namespaces",
  "events",
  "decision_categories",
  "decisions",
  "technologies",
  "opinion_modifiers",
  "dynamic_modifiers",
  "super_events",
  "military_industrial_organizations",
  "history_files",
] as const;

export interface DeleteCountryTarget {
  id: number;
  tag: string;
  name?: string;
}

interface DeleteCountryDialogProps {
  projectId: number;
  country: DeleteCountryTarget | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;

  onDeleted?: (tag: string) => void;
}

export function DeleteCountryDialog({
  projectId,
  country,
  open,
  onOpenChange,
  onDeleted,
}: DeleteCountryDialogProps) {
  const t = useTranslations("countrySwitcher.deleteCountry");
  const fetchProject = useProjectStore((s) => s.fetchProject);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const {
    data: preview,
    error: previewError,
    isLoading,
  } = useSWR(
    open && country
      ? ["countries/delete-preview", projectId, country.id]
      : null,
    () => countriesApi.deletePreview(projectId, country!.id),
    { revalidateOnFocus: false },
  );

  const label = country
    ? country.name && country.name !== country.tag
      ? `${country.name} (${country.tag})`
      : country.tag
    : "";
  const blockedLast = Boolean(preview?.will_be_last_country);
  const blockedStates = Boolean(
    preview && !preview.vanilla_tag && preview.owned_states > 0,
  );
  const cascade = preview
    ? CASCADE_KEYS.map(
        (key) => [key, Number(preview.cascade_counts?.[key] ?? 0)] as const,
      ).filter(([, n]) => n > 0)
    : [];

  const close = (next: boolean) => {
    if (busy) return;
    if (!next) setError(null);
    onOpenChange(next);
  };

  const remove = async () => {
    if (!country || !preview || blockedLast || blockedStates) return;
    setBusy(true);
    setError(null);
    try {
      await countriesApi.delete(projectId, country.id);

      await fetchProject(projectId);
      toast.success(t("deleted", { country: label }));
      onDeleted?.(country.tag);
      onOpenChange(false);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={close} disablePointerDismissal>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("title", { country: label })}</DialogTitle>
          <DialogDescription>{t("description")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3 text-sm">
          {isLoading && (
            <div
              className="h-16 rounded-md bg-muted/60 animate-pulse"
              aria-hidden
            />
          )}
          {previewError && (
            <p role="alert" className="text-destructive">
              {t("previewFailed", { error: (previewError as Error).message })}
            </p>
          )}
          {preview && blockedLast && (
            <p
              role="alert"
              className="rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-warning"
            >
              {t("lastCountry")}
            </p>
          )}
          {preview && !blockedLast && blockedStates && (
            <p
              role="alert"
              className="rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-warning"
            >
              {t("ownsStates", { count: preview.owned_states })}
            </p>
          )}
          {preview && !blockedLast && !blockedStates && (
            <>
              {cascade.length > 0 ? (
                <div>
                  <p>{t("cascadeIntro")}</p>
                  <ul className="mt-1.5 grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-muted-foreground">
                    {cascade.map(([key, n]) => (
                      <li key={key} className="flex justify-between gap-2">
                        <span>{t(`kind.${key}`)}</span>
                        <span className="tabular-nums">{n}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              ) : (
                <p className="text-muted-foreground">{t("nothingElse")}</p>
              )}
              <p className="text-xs text-muted-foreground">
                {preview.vanilla_tag ? t("vanillaNote") : t("customNote")}
              </p>
            </>
          )}
          {error && (
            <p role="alert" className="text-destructive">
              {error}
            </p>
          )}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => close(false)}
            disabled={busy}
          >
            {t("cancel")}
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={() => void remove()}
            disabled={!preview || blockedLast || blockedStates || busy}
          >
            {busy ? t("deleting") : t("confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
