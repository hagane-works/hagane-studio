"use client";

import { useParams } from "@/platform/navigation";
import { MIOsContent } from "@/components/workbench/MIOsDialog";

export default function MIOPage() {
  const params = useParams();
  const projectId = Number(params.id);
  return <MIOsContent projectId={projectId} />;
}
