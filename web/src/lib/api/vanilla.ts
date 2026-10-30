import { request } from "./_client";
import { requestGameIndexed } from "./gameIndex";

export const vanillaDataApi = {
  equipmentArchetypes: () =>
    requestGameIndexed<{ total: number; archetypes: string[] }>(
      `/go/vanilla/equipment-archetypes`,
    ),
  subunits: () =>
    request<{
      total: number;
      stat_keys: string[];
      subunits: Record<string, import("@/types").VanillaSubunit>;
    }>(`/go/vanilla/subunits`),
};

export interface VanillaEventEntry {
  ns: string;
  id: string;
  type: string;
  title: string;
  picture: string;
  is_triggered_only: boolean;
  fire_only_once: boolean;
  hidden: boolean;
  major: boolean;
  option_count: number;
  file: string;
  line: number;
}

export interface VanillaEventListResponse {
  total_matching: number;
  limit: number;
  offset: number;
  items: VanillaEventEntry[];
  meta?: {
    total: number;
    generated_at?: string;
    namespaces: number;
  };
  _warning?: string;
}

export interface VanillaEventDetailResponse {
  entry: VanillaEventEntry;
  body: string;
}

export const vanillaEventsApi = {
  list: (params: {
    q?: string;
    event_type?: string;
    namespace_prefix?: string;
    limit?: number;
    offset?: number;
  }) => {
    const qs = new URLSearchParams();
    if (params.q) qs.set("q", params.q);
    if (params.event_type) qs.set("event_type", params.event_type);
    if (params.namespace_prefix)
      qs.set("namespace_prefix", params.namespace_prefix);
    if (params.limit !== undefined) qs.set("limit", String(params.limit));
    if (params.offset !== undefined) qs.set("offset", String(params.offset));
    const suffix = qs.toString() ? `?${qs.toString()}` : "";
    return request<VanillaEventListResponse>(`/go/vanilla/events${suffix}`);
  },
  detail: (ns: string, num: number) =>
    request<VanillaEventDetailResponse>(
      `/go/vanilla/events/${encodeURIComponent(ns)}/${num}`,
    ),
};

export interface VanillaIdeaEntry {
  category: string;
  idea_id: string;
  picture: string | null;
  file: string;
  line: number;
}

export interface VanillaIdeaListResponse {
  total_matching: number;
  limit: number;
  offset: number;
  items: VanillaIdeaEntry[];
  meta?: {
    total: number;
    generated_at?: string;
    categories: number;
  };
  _warning?: string;
}

export const vanillaIdeasApi = {
  list: (params: {
    q?: string;
    category?: string;
    has_picture?: boolean;
    limit?: number;
    offset?: number;
  }) => {
    const qs = new URLSearchParams();
    if (params.q) qs.set("q", params.q);
    if (params.category) qs.set("category", params.category);
    if (params.has_picture !== undefined)
      qs.set("has_picture", String(params.has_picture));
    if (params.limit !== undefined) qs.set("limit", String(params.limit));
    if (params.offset !== undefined) qs.set("offset", String(params.offset));
    const suffix = qs.toString() ? `?${qs.toString()}` : "";
    return requestGameIndexed<VanillaIdeaListResponse>(
      `/go/vanilla/ideas${suffix}`,
    );
  },
  detail: (category: string, ideaId: string) =>
    requestGameIndexed<{ entry: VanillaIdeaEntry }>(
      `/go/vanilla/ideas/${encodeURIComponent(category)}/${encodeURIComponent(ideaId)}`,
    ),
};
