export type EventType =
  | "country_event"
  | "news_event"
  | "state_event"
  | "unit_leader_event"
  | "operative_leader_event";

export const EVENT_TYPE_OPTIONS: {
  value: EventType;
  label: string;
  label_i18n_key: string;
}[] = [
  {
    value: "country_event",
    label: "Country Event / 国家事件",
    label_i18n_key: "countryEvent",
  },
  {
    value: "news_event",
    label: "News Event / 新闻事件",
    label_i18n_key: "newsEvent",
  },
  {
    value: "state_event",
    label: "State Event / 州事件",
    label_i18n_key: "stateEvent",
  },
  {
    value: "unit_leader_event",
    label: "Unit Leader Event / 将领事件",
    label_i18n_key: "unitLeaderEvent",
  },
  {
    value: "operative_leader_event",
    label: "Operative Event / 间谍事件",
    label_i18n_key: "operativeEvent",
  },
];

export interface EventNamespace {
  id: number;
  project_id: number;
  namespace: string;
  country_tag: string;
  scope_kind: string;
  shared_country_tags: string[] | null;
  created_at: string;
  updated_at: string;
  event_count: number;
}

export interface EventOption {
  name: string;
  trigger: string;
  effects: string;
  hidden_effects: string;
  ai_chance: {
    base?: number;
    factor?: number;
    modifiers?: { factor?: number; trigger?: string }[];
  };

  name_en?: string;
  name_zh?: string;
  display_name_locs?: Record<string, string>;

  effect_tooltip_en?: string;
  effect_tooltip_zh?: string;
  effect_tooltip_locs?: Record<string, string>;
}

export interface GameEvent {
  id: number;
  project_id: number;
  namespace_id: number;
  event_id: string;
  event_number: number;
  event_type: EventType;
  title: string | { text: string; trigger: string }[];
  description: string | { text: string; trigger: string }[];
  picture: string;
  is_triggered_only: boolean;
  fire_only_once: boolean;
  hidden: boolean;
  major: boolean;
  trigger: string;
  mean_time_to_happen: string;
  immediate: string;
  after: string;
  options: EventOption[];
  timeout_days: number;
  raw_extras: Record<string, unknown>;
  country_tag: string;
  scope_kind: string;
  shared_country_tags: string[] | null;
  title_en: string;
  title_zh: string;
  desc_en: string;
  desc_zh: string;

  display_name_locs: Record<string, string>;

  description_locs: Record<string, string>;

  auto_fire_on_startup: boolean;
  auto_fire_after_days: number;

  on_action_attach?: string;

  source?: string;
  created_at: string;
  updated_at: string;
}

export const EVENT_ON_ACTION_HOOKS: {
  value: string;
  label: string;
  label_i18n_key: string;
}[] = [
  {
    value: "",
    label: "无 — 用 trigger/MTTH 触发 (None — use trigger/MTTH)",
    label_i18n_key: "triggerMtth",
  },
  {
    value: "on_startup",
    label: "on_startup · 开局自动触发",
    label_i18n_key: "onStartup",
  },
  { value: "on_war", label: "on_war · 开战时", label_i18n_key: "onWar" },
  {
    value: "on_capitulation",
    label: "on_capitulation · 投降后",
    label_i18n_key: "onCapitulation",
  },
  {
    value: "on_peaceconference_ended",
    label: "on_peaceconference_ended · 和平结束后",
    label_i18n_key: "onPeaceconferenceEnded",
  },
  {
    value: "on_monthly",
    label: "on_monthly · 每月结算",
    label_i18n_key: "onMonthly",
  },
  {
    value: "on_weekly",
    label: "on_weekly · 每周结算",
    label_i18n_key: "onWeekly",
  },
  {
    value: "on_new_term_election",
    label: "on_new_term_election · 选举",
    label_i18n_key: "onNewTermElection",
  },
  {
    value: "on_state_control_changed",
    label: "on_state_control_changed · 州控制变更",
    label_i18n_key: "onStateControlChanged",
  },
  {
    value: "on_unit_leader_promote_from_ranks_veteran",
    label: "on_unit_leader_promote_from_ranks_veteran · 老兵将领晋升",
    label_i18n_key: "onUnitLeaderPromoteFromRanksVeteran",
  },
  {
    value: "on_unit_leader_promote_from_ranks_green",
    label: "on_unit_leader_promote_from_ranks_green · 新兵将领晋升",
    label_i18n_key: "onUnitLeaderPromoteFromRanksGreen",
  },
  {
    value: "on_ace_promoted",
    label: "on_ace_promoted · 王牌飞行员晋升",
    label_i18n_key: "onAcePromoted",
  },
  {
    value: "on_government_change",
    label: "on_government_change · 政府变更",
    label_i18n_key: "onGovernmentChange",
  },
  {
    value: "on_war_relation_added",
    label: "on_war_relation_added · 战争关系加",
    label_i18n_key: "onWarRelationAdded",
  },
  {
    value: "on_release_as_puppet",
    label: "on_release_as_puppet · 释放为傀儡",
    label_i18n_key: "onReleaseAsPuppet",
  },
  {
    value: "on_release_as_free",
    label: "on_release_as_free · 释放为自由国家",
    label_i18n_key: "onReleaseAsFree",
  },
  {
    value: "on_civil_war_end",
    label: "on_civil_war_end · 内战结束",
    label_i18n_key: "onCivilWarEnd",
  },
  {
    value: "on_civil_war_end_before_annexation",
    label: "on_civil_war_end_before_annexation · 内战吞并前",
    label_i18n_key: "onCivilWarEndBeforeAnnexation",
  },
  {
    value: "on_annex",
    label: "on_annex · 吞并国家",
    label_i18n_key: "onAnnex",
  },
  {
    value: "on_subject_free",
    label: "on_subject_free · 附属国独立",
    label_i18n_key: "onSubjectFree",
  },
  {
    value: "on_subject_annexed",
    label: "on_subject_annexed · 宗主吞并附属",
    label_i18n_key: "onSubjectAnnexed",
  },
  {
    value: "on_puppet",
    label: "on_puppet · 变附属国",
    label_i18n_key: "onPuppet",
  },
  {
    value: "on_join_faction",
    label: "on_join_faction · 加入阵营",
    label_i18n_key: "onJoinFaction",
  },
  {
    value: "on_create_faction",
    label: "on_create_faction · 创建阵营",
    label_i18n_key: "onCreateFaction",
  },
  {
    value: "on_leave_faction",
    label: "on_leave_faction · 离开阵营",
    label_i18n_key: "onLeaveFaction",
  },
  {
    value: "on_offer_join_faction",
    label: "on_offer_join_faction · 邀请加入阵营",
    label_i18n_key: "onOfferJoinFaction",
  },
  {
    value: "on_naval_invasion",
    label: "on_naval_invasion · 登陆战",
    label_i18n_key: "onNavalInvasion",
  },
  {
    value: "on_paradrop",
    label: "on_paradrop · 空降",
    label_i18n_key: "onParadrop",
  },
  {
    value: "on_border_war_lost",
    label: "on_border_war_lost · 边境战败",
    label_i18n_key: "onBorderWarLost",
  },
  {
    value: "on_assume_faction_leadership",
    label: "on_assume_faction_leadership · 接任阵营领袖",
    label_i18n_key: "onAssumeFactionLeadership",
  },
  {
    value: "on_government_exiled",
    label: "on_government_exiled · 流亡政府",
    label_i18n_key: "onGovernmentExiled",
  },
];
