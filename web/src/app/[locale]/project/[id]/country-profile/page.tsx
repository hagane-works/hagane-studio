"use client";

import { useCallback } from "react";
import { useParams } from "@/platform/navigation";
import { useRouter } from "@/i18n/navigation";
import { CountryProfilePage as CountryProfileContent } from "@/components/workbench/country-profile";
import {
  POLITICS_WORKSPACE_ROUTES,
  politicsWorkspaceHref,
} from "@/lib/workbench/politicsWorkspaceRoutes";
import { useProjectStore } from "@/lib/store/projectStore";

export default function CountryProfilePage() {
  const params = useParams();
  const router = useRouter();
  const projectId = Number(params.id);
  const fetchProject = useProjectStore((state) => state.fetchProject);
  const refreshProject = useCallback(
    () => fetchProject(projectId),
    [fetchProject, projectId],
  );

  return (
    <section className="h-full overflow-y-auto bg-background p-5">
      <CountryProfileContent
        projectId={projectId}
        onChanged={refreshProject}
        onOpenPolitics={() =>
          router.push(
            politicsWorkspaceHref(
              projectId,
              POLITICS_WORKSPACE_ROUTES.countryPolitics,
            ),
          )
        }
      />
    </section>
  );
}
