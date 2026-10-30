import { useState } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { Gamepad2, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { exportApi } from "@/lib/api/export";
import { readableError } from "@/lib/api/_client";
import { useAppStatus } from "./SetupGate";

export function InstallModPanel({
  projectId,
  includeImported,
  replacePaths,
}: {
  projectId: number;
  includeImported: boolean;
  replacePaths: string[];
}) {
  const t = useTranslations("desktop.install");
  const { data: status } = useAppStatus();
  const [busy, setBusy] = useState(false);
  const [installedTo, setInstalledTo] = useState<string | null>(null);

  async function install() {
    setBusy(true);
    try {
      const result = await exportApi.install(
        projectId,
        includeImported,
        replacePaths,
      );
      setInstalledTo(result.mod_folder);
      toast.success(t("done"), { description: result.mod_folder });
    } catch (reason) {
      toast.error(t("failed"), { description: readableError(reason) });
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="flex flex-wrap items-center gap-3 rounded-lg border border-border bg-card px-4 py-3">
      <Gamepad2
        className="h-5 w-5 shrink-0 text-muted-foreground"
        aria-hidden
      />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{t("title")}</p>
        <p
          className="truncate text-xs text-muted-foreground"
          title={installedTo ?? status?.mod_dir}
        >
          {installedTo
            ? t("installedTo", { folder: installedTo })
            : t("hint", { folder: status?.mod_dir ?? "" })}
        </p>
      </div>
      <Button variant="outline" onClick={install} disabled={busy}>
        {busy && <Loader2 className="h-4 w-4 animate-spin" />}
        {busy ? t("installing") : installedTo ? t("reinstall") : t("install")}
      </Button>
    </section>
  );
}
