"use client";

import { useTranslations } from "use-intl";
import { Link, usePathname } from "@/i18n/navigation";
import { OpenInGameLink } from "@/components/workbench/OpenInGameLink";
import { gameViewHref } from "@/lib/gameViewLinks";
import { confirmDiscardIfDirty } from "@/lib/hooks/useUnsavedGuard";
import { useActiveCountry } from "@/lib/store/projectStore";
import {
  POLITICS_WORKSPACE_ROUTES,
  politicsWorkspaceHref,
} from "@/lib/workbench/politicsWorkspaceRoutes";
import { cn } from "@/lib/utils";

export function PoliticsWorkspaceNav({ projectId }: { projectId: number }) {
  const t = useTranslations("politicsPanel");
  const pathname = usePathname();
  const currentRoute = pathname.split("/").pop() ?? "";
  const activeCountry = useActiveCountry();
  const tabs = [
    {
      route: POLITICS_WORKSPACE_ROUTES.countryPolitics,
      label: t("countryPoliticsTab"),
      scope: t("countryPoliticsScope"),
    },
    {
      route: POLITICS_WORKSPACE_ROUTES.ideologies,
      label: t("ideologyLibraryTab"),
      scope: t("ideologyLibraryScope"),
    },
  ] as const;

  return (
    <nav
      aria-label={t("title")}
      className="flex shrink-0 items-center gap-1 border-b border-border bg-background px-5"
    >
      {tabs.map((tab) => {
        const active = currentRoute === tab.route;
        return (
          <Link
            key={tab.route}
            href={politicsWorkspaceHref(projectId, tab.route)}
            onNavigate={(event) => {
              if (!confirmDiscardIfDirty()) event.preventDefault();
            }}
            aria-current={active ? "page" : undefined}
            className={cn(
              "ui-interactive -mb-px flex min-h-11 items-center gap-2 border-b-2 px-4 text-sm",
              active
                ? "border-accent text-foreground"
                : "border-transparent text-muted-foreground hover:text-foreground",
            )}
          >
            <span>{tab.label}</span>
            <span className="rounded-full bg-muted px-2 py-0.5 text-micro text-muted-foreground">
              {tab.scope}
            </span>
          </Link>
        );
      })}

      <OpenInGameLink
        href={gameViewHref(projectId, {
          panel: "countrypoliticsview",
          tag: activeCountry,
        })}
        className="ml-auto"
      />
    </nav>
  );
}
