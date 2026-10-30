"use client";

import type { ReactNode } from "react";
import { Globe, GlobeLock, Link2 } from "lucide-react";
import { useTranslations } from "use-intl";
import { InfoTooltip, TooltipProvider } from "@/components/ui/tooltip";
import type { ScopedEntity } from "@/lib/utils/scope";
import { getScopeSource } from "@/lib/utils/scope";

interface ScopeBadgeProps {
  entity: ScopedEntity;
  activeCountry: string | null | undefined;

  showGlyph?: boolean;
}

export function ScopeBadge({
  entity,
  activeCountry,
  showGlyph = true,
}: ScopeBadgeProps) {
  const t = useTranslations("scopeBadge");
  const source = getScopeSource(entity, activeCountry);

  const config: Record<
    "own" | "explicit" | "global" | "global_except",
    { glyph: ReactNode; label: string; title: string }
  > = {
    own: {
      glyph: (
        <span className="inline-block h-1.5 w-1.5 shrink-0 rounded-full bg-muted-foreground/60" />
      ),
      label: t("ownLabel"),
      title: t("ownTitle", { tag: entity.country_tag ?? "—" }),
    },
    explicit: {
      glyph: <Link2 className="h-3 w-3 shrink-0" aria-hidden />,
      label: t("explicitLabel"),
      title: t("explicitTitle", {
        tags: (entity.shared_country_tags ?? []).join(", ") || "—",
      }),
    },
    global: {
      glyph: <Globe className="h-3 w-3 shrink-0" aria-hidden />,
      label: t("globalLabel"),
      title: t("globalTitle"),
    },
    global_except: {
      glyph: <GlobeLock className="h-3 w-3 shrink-0" aria-hidden />,
      label: t("globalExceptLabel"),
      title: t("globalExceptTitle", {
        tags: (entity.shared_country_tags ?? []).join(", ") || "—",
      }),
    },
  };

  const { glyph, title } = config[source];

  if (!showGlyph) {
    return <span title={title} aria-label={title} />;
  }

  return (
    <span
      className="inline-flex items-center text-xs leading-none shrink-0"
      title={title}
      aria-label={title}
    >
      {glyph}
    </span>
  );
}

export function ScopeSectionHeader({
  variant,
  count,
}: {
  variant: "own" | "shared" | "all";
  count: number;
}) {
  const t = useTranslations("scopeBadge");
  const config = {
    own: {
      label: t("sectionOwnLabel"),
      hint: t("sectionOwnHint"),
      tone: "text-foreground",
    },
    shared: {
      label: t("sectionSharedLabel"),
      hint: t("sectionSharedHint"),
      tone: "text-amber-600 dark:text-amber-400",
    },

    all: {
      label: t("sectionAllLabel"),
      hint: t("sectionAllHint"),
      tone: "text-foreground",
    },
  } as const;
  const c = config[variant];
  return (
    <div className="flex items-center gap-1.5 px-1 mt-2 mb-2 first:mt-0">
      <span className={`text-xs font-semibold ${c.tone}`}>{c.label}</span>
      <span className="text-xs text-muted-foreground">{count}</span>

      <TooltipProvider>
        <InfoTooltip content={c.hint} />
      </TooltipProvider>
    </div>
  );
}
