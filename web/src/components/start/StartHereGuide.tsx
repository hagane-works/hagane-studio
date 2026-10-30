"use client";

import type { ReactNode } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { ArrowRight } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  CountryFlag,
  isResolvedCountryName,
} from "@/components/workbench/CountryFlag";
import { useOpenAddCountryDialog } from "@/components/workbench/AddCountryDialogContext";
import { countriesApi } from "@/lib/api";
import { useProjectStore, useActiveCountry } from "@/lib/store/projectStore";
import { useVanillaTagName } from "@/lib/vanillaCountryTags";
import { cn } from "@/lib/utils";

function GuideCard({
  number,
  title,
  children,
}: {
  number: number;
  title: string;
  children: ReactNode;
}) {
  return (
    <section
      className={cn(
        "flex flex-col gap-3 rounded-lg border border-border bg-card p-4",
        number === 3 && "border-accent/40 bg-accent/5",
      )}
    >
      <h2 className="flex items-start gap-2 text-base font-semibold">
        <span
          aria-hidden
          className="mt-0.5 inline-flex size-5 shrink-0 items-center justify-center rounded-full bg-accent/10 text-xs font-semibold text-accent"
        >
          {number}
        </span>
        <span className="min-w-0 break-words">{title}</span>
      </h2>
      <div className="flex flex-col items-start gap-3 pl-7">{children}</div>
    </section>
  );
}

const listClass =
  "list-disc space-y-1 pl-4 text-sm leading-relaxed text-muted-foreground marker:text-border";

export function StartHereGuide({ projectId }: { projectId: number }) {
  const t = useTranslations("startHere");
  const vanillaName = useVanillaTagName();
  const project = useProjectStore((s) =>
    s.currentProject?.id === projectId ? s.currentProject : null,
  );
  const active = useActiveCountry();
  const openAddCountry = useOpenAddCountryDialog();
  const tags = project?.country_tags ?? [];
  const tag = active && tags.includes(active) ? active : tags[0];

  const { data: grouped } = useSWR(
    project && tags.length > 1 ? ["countries/grouped", projectId] : null,
    () => countriesApi.grouped(projectId),
  );
  const country = [
    ...(grouped?.authored ?? []),
    ...(grouped?.reference ?? []),
  ].find((item) => item.tag === tag);
  const storedName =
    country?.name ||
    (project && tag === project.country_tag ? project.country_name : "");
  const name = isResolvedCountryName(storedName, tag)
    ? storedName
    : (vanillaName(tag) ?? tag ?? "");
  const baseMod = project?.dependencies?.filter(Boolean).join(", ") ?? "";
  const rich = (key: string, values: Record<string, string> = {}) =>
    t.rich(key, {
      ...values,
      b: (chunks) => (
        <span className="font-medium text-foreground">{chunks}</span>
      ),
    });

  if (!project)
    return (
      <div aria-busy="true" className="h-full animate-pulse bg-background" />
    );

  return (
    <div className="h-full overflow-y-auto">
      <div className="mx-auto w-full max-w-3xl space-y-6 px-6 py-8 sm:py-10">
        <header className="space-y-2">
          <h1 className="font-heading text-2xl font-semibold sm:text-3xl">
            {t("title")}
          </h1>
          <p className="text-sm text-muted-foreground">{t("subtitle")}</p>
        </header>
        <div className="space-y-3">
          <GuideCard number={1} title={t("c1.title")}>
            <ul className={listClass}>
              <li>{t("c1.b1")}</li>
              <li>{t("c1.b2")}</li>
              <li>
                {t(project.is_loaded_existing_mod ? "c1.b3Loaded" : "c1.b3")}
              </li>
            </ul>
            {tag ? (
              <p className="inline-flex max-w-full items-center gap-1.5 rounded-full border border-border bg-muted/40 px-2.5 py-1 text-xs text-foreground">
                <CountryFlag color={country?.color} />
                <span className="min-w-0 break-words">
                  {t("c1.currentCountry", { tag, name })}
                </span>
              </p>
            ) : (
              <Button size="sm" onClick={openAddCountry}>
                {t("c1.addCountry")}
              </Button>
            )}
          </GuideCard>
          <GuideCard number={2} title={t("c2.title")}>
            <ul className={listClass}>
              <li>{rich("c2.b1")}</li>
              {baseMod && <li>{rich("c2.b2Base", { baseMod })}</li>}
            </ul>
          </GuideCard>
          <GuideCard number={3} title={t("local.title")}>
            <ul className={listClass}>
              <li>{rich("local.b1")}</li>
              <li>{t("local.b2")}</li>
            </ul>
          </GuideCard>
        </div>
        <div className="flex flex-wrap items-center gap-4">
          {tag ? (
            <Link
              className={buttonVariants()}
              href={`/project/${projectId}/characters`}
            >
              {t("ctaCharacters")}
              <ArrowRight className="h-3.5 w-3.5" />
            </Link>
          ) : (
            <Button onClick={openAddCountry}>
              {t("ctaAddFirst")}
              <ArrowRight className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
