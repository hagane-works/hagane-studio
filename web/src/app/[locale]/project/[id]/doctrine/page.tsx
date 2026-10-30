"use client";

import { useParams } from "@/platform/navigation";
import { DoctrinesContent } from "@/components/workbench/DoctrinesContent";

export default function DoctrinePage() {
  const params = useParams();
  const projectId = Number(params.id);
  return <DoctrinesContent projectId={projectId} />;
}
