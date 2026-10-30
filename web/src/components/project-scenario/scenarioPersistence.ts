import { projectsApi } from "@/lib/api";
import { useProjectStore } from "@/lib/store/projectStore";
import type { Project } from "@/types";

export type ScenarioProjectPatch = Partial<{
  start_date: string;
  end_date: string;
  scenario_history_policy: NonNullable<Project["scenario_history_policy"]>;
}>;

export class ScenarioReadbackMismatchError extends Error {
  constructor() {
    super("scenario settings did not match the saved project");
    this.name = "ScenarioReadbackMismatchError";
  }
}

function normalizedDate(value: string | null | undefined): string {
  return (value || "").trim();
}

export async function saveScenarioProjectWithReadback(
  projectId: number,
  patch: ScenarioProjectPatch,
): Promise<Project> {
  await projectsApi.update(projectId, patch);
  const readback = await projectsApi.get(projectId);

  if (
    (patch.start_date !== undefined &&
      normalizedDate(readback.start_date) !==
        normalizedDate(patch.start_date)) ||
    (patch.end_date !== undefined &&
      normalizedDate(readback.end_date) !== normalizedDate(patch.end_date)) ||
    (patch.scenario_history_policy !== undefined &&
      readback.scenario_history_policy !== patch.scenario_history_policy)
  ) {
    throw new ScenarioReadbackMismatchError();
  }

  return readback;
}

export function applyProjectReadback(readback: Project): void {
  useProjectStore.setState((state) => ({
    currentProject:
      state.currentProject?.id === readback.id
        ? readback
        : state.currentProject,
    projects: state.projects.map((project) =>
      project.id === readback.id ? readback : project,
    ),
    error: null,
  }));
}
