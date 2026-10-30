"use client";

import { useCallback } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";

import type { ReseedDecision } from "./draftReseed";

export type ReseedNoticeInput = Pick<
  ReseedDecision<unknown, unknown>,
  "reason" | "clobberedDirtyDraft"
>;

export function useApplyReseedNotice(): (d: ReseedNoticeInput) => void {
  const t = useTranslations("editorWiring");
  return useCallback(
    (d: ReseedNoticeInput) => {
      if (d.reason !== "content") return;
      toast.success(t("applyReseedTitle"), {
        description: d.clobberedDirtyDraft
          ? t("applyReseedClobbered")
          : t("applyReseedSynced"),
      });
    },
    [t],
  );
}
