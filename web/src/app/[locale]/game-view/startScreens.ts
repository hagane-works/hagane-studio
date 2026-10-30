import { COUNTRY_PICKER } from "./CountrySelectLayer";
import { SCENARIO_PICKER } from "./ScenarioSelectLayer";

export function onStartScreens(panel: string | null): boolean {
  return panel === SCENARIO_PICKER || panel === COUNTRY_PICKER;
}

export function startScreensToggle(panel: string | null): string {
  return onStartScreens(panel) ? "countrypoliticsview" : SCENARIO_PICKER;
}
