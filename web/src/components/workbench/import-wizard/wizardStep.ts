import type { Phase } from "./types";

export type WizardStep = 1 | 2 | 3 | 4;

export interface WizardStepInputs {
  phase: Phase;

  showPreflight: boolean;

  reImporting: boolean;

  countryPreparationInFlight: boolean;
}

export function deriveWizardStep({
  phase,
  showPreflight,
  reImporting,
  countryPreparationInFlight,
}: WizardStepInputs): WizardStep {
  if (reImporting || countryPreparationInFlight) return 4;
  if (phase === "parsing" || phase === "uploading" || phase === "done")
    return 4;
  if (phase === "scanned") return showPreflight ? 3 : 2;

  return 1;
}
