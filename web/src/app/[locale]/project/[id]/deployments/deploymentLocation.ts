import type { State } from "@/types";

export function stateForProvince(
  province: number | null,
  states: State[],
): number | null {
  if (province == null) return null;
  return (
    states.find((state) => (state.provinces ?? []).includes(province))
      ?.state_id ?? null
  );
}

export function isProvinceAvailable(
  province: number | null,
  states: State[],
): boolean {
  return province == null || stateForProvince(province, states) !== null;
}
