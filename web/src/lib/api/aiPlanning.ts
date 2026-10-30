import { request } from "./_client";

export const PLAN_BLOCK_FIELDS = [
  "allowed",
  "enable",
  "abort",
  "ai_national_focuses",
  "focus_factors",
  "research",
  "ideas",
  "traits",
  "ai_strategy",
  "weight",
] as const;
export type PlanBlockField = (typeof PLAN_BLOCK_FIELDS)[number];

export interface AiStrategyPlanDraft {
  plan_id: string;
  name: string;
  desc: string;
  allowed: string[];
  enable: string[];
  abort: string[];
  ai_national_focuses: string[];
  focus_factors: string[];
  research: string[];
  ideas: string[];
  traits: string[];
  ai_strategy: string[];
  weight: string[];
  field_order: string[];
}

export interface AiStrategyPlan extends AiStrategyPlanDraft {
  id: number;
  project_id: number;
  raw_block: string;
  source_file: string;
  source: string;
  created_at: string;
  updated_at: string;
}

export interface AiTargetTemplate {
  id: string;
  body: string;
}

export interface AiTemplateDraft {
  template_id: string;
  role: string;
  blocked_for: string;
  available_for: string;
  upgrade_prio: string;
  enable: string;
  front_role_override: string;
  target_templates: AiTargetTemplate[];
  field_order: string[];
}

export interface AiTemplate extends AiTemplateDraft {
  id: number;
  project_id: number;
  raw_block: string;
  source_file: string;
  source: string;
  created_at: string;
  updated_at: string;
}

export function newAiStrategyPlanDraft(
  planId = "new_ai_plan",
): AiStrategyPlanDraft {
  return {
    plan_id: planId,
    name: "New AI plan",
    desc: "",
    allowed: ["{\n}"],
    enable: ["{\n}"],
    abort: ["{\n}"],
    ai_national_focuses: [],
    focus_factors: [],
    research: [],
    ideas: [],
    traits: [],
    ai_strategy: [],
    weight: ["{\n\tfactor = 1\n}"],
    field_order: ["name", "desc", "allowed", "enable", "abort", "weight"],
  };
}

export function newAiTemplateDraft(
  templateId = "new_ai_template",
): AiTemplateDraft {
  return {
    template_id: templateId,
    role: "infantry",
    blocked_for: "",
    available_for: "",
    upgrade_prio: "{\n\tfactor = 1\n}",
    enable: "",
    front_role_override: "",
    target_templates: [
      {
        id: "infantry_default",
        body: "{\n\tupgrade_prio = {\n\t\tfactor = 1\n\t}\n\ttarget_template = {\n\t\tsupport = {\n\t\t}\n\t\tregiments = {\n\t\t\tinfantry = 6\n\t\t}\n\t}\n}",
      },
    ],
    field_order: ["role", "upgrade_prio", "target:infantry_default"],
  };
}

export const aiStrategyPlanToDraft = (
  row: AiStrategyPlan,
): AiStrategyPlanDraft =>
  ({
    plan_id: row.plan_id,
    name: row.name,
    desc: row.desc,
    ...Object.fromEntries(
      PLAN_BLOCK_FIELDS.map((field) => [field, [...row[field]]]),
    ),
    field_order: [...row.field_order],
  }) as AiStrategyPlanDraft;

export const aiTemplateToDraft = (row: AiTemplate): AiTemplateDraft => ({
  template_id: row.template_id,
  role: row.role,
  blocked_for: row.blocked_for,
  available_for: row.available_for,
  upgrade_prio: row.upgrade_prio,
  enable: row.enable,
  front_role_override: row.front_role_override,
  target_templates: row.target_templates.map((item) => ({ ...item })),
  field_order: [...row.field_order],
});

function quote(value: string): string {
  return `"${value.replaceAll("\\", "\\\\").replaceAll('"', '\\"')}"`;
}

function assignment(key: string, rawValue: string): string[] {
  const parts = rawValue.trim().split("\n");
  return [
    `\t${key} = ${parts[0]}`,
    ...parts.slice(1).map((line) => `\t${line}`),
  ];
}

