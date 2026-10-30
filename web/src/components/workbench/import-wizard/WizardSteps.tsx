import { Check } from "lucide-react";
import { useTranslations } from "use-intl";
import type { WizardStep } from "./wizardStep";

export interface WizardStepsProps {
  current: WizardStep;

  finished: boolean;
}

const STEP_KEYS = [
  "wizardStep1",
  "wizardStep2",
  "wizardStep3",
  "wizardStep4",
] as const;

export function WizardSteps({ current, finished }: WizardStepsProps) {
  const t = useTranslations("unifiedModImport");
  return (
    <ol className="flex items-center gap-1" aria-label={t("wizardAria")}>
      {STEP_KEYS.map((key, i) => {
        const stepNo = (i + 1) as WizardStep;
        const isDone = finished || stepNo < current;
        const isCurrent = !finished && stepNo === current;
        const isRunning = isCurrent && stepNo === 4;
        return (
          <li key={key} className="flex min-w-0 items-center gap-1">
            {i > 0 && (
              <span
                aria-hidden
                className={`h-px w-4 shrink-0 sm:w-6 ${
                  isDone || isCurrent ? "bg-primary/50" : "bg-border"
                }`}
              />
            )}
            <span
              className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-full border text-micro font-semibold ${
                isDone
                  ? "border-primary/50 bg-primary/10 text-primary"
                  : isCurrent
                    ? `border-primary bg-primary text-primary-foreground ${
                        isRunning ? "animate-pulse-ring" : ""
                      }`
                    : "border-border text-muted-foreground"
              }`}
              aria-hidden
            >
              {isDone ? (
                <Check className="h-3 w-3" strokeWidth={2.5} />
              ) : (
                stepNo
              )}
            </span>

            <span
              className={`truncate text-xs ${
                isCurrent
                  ? "text-foreground"
                  : isDone
                    ? "text-muted-foreground"
                    : "text-muted-foreground/70"
              }`}
              aria-current={isCurrent ? "step" : undefined}
            >
              {t(key)}
            </span>
          </li>
        );
      })}
    </ol>
  );
}
