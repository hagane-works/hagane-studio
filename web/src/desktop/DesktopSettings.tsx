import { useState } from "react";
import { useTranslations } from "use-intl";
import { useSWRConfig } from "swr";
import { ExternalLink, FolderOpen, Power, Settings } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { readableError } from "@/lib/api/_client";
import { appApi, type AppStatus, type CpuSetting } from "./appApi";
import { FolderPicker } from "./FolderPicker";
import { GameIndexStatusPanel } from "./GameIndexStatus";
import { useConfirmClose } from "./programWindow";
import { APP_STATUS_KEY, useAppStatus } from "./SetupGate";
import { AvatarSettings } from "./AvatarSettings";
import { avatarSource, useAvatarStore } from "./avatar";

type Field = "projects_dir" | "game_dir" | "mod_dir" | "cwtools_dir";

const CWTOOLS_SOURCE = "https://github.com/cwtools/cwtools-hoi4-config";

export function DesktopSettingsButton({ className }: { className?: string }) {
  const t = useTranslations("desktop.settings");
  const [open, setOpen] = useState(false);
  const avatar = useAvatarStore((state) => state.selection);
  return (
    <>
      <button
        type="button"
        className={className}
        onClick={() => setOpen(true)}
        aria-label={t("open")}
        title={t("open")}
      >
        <img
          src={avatarSource(avatar)}
          alt=""
          className={`size-7 rounded-full border border-border bg-muted ${avatar.kind === "pet" ? "object-contain p-0.5" : "object-cover"}`}
        />
        <Settings className="size-3.5" aria-hidden />
      </button>
      <DesktopSettingsDialog open={open} onOpenChange={setOpen} />
    </>
  );
}

function DesktopSettingsDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations("desktop.settings");
  const tIndex = useTranslations("desktop.gameIndex");
  const { data } = useAppStatus();
  const { mutate } = useSWRConfig();
  const [picking, setPicking] = useState<Field | null>(null);
  const [error, setError] = useState("");
  const [quitting, setQuitting] = useState(false);
  const confirmClose = useConfirmClose();

  const [portDraft, setPortDraft] = useState<string | null>(null);
  const portText = portDraft ?? String(data?.configured_port ?? "");

  const port = Number(portText.trim());
  const portValid =
    port === 0 || (Number.isInteger(port) && port >= 1024 && port <= 65535);

  async function save(patch: Partial<AppStatus["settings"]>) {
    setError("");
    try {
      const next = await appApi.updateSettings(patch);
      await mutate(APP_STATUS_KEY, next, false);
      if (patch.projects_dir !== undefined || patch.game_dir !== undefined) {
        window.location.reload();
      }
      setPicking(null);
      return true;
    } catch (reason) {
      setError(readableError(reason));
      return false;
    }
  }

  const rows: { field: Field; label: string; value: string }[] = data
    ? [
        {
          field: "projects_dir",
          label: t("projectsDir"),
          value: data.settings.projects_dir,
        },
        {
          field: "game_dir",
          label: t("gameDir"),
          value: data.game
            ? `${data.game.dir} (${data.game.version || "?"})`
            : data.settings.game_dir,
        },
        { field: "mod_dir", label: t("modDir"), value: data.mod_dir },
      ]
    : [];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="rounded-md border bg-muted/30 px-3 py-2">
            <div className="text-sm font-medium">{t("legalTitle")}</div>
            <p className="mt-1 text-xs text-muted-foreground">
              {t("legalNotice")}
            </p>
          </div>
          <AvatarSettings />
          {rows.map((row) => (
            <div key={row.field} className="min-w-0 space-y-1.5">
              <div className="text-sm font-medium">{row.label}</div>
              <div className="flex items-start gap-3">
                <code className="min-w-0 flex-1 break-all rounded-md bg-muted px-3 py-2 text-xs">
                  {row.value || "—"}
                </code>
                <Button variant="outline" onClick={() => setPicking(row.field)}>
                  <FolderOpen className="h-4 w-4" />
                  {t("change")}
                </Button>
              </div>
            </div>
          ))}
          <div className="min-w-0 space-y-1.5">
            <div className="text-sm font-medium">{tIndex("title")}</div>
            <GameIndexStatusPanel />
          </div>
          {data && (
            <div className="min-w-0 space-y-1.5">
              <div className="text-sm font-medium">{t("cwtools")}</div>
              <div className="flex items-start gap-3">
                <code className="min-w-0 flex-1 break-all rounded-md bg-muted px-3 py-2 text-xs">
                  {data.cwtools.dir || t("cwtoolsMissing")}
                </code>
                <Button
                  variant="outline"
                  onClick={() => setPicking("cwtools_dir")}
                >
                  <FolderOpen className="h-4 w-4" />
                  {t("change")}
                </Button>
                {data.settings.cwtools_dir ? (
                  <Button
                    variant="outline"
                    onClick={() => void save({ cwtools_dir: "" })}
                  >
                    {t("cwtoolsAutomatic")}
                  </Button>
                ) : null}
              </div>
              {(data.cwtools.dir || data.cwtools.error) && (
                <p className="text-xs text-muted-foreground">
                  {data.cwtools.dir
                    ? t("cwtoolsModifiers", { count: data.cwtools.modifiers })
                    : data.cwtools.error}
                </p>
              )}
              <p className="text-xs text-muted-foreground">
                {t.rich("cwtoolsAbout", {
                  link: (chunks) => (
                    <a
                      href={CWTOOLS_SOURCE}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-accent ui-interactive hover:underline"
                    >
                      {chunks}
                      <ExternalLink className="h-3 w-3" aria-hidden />
                    </a>
                  ),
                })}
              </p>
            </div>
          )}
          {data && (
            <div className="min-w-0 space-y-1.5">
              <label
                htmlFor="desktop-port"
                className="block text-sm font-medium"
              >
                {t("port")}
              </label>
              <div className="flex items-start gap-3">
                <Input
                  id="desktop-port"
                  type="number"
                  inputMode="numeric"
                  min={1024}
                  max={65535}
                  className="w-32"
                  value={portText}
                  aria-invalid={!portValid}
                  onChange={(event) => setPortDraft(event.target.value)}
                />
                <Button
                  variant="outline"
                  disabled={portDraft === null || !portValid}
                  onClick={async () => {
                    if (await save({ port })) setPortDraft(null);
                  }}
                >
                  {t("portSave")}
                </Button>
              </div>

              <p className="text-xs text-muted-foreground">
                {t("portInUse", { port: String(data.port) })}
              </p>
              <p className="text-xs text-muted-foreground">{t("portAbout")}</p>
            </div>
          )}
          {data && (
            <div className="min-w-0 space-y-1.5">
              <label
                htmlFor="desktop-cpu"
                className="block text-sm font-medium"
              >
                {t("cpu")}
              </label>
              <Select
                id="desktop-cpu"
                className="w-64"
                value={data.cpu}
                onChange={(event) =>
                  void save({ cpu: event.target.value as CpuSetting })
                }
              >
                <option value="1">{t("cpuOne")}</option>

                <option value="50%">
                  {t("cpuHalf", {
                    cores: Math.max(1, Math.floor(data.cores / 2)),
                  })}
                </option>
                <option value="100%">
                  {t("cpuAll", { cores: data.cores })}
                </option>
              </Select>
              <p className="text-xs text-muted-foreground">{t("cpuAbout")}</p>
            </div>
          )}
          {error && <p className="text-sm text-destructive">{error}</p>}
          <div className="flex justify-end border-t pt-4">
            <Button
              variant="outline"
              disabled={quitting}
              onClick={async () => {
                if (!(await confirmClose())) return;
                setQuitting(true);
                await appApi.quit().catch(() => undefined);
                window.close();
              }}
            >
              <Power className="h-4 w-4" />
              {t("quit")}
            </Button>
          </div>
        </div>
        <FolderPicker
          open={picking !== null}
          onOpenChange={(next) => !next && setPicking(null)}
          title={
            picking === "cwtools_dir"
              ? t("cwtools")
              : (rows.find((row) => row.field === picking)?.label ?? "")
          }
          initialPath={picking && data ? initialFor(picking, data) : ""}
          onPick={(path) => picking && void save({ [picking]: path })}
        />
      </DialogContent>
    </Dialog>
  );
}

function initialFor(field: Field, status: AppStatus): string {
  if (field === "game_dir") return status.game?.dir ?? status.settings.game_dir;
  if (field === "mod_dir") return status.mod_dir;
  if (field === "cwtools_dir")
    return status.cwtools.dir || (status.settings.cwtools_dir ?? "");
  return status.settings.projects_dir;
}
