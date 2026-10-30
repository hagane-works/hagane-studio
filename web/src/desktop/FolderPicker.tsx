import { useCallback, useEffect, useState } from "react";
import { useTranslations } from "use-intl";
import { ArrowUp, Folder, HardDrive, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { readableError } from "@/lib/api/_client";
import { appApi, type FolderListing } from "./appApi";

interface FolderPickerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;

  initialPath?: string;
  onPick: (path: string) => void;
}

export function FolderPicker({
  open,
  onOpenChange,
  title,
  description,
  initialPath = "",
  onPick,
}: FolderPickerProps) {
  const t = useTranslations("desktop.folderPicker");
  const [listing, setListing] = useState<FolderListing | null>(null);
  const [typed, setTyped] = useState(initialPath);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const load = useCallback(async (path: string) => {
    setLoading(true);
    setError("");
    try {
      const next = await appApi.folders(path);
      setListing(next);
      setTyped(next.path);
    } catch (reason) {
      setError(readableError(reason));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) void load(initialPath);
  }, [open, initialPath, load]);

  const atDrives = listing?.parent === null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && <DialogDescription>{description}</DialogDescription>}
        </DialogHeader>

        <form
          className="flex gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            void load(typed.trim());
          }}
        >
          <Button
            type="button"
            variant="outline"
            size="icon"
            aria-label={t("up")}
            disabled={!listing || atDrives}
            onClick={() => listing && void load(listing.parent ?? "")}
          >
            <ArrowUp className="h-4 w-4" />
          </Button>
          <Input
            value={typed}
            onChange={(event) => setTyped(event.target.value)}
            placeholder={t("pathPlaceholder")}
            spellCheck={false}
            className="font-mono text-xs"
          />
          <Button type="submit" variant="outline">
            {t("go")}
          </Button>
        </form>

        <div className="h-80 overflow-y-auto rounded-md border bg-card">
          {loading && (
            <div className="flex h-full items-center justify-center text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          )}
          {!loading && error && (
            <p className="p-4 text-sm text-destructive">{error}</p>
          )}
          {!loading && !error && listing && listing.entries.length === 0 && (
            <p className="p-4 text-sm text-muted-foreground">{t("empty")}</p>
          )}
          {!loading && !error && listing && (
            <ul className="divide-y">
              {listing.entries.map((entry) => (
                <li key={entry.path}>
                  <button
                    type="button"
                    className="ui-interactive flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-muted"
                    onClick={() => void load(entry.path)}
                  >
                    {atDrives ? (
                      <HardDrive className="h-4 w-4 shrink-0 text-muted-foreground" />
                    ) : (
                      <Folder className="h-4 w-4 shrink-0 text-muted-foreground" />
                    )}
                    <span className="truncate">{entry.name}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("cancel")}
          </Button>
          <Button
            disabled={!listing?.path}
            onClick={() => listing?.path && onPick(listing.path)}
          >
            {t("choose")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
