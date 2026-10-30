import { useState, type ReactNode } from "react";
import useSWR, { useSWRConfig } from "swr";
import { useTranslations } from "use-intl";
import { CheckCircle2, FolderOpen, Loader2, TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { readableError } from "@/lib/api/_client";
import {
  REFERENCE_TABLES_KEY,
  loadReferenceTables,
} from "@/lib/referenceTables";
import { appApi, type AppStatus } from "./appApi";
import { FolderPicker } from "./FolderPicker";
import { GAME_INDEX_KEY, GameIndexStatusPanel } from "./GameIndexStatus";

export const APP_STATUS_KEY = "/app/status";

export function useAppStatus() {
  return useSWR<AppStatus>(APP_STATUS_KEY, () => appApi.status(), {
    revalidateOnFocus: false,
  });
}

export function SetupGate({ children }: { children: ReactNode }) {
  const { data, error, mutate } = useAppStatus();

  const tables = useSWR(REFERENCE_TABLES_KEY, loadReferenceTables, {
    revalidateIfStale: false,
    revalidateOnFocus: false,
    revalidateOnReconnect: false,
  });
  const failure = error ?? tables.error;
  if (failure) {
    return (
      <p className="m-8 text-sm text-destructive">{readableError(failure)}</p>
    );
  }
  if (!data || !tables.data) {
    return (
      <div className="flex h-screen items-center justify-center text-muted-foreground">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    );
  }
  if (!data.ready)
    return (
      <SetupScreen status={data} onSaved={(next) => mutate(next, false)} />
    );
  return <>{children}</>;
}

function SetupScreen({
  status,
  onSaved,
}: {
  status: AppStatus;
  onSaved: (next: AppStatus) => void;
}) {
  const t = useTranslations("desktop.setup");
  const tIndex = useTranslations("desktop.gameIndex");
  const { mutate } = useSWRConfig();
  const [picking, setPicking] = useState<"projects" | "game" | null>(null);
  const [draft, setDraft] = useState(status);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  async function save(patch: Partial<AppStatus["settings"]>) {
    setSaving(true);
    setError("");
    try {
      const next = await appApi.updateSettings(patch);
      setDraft(next);

      if (patch.game_dir !== undefined) void mutate(GAME_INDEX_KEY);
      return next;
    } catch (reason) {
      setError(readableError(reason));
      return null;
    } finally {
      setSaving(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-2xl flex-col justify-center gap-8 p-8">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t("intro")}</p>
      </header>

      <section className="ui-panel space-y-3 p-5">
        <h2 className="text-sm font-semibold">{t("projectsDirTitle")}</h2>
        <p className="text-xs text-muted-foreground">{t("projectsDirHelp")}</p>
        <div className="flex items-center gap-3">
          <code className="min-w-0 flex-1 truncate rounded-md bg-muted px-3 py-2 text-xs">
            {draft.settings.projects_dir || t("notChosen")}
          </code>
          <Button variant="outline" onClick={() => setPicking("projects")}>
            <FolderOpen className="h-4 w-4" />
            {t("choose")}
          </Button>
        </div>
      </section>

      <section className="ui-panel space-y-3 p-5">
        <h2 className="text-sm font-semibold">{t("gameDirTitle")}</h2>
        <p className="text-xs text-muted-foreground">{t("gameDirHelp")}</p>
        <div className="flex items-center gap-3">
          {draft.game ? (
            <CheckCircle2 className="h-4 w-4 shrink-0 text-success" />
          ) : (
            <TriangleAlert className="h-4 w-4 shrink-0 text-warning" />
          )}
          <code className="min-w-0 flex-1 truncate rounded-md bg-muted px-3 py-2 text-xs">
            {draft.game
              ? t("gameFound", {
                  dir: draft.game.dir,
                  version: draft.game.version || "?",
                })
              : t("gameMissing")}
          </code>
          <Button variant="outline" onClick={() => setPicking("game")}>
            <FolderOpen className="h-4 w-4" />
            {t("choose")}
          </Button>
        </div>
      </section>

      {draft.game && (
        <section className="ui-panel space-y-3 p-5">
          <h2 className="text-sm font-semibold">{tIndex("title")}</h2>
          <GameIndexStatusPanel />
        </section>
      )}

      {error && <p className="text-sm text-destructive">{error}</p>}

      <div className="flex justify-end">
        <Button
          disabled={!draft.ready || saving}
          onClick={() => onSaved(draft)}
        >
          {t("start")}
        </Button>
      </div>

      <FolderPicker
        open={picking !== null}
        onOpenChange={(open) => !open && setPicking(null)}
        title={picking === "game" ? t("gameDirTitle") : t("projectsDirTitle")}
        initialPath={
          picking === "game"
            ? draft.settings.game_dir
            : draft.settings.projects_dir
        }
        onPick={async (path) => {
          const next = await save(
            picking === "game" ? { game_dir: path } : { projects_dir: path },
          );
          if (next) setPicking(null);
        }}
      />
    </main>
  );
}
