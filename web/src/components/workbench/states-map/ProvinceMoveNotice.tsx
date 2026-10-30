"use client";

import { useTranslations } from "use-intl";
import { ArrowRightLeft, AlertTriangle, Undo2, X } from "lucide-react";
import { useStateStore } from "@/lib/store/stateStore";

type Translator = (
  key: string,
  values?: Record<string, string | number>,
) => string;

function tr(
  t: Translator,
  key: string,
  values: Record<string, string | number>,
  fallback: string,
): string {
  try {
    const out = t(key, values);
    return out && !out.endsWith(key) ? out : fallback;
  } catch {
    return fallback;
  }
}

export interface ProvinceMoveNoticeProps {
  projectId: number;
}

export function ProvinceMoveNotice({ projectId }: ProvinceMoveNoticeProps) {
  const t = useTranslations("states") as unknown as Translator;
  const tUndo = useTranslations("undoBanner");
  const move = useStateStore((s) => s.lastProvinceMove);
  const failure = useStateStore((s) => s.provinceMoveFailure);
  const undoing = useStateStore((s) => s.undoingProvinceMove);
  const undoProvinceMove = useStateStore((s) => s.undoProvinceMove);
  const dismissProvinceMove = useStateStore((s) => s.dismissProvinceMove);

  if (!move && !failure) return null;

  const isFailure = failure != null;
  const provinces = (failure?.provinceIds ?? move?.movedProvinceIds ?? []).join(
    ", ",
  );
  const from = (move?.donors ?? []).map((d) => d.stateId).join(", ");
  const to = String(move?.target.stateId ?? "");

  let text: string;
  if (failure?.kind === "undo_failed") {
    text = `${tUndo("undoFailedPrefix")}${failure.detail}`;
  } else if (failure) {
    text = tr(
      t,
      "provinceMoveFailedNotice",
      { provinces, detail: failure.detail },
      `Couldn't move province ${provinces} · nothing was changed. ${failure.detail}`,
    );
  } else {
    text = tr(
      t,
      "provinceMovedNotice",
      { provinces, from, to },
      `Province ${provinces} moved from state ${from} to state ${to}.`,
    );
  }

  return (
    <div
      role="status"
      data-testid="province-move-notice"
      className={`fixed bottom-4 left-1/2 z-50 -translate-x-1/2 flex max-w-[min(32rem,calc(100vw-2rem))] items-start gap-2 rounded-lg border px-3 py-2 text-xs shadow-lg backdrop-blur ${
        isFailure
          ? "border-destructive/50 bg-destructive/10 text-destructive"
          : "border-border/60 bg-popover/95 text-foreground"
      }`}
    >
      {isFailure ? (
        <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
      ) : (
        <ArrowRightLeft
          className="mt-0.5 h-3.5 w-3.5 shrink-0 text-accent"
          aria-hidden
        />
      )}
      <p className="min-w-0 flex-1 break-words">{text}</p>
      {move && !isFailure && (
        <button
          type="button"
          onClick={() => void undoProvinceMove(projectId)}
          disabled={undoing}
          title={tUndo("undo")}
          className="ui-interactive inline-flex shrink-0 items-center gap-1 rounded-md border border-border/60 px-2 py-1 font-medium hover:bg-muted disabled:opacity-50"
        >
          <Undo2 className="h-3 w-3" aria-hidden />
          {undoing ? tUndo("undoing") : tUndo("undo")}
        </button>
      )}
      <button
        type="button"
        onClick={dismissProvinceMove}
        title={tUndo("dismissTitle")}
        aria-label={tUndo("dismissTitle")}
        className="ui-interactive shrink-0 rounded p-1 text-muted-foreground hover:text-foreground"
      >
        <X className="h-3 w-3" aria-hidden />
      </button>
    </div>
  );
}
