import type {
  WargoalListResponse,
  WargoalType,
  WargoalWrite,
} from "@/types/wargoal";

import { request } from "./_client";

export const wargoalsApi = {
  list: (projectId: number) =>
    request<WargoalListResponse>(`/go/projects/${projectId}/wargoals`),

  create: (projectId: number, data: WargoalWrite) =>
    request<WargoalType>(`/go/projects/${projectId}/wargoals`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (projectId: number, id: number, data: WargoalWrite) =>
    request<WargoalType>(`/go/projects/${projectId}/wargoals/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/wargoals/${id}`, {
      method: "DELETE",
    }),
};
