"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { TagSearchCombobox } from "@/components/TagSearchCombobox";
import {
  CustomTagInput,
  CUSTOM_TAG_REGEX,
} from "@/components/workbench/CustomTagInput";
import { ConflictWarningDialog } from "@/components/workbench/ConflictWarningDialog";
import type { CountryTag } from "@/types";
import { API_BASE, countriesApi } from "@/lib/api";
import { useProjectStore } from "@/lib/store/projectStore";
import { useVanillaCountryTags } from "@/lib/vanillaCountryTags";
import {
  GRAPHICAL_CULTURES,
  deriveGraphicalCulture2d,
  rgbToHex,
  hexToRgb,
} from "@/lib/graphicalCultures";

interface AddCountryDialogProps {
  projectId: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;

  onAdded?: (tag: string) => void;
}

type Tab = "vanilla" | "custom";

export function AddCountryDialog({
  projectId,
  open,
  onOpenChange,
  onAdded,
}: AddCountryDialogProps) {
  const t = useTranslations("addCountryDialog");
  const fetchProject = useProjectStore((s) => s.fetchProject);
  const setActiveCountryTag = useProjectStore((s) => s.setActiveCountryTag);
  const vanillaTags = useVanillaCountryTags();

  const [tab, setTab] = useState<Tab>("vanilla");

  const [selectedTag, setSelectedTag] = useState<CountryTag | null>(null);
  const [nameOverride, setNameOverride] = useState("");

  const [customTag, setCustomTag] = useState("");
  const [customName, setCustomName] = useState("");

  const [customColor, setCustomColor] = useState<number[]>([]);
  const [customCulture, setCustomCulture] = useState("");
  const [conflictOpen, setConflictOpen] = useState(false);

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reset = () => {
    setSelectedTag(null);
    setNameOverride("");
    setCustomTag("");
    setCustomName("");
    setCustomColor([]);
    setCustomCulture("");
    setConflictOpen(false);
    setError(null);
    setTab("vanilla");
  };

  const postCountry = async (
    tag: string,
    name: string,
    extra?: Record<string, unknown>,
  ) => {
    setSubmitting(true);
    setError(null);
    try {
      const res = await fetch(
        `${API_BASE}/go/projects/${projectId}/countries`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            tag,
            name: name.trim() || tag,
            ideology: "neutrality",
            ...(extra ?? {}),
          }),
        },
      );
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.detail || `HTTP ${res.status}`);
      }

      await fetchProject(projectId);
      setActiveCountryTag(tag);
      onAdded?.(tag);
      reset();
      onOpenChange(false);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  const takeInVanilla = async (tag: string) => {
    setSubmitting(true);
    setError(null);
    try {
      await countriesApi.takeIn(projectId, tag);
      await fetchProject(projectId);
      setActiveCountryTag(tag);
      onAdded?.(tag);
      reset();
      onOpenChange(false);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleVanillaSubmit = async () => {
    if (!selectedTag) return;
    const name = nameOverride.trim();
    if (name) await postCountry(selectedTag.tag, name);
    else await takeInVanilla(selectedTag.tag);
  };

  const customExtra = (): Record<string, unknown> => {
    const extra: Record<string, unknown> = {};
    if (customColor.length === 3) extra.color = customColor;
    if (customCulture) {
      extra.graphical_culture = customCulture;
      extra.graphical_culture_2d = deriveGraphicalCulture2d(customCulture);
    }
    return extra;
  };

  const handleCustomSubmit = async () => {
    const trimmed = customTag.trim();
    if (!CUSTOM_TAG_REGEX.test(trimmed)) return;

    if (vanillaTags.lookup(trimmed)) {
      setConflictOpen(true);
      return;
    }
    await postCountry(trimmed, customName, customExtra());
  };

  const handleCustomConfirmOverride = async () => {
    setConflictOpen(false);
    await postCountry(customTag.trim(), customName, customExtra());
  };

  const customValid = CUSTOM_TAG_REGEX.test(customTag.trim());

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(next) => {
          if (!next) reset();
          onOpenChange(next);
        }}
        disablePointerDismissal
      >
        <DialogContent className="overflow-visible">
          <DialogHeader>
            <DialogTitle>{t("title")}</DialogTitle>
          </DialogHeader>

          <div
            role="tablist"
            aria-label={t("sourceAriaLabel")}
            className="flex border-b border-border"
          >
            {(
              [
                { id: "vanilla", label: t("tabVanilla") },
                { id: "custom", label: t("tabCustom") },
              ] as const
            ).map((item) => {
              const active = tab === item.id;
              return (
                <button
                  key={item.id}
                  role="tab"
                  aria-selected={active}
                  type="button"
                  onClick={() => {
                    setTab(item.id);
                    setError(null);
                  }}
                  className={
                    "px-3 py-1.5 text-xs ui-interactive border-b-2 -mb-px " +
                    (active
                      ? "border-accent text-accent font-semibold"
                      : "border-transparent text-muted-foreground hover:text-foreground")
                  }
                >
                  {item.label}
                </button>
              );
            })}
          </div>

          {tab === "vanilla" && (
            <div className="space-y-4 pt-2">
              <div className="space-y-2">
                <Label>{t("countryTagLabel")}</Label>
                <TagSearchCombobox
                  value={selectedTag}
                  onChange={setSelectedTag}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("displayNameLabel")}</Label>
                <Input
                  placeholder={selectedTag?.name || t("vanillaNamePlaceholder")}
                  value={nameOverride}
                  onChange={(e) => setNameOverride(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  {t("vanillaNameHint")}
                </p>
              </div>
              {error && <p className="text-xs text-destructive">{error}</p>}
              <Button
                className="w-full"
                onClick={handleVanillaSubmit}
                disabled={!selectedTag || submitting}
              >
                {submitting ? t("adding") : t("addButton")}
              </Button>
            </div>
          )}

          {tab === "custom" && (
            <div className="space-y-4 pt-2">
              <p className="text-xs text-muted-foreground">{t("customHint")}</p>
              <CustomTagInput value={customTag} onChange={setCustomTag} />
              <div className="space-y-2">
                <Label>{t("displayNameLabel")}</Label>
                <Input
                  placeholder={t("customNamePlaceholder")}
                  value={customName}
                  onChange={(e) => setCustomName(e.target.value)}
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label>{t("mapColorLabel")}</Label>
                  <div className="flex items-center gap-2">
                    <input
                      type="color"
                      aria-label={t("mapColorAriaLabel")}
                      value={rgbToHex(
                        customColor.length === 3 ? customColor : null,
                      )}
                      onChange={(e) => setCustomColor(hexToRgb(e.target.value))}
                      className="h-8 w-12 cursor-pointer rounded border bg-transparent p-0.5"
                    />
                    <span className="font-mono text-xs text-muted-foreground">
                      {customColor.length === 3
                        ? `rgb(${customColor.join(", ")})`
                        : t("mapColorAuto")}
                    </span>
                  </div>
                </div>
                <div className="space-y-1">
                  <Label>{t("graphicalCultureLabel")}</Label>
                  <Select
                    aria-label={t("graphicalCultureAriaLabel")}
                    value={customCulture}
                    onChange={(e) => setCustomCulture(e.target.value)}
                    className="h-8 px-2 text-xs"
                  >
                    <option value="">{t("graphicalCultureAuto")}</option>
                    {GRAPHICAL_CULTURES.map((g) => (
                      <option key={g} value={g}>
                        {g}
                      </option>
                    ))}
                  </Select>
                </div>
              </div>
              {error && <p className="text-xs text-destructive">{error}</p>}
              <Button
                className="w-full"
                onClick={handleCustomSubmit}
                disabled={!customValid || submitting}
              >
                {submitting ? t("adding") : t("addButton")}
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>

      <ConflictWarningDialog
        open={conflictOpen}
        onOpenChange={setConflictOpen}
        tag={customTag.trim()}
        onConfirm={handleCustomConfirmOverride}
      />
    </>
  );
}
