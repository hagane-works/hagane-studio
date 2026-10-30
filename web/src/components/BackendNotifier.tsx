"use client";

import { useEffect } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";
import { WRITE_BLOCKED_EVENT } from "@/lib/api/_client";

export function BackendNotifier() {
  const t = useTranslations("createCollision");
  const tFiles = useTranslations("fileAccess");

  useEffect(() => {
    if (typeof window === "undefined") return;

    const onWarnings = (e: Event) => {
      const ce = e as CustomEvent<{ path: string; warnings: string[] }>;
      const warnings = ce.detail?.warnings ?? [];
      for (const w of warnings) {
        if (typeof w !== "string") continue;
        const isAutoSuffix = w.includes("auto-suffixed");
        toast.info(isAutoSuffix ? t("autoSuffixedTitle") : w, {
          description: isAutoSuffix ? w : undefined,
        });
      }
    };

    const onCollision = (e: Event) => {
      const ce = e as CustomEvent<{ path: string; message: string }>;
      const msg = ce.detail?.message ?? t("alreadyExistsFallback");
      toast.error(t("alreadyExistsTitle"), { description: msg });
    };

    const onWriteBlocked = (e: Event) => {
      const ce = e as CustomEvent<{ folder: string }>;
      toast.error(tFiles("writeBlockedTitle"), {
        id: WRITE_BLOCKED_EVENT,
        duration: 20000,
        description: tFiles("writeBlockedDescription", {
          folder: ce.detail?.folder ?? "",
        }),
      });
    };

    window.addEventListener("api-backend-warnings", onWarnings);
    window.addEventListener("api-collision-error", onCollision);
    window.addEventListener(WRITE_BLOCKED_EVENT, onWriteBlocked);
    return () => {
      window.removeEventListener("api-backend-warnings", onWarnings);
      window.removeEventListener("api-collision-error", onCollision);
      window.removeEventListener(WRITE_BLOCKED_EVENT, onWriteBlocked);
    };
  }, [t, tFiles]);

  return null;
}
