import { request } from "./_client";
import type { UiPanel, UiPanelFiles } from "@/types";

export const uiPanelsApi = {
  list: (projectId: number) =>
    request<{ panels: UiPanel[]; total: number }>(
      `/go/projects/${projectId}/ui-panels`,
    ),
  get: (projectId: number, id: number) =>
    request<UiPanel>(`/go/projects/${projectId}/ui-panels/${id}`),
  create: (projectId: number, data: Partial<UiPanel>) =>
    request<UiPanel>(`/go/projects/${projectId}/ui-panels`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  update: (projectId: number, id: number, data: Partial<UiPanel>) =>
    request<UiPanel>(`/go/projects/${projectId}/ui-panels/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    }),
  delete: (projectId: number, id: number) =>
    request<void>(`/go/projects/${projectId}/ui-panels/${id}`, {
      method: "DELETE",
    }),

  files: (projectId: number, id: number) =>
    request<UiPanelFiles>(`/go/projects/${projectId}/ui-panels/${id}/files`),
};
