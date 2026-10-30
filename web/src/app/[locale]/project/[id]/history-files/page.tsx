"use client";

import { useParams } from "@/platform/navigation";
import { HistoryFilesContent } from "@/components/workbench/HistoryFilesContent";

export default function HistoryFilesPage() {
  const params = useParams();
  const projectId = Number(params.id);
  return <HistoryFilesContent projectId={projectId} />;
}
