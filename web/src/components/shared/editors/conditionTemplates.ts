export interface ConditionTemplate {
  label_en: string;
  label_zh: string;
  code: string;
  desc_en: string;
  desc_zh: string;
}

export const CONDITION_TEMPLATES: ConditionTemplate[] = [
  {
    label_en: "Focus completed",
    label_zh: "已完成国策",
    code: "has_completed_focus = FOCUS_ID",
    desc_en:
      "Checks whether your country has completed the focus with the given ID. Most used for event triggers, decision visibility, and as a prerequisite for later focuses. FOCUS_ID must point to a focus defined in this mod or vanilla; a typo won't error but will always return false. Commonly paired with date > 1940.1.1 to bound the time window.",
    desc_zh:
      "检查本国是否已完成指定 ID 的国策。最常用于事件触发、决议可见条件、后续国策前置。FOCUS_ID 需指向本 mod 或原版已定义的国策，拼写错误不会报错而是永远返回 false。典型搭配 date > 1940.1.1 限定时间窗口。",
  },
  {
    label_en: "Not at war",
    label_zh: "未处于战争",
    code: "has_war = no",
    desc_en:
      "Requires your country to be in no war at all (including small border conflicts). Often gates large focuses/decisions — to stop the player switching major paths mid-war. A truce doesn't count as 'not at war'; it returns yes only once all wars end. Different from has_civil_war — civil war is queried separately.",
    desc_zh:
      "要求本国当前未处于任何战争（含小规模边境冲突）。常作为大型国策/决议的激活门槛——避免玩家打仗时切换重大路线。停战协议不算未战争；所有 war 都结束后才为 yes。与 has_civil_war 不同——内战需单独查询。",
  },
  {
    label_en: "At war",
    label_zh: "处于战争中",
    code: "has_war = yes",
    desc_en:
      "Requires your country to be in at least one war. Common as the trigger for wartime emergency events/decisions (war-industry mobilization, conscription drives). To filter 'at war with a specific country' use has_war_with = TAG for precision. Usually combined with war_support / manpower conditions to form a 'wartime' chain.",
    desc_zh:
      "要求本国当前处于至少一场战争。常用于战时紧急事件/决议（战时工业动员、征兵潮）的触发门槛。如要筛选'与特定国家交战'用 has_war_with = TAG 更精确。一般与 war_support / manpower 条件组合形成'战时剧情'链。",
  },
  {
    label_en: "Not in civil war",
    label_zh: "未内战",
    code: "has_civil_war = no",
    desc_en:
      "Requires your country to not be in a civil war. HOI4 civil wars are triggered by the start_civil_war effect (host vs breakaway faction) and are independent of has_war — having a civil war but no foreign war is entirely possible. Often used in focus chains as a 'only during a stable period' prerequisite. Returns no automatically once the civil war ends.",
    desc_zh:
      "要求本国当前未处于内战状态。HOI4 内战由 start_civil_war 效果触发（主国 vs 分裂派），与 has_war 独立——同时有内战但无外战完全可能。国策链中常作为'稳定期才能激活'的前置。内战结束后自动回 no。",
  },
  {
    label_en: "Date restriction",
    label_zh: "日期限制",
    code: "date > 1937.1.1",
    desc_en:
      "Checks the current in-game date. Format yyyy.m.d, supports > / < / =. Most used to control the timing of historical events/focuses — e.g. 'can't research the atom bomb before 1940'. The 1936 start counts from 1936.1.1; the 1939 scenario start must be accounted for before a date check. No range syntax — for a range, nest two date checks under AND.",
    desc_zh:
      "检查当前游戏日期。格式 yyyy.m.d，支持 > / < / =。最常用于控制历史事件/国策触发时机——如'1940 年前不能研究原子弹'。1936 开局从 1936.1.1 起计时；1939 剧本起点在 date 判断前要纳入考量。不支持区间写法，要范围时用 AND 嵌套两个 date。",
  },
  {
    label_en: "Stability requirement",
    label_zh: "稳定度要求",
    code: "has_stability > 0.5",
    desc_en:
      "Checks your country's stability (0.0~1.0). Stability affects factory output, political support, and event probability. Typical magnitudes: ~60-70% at start, 20-40% in war; focuses/decisions can grant +5%~15%. As a condition: > 0.5 median, > 0.7 high, < 0.3 turbulent. Compare with a decimal (0.5), not a percentage (50).",
    desc_zh:
      "检查本国稳定度（0.0~1.0）。稳定度影响工厂产量、政治支持、事件概率。典型量级：开局 60-70%、战时 20-40%、国策/决议可 +5%~15%。作为条件：> 0.5 中位、> 0.7 高稳定、< 0.3 动荡。比较用小数（0.5）不是百分比（50）。",
  },
  {
    label_en: "War support requirement",
    label_zh: "战争支持要求",
    code: "has_war_support > 0.5",
    desc_en:
      "Checks your country's war support (0.0~1.0). Determines whether you can conscript, use total-mobilization laws, wage unrestricted submarine warfare, etc. Low in peacetime (20-40%), spikes after major events / being attacked. Common thresholds: > 0.3 (puppet expedition) / > 0.5 (full conscription) / > 0.7 (total mobilization). Use a decimal.",
    desc_zh:
      "检查本国战争支持度（0.0~1.0）。决定能否征兵、用全面动员法、发动无限制潜艇战等激进政策。和平期低（20-40%），大事件/被袭击后飙升。常见阈值：> 0.3（傀儡远征）/ > 0.5（全面征兵）/ > 0.7（总动员）。数值用小数。",
  },
  {
    label_en: "Has country flag",
    label_zh: "拥有国家flag",
    code: "has_country_flag = FLAG_NAME",
    desc_en:
      "Checks whether your country holds a custom flag. A flag is a boolean state marker set by the author via set_country_flag, carrying 'X happened' across events/decisions/focuses. Often used to unlock a follow-up path after some event. Extended form: has_country_flag = { flag = X date > 30 } (held for at least 30 days). Clear with clr_country_flag.",
    desc_zh:
      "检查本国是否拥有某个自定义 flag。Flag 是脚本作者用 set_country_flag 设的布尔状态标记，跨事件/决议/国策传递'发生过 X'。常用于'某事件后解锁后续路径'。扩展形式 has_country_flag = { flag = X date > 30 }（至少持有 30 天）。用 clr_country_flag 清除。",
  },
  {
    label_en: "Is a given ideology",
    label_zh: "是某意识形态",
    code: "has_government = democratic",
    desc_en:
      "Checks your country's ruling party's main ideology. Valid values: democratic / fascism / communism / neutrality. Note it's the 'ruling party', not the 'most-supported party' — it only changes when set_politics changes ruling_party. Sub-ideologies (social_liberal / stalinism) aren't checked here; use has_government_sub_ideology or has_idea.",
    desc_zh:
      "检查本国当前执政党的主意识形态。合法值：democratic / fascism / communism / neutrality。注意是'执政党'不是'支持率最高党'——set_politics 改 ruling_party 才变。子意识形态（social_liberal / stalinism）不在此查，用 has_government_sub_ideology 或 has_idea。",
  },
  {
    label_en: "Party support",
    label_zh: "政党支持率",
    code: "democratic > 0.5",
    desc_en:
      "Checks a country's support for a given ideology (0.0~1.0). Valid forms: democratic / fascism / communism / neutrality. Different from has_government (ruling party) — support can be high without being in power. Often used as a prerequisite for taking power via a legal election (> 0.5) or a civil war erupting (opposition crosses a threshold).",
    desc_zh:
      "检查本国某意识形态的党派支持率（0.0~1.0）。合法形式：democratic / fascism / communism / neutrality。与 has_government（执政党）不同——支持率可高但未执政。常用于'合法选举上台'（> 0.5）或'内战爆发'（反对派突破阈值）前置。",
  },
  {
    label_en: "Is a puppet",
    label_zh: "是傀儡国",
    code: "is_puppet = yes",
    desc_en:
      "Checks whether your country is a puppet of any country (any autonomy level: colony, integrated puppet, political proxy, etc.). The check is very loose — any subordinate relationship counts. To filter a specific overlord use is_subject_of = TAG; for a specific autonomy use has_autonomy_state = autonomy_X. Often used for 'independence-war events only a puppet can trigger'.",
    desc_zh:
      "检查本国是否为任意国家的傀儡（任意 autonomy level：殖民地、集成傀儡、政治代理等）。判定极宽松——任何从属关系都算。筛具体宗主用 is_subject_of = TAG；筛具体 autonomy 用 has_autonomy_state = autonomy_X。常用于'傀儡才能触发的独立战争事件'。",
  },
  {
    label_en: "Not a puppet",
    label_zh: "非傀儡国",
    code: "is_puppet = no",
    desc_en:
      "Requires your country to not be a puppet (fully independent). The strict opposite of is_puppet = yes. Often a base gate for 'only sovereign states can join factions / declare war / use diplomacy' — HOI4 disables many diplomatic/military effects for puppets. Combine with has_war_with to filter 'independent sovereign + currently at war'.",
    desc_zh:
      "要求本国非傀儡（完全独立）。与 is_puppet = yes 严格相反。常作为'主权国家才能加入派系/宣战/用外交'的基础门槛——HOI4 大量外交/军事 effect 对傀儡禁用。配合 has_war_with 可筛'独立主权 + 正在战争'的剧情条件。",
  },
  {
    label_en: "Controls a state",
    label_zh: "控制省份",
    code: "controls_state = STATE_ID",
    desc_en:
      "Checks whether your country actually controls the given state (not just owns it on paper — it must be garrisoned with no foreign occupation). STATE_ID is a number (not a TAG, not a province ID). Often used for 'hold/retake a key state' story triggers. Distinction: owns_state (legal ownership) vs controls_state (actual control). When occupied in war, owns still belongs to the original country, controls switches to the occupier.",
    desc_zh:
      "检查本国是否实际控制指定 state（不只是法理拥有，必须驻军/无外军占领）。STATE_ID 为数字（非 TAG、非 province ID）。常用于'守住/夺回关键州'剧情触发。区别：owns_state（法理拥有）/ controls_state（实际控制）。战争中被占领时 owns 仍归原国、controls 变占领国。",
  },
  {
    label_en: "Manpower requirement",
    label_zh: "人力要求",
    code: "has_manpower > 100000",
    desc_en:
      "Checks your country's available manpower pool (recruited-but-not-deployed reserves) — not total population, not army size — the 'ready for the front line' number. Typical for a great power at start 50K-200K, 1M+ in total war. Common: > 100K (a mid-size country launches a campaign) / > 500K (can fight a protracted war). Check exhaustion with < 0.",
    desc_zh:
      "检查本国可用人力池（征兵未动员的预备役），不是总人口也不是陆军规模——是'随时能上前线'的数字。开局大国典型 50K-200K，全面战争 1M+。常见：> 100K（中等国家发动战役）/ > 500K（能打持久战）。耗竭检查用 < 0。",
  },
  {
    label_en: "Army size",
    label_zh: "陆军规模",
    code: "has_army_size = {\n\tsize > 20\n\ttype = infantry\n}",
    desc_en:
      "Block trigger: checks the number of division-level units. Sub-fields: size (count, supports > / < / =), type (optional, filter a unit type like infantry / armor / motorized), archetype (filter an equipment category). Without type it counts all divisions. Often used for 'have enough troops to deploy abroad'. Equivalent to num_divisions but more flexible. A full division ≈ 10K men.",
    desc_zh:
      "块 trigger：检查陆军师级单位数量。子字段：size（数量，支持 > / < / =）、type（可选，筛兵种如 infantry / armor / motorized）、archetype（筛装备大类）。不写 type 时数所有师。常用于'有足够部队才能远征'。与 num_divisions 等价但更灵活。一个满编师 ≈ 10K 人。",
  },
  {
    label_en: "NOT condition",
    label_zh: "NOT条件",
    code: "NOT = {\n\thas_idea = IDEA_NAME\n}",
    desc_en:
      "A scope-control clause — inverts the result of its inner conditions. If any sub-condition inside the NOT block is true, the whole NOT is false. Common pitfall: NOT = { A B } is NOT(A AND B) — it passes if either A or B is false. For 'neither holds' use NOT = { OR = { A B } }. When deeply nested, consider extracting a scripted_trigger for readability.",
    desc_zh:
      "作用域控制子句——反转内部条件结果。NOT 块内任一子条件为真则整个 NOT 为假。常见误区：NOT = { A B } 是 NOT(A AND B)，A 或 B 任一为 false 就通过。要'都不满足'用 NOT = { OR = { A B } }。嵌套深时考虑拆 scripted_trigger 提高可读性。",
  },
  {
    label_en: "OR condition",
    label_zh: "OR条件",
    code: "OR = {\n\thas_completed_focus = FOCUS_A\n\thas_completed_focus = FOCUS_B\n}",
    desc_en:
      "A scope-control clause — the whole OR is true when any sub-condition in the block is true. Most used for multiple-path prerequisites (completing focus A or B both activate what follows) or fallback conditions. Companions: AND (implicit by default; an outer brace is equivalent), NOT (inverts), count_triggers (require N met). Watch indentation when nesting; large mods wrap chains in a scripted_trigger.",
    desc_zh:
      "作用域控制子句——块内任一子条件为真时整个 OR 为真。最常用于'多路径前置'（完成 focus A 或 B 都能激活后续）或'兜底条件'。配套 AND（默认隐含，外层大括号等效）、NOT（反转）、count_triggers（要求满足 N 个）。嵌套时注意缩进，大 mod 常用 scripted_trigger 封装避免长链。",
  },
  {
    label_en: "Other-country condition",
    label_zh: "他国条件",
    code: "TAG = {\n\texists = yes\n\thas_war = no\n}",
    desc_en:
      "A scope change: switches the condition scope to the given TAG country, querying that country's state inside. TAG can be a 3-letter country code (USA / GER / SOV) or a scope var (FROM / ROOT / PREV / event_target). Common uses: check whether an ally has joined the war, or an enemy's government type. FROM in an effect chain means 'the triggerer', ROOT is the 'outermost scope'. Put exists = yes first to avoid accessing a non-existent TAG.",
    desc_zh:
      "scope change：把条件作用域切换到指定 TAG 国家，内部条件查询该国状态。TAG 可为 3 字母国家代码（USA / GER / SOV）或 scope var（FROM / ROOT / PREV / event_target）。常见用法：检查盟友是否参战、敌国政府类型。FROM 在 effect 链里指'触发者'，ROOT 是'最外层作用域'。先 exists = yes 避免访问不存在的 TAG。",
  },
  {
    label_en: "Variable value check",
    label_zh: "变量值检查",
    code: "check_variable = { var = VAR_NAME value = 1 compare = greater_than }",
    desc_en:
      "Queries a numeric variable set by set_variable. Full syntax { var = NAME value = N compare = OP }, also short forms { NAME > N } / { NAME < N } / { NAME = N }. Valid compare values: less_than / less_than_or_equals / equals / greater_than_or_equals / greater_than / not_equals. Can compare two variables (put another variable name in the value field). Often used for mission/decision progress checks.",
    desc_zh:
      "查询 set_variable 设的数值变量。完整语法 { var = NAME value = N compare = OP },也可简写 { NAME > N } / { NAME < N } / { NAME = N }。compare 合法值:less_than / less_than_or_equals / equals / greater_than_or_equals / greater_than / not_equals。可比较两个变量(value 字段填另一个变量名)。常用于 mission/decision 的进度条件检查。",
  },
  {
    label_en: "Has state flag",
    label_zh: "拥有 state flag",
    code: "has_state_flag = FLAG_NAME",
    desc_en:
      "Queries a state-level boolean marker within a state scope. Same mechanism as has_country_flag but on a state rather than a country. Used with set_state_flag / clr_state_flag. Common uses: state-level mission progress (has a state finished surveying) / regional story markers. State-scope switch: `every_owned_state = { limit = { has_state_flag = X } ... }`.",
    desc_zh:
      "state-scope 内查询 state-level 布尔标记。跟 has_country_flag 同款机制但作用于 state 而非 country。配 set_state_flag / clr_state_flag 用。常见用法:州级任务进度(state 是否完成探勘)/ 区域剧情标记。state-scope 切换:`every_owned_state = { limit = { has_state_flag = X } ... }`。",
  },
  {
    label_en: "Fully controls state",
    label_zh: "完全控制 state",
    code: "is_fully_controlled_by = ROOT",
    desc_en:
      "Within a state scope, queries whether the state is fully held by the TAG (legal ownership + actual occupation + no enemy cores). Distinct from owns_state (legal only, may not control) / controls_state (actual occupation, may not own). The strictest 'fully ours' check. Often used for mission conditions like 'all my core states are mine'. The target can be ROOT / FROM / TAG.",
    desc_zh:
      "state-scope 内查询该 state 是否完全归 TAG 控制(法理拥有 + 实际占领 + 无敌军核心)。区别 owns_state(只法理拥有 不一定控制) / controls_state(实际占领 不一定法理)。是最严格的 'fully ours' 检查。常用于 mission 'all my core states are mine' 类条件。TARGET 可写 ROOT / FROM / TAG。",
  },
  {
    label_en: "DLC check",
    label_zh: "DLC 检查",
    code: 'has_dlc = "Together for Victory"',
    desc_en:
      "Queries whether the user has the given DLC enabled. The value is the DLC's full English name (in double quotes, case-sensitive). Common DLC names: 'Together for Victory' / 'Death or Dishonor' / 'Waking the Tiger' / 'Man the Guns' / 'La Resistance' / 'Battle for the Bosporus' / 'No Step Back' / 'By Blood Alone' / 'Arms Against Tyranny' / 'Trial of Allegiance' / 'Götterdämmerung'. Often used as a fallback for content only available with a DLC.",
    desc_zh:
      "查询用户是否启用了指定 DLC。值是 DLC 完整英文名(双引号包裹,严格区分大小写)。常用 DLC 名:'Together for Victory' / 'Death or Dishonor' / 'Waking the Tiger' / 'Man the Guns' / 'La Resistance' / 'Battle for the Bosporus' / 'No Step Back' / 'By Blood Alone' / 'Arms Against Tyranny' / 'Trial of Allegiance' / 'Götterdämmerung'。常用于 mod 'DLC 解锁后才有的内容' 兜底。",
  },
  {
    label_en: "Any owned state matches",
    label_zh: "任意国有州满足",
    code: "any_owned_state = {\n\tinfrastructure > 5\n\tis_coastal = yes\n}",
    desc_en:
      "Trigger: true if any one owned state of your country meets the inner conditions. It's a trigger (condition), not an effect (to execute → use random_owned_state). Companion all_owned_state (all must match). Same family: any_controlled_state / any_state (any on the whole map) / any_neighbor_country. Often used for gates like 'is there any coastal state where I can build a port'.",
    desc_zh:
      "trigger:本国任意一个 owned state 满足内部条件就为 true。是 trigger(条件)不是 effect(执行 → 用 random_owned_state)。配套 all_owned_state(所有都要满足)。同家族 any_controlled_state / any_state(全地图任意)/ any_neighbor_country。常用于'有没有任一海岸州可建港口'类条件门槛。",
  },
];
