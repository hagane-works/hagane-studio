import { request } from "@/lib/api/_client";
import type {
  IdeologyRegistry,
  IdeologySubEntry,
} from "@/lib/ideology/registry";

type Colors = Readonly<Record<string, string>>;

export interface ReferenceTables {
  ideologies: Pick<IdeologyRegistry, "groups" | "sub_ideologies">;

  text_colors: { global: Colors; fonts: Readonly<Record<string, Colors>> };
}

export const REFERENCE_TABLES_KEY = "/app/reference-tables";

const NO_COLORS: Colors = {};
let tables: ReferenceTables = {
  ideologies: { groups: [], sub_ideologies: [] },
  text_colors: { global: NO_COLORS, fonts: {} },
};

export async function loadReferenceTables(): Promise<ReferenceTables> {
  setReferenceTables(await request<unknown>(REFERENCE_TABLES_KEY));
  return tables;
}

export function setReferenceTables(raw: unknown): void {
  const root = record(raw);
  const ideologies = record(root.ideologies);
  const colors = record(root.text_colors);
  tables = {
    ideologies: {
      groups: list(ideologies.groups).filter(
        (group): group is string => typeof group === "string",
      ),
      sub_ideologies: list(ideologies.sub_ideologies).flatMap(
        (entry): IdeologySubEntry[] => {
          const sub = record(entry);
          if (typeof sub.id !== "string" || !sub.id) return [];
          const out: IdeologySubEntry = {
            id: sub.id,
            group: typeof sub.group === "string" ? sub.group : "",
          };
          if (sub.name && typeof sub.name === "object")
            out.name = sub.name as IdeologySubEntry["name"];
          if (sub.random_disabled === true) out.random_disabled = true;
          return [out];
        },
      ),
    },
    text_colors: {
      global: colorMap(colors.global),
      fonts: Object.fromEntries(
        Object.entries(record(colors.fonts)).map(([font, map]) => [
          font,
          colorMap(map),
        ]),
      ),
    },
  };
}

export const vanillaIdeologies = () => tables.ideologies;

export const textColors = () => tables.text_colors.global;

export const fontTextColors = (font: string): Colors =>
  tables.text_colors.fonts[font] ?? NO_COLORS;

function record(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}

function list(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function colorMap(value: unknown): Colors {
  return Object.fromEntries(
    Object.entries(record(value)).filter(
      (entry): entry is [string, string] => typeof entry[1] === "string",
    ),
  );
}
