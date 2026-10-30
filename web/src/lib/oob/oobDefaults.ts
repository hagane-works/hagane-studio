export const VANILLA_START_YEAR = 1936;

export function startYearFromStartDate(
  startDate?: string | null,
): number | null {
  const head =
    String(startDate ?? "")
      .split(".")[0]
      ?.trim() ?? "";
  if (!/^\d{1,4}$/.test(head)) return null;
  const year = Number(head);
  return year > 0 ? year : null;
}

export const tagFromOob = (oob?: string | null): string =>
  String(oob ?? "")
    .split("_", 1)[0]
    .trim()
    .toUpperCase()
    .slice(0, 10);

export function normalizeCountryTag(raw?: string | null): string {
  const tag = String(raw ?? "")
    .trim()
    .toUpperCase();
  if (!tag || tag.includes("_")) return "";

  return tag.slice(0, 10);
}

export function defaultOobFileName(
  countryTag?: string | null,
  startDate?: string | null,
): string {
  const tag = normalizeCountryTag(countryTag);
  if (!tag) return "";
  return `${tag}_${startYearFromStartDate(startDate) ?? VANILLA_START_YEAR}`;
}

export function resolveOobFileName(
  existingOobFile?: string | null,
  countryTag?: string | null,
  startDate?: string | null,
): string {
  const existing = String(existingOobFile ?? "").trim();
  if (existing) return existing;
  return defaultOobFileName(countryTag, startDate);
}

function asComment(line: string): string {
  const text = String(line ?? "").trim();
  if (!text) return "#";
  return text.startsWith("#") ? text : `# ${text}`;
}

const UNITS_EXAMPLE_COMMENT =
  '# units = { division = { name = "1. Division" location = 1 division_template = "Template" } }';

const SET_OOB_PLACEHOLDER_COMMENT = '# set_oob = "<TAG>_<year>"';

export interface OobStarterOptions {
  comment: string;

  unknownTagComment: string;

  countryTag?: string | null;

  oobFile?: string | null;

  startDate?: string | null;
}

export function oobStarterScript(options: OobStarterOptions): string {
  const target = resolveOobFileName(
    options.oobFile,
    options.countryTag,
    options.startDate,
  );
  const lines = [asComment(options.comment)];
  if (target) {
    lines.push(`set_oob = "${target}"`);
  } else {
    lines.push(
      asComment(options.unknownTagComment),
      SET_OOB_PLACEHOLDER_COMMENT,
    );
  }
  lines.push(UNITS_EXAMPLE_COMMENT);
  return lines.join("\n");
}

export interface DefaultOobFileInput {
  countryTag?: string | null;

  startDate?: string | null;

  existing?: (string | null | undefined)[];
}

export function pickDefaultOobFile(input: DefaultOobFileInput): string {
  const candidates = (input.existing ?? [])
    .map((f) => String(f ?? "").trim())
    .filter(Boolean);
  const tag = normalizeCountryTag(input.countryTag);
  if (tag) {
    const owned = candidates.find((f) => tagFromOob(f) === tag);
    return owned ?? defaultOobFileName(tag, input.startDate);
  }
  return candidates[0] ?? "";
}
