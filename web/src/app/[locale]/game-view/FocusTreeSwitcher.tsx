"use client";

import { useState } from "react";
import useSWR from "swr";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { focusTreesApi } from "@/lib/api/focus-tech";
import { isGameFolderMissing } from "@/lib/api/gameIndex";
import { guiRenderApi } from "@/lib/guiRenderApi";

export function FocusTreeSwitcher({
  projectId,
  tag,
  current,
  onChoose,
}: {
  projectId: number;
  tag: string;

  current: string | null;
  onChoose: (treeId: string) => void;
}) {
  const t = useTranslations("gameViewContent.focusTree");
  const tEdit = useTranslations("gameViewEdit");
  const tCommon = useTranslations("common");
  const choices = useSWR(
    ["game-view-focus-choices", projectId, tag],
    () => {
      guiRenderApi.setProjectId(projectId);
      return guiRenderApi.focusTreeChoices(tag);
    },
    { revalidateOnFocus: false },
  );
  const [open, setOpen] = useState(false);
  const [newId, setNewId] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const list = choices.data ?? [];
  const loaded = list[0]?.id;

  const choose = (treeId: string) => {
    setOpen(false);
    onChoose(treeId);
  };

  const create = async () => {
    setBusy(true);
    setError(null);
    try {
      const tree = await focusTreesApi.create(projectId, {
        country_tag: tag,
        ...(newId.trim() ? { tree_id: newId.trim() } : {}),
      });
      setNewId("");
      await choices.mutate();
      choose(tree.tree_id);
      toast.success(t("created", { id: tree.tree_id }));
    } catch (err) {
      setError(
        isGameFolderMissing(err)
          ? tCommon("gameFolderMissing")
          : err instanceof Error
            ? err.message
            : String(err),
      );
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="w-full rounded-md border border-border bg-card p-3 text-card-foreground shadow-lg">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-full items-center justify-between gap-2 text-left"
      >
        <span className="min-w-0">
          <span className="block text-micro text-muted-foreground">
            {t("label", { tag })}
          </span>
          <span className="block truncate font-mono text-sm">
            {current ?? "—"}
          </span>
        </span>
        <span className="text-muted-foreground">{open ? "▴" : "▾"}</span>
      </button>
      {open && (
        <div className="mt-3 space-y-3">
          {choices.isLoading && (
            <p className="text-micro text-muted-foreground">
              {tEdit("person.loadingList")}
            </p>
          )}
          {!choices.isLoading && !list.length && (
            <p className="text-micro text-muted-foreground">
              {t("none", { tag })}
            </p>
          )}
          <ul className="max-h-64 space-y-1 overflow-y-auto">
            {list.map((choice) => (
              <li key={choice.id}>
                <button
                  type="button"
                  onClick={() => choose(choice.id)}
                  className={`flex w-full items-center justify-between gap-2 rounded px-2 py-1 text-left text-micro ${
                    choice.id === current
                      ? "bg-primary text-primary-foreground"
                      : "hover:bg-muted"
                  }`}
                >
                  <span className="truncate font-mono">{choice.id}</span>
                  <span className="shrink-0">
                    {t("count", { count: choice.focuses })}
                    {choice.id === loaded ? ` · ${t("gameDefault")}` : ""}
                  </span>
                </button>
              </li>
            ))}
          </ul>
          <form
            className="space-y-2"
            onSubmit={(e) => {
              e.preventDefault();
              void create();
            }}
          >
            <input
              value={newId}
              disabled={busy}
              onChange={(e) => setNewId(e.target.value)}
              placeholder={t("newIdPlaceholder")}
              className="w-full rounded border border-input bg-background px-2 py-1 font-mono text-sm"
            />
            <button
              type="submit"
              disabled={busy}
              className="rounded bg-primary px-3 py-1 text-micro text-primary-foreground hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {t("create")}
            </button>
            <p className="text-micro text-muted-foreground">
              {t("createNote", { tag })}
            </p>
          </form>
          {error && <p className="text-micro text-destructive">{error}</p>}
        </div>
      )}
    </div>
  );
}
