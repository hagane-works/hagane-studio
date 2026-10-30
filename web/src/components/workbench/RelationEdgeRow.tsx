"use client";

import { useTranslations } from "use-intl";
import { useRouter } from "@/i18n/navigation";
import {
  User,
  Lightbulb,
  Target,
  MessageSquare,
  Gavel,
  Cog,
  Tag,
  Factory,
  Smile,
  HelpCircle,
  ArrowRight,
  ArrowLeft,
  type LucideIcon,
} from "lucide-react";
import type { RefEdge } from "@/lib/api";
import { CountryTagChip } from "@/components/workbench/CountryTagChip";
import { canJump, jumpToSource, jumpToTarget } from "@/lib/nav/relationNav";

const TYPE_ICON: Record<string, LucideIcon> = {
  character: User,
  idea: Lightbulb,
  focus: Target,
  event: MessageSquare,
  decision: Gavel,
  technology: Cog,
  trait: Tag,
  mio: Factory,
  opinion_modifier: Smile,
};

export interface RelationEdgeRowProps {
  edge: RefEdge;

  direction: "outgoing" | "incoming";
  projectId: number;

  isReferenceCountry?: (tag: string | null) => boolean;
}

export function RelationEdgeRow({
  edge,
  direction,
  projectId,
  isReferenceCountry,
}: RelationEdgeRowProps) {
  const t = useTranslations("relations");
  const router = useRouter();

  const otherType =
    direction === "outgoing" ? edge.target_type : edge.source_kind;
  const otherId = direction === "outgoing" ? edge.target_id : edge.source_name;

  const Icon = TYPE_ICON[otherType] ?? HelpCircle;
  const DirIcon = direction === "outgoing" ? ArrowRight : ArrowLeft;

  const jumpable = canJump(otherType);

  const dangling = direction === "outgoing" && edge.resolved === false;

  const otherTag = direction === "outgoing" ? edge.target_tag : edge.source_tag;
  const dimmed = isReferenceCountry?.(otherTag) ?? false;

  const handleClick = () => {
    if (!jumpable) return;
    if (direction === "outgoing") {
      jumpToTarget(router, projectId, edge.target_type, edge.target_id);
    } else {
      jumpToSource(router, projectId, edge.source_kind, edge.source_id);
    }
  };

  const typeLabel = t(`type.${otherType}`);

  const inner = (
    <>
      <DirIcon
        className="w-3 h-3 shrink-0 text-muted-foreground/60"
        aria-hidden
      />
      <Icon
        className="w-3.5 h-3.5 shrink-0 text-muted-foreground"
        aria-hidden
      />
      <span className="font-mono text-xs truncate">{otherId}</span>
      <span className="text-micro text-muted-foreground shrink-0">
        {typeLabel}
      </span>
      <span
        className="font-mono text-micro text-muted-foreground shrink-0"
        title={t("viaField", { field: edge.field_name })}
      >
        {edge.effect_name}
      </span>
      {edge.source_tag && (
        <CountryTagChip tag={edge.source_tag} className="shrink-0" />
      )}
      {dangling && (
        <span
          className="ml-auto text-micro text-muted-foreground italic shrink-0"
          title={t("danglingHint")}
        >
          {t("danglingLabel")}
        </span>
      )}
    </>
  );

  const baseClass =
    "w-full flex items-center gap-1.5 px-2 py-1 rounded text-left" +
    (dimmed ? " opacity-60" : "");

  if (jumpable) {
    return (
      <button
        type="button"
        onClick={handleClick}
        className={
          baseClass +
          " hover:bg-muted ui-interactive" +
          (dimmed ? " hover:opacity-100" : "")
        }
        title={
          direction === "outgoing"
            ? t("openTarget", { id: otherId })
            : t("openSource", { id: otherId })
        }
      >
        {inner}
      </button>
    );
  }

  return (
    <div
      className={baseClass + " cursor-default"}
      title={t("noEditor", { type: typeLabel })}
    >
      {inner}
    </div>
  );
}
