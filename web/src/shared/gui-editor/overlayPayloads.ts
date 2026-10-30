import type { TextKind } from "@/lib/guiRenderApi";

export interface BuildSetTextPayloadOptions {
  target: string;
  originalText: string | null | undefined;
  originalKind: TextKind | null | undefined;
  langTexts: Record<string, string>;
  existingPayload?: Record<string, unknown> | null;
}

export function stableGuiLocalisationKey(target: string): string {
  const readable =
    target
      .normalize("NFKD")
      .replace(/[\u0300-\u036f]/g, "")
      .replace(/[^A-Za-z0-9]+/g, "_")
      .replace(/^_+|_+$/g, "")
      .toUpperCase()
      .slice(0, 96) || "NODE";

  let hash = 0x811c9dc5;
  for (let index = 0; index < target.length; index += 1) {
    hash ^= target.charCodeAt(index);
    hash = Math.imul(hash, 0x01000193);
  }
  const suffix = (hash >>> 0).toString(16).padStart(8, "0").toUpperCase();
  return `GUI_${readable}_${suffix}`;
}

function nonEmptyString(value: unknown): string | null {
  if (typeof value !== "string") return null;
  const trimmed = value.trim();
  return trimmed ? trimmed : null;
}

export function buildSetTextPayload({
  target,
  originalText,
  originalKind,
  langTexts,
  existingPayload,
}: BuildSetTextPayloadOptions): Record<string, unknown> {
  const existingKey = nonEmptyString(existingPayload?.key);
  if (existingKey) {
    return {
      key: existingKey,
      is_new:
        typeof existingPayload?.is_new === "boolean"
          ? existingPayload.is_new
          : true,
      lang_texts: { ...langTexts },
    };
  }

  const originalKey = nonEmptyString(originalText);
  const canReuseOriginal =
    originalKey !== null && (originalKind === "loc" || originalKind === "key");
  return {
    key: canReuseOriginal ? originalKey : stableGuiLocalisationKey(target),
    is_new: !canReuseOriginal,
    lang_texts: { ...langTexts },
  };
}

export function buildSwapSpritePayload(
  gfxRef: string,
): Record<string, unknown> {
  return { gfx_ref: gfxRef };
}

export function readSetTextLangTexts(
  payload: Record<string, unknown> | null | undefined,
  fallbackDisplay: string,
): Record<string, string> {
  const result: Record<string, string> = {};
  const legacyLangTexts = payload?.lang_texts;
  if (
    legacyLangTexts &&
    typeof legacyLangTexts === "object" &&
    !Array.isArray(legacyLangTexts)
  ) {
    for (const [language, value] of Object.entries(legacyLangTexts)) {
      if (typeof value === "string") result[language] = value;
    }
  }

  if (result.en === undefined && typeof payload?.text === "string") {
    result.en = payload.text;
  }
  if (result.en === undefined && fallbackDisplay !== "") {
    result.en = fallbackDisplay;
  }
  return result;
}
