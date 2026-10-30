import { useSyncExternalStore } from "react";
import { requestGameIndexed, sessionGameData } from "@/lib/api/gameIndex";

export type UnitNames = Readonly<
  Record<string, Readonly<Record<string, string>>>
>;

export interface VanillaUnitNames {
  subunits: UnitNames;
  equipment: UnitNames;
}

const NO_NAMES: VanillaUnitNames = { subunits: {}, equipment: {} };

const unitNames = sessionGameData(
  () =>
    requestGameIndexed<{
      subunits: Record<string, Record<string, string>>;
      equipment: Record<string, Record<string, string>>;
    }>("/go/vanilla/unit-names").then((data): VanillaUnitNames => ({
      subunits: data.subunits ?? {},
      equipment: data.equipment ?? {},
    })),
  NO_NAMES,
);

export function useVanillaUnitNames(): VanillaUnitNames {
  return useSyncExternalStore(
    unitNames.subscribe,
    unitNames.current,
    unitNames.current,
  );
}
