"use client";

import { useParams } from "@/platform/navigation";
import { ScriptedSnippetsContent } from "@/components/workbench/ScriptedSnippetsContent";

export default function ScriptedSnippetsPage() {
  const params = useParams();
  const projectId = Number(params.id);
  return <ScriptedSnippetsContent projectId={projectId} />;
}
