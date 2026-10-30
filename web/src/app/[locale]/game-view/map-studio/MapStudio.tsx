"use client";

import { useState } from "react";
import { MapEditor } from "./MapEditor";
import { ModMapImport } from "./ModMapImport";
import { NewWorldStudio } from "./NewWorldStudio";

export function MapStudio({
  projectId,
  date,
  width,
  height,
  mapVersion,
  onClose,
  onWorldSaved,
  onMapChanged,
}: {
  projectId: number;

  date: string;

  width: number;
  height: number;

  mapVersion: number;
  onClose: () => void;

  onWorldSaved: (states: number) => Promise<void> | void;

  onMapChanged?: () => void;
}) {
  const [way, setWay] = useState<"edit" | "new" | "mod">("edit");
  if (way === "new") {
    return (
      <NewWorldStudio
        projectId={projectId}
        date={date}
        width={width}
        height={height}
        onClose={onClose}
        onBack={() => setWay("edit")}
        onSaved={onWorldSaved}
      />
    );
  }
  if (way === "mod") {
    return (
      <ModMapImport
        projectId={projectId}
        date={date}
        width={width}
        height={height}
        onClose={onClose}
        onBack={() => setWay("edit")}
        onSaved={onWorldSaved}
      />
    );
  }
  return (
    <MapEditor
      projectId={projectId}
      date={date}
      width={width}
      height={height}
      mapVersion={mapVersion}
      onClose={onClose}
      onNewWorld={() => setWay("new")}
      onImportMod={() => setWay("mod")}
      onMapChanged={onMapChanged}
    />
  );
}
