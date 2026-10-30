"use client";

import { Gamepad2 } from "lucide-react";
import { useTranslations } from "use-intl";
import { Link } from "@/i18n/navigation";
import { gameViewTargetOf } from "@/lib/gameViewLinks";

export function OpenInGameLink({
  href,
  className = "",
}: {
  href: string;
  className?: string;
}) {
  const t = useTranslations("workbenchShell");
  return (
    <Link
      href={href}
      target={gameViewTargetOf(href)}
      className={`inline-flex shrink-0 items-center gap-1 rounded px-1.5 py-0.5 text-xs text-accent hover:bg-muted ui-interactive ${className}`}
    >
      <Gamepad2 className="h-3.5 w-3.5" aria-hidden />
      {t("openInGame")}
    </Link>
  );
}
