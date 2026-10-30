"use client";

import { useParams } from "@/platform/navigation";
import { ScriptedLocalisationsContent } from "@/components/workbench/ScriptedLocalisationsContent";

export default function ScriptedLocalisationsPage() {
  const params = useParams();
  const projectId = Number(params.id);
  return <ScriptedLocalisationsContent projectId={projectId} />;
}
