"use client";

import { useEffect } from "react";
import { mutate } from "swr";
import { countriesApi } from "@/lib/api/scenario";
import { WORKBENCH_COUNTRY_PARAM } from "@/lib/gameViewLinks";
import { useProjectStore } from "@/lib/store/projectStore";

export function useLinkedCountry(
  projectId: number,
  projectLoaded: boolean,
): void {
  useEffect(() => {
    if (!projectLoaded) return;
    const url = new URL(window.location.href);
    const tag = url.searchParams
      .get(WORKBENCH_COUNTRY_PARAM)
      ?.trim()
      .toUpperCase();
    if (!tag) return;
    let cancelled = false;
    void (async () => {
      const store = useProjectStore.getState();
      if (!(store.currentProject?.country_tags ?? []).includes(tag)) {
        try {
          await countriesApi.takeIn(projectId, tag);
          await store.fetchProject(projectId);
          void mutate(["countries/grouped", projectId]);
        } catch {}
      }
      if (cancelled) return;
      const tags =
        useProjectStore.getState().currentProject?.country_tags ?? [];
      if (tags.includes(tag))
        useProjectStore.getState().setActiveCountryTag(tag);
      url.searchParams.delete(WORKBENCH_COUNTRY_PARAM);
      window.history.replaceState(
        window.history.state,
        "",
        url.pathname + url.search + url.hash,
      );
    })();
    return () => {
      cancelled = true;
    };
  }, [projectId, projectLoaded]);
}
