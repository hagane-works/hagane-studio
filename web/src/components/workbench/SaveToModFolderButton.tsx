"use client";

import { useState, useRef, useEffect } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { charactersApi } from "@/lib/api";

import { canWriteToFolder } from "@/lib/fsAccess";
import { BrowserCompatHint } from "@/components/workbench/BrowserCompatHint";

interface SaveToModFolderButtonProps {
  projectId: number;
  characterId: number;
  characterTag: string;
}

type FSAccessWindow = Window & {
  showDirectoryPicker?: (options?: {
    mode?: "read" | "readwrite";
  }) => Promise<FileSystemDirectoryHandle>;
};

async function getDirSubpath(
  root: FileSystemDirectoryHandle,
  path: string,
): Promise<{ dir: FileSystemDirectoryHandle; filename: string }> {
  const parts = path.split("/").filter(Boolean);
  const filename = parts.pop()!;
  let current = root;
  for (const part of parts) {
    current = await current.getDirectoryHandle(part, { create: true });
  }
  return { dir: current, filename };
}

async function writeFile(
  root: FileSystemDirectoryHandle,
  path: string,
  content: string,
  bom: boolean = false,
): Promise<void> {
  const { dir, filename } = await getDirSubpath(root, path);
  const handle = await dir.getFileHandle(filename, { create: true });
  const writable = await handle.createWritable();

  if (bom) {
    await writable.write(new Uint8Array([0xef, 0xbb, 0xbf]));
  }
  await writable.write(content);
  await writable.close();
}

export function SaveToModFolderButton({
  projectId,
  characterId,
  characterTag,
}: SaveToModFolderButtonProps) {
  const t = useTranslations("saveToModFolderButton");
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<"" | "saved" | "error">("");
  const [error, setError] = useState<string | null>(null);

  const dirHandleRef = useRef<FileSystemDirectoryHandle | null>(null);
  const dirNameRef = useRef<string>("");

  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  const fsAccessSupported = canWriteToFolder();

  const pickFolder = async () => {
    const picker = (window as FSAccessWindow).showDirectoryPicker;
    if (!picker) throw new Error("File System Access API unavailable");
    const handle = await picker({ mode: "readwrite" });
    dirHandleRef.current = handle;
    dirNameRef.current = handle.name;
  };

  const handleSave = async () => {
    if (!fsAccessSupported) {
      return;
    }
    setSaving(true);
    setError(null);
    setStatus("");
    try {
      if (!dirHandleRef.current) {
        await pickFolder();
      }
      const dir = dirHandleRef.current;
      if (!dir) throw new Error("No mod folder selected");

      const res = await charactersApi.previewFiles(projectId, characterId);
      const files = res.files;

      await writeFile(dir, files.script.path, files.script.content);
      await writeFile(dir, files.loc_en.path, files.loc_en.content, true);
      await writeFile(dir, files.loc_zh.path, files.loc_zh.content, true);
      await writeFile(dir, files.gfx.path, files.gfx.content);

      setStatus("saved");
      setTimeout(() => setStatus(""), 2000);

      toast.success(t("savedToFolder") + (dir.name ? ` · ${dir.name}` : ""));
    } catch (e) {
      if ((e as Error).name === "AbortError") {
        setSaving(false);
        return;
      }
      const msg = e instanceof Error ? e.message : String(e);
      setError(msg);
      setStatus("error");
      toast.error(t("saveFailed", { msg }));
    } finally {
      setSaving(false);
    }
  };

  const changeFolder = async () => {
    setError(null);
    setStatus("");
    try {
      await pickFolder();
    } catch (e) {
      if ((e as Error).name === "AbortError") return;
      setError(e instanceof Error ? e.message : String(e));
      setStatus("error");
    }
  };

  if (mounted && !fsAccessSupported) {
    return <BrowserCompatHint mode="folderWrite" />;
  }

  return (
    <div className="flex items-center gap-2">
      <Button
        size="sm"
        variant="outline"
        onClick={handleSave}
        disabled={saving}
        title={
          dirHandleRef.current
            ? t("titleWithFolder", {
                tag: characterTag,
                dir: dirNameRef.current,
              })
            : t("titleNoFolder")
        }
      >
        {saving
          ? t("saving")
          : dirHandleRef.current
            ? t("saveToFolderNamed", { dir: dirNameRef.current })
            : t("saveToModFolder")}
      </Button>
      {dirHandleRef.current && !saving && (
        <button
          type="button"
          onClick={() => void changeFolder()}
          className="text-micro text-muted-foreground hover:text-foreground underline-offset-2 hover:underline"
          title={t("changeFolderTitle")}
        >
          {t("changeFolder")}
        </button>
      )}
      {status === "saved" && (
        <span className="text-xs text-success">{t("saved")}</span>
      )}
      {status === "error" && error && (
        <span
          className="text-xs text-destructive truncate max-w-[300px]"
          title={error}
        >
          {error}
        </span>
      )}
    </div>
  );
}
