"use client";

import { useParams } from "@/platform/navigation";
import { SuperEventsContent } from "@/components/workbench/SuperEventsContent";

export default function SuperEventsPage() {
  const params = useParams();
  const projectId = Number(params.id);
  return <SuperEventsContent projectId={projectId} />;
}
