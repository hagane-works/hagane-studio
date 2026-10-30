import type { Technology } from "@/types";

export interface TechUpdateOp {
  id: number;
  data: Partial<Technology>;
}

export function buildConnectOps(
  techs: Technology[],
  targetDbId: number,
  sourceTechIdStr: string,
): TechUpdateOp[] {
  const target = techs.find((t) => t.id === targetDbId);
  if (!target) return [];

  if (target.tech_id === sourceTechIdStr) return [];
  const current = target.prerequisites || [];
  if (current.includes(sourceTechIdStr)) return [];

  const ops: TechUpdateOp[] = [
    { id: target.id, data: { prerequisites: [...current, sourceTechIdStr] } },
  ];

  const source = techs.find((t) => t.tech_id === sourceTechIdStr);
  if (
    source &&
    !(source.path ?? []).some((e) => e.leads_to_tech === target.tech_id)
  ) {
    ops.push({
      id: source.id,
      data: {
        path: [
          ...(source.path ?? []),
          { leads_to_tech: target.tech_id, research_cost_coeff: 1.0 },
        ],
      },
    });
  }
  return ops;
}

export function buildDisconnectOps(
  techs: Technology[],
  targetDbId: number,
  sourceTechIdStr: string,
): TechUpdateOp[] {
  const target = techs.find((t) => t.id === targetDbId);
  if (!target) return [];

  const ops: TechUpdateOp[] = [];
  const source = techs.find((t) => t.tech_id === sourceTechIdStr);
  if (source?.path?.some((e) => e.leads_to_tech === target.tech_id)) {
    ops.push({
      id: source.id,
      data: {
        path: source.path.filter((e) => e.leads_to_tech !== target.tech_id),
      },
    });
  }
  ops.push({
    id: target.id,
    data: {
      prerequisites: (target.prerequisites || []).filter(
        (p) => p !== sourceTechIdStr,
      ),
    },
  });
  return ops;
}
