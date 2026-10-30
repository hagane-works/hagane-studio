"use client";

import { useTranslations } from "use-intl";
import type { GuiTechnology, TechState } from "./TechTreeLayer";
import { GamePanel } from "./GamePanel";
import { researchDays } from "./techEdit";

function Fact({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded border border-border/70 bg-muted/25 px-2.5 py-2">
      <dt className="text-micro text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 break-words text-sm text-foreground">{value}</dd>
    </div>
  );
}

function stateOf(tech: GuiTechnology): TechState {
  return tech.state ?? (tech.researched ? "researched" : "unavailable");
}

export function TechInspectPanel({
  tech,
  baseTechCost,
  onRequestEdit,
  onClose,
}: {
  tech: GuiTechnology;
  baseTechCost?: number;
  onRequestEdit: () => void;
  onClose: () => void;
}) {
  const t = useTranslations("gameViewControls");
  const tTech = useTranslations("gameViewResearch.tech");
  const days = researchDays(tech.cost, baseTechCost);
  const none = t("inspect.none");
  const joined = (values: readonly string[] | undefined) =>
    values?.length ? values.join(", ") : none;

  return (
    <GamePanel
      title={tech.card_label || tech.label || tech.name}
      kind={tTech("kind", { id: tech.name })}
      source={tech.added_by_project ? "项目" : "原版"}
      workbenchHref={null}
      wide={false}
      onClose={onClose}
    >
      <p className="mt-3 rounded border border-border/70 bg-muted/25 px-3 py-2 text-xs leading-5 text-muted-foreground">
        {t("inspect.viewHint")}
      </p>
      <dl className="mt-3 grid grid-cols-2 gap-2">
        <Fact label={t("inspect.startYear")} value={tech.start_year} />
        <Fact
          label={t("inspect.status")}
          value={t(`inspect.states.${stateOf(tech)}`)}
        />
        <Fact label={t("inspect.researchCost")} value={tech.cost} />
        <Fact
          label={t("inspect.approxDays")}
          value={days === null ? none : t("inspect.days", { days })}
        />
        <Fact label={t("inspect.position")} value={`${tech.x}, ${tech.y}`} />
        <Fact label={t("inspect.icon")} value={tech.icon || none} />
      </dl>
      <dl className="mt-2 space-y-2">
        <Fact label={t("inspect.leadsTo")} value={joined(tech.leads_to)} />
        <Fact label={t("inspect.mutuallyExclusive")} value={joined(tech.xor)} />
        <Fact
          label={t("inspect.subTechnologies")}
          value={joined(tech.sub_technologies?.map((item) => item.name))}
        />
      </dl>
      <button
        type="button"
        onClick={onRequestEdit}
        className="mt-4 w-full rounded-md border border-accent/60 bg-accent/10 px-3 py-2 text-sm text-accent hover:bg-accent/20"
      >
        {t("inspect.enterEdit")}
      </button>
    </GamePanel>
  );
}
