export const CUSTOM_VOCAB_SENTINEL = "__custom_vocab__";

export interface VocabularySelection {
  selectValue: string;

  freeText: boolean;

  isCustom: boolean;

  showUnknownNote: boolean;
}

export function resolveVocabularySelection(
  value: string | null | undefined,
  vanillaValues: readonly string[],
  baseModValues: readonly string[],
  explicitCustomMode = false,
): VocabularySelection {
  const v = (value ?? "").trim();
  const isVanilla = v !== "" && vanillaValues.includes(v);
  const isBaseMod = v !== "" && baseModValues.includes(v);

  if (v === "" && !explicitCustomMode) {
    return {
      selectValue: "",
      freeText: false,
      isCustom: false,
      showUnknownNote: false,
    };
  }

  const freeText = explicitCustomMode || (v !== "" && !isVanilla && !isBaseMod);

  return {
    selectValue: freeText ? CUSTOM_VOCAB_SENTINEL : v,
    freeText,

    isCustom: v !== "" && !isVanilla,
    showUnknownNote:
      v !== "" && !isVanilla && baseModValues.length > 0 && !isBaseMod,
  };
}

export function baseModOnlyOptions(
  vanillaValues: readonly string[],
  baseModValues: readonly string[],
): string[] {
  const seen = new Set(vanillaValues);
  const out: string[] = [];
  for (const v of baseModValues) {
    const t = (v ?? "").trim();
    if (!t || seen.has(t)) continue;
    seen.add(t);
    out.push(t);
  }
  return out.sort();
}
