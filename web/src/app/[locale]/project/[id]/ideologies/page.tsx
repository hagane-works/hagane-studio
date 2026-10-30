"use client";

import { useCallback } from "react";
import { useParams, useSearchParams } from "@/platform/navigation";
import { useRouter } from "@/i18n/navigation";
import { IdeologyLibraryWorkbench } from "@/components/workbench/politics/ideologies";
import { PoliticsWorkspaceFrame } from "@/components/workbench/politics/PoliticsWorkspaceFrame";
import { useProjectStore } from "@/lib/store/projectStore";
import {
  POLITICS_WORKSPACE_ROUTES,
  politicsWorkspaceHref,
} from "@/lib/workbench/politicsWorkspaceRoutes";

export default function IdeologiesPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const router = useRouter();
  const projectId = Number(params.id);
  const fetchProject = useProjectStore((state) => state.fetchProject);
  const refreshProject = useCallback(
    () => fetchProject(projectId),
    [fetchProject, projectId],
  );
  const initialItemId = searchParams.get("item") ?? undefined;

  const syncSelectedItem = useCallback((scriptId: string) => {
    if (typeof window === "undefined") return;
    const url = new URL(window.location.href);
    if (url.searchParams.get("item") === scriptId) return;
    url.searchParams.set("item", scriptId);
    window.history.replaceState(window.history.state, "", url);
  }, []);

  return (
    <PoliticsWorkspaceFrame projectId={projectId}>
      <div className="h-full overflow-y-auto p-5">
        <IdeologyLibraryWorkbench
          projectId={projectId}
          onChanged={refreshProject}
          initialItemId={initialItemId}
          onItemChange={syncSelectedItem}
          onOpenCountryPolitics={() =>
            router.push(
              politicsWorkspaceHref(
                projectId,
                POLITICS_WORKSPACE_ROUTES.countryPolitics,
              ),
            )
          }
        />
      </div>
    </PoliticsWorkspaceFrame>
  );
}
