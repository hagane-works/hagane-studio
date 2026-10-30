"use client";

import useSWR from "swr";
import { relationsApi, type EntityRelations } from "@/lib/api";

export function useEntityRelations(
  projectId: number | null | undefined,
  kind: string | null | undefined,
  sourceId: number | null | undefined,
) {
  const enabled =
    !!projectId && !!kind && sourceId !== null && sourceId !== undefined;
  return useSWR<EntityRelations>(
    enabled ? ["relations/entity", projectId, kind, sourceId] : null,
    () =>
      relationsApi.entity(
        projectId as number,
        kind as string,
        sourceId as number,
      ),
  );
}

export function useCountryRelations(
  projectId: number | null | undefined,
  tag: string | null | undefined,
) {
  const enabled = !!projectId && !!tag;
  return useSWR<EntityRelations>(
    enabled ? ["relations/country", projectId, tag] : null,
    () => relationsApi.country(projectId as number, tag as string),
  );
}
