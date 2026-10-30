"use client";

import { useParams } from "@/platform/navigation";
import { BookmarksContent } from "@/components/workbench/BookmarksContent";

export default function BookmarksPage() {
  const params = useParams();
  const projectId = Number(params.id);
  return <BookmarksContent key={projectId} projectId={projectId} />;
}
