export interface OpinionModifier {
  id: number;
  project_id: number;
  modifier_id: string;
  country_tag: string;
  value: number | string | null;
  decay: number | string | null;
  days: number | string | null;
  months: number | string | null;
  years: number | string | null;
  min_trust: number | string | null;
  max_trust: number | string | null;
  trade: boolean | null;
  body: Record<string, unknown>;
  raw_extras: Record<string, unknown>;
  source: "created" | "imported" | string;
  created_at?: string | null;
  updated_at?: string | null;
}

export interface OpinionModifierWrite {
  modifier_id?: string;
  country_tag?: string;
  value?: number | null;
  decay?: number | null;
  days?: number | null;
  months?: number | null;
  years?: number | null;
  min_trust?: number | null;
  max_trust?: number | null;
  trade?: boolean | null;
}
