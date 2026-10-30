"use client";

import { useState, useRef, useEffect, useMemo } from "react";
import Image from "@/platform/image";
import { useTranslations } from "use-intl";
import { Button } from "@/components/ui/button";
import { API_BASE, charactersApi } from "@/lib/api";
import { useBaseModDepKeys, artUrl, gfxUrl } from "@/lib/api/basemod";
import { useConfirm } from "@/components/shared/ConfirmDialogProvider";

interface PortraitUploadProps {
  projectId: number;
  characterId: number;
  largePortrait: string;
  smallPortrait: string;

  isPortraitModified: boolean;
  characterTag: string;

  showAdvisor: boolean;
  onUploaded: () => void;
}

export function PortraitUpload({
  projectId,
  characterId,
  largePortrait,
  smallPortrait,
  isPortraitModified,
  characterTag,
  showAdvisor,
  onUploaded,
}: PortraitUploadProps) {
  const t = useTranslations("portraitUpload");
  const confirmDialog = useConfirm();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [cacheBust, setCacheBust] = useState(Date.now());

  const [portraitStage, setPortraitStage] = useState(0);
  const [advisorStage, setAdvisorStage] = useState(0);

  useEffect(() => {
    setCacheBust(Date.now());
    setPortraitStage(0);
    setAdvisorStage(0);
  }, [projectId, characterId, largePortrait, smallPortrait]);

  const commitPortrait = async (file: File) => {
    await charactersApi.uploadPortrait(projectId, characterId, file);
    setCacheBust(Date.now());
    setPortraitStage(0);
    setAdvisorStage(0);
    onUploaded();
  };

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setUploading(true);
    setError(null);
    try {
      await commitPortrait(file);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  const handleReset = async () => {
    if (
      !(await confirmDialog({
        body: t("resetConfirm", { tag: characterTag }),
        danger: true,
      }))
    )
      return;
    setResetting(true);
    setError(null);
    try {
      await charactersApi.resetPortrait(projectId, characterId);
      setCacheBust(Date.now());
      setPortraitStage(0);
      setAdvisorStage(0);
      onUploaded();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setResetting(false);
    }
  };

  const portraitIsUnmatched = largePortrait.startsWith("GFX_");
  const advisorIsUnmatched = smallPortrait.startsWith("GFX_");

  const inferredPortrait = largePortrait || `portrait_${characterTag}.png`;
  const inferredAdvisor = smallPortrait || `advisor_${characterTag}.png`;

  const baseModDepKeys = useBaseModDepKeys();

  const baseModArt = (id: number, dk: string, ref: string) =>
    ref.startsWith("GFX_") ? gfxUrl(id, dk, ref) : artUrl(id, dk, ref);
  const baseModPortraitUrls = useMemo(
    () =>
      largePortrait && baseModDepKeys.length
        ? baseModDepKeys.map((dk) => baseModArt(projectId, dk, largePortrait))
        : [],
    [projectId, largePortrait, baseModDepKeys],
  );
  const baseModAdvisorUrls = useMemo(
    () =>
      showAdvisor && smallPortrait && baseModDepKeys.length
        ? baseModDepKeys.map((dk) => baseModArt(projectId, dk, smallPortrait))
        : [],
    [projectId, showAdvisor, smallPortrait, baseModDepKeys],
  );

  const portraitCandidates = portraitIsUnmatched
    ? [
        `${API_BASE}/go/projects/${projectId}/characters/${characterId}/assets/${encodeURIComponent(largePortrait)}?t=${cacheBust}`,
        ...baseModPortraitUrls,
      ]
    : [
        `${API_BASE}/go/projects/${projectId}/characters/${characterId}/assets/${inferredPortrait}?t=${cacheBust}`,
        ...baseModPortraitUrls,
      ];
  const portraitUrl =
    portraitStage < portraitCandidates.length
      ? portraitCandidates[portraitStage]
      : null;

  const advisorCandidates = !showAdvisor
    ? []
    : advisorIsUnmatched
      ? baseModAdvisorUrls
      : [
          `${API_BASE}/go/projects/${projectId}/characters/${characterId}/assets/${inferredAdvisor}?t=${cacheBust}`,
          ...baseModAdvisorUrls,
        ];
  const advisorUrl =
    advisorStage < advisorCandidates.length
      ? advisorCandidates[advisorStage]
      : null;

  const hasUnmatchedGfx =
    portraitIsUnmatched || (showAdvisor && advisorIsUnmatched);

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
        {t("heading")}
      </h3>

      <div>
        <input
          ref={fileInputRef}
          type="file"

          accept="image/*,.dds,.tga"
          onChange={handleFileSelect}
          className="hidden"
        />

        <Button
          variant="outline"
          className="w-full border-input bg-card px-3 text-sm hover:bg-muted/40 dark:border-input dark:bg-card dark:hover:bg-muted/40"
          onClick={() => fileInputRef.current?.click()}
          disabled={uploading || resetting}
        >
          {uploading ? t("uploading") : t("uploadButton")}
        </Button>
        <p className="text-xs text-muted-foreground mt-1">{t("ratioHint")}</p>

        {(isPortraitModified ||
          (largePortrait !== "" && !portraitIsUnmatched)) && (
          <Button
            variant="ghost"
            size="sm"
            className="w-full text-xs mt-2 text-muted-foreground hover:text-foreground"
            onClick={handleReset}
            disabled={uploading || resetting}
          >
            {resetting ? t("resetting") : t("resetButton")}
          </Button>
        )}
      </div>

      {error && (
        <div className="text-xs text-destructive p-2 bg-destructive/10 rounded-md">
          {error}
        </div>
      )}

      {portraitUrl || advisorUrl ? (
        <div className="space-y-3">
          <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
            {t("generatedPreview")}
          </h4>

          {portraitUrl && (
            <div className="flex items-start gap-3">
              <Image
                key={portraitUrl}
                src={portraitUrl}
                alt={t("leaderPortraitAlt")}
                width={78}
                height={105}
                unoptimized
                className="border border-border rounded-md bg-muted shrink-0"
                onError={() => setPortraitStage((s) => s + 1)}
              />
              <div className="min-w-0">
                <div className="text-xs">{t("leaderPortrait")}</div>
                <div className="text-xs text-muted-foreground">156 x 210</div>
              </div>
            </div>
          )}

          {advisorUrl && (
            <div className="flex items-start gap-3">
              <Image
                key={advisorUrl}
                src={advisorUrl}
                alt={t("advisorIconAlt")}
                width={65}
                height={67}
                unoptimized
                className="border border-border rounded-md bg-muted shrink-0"
                onError={() => setAdvisorStage((s) => s + 1)}
              />
              <div className="min-w-0">
                <div className="text-xs">{t("advisorIcon")}</div>
                <div className="text-xs text-muted-foreground">65 x 67</div>
              </div>
            </div>
          )}
        </div>
      ) : hasUnmatchedGfx ? (
        <div className="text-center text-amber-600 dark:text-amber-400 text-sm py-6 border border-dashed border-amber-500/40 rounded-md bg-amber-500/5">
          <p className="mb-1 font-medium">{t("notImported")}</p>
          <p className="text-xs">
            {t("referencesSprite")}
            {largePortrait ? (
              <span className="block font-mono text-micro mt-1 text-amber-700 dark:text-amber-300">
                {largePortrait}
                {smallPortrait && smallPortrait !== largePortrait
                  ? ` · ${smallPortrait}`
                  : ""}
              </span>
            ) : null}
          </p>
          <p className="text-xs mt-2">{t("notImportedDetail")}</p>
        </div>
      ) : (
        <div className="text-center text-muted-foreground text-sm py-6 border border-dashed border-input bg-card rounded-lg">
          <p className="mb-1">{t("noPortrait")}</p>
          <p className="text-xs">{t("noPortraitHint")}</p>
        </div>
      )}
    </div>
  );
}
