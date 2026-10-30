"use client";

import { useTranslations } from "use-intl";
import { InfoTooltip } from "@/components/ui/tooltip";

export function Hoi4TextMarkupHint({ className }: { className?: string }) {
  const t = useTranslations("hoi4Markup");

  return (
    <InfoTooltip
      title={t("title")}
      content={t.raw("legend")}
      className={className}
    />
  );
}
