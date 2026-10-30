"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";
import useSWR from "swr";
import { Link2, AlertCircle } from "lucide-react";
import { useEntityRelations } from "@/lib/hooks/useEntityRelations";
import { RelationEdgeRow } from "@/components/workbench/RelationEdgeRow";
import { countriesApi, type EntityRelations, type RefEdge } from "@/lib/api";

export interface RelationsPanelProps {
  kind: string;

  sourceId: number | null;
  projectId: number;

  hideOutgoing?: boolean;
}

export function RelationsPanel({
  kind,
  sourceId,
  projectId,
  hideOutgoing = false,
}: RelationsPanelProps) {
  const t = useTranslations("relations");

  const { data, error, isLoading } = useEntityRelations(
    projectId,
    kind,
    sourceId,
  );

  const { data: grouped } = useSWR(
    projectId ? ["countries/grouped", projectId] : null,
    () => countriesApi.grouped(projectId),
  );
  const isReferenceCountry = useMemo(() => {
    const refTags = new Set((grouped?.reference ?? []).map((c) => c.tag));
    return (tag: string | null) => (tag ? refTags.has(tag) : false);
  }, [grouped]);

  if (sourceId === null) {
    return (
      <EmptyState icon={<Link2 className="h-6 w-6 text-muted-foreground/50" />}>
        {t("noSelection")}
      </EmptyState>
    );
  }

  if (isLoading && !data) {
    return (
      <div className="p-3 space-y-1.5">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="h-7 rounded bg-muted/40 animate-pulse" />
        ))}
        <p className="pt-2 text-micro text-muted-foreground text-center">
          {t("loading")}
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <EmptyState
        icon={<AlertCircle className="h-6 w-6 text-destructive/60" />}
      >
        {t("error")}
      </EmptyState>
    );
  }

  const rel: EntityRelations | undefined = data;
  const outgoing = rel?.outgoing ?? [];
  const incoming = rel?.incoming ?? [];
  const outTotal = rel?.outgoing_total ?? outgoing.length;
  const inTotal = rel?.incoming_total ?? incoming.length;
  const danglingCount = outgoing.filter((e) => e.resolved === false).length;

  const nothingToShow = hideOutgoing
    ? incoming.length === 0
    : outgoing.length === 0 && incoming.length === 0;
  if (nothingToShow) {
    return (
      <EmptyState icon={<Link2 className="h-6 w-6 text-muted-foreground/50" />}>
        {t("noRelations")}
      </EmptyState>
    );
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      {!hideOutgoing && (
        <Section
          title={t("outgoing")}
          total={outTotal}

          danglingCount={danglingCount}
          danglingLabel={t("danglingCount", { count: danglingCount })}
          edges={outgoing}
          direction="outgoing"
          emptyText={t("emptyOutgoing")}
          projectId={projectId}
          isReferenceCountry={isReferenceCountry}
        />
      )}
      <Section
        title={t("incoming")}
        total={inTotal}
        edges={incoming}
        direction="incoming"
        emptyText={t("emptyIncoming")}
        projectId={projectId}
        isReferenceCountry={isReferenceCountry}
      />
    </div>
  );
}

interface SectionProps {
  title: string;
  total: number;
  edges: RefEdge[];
  direction: "outgoing" | "incoming";
  emptyText: string;
  projectId: number;
  isReferenceCountry: (tag: string | null) => boolean;

  danglingCount?: number;
  danglingLabel?: string;
}

function Section({
  title,
  total,
  edges,
  direction,
  emptyText,
  projectId,
  isReferenceCountry,
  danglingCount = 0,
  danglingLabel,
}: SectionProps) {
  return (
    <section className="border-b border-border last:border-b-0">
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur px-2 py-1.5 flex items-center gap-1.5 text-micro uppercase tracking-widest font-semibold text-muted-foreground">
        <span>{title}</span>
        <span className="text-muted-foreground/60">({total})</span>
        {danglingCount > 0 && (
          <span className="ml-auto normal-case tracking-normal font-normal text-micro text-muted-foreground italic">
            {danglingLabel}
          </span>
        )}
      </div>
      {edges.length === 0 ? (
        <p className="px-2 py-2 text-micro text-muted-foreground italic">
          {emptyText}
        </p>
      ) : (
        <div className="py-1">
          {edges.map((edge, i) => (
            <RelationEdgeRow
              key={`${edge.source_kind}:${edge.source_id}:${edge.target_type}:${edge.target_id}:${edge.field_name}:${i}`}
              edge={edge}
              direction={direction}
              projectId={projectId}
              isReferenceCountry={isReferenceCountry}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function EmptyState({
  icon,
  children,
}: {
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-center h-full p-8">
      <div className="flex flex-col items-center gap-3 text-center max-w-[220px]">
        <div className="w-12 h-12 rounded-xl bg-muted flex items-center justify-center border border-border">
          {icon}
        </div>
        <p className="text-xs text-muted-foreground leading-relaxed">
          {children}
        </p>
      </div>
    </div>
  );
}