export function formatAiStrategyPlanScript(
  value: AiStrategyPlanDraft,
  options?: { source?: string; rawBlock?: string },
): string {
  if (options?.source === "imported" && options.rawBlock?.trim())
    return options.rawBlock.trim();
  const queues = Object.fromEntries(
    PLAN_BLOCK_FIELDS.map((field) => [field, [...value[field]]]),
  ) as Record<PlanBlockField, string[]>;
  const lines = [`${value.plan_id} = {`];
  const scalarDone = new Set<string>();
  for (const key of value.field_order) {
    if ((key === "name" || key === "desc") && !scalarDone.has(key)) {
      lines.push(`\t${key} = ${quote(value[key])}`);
      scalarDone.add(key);
    } else if ((PLAN_BLOCK_FIELDS as readonly string[]).includes(key)) {
      const field = key as PlanBlockField;
      const raw = queues[field].shift();
      if (raw) lines.push(...assignment(field, raw));
    }
  }
  for (const key of ["name", "desc"] as const) {
    if (!scalarDone.has(key) && value[key])
      lines.push(`\t${key} = ${quote(value[key])}`);
  }
  for (const field of PLAN_BLOCK_FIELDS) {
    for (const raw of queues[field]) lines.push(...assignment(field, raw));
  }
  lines.push("}");
  return lines.join("\n");
}

export function formatAiTemplateScript(
  value: AiTemplateDraft,
  options?: { source?: string; rawBlock?: string },
): string {
  if (options?.source === "imported" && options.rawBlock?.trim())
    return options.rawBlock.trim();
  const roleBlocks = new Set([
    "blocked_for",
    "available_for",
    "upgrade_prio",
    "enable",
  ]);
  const roleScalars = new Set(["role", "front_role_override"]);
  const fields = value as unknown as Record<string, string>;
  const targets = new Map(
    value.target_templates.map((item) => [item.id, item]),
  );
  const emitted = new Set<string>();
  const lines = [`${value.template_id} = {`];
  const emit = (key: string, raw: string) => {
    if (!raw) return;
    if (roleScalars.has(key)) lines.push(`\t${key} = ${raw}`);
    else lines.push(...assignment(key, raw));
  };
  for (const marker of value.field_order) {
    if (
      (roleBlocks.has(marker) || roleScalars.has(marker)) &&
      !emitted.has(marker)
    ) {
      emit(marker, fields[marker]);
      emitted.add(marker);
    } else if (marker.startsWith("target:")) {
      const id = marker.slice(7);
      const target = targets.get(id);
      if (target && !emitted.has(marker)) {
        emit(target.id, target.body);
        emitted.add(marker);
      }
    }
  }
  for (const key of [
    "role",
    "front_role_override",
    "blocked_for",
    "available_for",
    "upgrade_prio",
    "enable",
  ]) {
    if (!emitted.has(key)) emit(key, fields[key]);
  }
  for (const target of value.target_templates) {
    if (!emitted.has(`target:${target.id}`)) emit(target.id, target.body);
  }
  lines.push("}");
  return lines.join("\n");
}

export const aiPlanningApi = {
  listPlans: (projectId: number) =>
    request<{ ai_strategy_plans: AiStrategyPlan[]; total: number }>(
      `/go/projects/${projectId}/ai-strategy-plans`,
    ),
  createPlan: (projectId: number, data: AiStrategyPlanDraft) =>
    request<AiStrategyPlan>(`/go/projects/${projectId}/ai-strategy-plans`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  updatePlan: (projectId: number, id: number, data: AiStrategyPlanDraft) =>
    request<AiStrategyPlan>(
      `/go/projects/${projectId}/ai-strategy-plans/${id}`,
      { method: "PUT", body: JSON.stringify(data) },
    ),
  deletePlan: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/ai-strategy-plans/${id}`, {
      method: "DELETE",
    }),
  listTemplates: (projectId: number) =>
    request<{ ai_templates: AiTemplate[]; total: number }>(
      `/go/projects/${projectId}/ai-templates`,
    ),
  createTemplate: (projectId: number, data: AiTemplateDraft) =>
    request<AiTemplate>(`/go/projects/${projectId}/ai-templates`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  updateTemplate: (projectId: number, id: number, data: AiTemplateDraft) =>
    request<AiTemplate>(`/go/projects/${projectId}/ai-templates/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  deleteTemplate: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/ai-templates/${id}`, {
      method: "DELETE",
    }),
};
