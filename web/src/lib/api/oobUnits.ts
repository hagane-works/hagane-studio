import { request } from "./_client";
import type {
  AirWing,
  AirWingCreate,
  AirWingGroup,
  AirWingGroupCreate,
  AirWingGroupUpdate,
  AirWingUpdate,
  Fleet,
  FleetCreate,
  FleetUpdate,
  OobUnitsImportRequest,
  OobUnitsImportResult,
  OobUnitsPreview,
  OobUnitsSnapshot,
  Ship,
  ShipCreate,
  ShipUpdate,
  TaskForce,
  TaskForceCreate,
  TaskForceUpdate,
} from "@/types/oobUnits";

export interface OobUnitsFilter {
  oobFile?: string;
  sourceFile?: string;
}

const goBase = (projectId: number) => `/go/projects/${projectId}/oob-units`;

const withFilter = (path: string, filter?: OobUnitsFilter): string => {
  const params = new URLSearchParams();
  if (filter?.oobFile) params.set("oob_file", filter.oobFile);
  if (filter?.sourceFile) params.set("source_file", filter.sourceFile);
  const query = params.toString();
  return query ? `${path}?${query}` : path;
};

const write = <T>(path: string, method: "POST" | "PUT", body: unknown) =>
  request<T>(path, { method, body: JSON.stringify(body) });

const remove = (path: string) => request<void>(path, { method: "DELETE" });

export const oobUnitsApi = {
  snapshot: (projectId: number, filter?: OobUnitsFilter) =>
    request<OobUnitsSnapshot>(
      withFilter(`${goBase(projectId)}/snapshot`, filter),
    ),

  preview: (projectId: number, filter?: OobUnitsFilter) =>
    request<OobUnitsPreview>(
      withFilter(`${goBase(projectId)}/preview`, filter),
    ),

  importText: (projectId: number, data: OobUnitsImportRequest) =>
    write<OobUnitsImportResult>(`${goBase(projectId)}/import`, "POST", data),

  airWingGroups: {
    create: (projectId: number, data: AirWingGroupCreate) =>
      write<AirWingGroup>(`${goBase(projectId)}/air-wing-groups`, "POST", data),
    update: (projectId: number, rowId: number, data: AirWingGroupUpdate) =>
      write<AirWingGroup>(
        `${goBase(projectId)}/air-wing-groups/${rowId}`,
        "PUT",
        data,
      ),
    remove: (projectId: number, rowId: number) =>
      remove(`${goBase(projectId)}/air-wing-groups/${rowId}`),
  },

  airWings: {
    create: (projectId: number, data: AirWingCreate) =>
      write<AirWing>(`${goBase(projectId)}/air-wings`, "POST", data),
    update: (projectId: number, rowId: number, data: AirWingUpdate) =>
      write<AirWing>(`${goBase(projectId)}/air-wings/${rowId}`, "PUT", data),
    remove: (projectId: number, rowId: number) =>
      remove(`${goBase(projectId)}/air-wings/${rowId}`),
  },

  fleets: {
    create: (projectId: number, data: FleetCreate) =>
      write<Fleet>(`${goBase(projectId)}/fleets`, "POST", data),
    update: (projectId: number, rowId: number, data: FleetUpdate) =>
      write<Fleet>(`${goBase(projectId)}/fleets/${rowId}`, "PUT", data),
    remove: (projectId: number, rowId: number) =>
      remove(`${goBase(projectId)}/fleets/${rowId}`),
  },

  taskForces: {
    create: (projectId: number, data: TaskForceCreate) =>
      write<TaskForce>(`${goBase(projectId)}/task-forces`, "POST", data),
    update: (projectId: number, rowId: number, data: TaskForceUpdate) =>
      write<TaskForce>(
        `${goBase(projectId)}/task-forces/${rowId}`,
        "PUT",
        data,
      ),
    remove: (projectId: number, rowId: number) =>
      remove(`${goBase(projectId)}/task-forces/${rowId}`),
  },

  ships: {
    create: (projectId: number, data: ShipCreate) =>
      write<Ship>(`${goBase(projectId)}/ships`, "POST", data),
    update: (projectId: number, rowId: number, data: ShipUpdate) =>
      write<Ship>(`${goBase(projectId)}/ships/${rowId}`, "PUT", data),
    remove: (projectId: number, rowId: number) =>
      remove(`${goBase(projectId)}/ships/${rowId}`),
  },
};
