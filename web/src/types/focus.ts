import type { SaveDiagnostic } from "@/lib/saveDiagnostics";

export type FocusTreeKind = "primary" | "shared_pool";

export interface SharedFocusRef {
  focus_id: string;
  offset_x: number;
  offset_y: number;
}

export type InitialShowPosition =
  | { kind: "focus"; focus_id: string }
  | {
      kind: "coordinates";
      x: number;
      y: number;
      offsets: { x: number; y: number; trigger: string }[];
    }
  | { kind: "legacy_ambiguous"; raw: string }
  | { kind: "invalid"; raw: unknown; raw_text: string };

export interface FocusFilePolicy {
  source_file: string;
  origin_kind: string;
  tree_ids: string[];
  matched_tree_ids: string[];
  emit_mode: string;
  target_path: string;
  target_layer: string;
  decision_source: string;
}

export interface FocusTree {
  edit_revision?: string;
  diagnostics?: readonly SaveDiagnostic[];
  id: number;
  project_id: number;
  tree_id: string;
  country_tag: string | null;
  is_default: boolean;
  continuous_focus_position_x: number;
  continuous_focus_position_y: number;
  initial_show_position: InitialShowPosition | null;

  initial_show_focus: string;
  tree_kind: FocusTreeKind;
  shared_focus_refs: SharedFocusRef[];
  source_file: string;
  file_policy?: FocusFilePolicy | null;
  scope_kind: string;
  shared_country_tags: string[] | null;
  raw_extras?: Record<string, unknown>;
  nodes: FocusNode[];
}

export interface FocusNode {
  tree_edit_revision?: string;
  diagnostics?: readonly SaveDiagnostic[];
  tree_diagnostics?: readonly SaveDiagnostic[];
  id: number;
  focus_tree_id: number;
  focus_id: string;
  name: string;
  icon: string;
  cost: number;
  x: number;
  y: number;
  relative_position_id: string;
  prerequisites: string[][];
  mutually_exclusive: string[];
  completion_reward: string;
  available: string;
  bypass: string;
  cancel: string;
  select_effect: string;
  ai_will_do: string;
  search_filters: string[];

  cancel_if_invalid: boolean;
  continue_if_invalid: boolean;
  available_if_capitulated: boolean;

  will_lead_to_war_with: string[];
  raw_extras: Record<string, unknown>;
  allow_branch_ast: Record<string, unknown>;
  load_focus_tree_rules: Record<string, unknown>[];
  name_en: string;
  name_zh: string;
  desc_en: string;
  desc_zh: string;

  display_name_locs: Record<string, string>;

  description_locs: Record<string, string>;

  effect_tooltip_locs: Record<string, string>;

  source?: string;

  offset?: { trigger: string; x: number; y: number }[];
  dynamic_icon?: { icon: string; trigger: string }[];
  text_icon?: string;
}

export type FocusNodeCreateInput = Partial<FocusNode> & {
  on_conflict?: "error" | "suffix";
};
