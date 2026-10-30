import type { ReactNode } from "react";

import { PoliticsWorkspaceNav } from "./PoliticsWorkspaceNav";

export function PoliticsWorkspaceFrame({
  projectId,
  children,
}: {
  projectId: number;
  children: ReactNode;
}) {
  return (
    <section className="flex h-full min-h-0 flex-col bg-background">
      <PoliticsWorkspaceNav projectId={projectId} />
      <div className="min-h-0 flex-1">{children}</div>
    </section>
  );
}
