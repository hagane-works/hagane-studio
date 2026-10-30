"use client";

import { useEffect } from "react";
import { useTranslations } from "use-intl";
import { toast } from "sonner";

export function BackendNotifier() {
  const t = useTranslations("createCollision");

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

    window.addEventListener("api-backend-warnings", onWarnings);
    window.addEventListener("api-collision-error", onCollision);
    return () => {
      window.removeEventListener("api-backend-warnings", onWarnings);
      window.removeEventListener("api-collision-error", onCollision);
    };
  }, [t]);

  return null;
}
