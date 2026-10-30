import type { Dispatch, SetStateAction } from "react";
import { useTranslations } from "use-intl";
import type { ScanResults, Phase, SilentSeedResult } from "./types";

export interface SilentSeedBannerProps {
  phase: Phase;
  scan: ScanResults;
  silentSeedResult: SilentSeedResult | null;
  tImport: ReturnType<typeof useTranslations>;
  advancedCountrySeed: boolean;
  setAdvancedCountrySeed: Dispatch<SetStateAction<boolean>>;
}

export function SilentSeedBanner({
  phase,
  scan,
  silentSeedResult,
  tImport,
  advancedCountrySeed,
  setAdvancedCountrySeed,
}: SilentSeedBannerProps) {
  const seedInFlight =
    silentSeedResult?.phase === "auditing" ||
    silentSeedResult?.phase === "creating";
  return (
    <>
      {phase === "scanned" &&
        scan.country_tags.fileCount > 0 &&
        !seedInFlight && (
          <div className="rounded border border-border bg-muted/40 p-3 text-xs space-y-1">
            {silentSeedResult?.phase === "done" && (
              <div className="space-y-1">
                <p className="text-green-600 dark:text-green-400">
                  {tImport("silentSeed.doneCount", {
                    created: silentSeedResult.created,
                  })}
                  {silentSeedResult.skippedDynamic > 0 &&
                    tImport("silentSeed.doneDynamicSkip", {
                      n: silentSeedResult.skippedDynamic,
                    })}
                  {silentSeedResult.skippedExisting > 0 &&
                    tImport("silentSeed.doneExistingSkip", {
                      n: silentSeedResult.skippedExisting,
                    })}
                  {silentSeedResult.presetName &&
                    tImport("silentSeed.doneIdentifiedPreset", {
                      preset: silentSeedResult.presetName,
                    })}
                  {tImport("silentSeed.doneRemoveHint")}
                </p>

                {silentSeedResult.samples &&
                  silentSeedResult.samples.length > 0 && (
                    <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-muted-foreground">
                      <span className="text-xs uppercase tracking-wide">
                        {tImport("silentSeed.samplesLabel")}
                      </span>
                      {silentSeedResult.samples.map((s) => (
                        <span key={s.tag} className="flex items-center gap-1">
                          {s.color && (
                            <span
                              className="inline-block h-3 w-3 rounded-sm border border-border/60"
                              style={{
                                backgroundColor: `rgb(${s.color.join(",")})`,
                              }}
                              title={`RGB ${s.color.join(", ")}`}
                            />
                          )}
                          <span className="text-foreground">{s.name}</span>
                          <span className="font-mono text-micro">
                            ({s.tag})
                          </span>
                        </span>
                      ))}
                    </div>
                  )}
              </div>
            )}
            {silentSeedResult?.phase === "error" && (
              <p className="text-red-500">
                {tImport("silentSeed.createFailed", {
                  err: silentSeedResult.errorMsg ?? "",
                })}
              </p>
            )}
            {!silentSeedResult && (
              <p className="text-muted-foreground">
                {tImport("detected", {
                  n:
                    scan.country_tags.declaredTagCount ??
                    scan.country_tags.fileCount,
                })}
              </p>
            )}
            <label className="flex items-center gap-1.5 cursor-pointer text-muted-foreground hover:text-foreground">
              <input
                type="checkbox"
                checked={advancedCountrySeed}
                onChange={(e) => setAdvancedCountrySeed(e.target.checked)}
              />
              <span>{tImport("advancedToggle")}</span>
            </label>
          </div>
        )}
    </>
  );
}
