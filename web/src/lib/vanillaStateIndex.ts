import { useSyncExternalStore } from "react";
import { requestGameIndexed, sessionGameData } from "@/lib/api/gameIndex";

export interface VanillaStateContent {
  navalBases?: number;
  bunkers?: number;
  coastalBunkers?: number;
  dams?: number;

  landmarks?: string[];

  datedBlocks?: number;

  hasDlc?: number;

  forceLink?: boolean;

  resistance?: number;

  otherContent?: number;
}

export interface VanillaStateIndex {
  content: Readonly<Record<number, VanillaStateContent>>;

  dlc: Readonly<Record<number, readonly string[]>>;
}

export const EMPTY_VANILLA_STATE_INDEX: VanillaStateIndex = {
  content: {},
  dlc: {},
};

const stateIndex = sessionGameData(
  () =>
    Promise.all([
      requestGameIndexed<{ states: Record<number, VanillaStateContent> }>(
        "/go/vanilla/state-content",
      ),
      requestGameIndexed<{ states: Record<number, string[]> }>(
        "/go/vanilla/dlc-states",
      ),
    ]).then(([content, dlc]): VanillaStateIndex => ({
      content: content.states ?? {},
      dlc: dlc.states ?? {},
    })),
  EMPTY_VANILLA_STATE_INDEX,
);

export function useVanillaStateIndex(): VanillaStateIndex {
  return useSyncExternalStore(
    stateIndex.subscribe,
    stateIndex.current,
    stateIndex.current,
  );
}
