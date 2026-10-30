"use client";

import { useParams } from "@/platform/navigation";
import { StartHereGuide } from "@/components/start/StartHereGuide";

export default function StartPage() {
  const params = useParams();
  return <StartHereGuide projectId={Number(params.id)} />;
}
