import type {
  OpinionModifier,
  OpinionModifierWrite,
} from "@/types/opinionModifier";

import { request } from "./_client";

export const opinionModifiersApi = {
  list: (projectId: number) =>
    request<OpinionModifier[]>(`/go/projects/${projectId}/opinion-modifiers`),

  create: (projectId: number, data: OpinionModifierWrite) =>
    request<OpinionModifier>(`/go/projects/${projectId}/opinion-modifiers`, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  update: (projectId: number, id: number, data: OpinionModifierWrite) =>
    request<OpinionModifier>(
      `/go/projects/${projectId}/opinion-modifiers/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(data),
      },
    ),

  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/opinion-modifiers/${id}`, {
      method: "DELETE",
    }),
};
