"use client";

import { useLocale, useTranslations } from "use-intl";
import { MapProblemError, mapProblemMessage } from "./mapProblems";

export function useMapErrorText(): (err: unknown) => string {
  const t = useTranslations("mapStudio");
  const locale = useLocale();
  return (err) =>
    err instanceof MapProblemError
      ? mapProblemMessage(err.problem, t, locale)
      : err instanceof Error
        ? err.message
        : String(err);
}
