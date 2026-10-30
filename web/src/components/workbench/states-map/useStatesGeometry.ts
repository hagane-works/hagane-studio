"use client";

import useSWR from "swr";
import { statesApi } from "@/lib/api";
import type { StatesGeometry, StateGeoShape } from "@/types";

export type { StatesGeometry, StateGeoShape };

export interface UseStatesGeometryResult {
  geometry: StatesGeometry | null;

  source: string | null;

  available: boolean;
  loading: boolean;
  error: Error | undefined;
}

export function useStatesGeometry(projectId: number): UseStatesGeometryResult {
  const { data, error, isLoading } = useSWR(
    projectId ? (["states/geometry", projectId] as const) : null,
    () => statesApi.geometry(projectId),
    {
      revalidateOnFocus: false,
      revalidateIfStale: false,
      shouldRetryOnError: false,
    },
  );

  return {
    geometry: data?.geometry ?? null,
    source: data?.source ?? null,
    available: data?.available ?? false,
    loading: isLoading,
    error: error as Error | undefined,
  };
}
