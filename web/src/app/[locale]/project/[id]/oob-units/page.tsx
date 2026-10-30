"use client";

import { useParams } from "@/platform/navigation";

import { OobUnitsEditor } from "@/components/workbench/oob-units/OobUnitsEditor";

export default function OobUnitsPage() {
  const { id } = useParams<{ id: string }>();
  return <OobUnitsEditor projectId={Number(id)} />;
}
