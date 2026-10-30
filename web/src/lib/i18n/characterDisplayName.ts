const LEGACY_CHARACTER_PLACEHOLDERS = new Set(["new character"]);

export function visibleCharacterName(value: string | null | undefined): string {
  const trimmed = (value ?? "").trim();
  return LEGACY_CHARACTER_PLACEHOLDERS.has(trimmed.toLocaleLowerCase())
    ? ""
    : trimmed;
}
