"use client";

import { useCallback } from "react";
import { useParams } from "@/platform/navigation";
import { useRouter } from "@/i18n/navigation";
import { CountryPoliticsPage as CountryPoliticsContent } from "@/components/workbench/politics/country-politics";
import { PoliticsWorkspaceFrame } from "@/components/workbench/politics/PoliticsWorkspaceFrame";
import {
  POLITICS_WORKSPACE_ROUTES,
  politicsWorkspaceHref,
} from "@/lib/workbench/politicsWorkspaceRoutes";
import { useProjectStore } from "@/lib/store/projectStore";

export default function CountryPoliticsPage() {
  const params = useParams();
  const router = useRouter();
  const projectId = Number(params.id);
  const fetchProject = useProjectStore((state) => state.fetchProject);
  const refreshProject = useCallback(
    () => fetchProject(projectId),
    [fetchProject, projectId],
  );

  return (
    <PoliticsWorkspaceFrame projectId={projectId}>
      <div className="h-full overflow-y-auto p-5">
        <CountryPoliticsContent
          projectId={projectId}
          onChanged={refreshProject}
          onOpenIdeologies={() =>
            router.push(
              politicsWorkspaceHref(
                projectId,
                POLITICS_WORKSPACE_ROUTES.ideologies,
              ),
            )
          }
        />
      </div>
    </PoliticsWorkspaceFrame>
  );
}
