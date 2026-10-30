import { describe, expect, it, vi } from "vitest";

vi.mock("./_client", () => ({
  API_BASE: "/api",
  request: vi.fn(async (path: string) => ({ path })),
}));

import {
  aiPlanningApi,
  newAiStrategyPlanDraft,
  newAiTemplateDraft,
} from "./aiPlanning";
import { request } from "./_client";

describe("AI planning Go routing", () => {
  it("routes both CRUD workbenches to the Go project service", async () => {
    vi.mocked(request).mockClear();
    await aiPlanningApi.listPlans(182);
    await aiPlanningApi.createPlan(182, newAiStrategyPlanDraft("new_plan"));
    await aiPlanningApi.updatePlan(
      182,
      6,
      newAiStrategyPlanDraft("updated_plan"),
    );
    await aiPlanningApi.deletePlan(182, 6);
    await aiPlanningApi.listTemplates(182);
    await aiPlanningApi.createTemplate(182, newAiTemplateDraft("new_template"));
    await aiPlanningApi.updateTemplate(
      182,
      7,
      newAiTemplateDraft("updated_template"),
    );
    await aiPlanningApi.deleteTemplate(182, 7);

    expect(vi.mocked(request).mock.calls.map(([path]) => path)).toEqual([
      "/go/projects/182/ai-strategy-plans",
      "/go/projects/182/ai-strategy-plans",
      "/go/projects/182/ai-strategy-plans/6",
      "/go/projects/182/ai-strategy-plans/6",
      "/go/projects/182/ai-templates",
      "/go/projects/182/ai-templates",
      "/go/projects/182/ai-templates/7",
      "/go/projects/182/ai-templates/7",
    ]);
  });
});
