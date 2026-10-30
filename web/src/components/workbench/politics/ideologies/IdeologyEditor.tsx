"use client";

import { useEffect, useId, useRef } from "react";
import { useTranslations } from "use-intl";

import { DescriptionLocsTabs } from "@/components/shared/editors/DescriptionLocsTabs";
import { DisplayNameLocsTabs } from "@/components/shared/editors/DisplayNameLocsTabs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ideologyIconsApi } from "@/lib/api/tooling";
import type { PoliticsIdeology } from "@/lib/api/politics";
import type { PoliticsPreviewTarget } from "./PoliticsInGamePreview";
import {
  rootBodyColor,
  rootBodyDraftError,
  rootBodyWithColor,
} from "./rootBody";
import {
  normalizedLocaleMap,
  targetKey,
  type IdeologyDefinitionDraft,
} from "./ideologyDefinition";

export interface IdeologyEditorProps {
  projectId: number;
  ideology: PoliticsIdeology;
  displayName: string;
  sourceLabel: string;
  draft: IdeologyDefinitionDraft;
  exportLanguages: readonly string[];
  onChange: (draft: IdeologyDefinitionDraft) => void;
  saving: boolean;
  onSave: () => void;

  iconAvailable?: boolean;
  iconBust?: number;
  uploadingIcon?: boolean;
  onUploadIcon?: (file: File) => void;

  onOpenScript?: () => void;
  focusTarget?: PoliticsPreviewTarget | null;
  onFocusTarget?: (target: PoliticsPreviewTarget) => void;
}

