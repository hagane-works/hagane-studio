"use client";

import { useState } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { ArrowUpDown, Copy, ExternalLink, Rocket } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { ModHealthSection } from "@/components/workbench/mod-health/ModHealthSection";
import type { BaseInstall } from "@/lib/api";

type OsKey = "windows" | "macos" | "linux";
const MOD_FOLDER_PATHS: { os: OsKey; labelKey: string; path: string }[] = [
  {
    os: "windows",
    labelKey: "closureOsWindows",
    path: "Documents\\Paradox Interactive\\Hearts of Iron IV\\mod\\",
  },
  {
    os: "macos",
    labelKey: "closureOsMacos",
    path: "~/Documents/Paradox Interactive/Hearts of Iron IV/mod/",
  },
  {
    os: "linux",
    labelKey: "closureOsLinux",
    path: "~/.local/share/Paradox Interactive/Hearts of Iron IV/mod/",
  },
];

function detectOs(): OsKey | null {
  if (typeof navigator === "undefined") return null;
  const ua = `${navigator.userAgent} ${navigator.platform ?? ""}`.toLowerCase();
  if (ua.includes("win")) return "windows";
  if (ua.includes("mac")) return "macos";
  if (ua.includes("linux") || ua.includes("x11")) return "linux";
  return null;
}

export function InGameVerifyDialog({
  open,
  onOpenChange,
  projectId,
  exportedFilename,
  hasDependencies,
  baseInstalls,
  dependencies,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: number;
  exportedFilename: string | null;
  hasDependencies: boolean;
  baseInstalls: BaseInstall[];
  dependencies: string[];
}) {
  const t = useTranslations("export");
  const [detectedOs] = useState<OsKey | null>(() => detectOs());

  const installList =
    baseInstalls.length > 0
      ? baseInstalls
      : dependencies.map((d) => ({
          dependency: d,
          matched: false,
          display_name: d,
          version: "",
          workshop_url: "",
          mod_key: "",
        }));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("ui.dialogVerify.title")}</DialogTitle>
          <DialogDescription>{t("ui.dialogVerify.sub")}</DialogDescription>
        </DialogHeader>

        <div className="max-h-[62vh] space-y-5 overflow-y-auto pr-1">
          <section className="space-y-2.5">
            <h4 className="text-sm font-semibold text-foreground">
              {t("ui.dialogVerify.installTitle")}
            </h4>
            <p className="text-xs text-muted-foreground">
              {t("ui.dialogVerify.installStep1")}{" "}
              {exportedFilename ? (
                <code className="font-mono text-foreground/90">
                  {exportedFilename}
                </code>
              ) : null}
            </p>
            <ul className="space-y-1">
              {MOD_FOLDER_PATHS.map((mf) => {
                const isMatch = detectedOs === mf.os;
                return (
                  <li
                    key={mf.os}
                    className={`flex flex-col gap-0.5 rounded-md border px-2.5 py-1.5 ${
                      isMatch
                        ? "border-accent/50 bg-accent/5"
                        : "border-border/60"
                    }`}
                  >
                    <span className="flex items-center gap-1.5 text-micro uppercase tracking-wide text-muted-foreground">
                      {t(mf.labelKey)}
                      {isMatch ? (
                        <span className="normal-case tracking-normal text-accent">
                          {t("closureOsDetected")}
                        </span>
                      ) : null}
                    </span>
                    <div className="flex items-center gap-2">
                      <code className="min-w-0 flex-1 break-all font-mono text-xs text-foreground/90">
                        {mf.path}
                      </code>
                      <button
                        type="button"
                        onClick={() => {
                          navigator.clipboard?.writeText(mf.path);
                          toast.success(t("closurePathCopied"));
                        }}
                        className="inline-flex shrink-0 items-center gap-1 text-xs text-muted-foreground ui-interactive hover:text-foreground"
                      >
                        <Copy className="h-3 w-3" aria-hidden />
                        {t("closureCopyPath")}
                      </button>
                    </div>
                  </li>
                );
              })}
            </ul>
            <p className="text-xs text-muted-foreground">
              {t("ui.dialogVerify.installStep2")}
            </p>

            {hasDependencies ? (
              <div className="space-y-2 rounded-lg border border-accent/40 bg-accent/5 p-3">
                <div className="flex items-center gap-2">
                  <ArrowUpDown
                    className="h-4 w-4 shrink-0 text-accent"
                    aria-hidden
                  />
                  <p className="text-sm font-semibold text-foreground">
                    {t("stage8Title")}
                  </p>
                </div>
                <p className="text-xs text-muted-foreground">
                  {t("stage8Intro")}
                </p>
                <ul className="space-y-1.5">
                  {installList.map((bi) => (
                    <li
                      key={bi.dependency}
                      className="space-y-1 rounded-md border border-border/60 bg-card px-2.5 py-1.5"
                    >
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className="text-sm font-medium text-foreground">
                          {bi.display_name}
                        </span>
                        {bi.version ? (
                          <span className="text-micro text-muted-foreground">
                            {t("stage8Version", { version: bi.version })}
                          </span>
                        ) : null}
                      </div>
                      {bi.workshop_url ? (
                        <div className="flex flex-wrap items-center gap-3">
                          <a
                            href={bi.workshop_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1 text-xs text-accent ui-interactive hover:underline"
                          >
                            <ExternalLink className="h-3 w-3" aria-hidden />
                            {t("stage8WorkshopLink")}
                          </a>
                          <button
                            type="button"
                            onClick={() => {
                              navigator.clipboard?.writeText(bi.workshop_url);
                              toast.success(t("stage8Copied"));
                            }}
                            className="inline-flex items-center gap-1 text-xs text-muted-foreground ui-interactive hover:text-foreground"
                          >
                            <Copy className="h-3 w-3" aria-hidden />
                            {t("stage8CopyUrl")}
                          </button>
                        </div>
                      ) : (
                        <p className="text-xs text-muted-foreground">
                          {t("stage8UnregisteredNote", {
                            dependency: bi.dependency,
                          })}
                        </p>
                      )}
                    </li>
                  ))}
                </ul>
                <p className="text-xs text-muted-foreground">
                  {t("stage8Step2")}
                </p>
              </div>
            ) : null}

            <div className="flex flex-wrap items-center gap-3">
              <a
                href="steam://run/394360"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-accent-foreground ui-interactive ui-focus-ring hover:bg-accent/90"
              >
                <Rocket className="h-3.5 w-3.5" aria-hidden />
                {t("closureLaunchGame")}
              </a>
              <span className="text-micro text-muted-foreground">
                {t("closureLaunchHint")}
              </span>
            </div>
          </section>

          <section className="space-y-2 border-t border-border pt-4">
            <h4 className="text-sm font-semibold text-foreground">
              {t("ui.dialogVerify.logsTitle")}
            </h4>
            <ModHealthSection projectId={projectId} />
          </section>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
          >
            {t("ui.dialogCheck.close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
