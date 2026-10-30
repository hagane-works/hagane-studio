export interface EffectTemplate {
  label_en: string;
  label_zh: string;
  code: string;
  desc_en?: string;
  desc_zh?: string;
}

export const EFFECT_TEMPLATES: EffectTemplate[] = [
  {
    label_en: "Political power",
    label_zh: "政治点数",
    code: "add_political_power = 100",
    desc_en:
      "Gives your country political power (PP). PP is spent appointing advisors, switching laws, and enacting decisions. The value can be positive or negative — a negative subtracts PP. Typical magnitudes: completing a focus +50~150, large decisions +200~400; passive gain ~2 PP/week (BASE_POLITICAL_POWER_INCREASE + modifiers). Cap +2000, floor -500 (defines/POLITICAL_POWER_UPPER_CAP/LOWER_CAP), clamped automatically.",
    desc_zh:
      "给本国增加政治点数（PP）。PP 用于任命顾问、切换法案、执行决议。数值可正可负——负数扣 PP。典型量级：完成国策 +50~150、大型决议 +200~400；被动积累 2 PP/周（由 BASE_POLITICAL_POWER_INCREASE + 各种 modifier 修正）。上限 +2000、下限 -500（defines/POLITICAL_POWER_UPPER_CAP/LOWER_CAP），超出自动截断。",
  },
  {
    label_en: "Stability",
    label_zh: "稳定度",
    code: "add_stability = 0.05",
    desc_en:
      "Adjusts your country's stability, as a 0.0~1.0 decimal (not a percentage). 0.05 = +5%. Stability affects factory output, political support, and event probability. Typical magnitudes: small events ±0.01~0.05, major focuses ±0.10, crisis events -0.15~-0.30. The result is clamped to 0.0~1.0. Negatives punish player choices (coups/riots/defeats).",
    desc_zh:
      "调整本国稳定度，单位是 0.0~1.0 小数（不是百分比）。0.05 = +5%。稳定度影响工厂产量、政治支持、事件概率。典型量级：小事件 ±0.01~0.05、重大国策 ±0.10、危机事件 -0.15~-0.30。结果被 0.0~1.0 自动截断。负值用于惩罚玩家决策（政变/暴动/战败）。",
  },
  {
    label_en: "War support",
    label_zh: "战争支持",
    code: "add_war_support = 0.05",
    desc_en:
      "Adjusts your country's war support, in the same 0.0~1.0 decimal as stability. War support determines whether you can conscript, use total-mobilization laws, and unlock aggressive military decisions. Typical magnitudes: small events ±0.01~0.05, being attacked / major events +0.15~0.30, defeat/surrender events -0.20~-0.40. Clamped to 0.0~1.0. Often combined with conscription-law effects to drive a mobilization arc.",
    desc_zh:
      "调整本国战争支持度，单位同稳定度的 0.0~1.0 小数。战争支持度决定能否征兵、使用总动员法、解锁激进军事决议。典型量级：小事件 ±0.01~0.05、被袭击/大事件 +0.15~0.30、战败/投降事件 -0.20~-0.40。0.0~1.0 自动截断。常与征兵法 effect 组合推动动员剧情。",
  },
  {
    label_en: "Factory (1 mil / 1 civ)",
    label_zh: "工厂 (1军/1民)",
    code: "random_owned_controlled_state = {\n\tlimit = {\n\t\tfree_building_slots = {\n\t\t\tbuilding = arms_factory\n\t\t\tsize > 0\n\t\t\tinclude_locked = yes\n\t\t}\n\t}\n\tadd_extra_state_shared_building_slots = 2\n\tadd_building_construction = {\n\t\ttype = arms_factory\n\t\tlevel = 1\n\t\tinstant_build = yes\n\t}\n\tadd_building_construction = {\n\t\ttype = industrial_complex\n\t\tlevel = 1\n\t\tinstant_build = yes\n\t}\n}",
    desc_en:
      "Scope change: picks one random state your country owns and actually controls, then runs the inner effect. Conditions inside limit = {} (e.g. free_building_slots > 0) decide which states enter the random pool. Common for 'build a factory / add infrastructure in any qualifying state'. Variants: every_owned_controlled_state (run on all matching states), random_state (random across the whole map). This template: +2 shared slots + 1 mil factory + 1 civ factory.",
    desc_zh:
      "scope change：随机选一个本国拥有并实际控制的 state 执行内部 effect。limit = {} 内写筛选条件（如 free_building_slots > 0），只有满足的 state 才进入随机池。常用于'在任意符合条件州建厂/加基建'。变体：every_owned_controlled_state（对所有满足 state 执行）、random_state（对全地图随机）。模板示例：加 2 共享槽 + 1 军工 + 1 民工。",
  },
  {
    label_en: "Activate mission",
    label_zh: "解锁决议",
    code: "activate_mission = MISSION_ID",
    desc_en:
      "Activates the given mission (a countdown task that fires periodically). Missions are defined under common/decisions/ categories, identified by the `days_mission_timeout = N` field (not finishing within N days counts as failure/timeout). An optional `activation = {}` block defines MTTH auto-activation conditions; this effect bypasses that and forces it open. Common for time-limited tasks triggered by a focus chain (e.g. GER_mefo_bills_mission, Maginot Line construction).",
    desc_zh:
      "激活指定 mission（定期触发的倒计时任务）。Mission 定义在 common/decisions/ 分类下、以 `days_mission_timeout = N` 字段作为标识（N 天内完不成视为失败/超时）。可选 `activation = {}` 块定义 MTTH 自动激活条件；本 effect 绕过该条件强制开启。常用于国策链触发限时任务（如 GER_mefo_bills_mission、马奇诺防线建设）。",
  },
  {
    label_en: "Fire event",
    label_zh: "触发事件",
    code: "country_event = {\n\tid = EVENT_NAMESPACE.1\n\tdays = 1\n}",
    desc_en:
      "Fires the given event for your country. id format is `namespace.number`, matching an add_namespace in events/xxx.txt. Optional params: days (integer delay), hours, random_hours, random_days (random window). Cross-country firing needs an outer scope change: TAG = { country_event = { ... } }. Note: events with is_triggered_only = yes can only fire via this effect, and aren't scanned by MTTH.",
    desc_zh:
      "给本国触发指定事件。id 格式 `命名空间.编号`，对应 events/xxx.txt 中 add_namespace。可选参数：days（整数延迟）、hours、random_hours、random_days（随机窗口）。跨国触发需外包 scope change：TAG = { country_event = { ... } }。注意：is_triggered_only = yes 的事件只能通过此 effect 触发，不会被 MTTH 自动扫描。",
  },
  {
    label_en: "Research bonus",
    label_zh: "研究加成",
    code: "add_tech_bonus = {\n\tname = BONUS_NAME\n\tbonus = 1.0\n\tuses = 2\n\tcategory = infantry_tech\n}",
    desc_en:
      "Grants a research bonus to the current research. Sub-fields: name (bonus identifier, queryable via has_tech_bonus), bonus (coefficient, 0.5 = +50%), uses (number of uses, 1 = single), category (filter a tech category like infantry_tech) or technology (a single tech ID). Typical magnitudes: small focus 0.5 × 1~2 uses, large focus 1.0 × 2~3 uses. ahead_reduction can lower the ahead-of-time penalty.",
    desc_zh:
      "给当期研究加成。子字段：name（bonus 标识，可用 has_tech_bonus 查询）、bonus（系数 0.5 = +50%）、uses（可用次数，1 = 单项）、category（筛科技大类如 infantry_tech）或 technology（单科技 ID）。典型量级：小国策 0.5 × 1~2 uses、大国策 1.0 × 2~3 uses。ahead_reduction 可减超时代惩罚。",
  },
  {
    label_en: "Unlock advisor",
    label_zh: "解锁角色",
    code: "TAG = {\n\tactivate_advisor = CHARACTER_TAG\n}",
    desc_en:
      "Activates an advisor of the target country (the character must already define an advisor role block in common/characters/). Scope change to TAG first, then call activate_advisor with the character_id. Common for 'unlock a hidden advisor after a focus completes'. Distinctions: activate_advisor puts the advisor into the usable slot pool; add_advisor_role (the Level B API) supports a temporary side role; deactivate_advisor removes it.",
    desc_zh:
      "激活目标国某个顾问（该角色需已在 common/characters/ 定义 advisor 身份块）。先 scope change 到 TAG，再调 activate_advisor 传 character_id。常用于'国策完成后解锁隐藏顾问'。区别：activate_advisor 让 advisor 进入顾问槽可用池；add_advisor_role（Level B 新 API）支持临时副职；deactivate_advisor 反向移除。",
  },
  {
    label_en: "Recruit character",
    label_zh: "招募将领",
    code: "TAG = {\n\trecruit_character = CHARACTER_TAG\n}",
    desc_en:
      "Recruits a character of the target country (the character_id must already be defined in common/characters/). Covers every role: country leader, general, field marshal, admiral, scientist, advisor. Difference from activate_advisor: recruit_character is generic recruitment (puts the character into the usable pool) and doesn't auto-assign a post; activate_advisor only targets advisors and auto-fills the advisor slot. Common for 'a focus/event unlocks a famous general/scientist'.",
    desc_zh:
      "招募目标国某个角色（character_id 需已在 common/characters/ 定义）。涵盖全部身份：国家领导人、将领、元帅、海军、科学家、顾问。和 activate_advisor 区别：recruit_character 通用招募（让角色进入可用池），不自动分配岗位；activate_advisor 只针对 advisor 且自动上顾问槽。常用于'国策/事件解锁名将/科学家'。",
  },
  {
    label_en: "Declare war / create wargoal",
    label_zh: "宣战/发起战争",
    code: "create_wargoal = {\n\ttype = annex_everything\n\ttarget = TARGET_TAG\n}",
    desc_en:
      "Creates a war goal, establishing a legal casus belli but **not** starting the war immediately. It must be paired with declare_war_on = TAG to actually trigger. Common type values: annex_everything (annex) / puppet_wargoal_focus (puppet) / liberate (free a puppet) / take_state_focus (take states, needs generator params). target is the 3-letter TAG of the target. Large mods pair it with ai_strategy to steer the AI per the script.",
    desc_zh:
      "创建战争目标，建立合法开战理由但**不立即开战**。需配合 declare_war_on = TAG 才实际触发。type 常用值：annex_everything（吞并）/ puppet_wargoal_focus（傀儡化）/ liberate（解放傀儡）/ take_state_focus（割州，需配 generator 参数）。target 为目标 3 字母 TAG。大 mod 常用 ai_strategy 配合引导 AI 按剧本行动。",
  },
  {
    label_en: "Add manpower",
    label_zh: "添加人力",
    code: "add_manpower = 50000",
    desc_en:
      "Adds recruited reserves to your country's manpower pool. The value is a head count (integer), positive or negative — a negative subtracts manpower. Typical magnitudes: small events +5K~20K, large recruitment focuses +50K~200K, failure costs -10K~-50K. Note: it's the conscription pool, not total population or army size. The pool cap is set by conscription law; excess doesn't stack; exhaustion makes has_manpower < 0 true.",
    desc_zh:
      "给本国人力池加征兵预备役。数值为人数（整数），可正可负——负数扣人力。典型量级：小事件 +5K~20K、大型募兵国策 +50K~200K、失败代价 -10K~-50K。注意：是征兵池不是总人口，也不是陆军规模。人力池上限由征兵法决定，超出不叠加；耗竭时 has_manpower < 0 为真。",
  },
  {
    label_en: "Set politics",
    label_zh: "设定政治",
    code: "set_politics = {\n\truling_party = democratic\n\telections_allowed = yes\n}",
    desc_en:
      "Switches your country's ruling party. ruling_party valid values: democratic / fascism / communism / neutrality. Optional sub-fields: elections_allowed (whether elections run), last_election (date of the last election), election_frequency (months between elections). It doesn't change the support distribution — to adjust support too, use add_popularity or set_popularities. Common after a coup/civil-war victory; routine switches need a PP-cost event.",
    desc_zh:
      "切换本国执政党。ruling_party 合法值：democratic / fascism / communism / neutrality。可选子字段：elections_allowed（是否开选举）、last_election（上次选举日期）、election_frequency（选举频率月数）。不改变支持率分布——要同步调支持率用 add_popularity 或 set_popularities。政变/内战胜利后常用；日常切换需 PP 成本事件配合。",
  },
  {
    label_en: "Add opinion modifier",
    label_zh: "添加外交修正",
    code: "add_opinion_modifier = {\n\ttarget = TARGET_TAG\n\tmodifier = OPINION_MODIFIER_NAME\n}",
    desc_en:
      "Adds a diplomatic opinion modifier toward a country (the modifier must be predefined in common/opinion_modifiers/). target = TAG is the country whose opinion changes; modifier is the opinion_modifier id (common vanilla: large_increase/medium_increase/small_increase + the reverse large_decrease etc.). Common for improving relations on focus completion/events. Distinct from targeted_modifier (a country modifier embedded in an idea/decision) — opinion_modifier is the standalone diplomatic-relations system.",
    desc_zh:
      "对指定国家添加外交修正(modifier 需在 common/opinion_modifiers/ 预定义)。target = TAG 是被改观感的国家,modifier 是 opinion_modifier id(常用 vanilla:large_increase/medium_increase/small_increase + 反向 large_decrease 等)。常用于国策完成/事件改善关系。区别 targeted_modifier(idea/decision 内嵌 country modifier)— opinion_modifier 是独立外交关系系统。",
  },
  {
    label_en: "Remove opinion modifier",
    label_zh: "移除外交修正",
    code: "remove_opinion_modifier = {\n\ttarget = TARGET_TAG\n\tmodifier = OPINION_MODIFIER_NAME\n}",
    desc_en:
      "Removes an opinion modifier previously added with add_opinion_modifier. target and modifier must match the values used when adding for the removal to match. Common when declaring war / breaking with an ally (to withdraw a friendly buff), or when switching focus paths (to clear a prerequisite modifier). If the modifier has no decay field in common/opinion_modifiers/, it must be removed explicitly or it stacks permanently.",
    desc_zh:
      "撤销之前用 add_opinion_modifier 添加的外交修正。target 跟 modifier 必须跟添加时一致才能匹配清除。常用于战争声明/盟友破裂时撤回友好 buff,或国策路径切换时清前置 modifier。如果 modifier 在 common/opinion_modifiers/ 没有 decay 字段,必须显式 remove,否则永久叠加。",
  },
  {
    label_en: "Set country flag",
    label_zh: "设定国家flag",
    code: "set_country_flag = FLAG_NAME",
    desc_en:
      "Sets a country-level boolean flag. Optional days field for auto-expiry: set_country_flag = { flag = FLAG_NAME days = 30 }. Query with has_country_flag = FLAG_NAME. Common pattern: a decision sets a flag → a modifier applies during that period → days_remove clears it on expiry via clr_country_flag. FLAG_NAME uses snake_case, author-defined, no predefinition needed.",
    desc_zh:
      "设定国家级布尔标记。可选 days 字段自动过期:set_country_flag = { flag = FLAG_NAME days = 30 }。配 has_country_flag = FLAG_NAME 用查询。常见用法:决议触发设 flag → modifier 期间生效 → days_remove 到期 clr_country_flag 清。FLAG_NAME 用 snake_case,作者自定义,不需要预定义。",
  },
  {
    label_en: "Clear country flag",
    label_zh: "清除国家flag",
    code: "clr_country_flag = FLAG_NAME",
    desc_en:
      "Clears a country-level boolean flag previously set with set_country_flag. FLAG_NAME must match the one used when setting. Often used inside a hidden_effect block for state-machine resets (to avoid exposing internal flags in a tooltip). If set_country_flag set a days auto-expiry, an explicit clr_country_flag isn't needed. The state-scope equivalent is clr_state_flag.",
    desc_zh:
      "清除之前用 set_country_flag 设的国家级布尔标记。FLAG_NAME 必须跟设时一致。常配 hidden_effect 块用于状态机重置(避免 tooltip 暴露内部 flag)。如果 set_country_flag 设了 days 自动过期,不需要显式 clr_country_flag。state-scope 等价 effect 是 clr_state_flag。",
  },
  {
    label_en: "Set variable",
    label_zh: "变量赋值",
    code: "set_variable = {\n\tvar = VAR_NAME\n\tvalue = 1\n}",
    desc_en:
      "Assigns a numeric variable. VAR_NAME is a snake_case custom variable name (no predefinition needed). value can be a constant / @constant / another variable. Common for passing numbers across entities, caching computations, driving dynamic modifiers (with dynamic_modifier). In a state scope the variable lives on the state, in a country scope on the country. Query with check_variable. set_temp_variable is a function-local variable.",
    desc_zh:
      "数值变量赋值。VAR_NAME 是 snake_case 自定义变量名(无需预定义)。value 可以是常数 / @常数 / 另一变量。常用于跨 entity 数值传递、计算缓存、动态修正驱动(配 dynamic_modifier)。state-scope 内变量挂在 state 上,country-scope 挂在 country 上。配 check_variable 查询。set_temp_variable 是函数局部变量。",
  },
  {
    label_en: "Add to variable",
    label_zh: "变量增减",
    code: "add_to_variable = {\n\tvar = VAR_NAME\n\tvalue = 1\n}",
    desc_en:
      "Adds to / subtracts from a variable (a negative value subtracts). The variable must first be initialized with set_variable (otherwise it accumulates from 0, which may not be what you want). Pair with while_loop_effect for loop counting (each pass add_to_temp_variable counter += 1). Same family as multiply_variable / divide_variable / modulo_variable. Variable values persist and don't reset.",
    desc_zh:
      "变量加减(value 负数即减)。前提变量必须先 set_variable 初始化(否则从 0 起累加,可能不是你想要的)。配 while_loop_effect 做循环计数(每次 add_to_temp_variable counter += 1)。和 multiply_variable / divide_variable / modulo_variable 同家族。变量数值持久化,不重置。",
  },
  {
    label_en: "Hidden effect block",
    label_zh: "隐藏效果块",
    code: "hidden_effect = {\n\tclr_country_flag = my_flag\n\tset_variable = { var = counter value = 0 }\n}",
    desc_en:
      "Wrapper: hides the inner effect from the tooltip (doesn't tell the player what it did). Common in immediate blocks to clear flags / reset variables / update a state machine internally. The inverse of effect_tooltip (show a tooltip but don't actually execute). A standard HOI4 idiom — vanilla uses it heavily to hide 'pure mechanical' state operations and avoid tooltip noise.",
    desc_zh:
      "wrapper:隐藏内部 effect 不显示在 tooltip(不告诉玩家做了啥)。常用于 immediate 块清 flag/重置变量/状态机内部更新。配 effect_tooltip 反向使用(显示 tooltip 但不实际执行)。HOI4 标准 idiom — vanilla 大量用于隐藏 'pure mechanical' state 操作避免 tooltip 噪音。",
  },
  {
    label_en: "Tooltip-only block (no execution)",
    label_zh: "效果展示块(不执行)",
    code: "effect_tooltip = {\n\tswap_ideas = { remove_idea = old_idea add_idea = new_idea }\n}",
    desc_en:
      "Wrapper: shows tooltip content but doesn't actually run the effect. Common in multi-branch if statements to give the player consistent expectations (every branch shows the same outcome, but only one branch actually executes). Or where a focus preview / decision description needs a tooltip without a double fire. The inverse of hidden_effect (execute but don't show). An indispensable HOI4 idiom.",
    desc_zh:
      "wrapper:展示 tooltip 内容但不实际执行 effect。常用于多分支 if 中给玩家一致预期(每个分支都显示同样结果,但实际只一个分支真执行)。或国策预览/decision 描述需要 tooltip 不需要双重 fire。配 hidden_effect 反向(执行但不显示)。HOI4 标准 idiom 不可缺。",
  },
  {
    label_en: "Conditional loop",
    label_zh: "条件循环",
    code: "set_temp_variable = { counter = 0 }\nwhile_loop_effect = {\n\tlimit = { check_variable = { counter < 5 } }\n\trandom_owned_state = {\n\t\tadd_building_construction = { type = arms_factory level = 1 instant_build = yes }\n\t}\n\tadd_to_temp_variable = { counter = 1 }\n}",
    desc_en:
      "Repeats the inner effect until limit is no longer met. Common for 'pick N random states and do X' / batch assignment / counted iteration. A break = name field can explicitly break out early. Note: HOI4 has a built-in infinite-loop guard (default max 1024 iterations), aborting automatically beyond that. Pair check_variable + add_to_temp_variable for the counter. For advanced array loops see reference: 高级代码/数组循环.",
    desc_zh:
      "重复执行内部 effect 直到 limit 不满足。常用于 '随机选 N 个 state 做 X' / 批量分配 / 计数迭代。break = name 字段可显式提前中断。注意:HOI4 内置防死循环(默认 max iterations 1024),超过自动 abort。配 check_variable + add_to_temp_variable 做计数器。advanced 用法 reference 见 高级代码/数组循环。",
  },
  {
    label_en: "Save event target",
    label_zh: "保存事件目标",
    code: "random_owned_state = { save_event_target_as = my_state }\n# 后续可用:event_target:my_state = { add_buildings = { ... } }",
    desc_en:
      "Inside immediate / hidden_effect, marks a scope as a named target; later effects / options reference it via event_target:NAME. It auto-clears when the event finishes (valid within the same event). Or use save_global_event_target_as to persist across events (clear explicitly with clear_global_event_target). Common idiom: lock 'this_specific_state' while iterating states, or pick 1 of multiple countries and save it as a target. See reference: 高级代码/event_target讲解.",
    desc_zh:
      "在 immediate / hidden_effect 内标记 scope 为命名 target,后续 effect / option 用 event_target:NAME 引用。event 完成自动清(同 event 内有效)。或 save_global_event_target_as 跨 event 持久化(用 clear_global_event_target 显式清)。常用 idiom:遍历 state 时锁定 'this_specific_state',或多 country 选 1 个保存为 target。reference 见 高级代码/event_target讲解。",
  },
  {
    label_en: "Add AI strategy",
    label_zh: "添加 AI 战略",
    code: "add_ai_strategy = {\n\ttype = befriend\n\tid = TARGET_TAG\n\tvalue = 100\n}",
    desc_en:
      "Hints a strategic tendency to the AI (not forced — just a weight adjustment). Common type values: befriend / antagonize / conquer / contain / declare_war / alliance / influence / area_priority / prepare_for_war, plus 40+ types. value is a weight (commonly 50-200, negative reverses). Common for steering the AI to act per the script. For details see reference: AI战略编写模板扩充版.txt.",
    desc_zh:
      "向 AI 提示战略倾向(不强制,只是修正权重)。常见 type:befriend(友好) / antagonize(敌对) / conquer(征服) / contain(遏制) / declare_war(直接宣战) / alliance(结盟) / influence(渗透) / area_priority(关注区域) / prepare_for_war(备战) 等 40+ type。value 是权重数值(常 50-200,负值反向)。常用于剧本引导 AI 按设计行动。详细见 reference AI战略编写模板扩充版.txt。",
  },
  {
    label_en: "Add named threat",
    label_zh: "添加命名威胁",
    code: "add_named_threat = {\n\tthreat = 2\n\tname = THREAT_LOC_KEY\n}",
    desc_en:
      "Adds a contribution to global world tension (default +1 = +1% tension); the threat field is the value. name is a localisation key shown in the world-tension tooltip (telling the player 'country XX did YY, causing +N% tension'). Common for aggressive-action storylines / a mod's custom tension driver. The reverse is reduce_world_tension.",
    desc_zh:
      "对全球 world tension 加贡献(默认 +1 = +1% tension),threat 字段是数值。name 是 localisation key,显示在 world tension tooltip 上(告诉玩家 'XX 国干了 YY 导致 +N% 紧张')。常用于侵略行为剧情 / mod 自定义 tension 推动器。配套 reduce_world_tension reverse。",
  },
];