export function IdeologyEditor({
  projectId,
  ideology,
  displayName,
  sourceLabel,
  draft,
  exportLanguages,
  onChange,
  saving,
  onSave,
  iconAvailable = false,
  iconBust = 0,
  uploadingIcon = false,
  onUploadIcon,
  onOpenScript,
  focusTarget,
  onFocusTarget,
}: IdeologyEditorProps) {
  const t = useTranslations("politicsPanel");
  const id = useId();
  const nameFieldRef = useRef<HTMLDivElement>(null);
  const descriptionFieldRef = useRef<HTMLDivElement>(null);
  const rootColorRef = useRef<HTMLInputElement>(null);
  const iconUploadLabelRef = useRef<HTMLLabelElement>(null);
  const rootBodyError =
    ideology.kind === "root" ? rootBodyDraftError(draft.rootBody) : null;
  const editable = ideology.source === "project_managed";

  useEffect(() => {
    if (!focusTarget) return;
    const element =
      focusTarget === "name"
        ? (nameFieldRef.current?.querySelector<HTMLInputElement>("input") ??
          null)
        : focusTarget === "description"
          ? (descriptionFieldRef.current?.querySelector<HTMLTextAreaElement>(
              "textarea",
            ) ?? null)
          : focusTarget === "color"
            ? rootColorRef.current
            : iconUploadLabelRef.current;
    element?.focus();
    element?.scrollIntoView?.({ block: "center", behavior: "smooth" });
  }, [focusTarget]);

  const updateRootColor = (color: string) => {
    onChange({
      ...draft,
      rootColor: color,
      rootBody: rootBodyWithColor(draft.rootBody, color),
    });
  };

  return (
    <div className="space-y-4" data-testid="ideology-workbench-editor">
      <section className="ui-panel p-5" aria-labelledby={`${id}-title`}>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs text-muted-foreground">
              {ideology.kind === "root"
                ? t("workbench.kindRoot")
                : t("workbench.kindSubtype")}
            </p>
            <h2
              id={`${id}-title`}
              className="break-words text-lg font-semibold tracking-tight"
            >
              {displayName}
            </h2>
            <p className="break-all font-mono text-xs text-muted-foreground">
              {ideology.script_id} · {t("stableIdLabel")}
            </p>
          </div>
          <span className="rounded-full border border-border bg-muted px-3 py-1 text-xs text-muted-foreground">
            {sourceLabel}
          </span>
        </div>

        {!editable ? (
          <div className="mt-4 rounded-md border border-border bg-muted/30 p-4">
            <h3 className="text-sm font-semibold">
              {t("workbench.readOnlyTitle")}
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("workbench.readOnlyReason", { source: sourceLabel })}
            </p>
          </div>
        ) : (
          <div className="mt-4 space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <div
                ref={nameFieldRef}
                className="space-y-1"
                onFocus={() => onFocusTarget?.("name")}
              >
                <Label>{t("displayNameLabel")}</Label>
                <DisplayNameLocsTabs
                  entityKey={`politics-edit:${targetKey(ideology)}:display-name`}
                  value={draft.name}
                  exportLanguages={[...exportLanguages]}
                  onChange={(name) => onChange({ ...draft, name })}
                  ariaLabel={t("displayNameLabel")}
                />
              </div>
              <div
                ref={descriptionFieldRef}
                className="space-y-1"
                onFocus={() => onFocusTarget?.("description")}
              >
                <Label>{t("workbench.descriptionLabel")}</Label>
                <DescriptionLocsTabs
                  entityKey={`politics-edit:${targetKey(ideology)}:description`}
                  value={draft.desc}
                  exportLanguages={[...exportLanguages]}
                  onChange={(desc) => onChange({ ...draft, desc })}
                  ariaLabel={t("workbench.descriptionLabel")}
                  rows={3}
                />
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              {ideology.kind === "subtype" && (
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={draft.randomDisabled}
                    onChange={(event) =>
                      onChange({
                        ...draft,
                        randomDisabled: event.target.checked,
                      })
                    }
                    className="ui-focus-ring size-4"
                  />
                  {t("workbench.randomDisabled")}
                </label>
              )}
              <Button
                type="button"
                size="xs"
                variant="outline"
                onClick={() => onChange({ ...draft, desc: {} })}
              >
                {t("workbench.clearAllDescriptions")}
              </Button>
            </div>

            {ideology.kind === "root" && (
              <details className="rounded-md border border-border bg-muted/20 p-3">
                <summary className="ui-focus-ring cursor-pointer text-sm font-medium">
                  {t("workbench.advancedSettings")}
                </summary>
                <div className="mt-3 space-y-3">
                  <div className="space-y-1">
                    <Label htmlFor={`${id}-root-color`}>
                      {t("workbench.groupColor")}
                    </Label>
                    <div className="flex items-center gap-3">
                      <Input
                        ref={rootColorRef}
                        id={`${id}-root-color`}
                        type="color"
                        value={draft.rootColor}
                        onFocus={() => onFocusTarget?.("color")}
                        onChange={(event) =>
                          updateRootColor(event.target.value)
                        }
                        className="h-10 w-20 cursor-pointer p-1"
                      />
                      <code className="text-xs text-muted-foreground">
                        {draft.rootColor.toUpperCase()}
                      </code>
                    </div>
                    <p className="text-micro text-muted-foreground">
                      {t("workbench.groupColorHint")}
                    </p>
                  </div>

                  {onOpenScript ? (
                    <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-border bg-card p-3">
                      <p className="text-xs text-muted-foreground">
                        {t("workbench.scriptTabHint")}
                      </p>
                      <Button
                        type="button"
                        size="xs"
                        variant="outline"
                        onClick={onOpenScript}
                      >
                        {t("workbench.openScriptTab")}
                      </Button>
                    </div>
                  ) : (
                    <div className="space-y-1">
                      <Label htmlFor={`${id}-root-body`}>
                        {t("workbench.advancedScript")}
                      </Label>
                      <Textarea
                        id={`${id}-root-body`}
                        value={draft.rootBody}
                        onChange={(event) => {
                          const rootBody = event.target.value;
                          onChange({
                            ...draft,
                            rootBody,
                            rootColor: rootBodyColor(rootBody),
                          });
                        }}
                        className="min-h-40 font-mono text-xs"
                      />
                      <p className="text-micro text-muted-foreground">
                        {t("workbench.advancedScriptHint")}
                      </p>
                    </div>
                  )}
                  {rootBodyError && (
                    <p role="alert" className="text-xs text-destructive">
                      {t(`workbench.rootBodyError.${rootBodyError}`)}
                    </p>
                  )}
                </div>
              </details>
            )}

            <div className="flex justify-end">
              <Button
                type="button"
                size="sm"
                disabled={
                  saving ||
                  Object.keys(normalizedLocaleMap(draft.name)).length === 0 ||
                  Boolean(rootBodyError)
                }
                onClick={onSave}
              >
                {saving ? t("workbench.saving") : t("workbench.saveChanges")}
              </Button>
            </div>
          </div>
        )}
      </section>

      {ideology.kind === "root" && (
        <section className="ui-panel p-4" aria-labelledby={`${id}-icon-title`}>
          <div className="flex items-center gap-4">
            <div className="flex size-12 shrink-0 items-center justify-center overflow-hidden rounded-md border border-border bg-muted">
              {iconAvailable ? (
                <img
                  src={`${ideologyIconsApi.previewUrl(projectId, ideology.script_id)}?bust=${iconBust}`}
                  alt={t("iconPreviewAlt", { name: displayName })}
                  width={48}
                  height={48}
                  className="size-12 object-cover"
                />
              ) : (
                <span className="px-1 text-center text-micro text-muted-foreground">
                  {t("iconPendingShort")}
                </span>
              )}
            </div>
            <div className="min-w-0 flex-1">
              <h3 id={`${id}-icon-title`} className="text-sm font-semibold">
                {t("visualIdentityTitle")}
              </h3>
              <p className="text-xs text-muted-foreground">
                {t("visualIdentityBoundary")}
              </p>
            </div>
            {onUploadIcon && (
              <label
                ref={iconUploadLabelRef}
                tabIndex={0}
                onFocus={() => onFocusTarget?.("icon")}
                className={`ui-interactive ui-focus-ring inline-flex h-8 cursor-pointer items-center rounded-md border border-border px-3 text-xs ${uploadingIcon ? "pointer-events-none opacity-50" : "hover:bg-muted"}`}
              >
                <input
                  type="file"
                  accept=".png,.jpg,.jpeg,.dds,image/png,image/jpeg"
                  className="sr-only"
                  disabled={uploadingIcon}
                  aria-label={t("iconUploadLabel", { name: displayName })}
                  onChange={(event) => {
                    const file = event.target.files?.[0];
                    if (file) onUploadIcon(file);
                    event.target.value = "";
                  }}
                />
                {uploadingIcon
                  ? t("iconUploading")
                  : iconAvailable
                    ? t("iconReplace")
                    : t("iconUpload")}
              </label>
            )}
          </div>
        </section>
      )}
    </div>
  );
}
