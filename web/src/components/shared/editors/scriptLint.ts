"use client";

import { useMemo } from "react";
import { useTranslations } from "use-intl";

import type { CwtoolRules } from "@/lib/api";
import {
  findUnknownScriptTokens,
  useVanillaScriptNames,
  type ScriptTokenIssue,
  type VanillaScriptNames,
} from "@/lib/vanillaScriptNames";
import { useLocaleStore, type Locale } from "@/lib/store/localeStore";
import { useCwtoolRules } from "@/shared/editors/cwtoolStore";
import {
  cwtoolValidate,
  type CwtoolMessageFormatter,
} from "@/shared/editors/cwtoolValidate";
import type { EditorMarker } from "@/shared/editors/ScriptEditor";

export const SCRIPTED_GUI_CUSTOM_TOKENS: ReadonlySet<string> = new Set([
  "visible",
  "triggered_by",
  "always",
  "ai_enabled",
  "ai_test_interval",
  "effects",
  "triggers",
  "properties",
  "dynamic_lists",
]);

export interface ScriptLintResult {
  markers: EditorMarker[];
  unknownTokens: ScriptTokenIssue[];
}

export const EMPTY_SCRIPT_LINT_RESULT: ScriptLintResult = {
  markers: [],
  unknownTokens: [],
};

export function lintScriptBlock(
  script: string | null | undefined,
  rules: CwtoolRules | null,
  vanilla: VanillaScriptNames,
  locale: Locale,
  formatMessage?: CwtoolMessageFormatter,
): ScriptLintResult {
  const normalizedScript = typeof script === "string" ? script : "";
  if (!normalizedScript.trim()) return EMPTY_SCRIPT_LINT_RESULT;
  return {
    markers: cwtoolValidate(
      normalizedScript,
      rules,
      locale === "zh" ? "zh" : "en",
      { formatMessage },
    ),
    unknownTokens: findUnknownScriptTokens(
      normalizedScript,
      vanilla,
      SCRIPTED_GUI_CUSTOM_TOKENS,
    ),
  };
}

export function useScriptLint(script: string): ScriptLintResult {
  const rules = useCwtoolRules();
  const vanilla = useVanillaScriptNames();
  const locale = useLocaleStore((state) => state.locale);
  const tMisc = useTranslations("misc");

  return useMemo(
    () =>
      lintScriptBlock(script, rules, vanilla, locale, (key, params) =>
        tMisc(key, params),
      ),
    [script, rules, vanilla, locale, tMisc],
  );
}
