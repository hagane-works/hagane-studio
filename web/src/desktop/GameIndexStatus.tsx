import { useState } from "react";
import useSWR from "swr";
import { useLocale, useTranslations } from "use-intl";
import { CheckCircle2, RefreshCw, TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Working } from "@/components/ui/Working";
import { readableError } from "@/lib/api/_client";
import { appApi, type GameIndexStatus } from "./appApi";

export const GAME_INDEX_KEY = "/app/game-index";

export function GameIndexStatusPanel() {
  const t = useTranslations("desktop.gameIndex");
  const locale = useLocale();
  const { data, mutate } = useSWR<GameIndexStatus>(
    GAME_INDEX_KEY,
    () => appApi.gameIndex(),
    {
      refreshInterval: (latest) => (latest?.state === "building" ? 1000 : 0),
      revalidateOnFocus: false,
    },
  );
  const [rebuilding, setRebuilding] = useState(false);
  const [error, setError] = useState("");

  async function rebuild() {
    setRebuilding(true);
    setError("");
    try {
      await mutate(await appApi.rebuildGameIndex(), false);
    } catch (reason) {
      setError(readableError(reason));
    } finally {
      setRebuilding(false);
    }
  }

  if (!data) return null;
  const canRebuild = data.state === "ready" || data.state === "failed";
  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">{t("about")}</p>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1 space-y-1.5 text-xs">
          {data.state === "missing" && (
            <p className="text-muted-foreground">{t("missing")}</p>
          )}
          {data.state === "building" && (
            <>
              <p className="flex items-center gap-2">
                <Working />
                {t("building", { done: data.done, total: data.total })}
              </p>
              <Progress
                value={data.total ? (data.done / data.total) * 100 : null}
                ariaLabel={t("title")}
                className="h-1.5"
              />
              {data.step && (
                <p className="text-muted-foreground">
                  {t("step", { step: data.step })}
                </p>
              )}
              <p className="text-muted-foreground">{t("buildingHint")}</p>
            </>
          )}
          {data.state === "ready" && (
            <p className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 shrink-0 text-success" />
              {t("ready", {
                version: data.game_version || "?",
                time: data.built_at
                  ? new Intl.DateTimeFormat(locale, {
                      dateStyle: "medium",
                      timeStyle: "short",
                    }).format(new Date(data.built_at))
                  : "?",
              })}
            </p>
          )}
          {data.state === "failed" && (
            <>
              <p className="flex items-center gap-2 text-destructive">
                <TriangleAlert className="h-4 w-4 shrink-0" />
                {t("failed")}
              </p>
              <code className="block break-all rounded-md bg-muted px-3 py-2">
                {data.error}
              </code>
            </>
          )}
          {error && <p className="text-destructive">{error}</p>}
        </div>
        {canRebuild && (
          <Button
            variant="outline"
            size="sm"
            disabled={rebuilding}
            onClick={rebuild}
          >
            <RefreshCw className="h-4 w-4" />
            {t("rebuild")}
          </Button>
        )}
      </div>
    </div>
  );
}
