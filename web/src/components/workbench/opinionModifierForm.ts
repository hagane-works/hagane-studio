import type {
  OpinionModifier,
  OpinionModifierWrite,
} from "@/types/opinionModifier";

export interface OpinionModifierDraft {
  modifier_id: string;
  country_tag: string;
  value: string;
  decay: string;
  days: string;
  months: string;
  years: string;
  min_trust: string;
  max_trust: string;
  trade: "" | "yes" | "no";
  preserved_known_fields: Partial<Record<OpinionModifierKnownField, string>>;
}

export type OpinionModifierNumericField =
  "value" | "decay" | "days" | "months" | "years" | "min_trust" | "max_trust";

export type OpinionModifierKnownField = OpinionModifierNumericField | "trade";

export const OPINION_MODIFIER_NUMERIC_FIELDS: readonly OpinionModifierNumericField[] =
  ["value", "decay", "days", "months", "years", "min_trust", "max_trust"];

export const EMPTY_OPINION_MODIFIER_DRAFT: OpinionModifierDraft = {
  modifier_id: "",
  country_tag: "",
  value: "",
  decay: "",
  days: "",
  months: "",
  years: "",
  min_trust: "",
  max_trust: "",
  trade: "",
  preserved_known_fields: {},
};

function scalarText(value: number | string | null | undefined): string {
  return value === null || value === undefined ? "" : String(value);
}

export function opinionModifierToDraft(
  row: OpinionModifier,
): OpinionModifierDraft {
  const preservedKnownFields: Partial<
    Record<OpinionModifierKnownField, string>
  > = {};
  const numericText = (field: OpinionModifierNumericField): string => {
    const raw = row.body?.[field];
    if (typeof raw === "string") {
      preservedKnownFields[field] = raw;
      return "";
    }
    return scalarText(row[field]);
  };
  const rawTrade = row.body?.trade;
  const normalizedTrade =
    typeof rawTrade === "string" ? rawTrade.trim().toLowerCase() : rawTrade;
  const tradeIsKnown =
    rawTrade === undefined ||
    typeof rawTrade === "boolean" ||
    normalizedTrade === "yes" ||
    normalizedTrade === "no" ||
    normalizedTrade === "true" ||
    normalizedTrade === "false" ||
    normalizedTrade === "1" ||
    normalizedTrade === "0";
  if (!tradeIsKnown && typeof rawTrade === "string") {
    preservedKnownFields.trade = rawTrade;
  }
  return {
    modifier_id: row.modifier_id,
    country_tag: row.country_tag ?? "",
    value: numericText("value"),
    decay: numericText("decay"),
    days: numericText("days"),
    months: numericText("months"),
    years: numericText("years"),
    min_trust: numericText("min_trust"),
    max_trust: numericText("max_trust"),
    trade: preservedKnownFields.trade
      ? ""
      : row.trade === null
        ? ""
        : row.trade
          ? "yes"
          : "no",
    preserved_known_fields: preservedKnownFields,
  };
}

function optionalNumber(value: string): number | null {
  const trimmed = value.trim();
  if (!trimmed) return null;
  const number = Number(trimmed);
  if (!Number.isFinite(number)) {
    throw new Error(`Invalid number: ${value}`);
  }
  return number;
}

export function opinionDraftToPayload(
  draft: OpinionModifierDraft,
): OpinionModifierWrite {
  const modifierId = draft.modifier_id.trim();
  if (!modifierId) throw new Error("modifier_id is required");
  const payload: OpinionModifierWrite = {
    modifier_id: modifierId,
    country_tag: draft.country_tag.trim().toUpperCase(),
  };
  for (const field of OPINION_MODIFIER_NUMERIC_FIELDS) {
    if (draft.preserved_known_fields[field] === undefined) {
      payload[field] = optionalNumber(draft[field]);
    }
  }
  if (draft.preserved_known_fields.trade === undefined) {
    payload.trade = draft.trade === "" ? null : draft.trade === "yes";
  }
  return payload;
}
