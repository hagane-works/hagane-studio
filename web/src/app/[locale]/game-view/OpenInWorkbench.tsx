"use client";

import { useTranslations } from "use-intl";
import { Link } from "@/i18n/navigation";
import { withWorkbenchCountry, workbenchTargetOf } from "@/lib/gameViewLinks";
import { useGameViewCountry } from "./countryTakeIn";

export function OpenInWorkbench({ href }: { href: string | null | undefined }) {
  const t = useTranslations("gameViewEdit.panel");
  const country = useGameViewCountry((s) => s.tag);
  if (!href) return null;
  const link = withWorkbenchCountry(href, country);
  return (
    <Link
      href={link}
      target={workbenchTargetOf(link)}
      className="mt-0.5 inline-block text-micro text-accent hover:underline"
    >
      {t("openInWorkbench")}
    </Link>
  );
}
