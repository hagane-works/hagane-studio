---
contentId: diplomacy-system
kind: guide
slug: diplomacy-system
locale: zh
status: published
title: "外交：开局关系、运行时效果与定义文件各写在哪"
summary: "外交在脚本里分三层：开局关系写在历史文件，运行中的变化写在国策或事件的效果里，宣战理由类型、好感修正、自治档三种「定义」各有自己的目录。分不清层就会把该在开局生效的东西写进国策，或者引用一个没定义过的好感修正。"
primaryTopic: diplomacy
topics: [diplomacy, politics]
scenarios: [focus, event, history]
gameTerms: [外交, 阵营, 宣战理由, 好感, 附庸, 傀儡, 自治, 保障, 互不侵犯, 宣称]
aliases: [外交怎么做, 阵营怎么建, 宣战理由怎么给, 好感怎么加, 傀儡怎么设, 附庸等级, diplomatic_relation, add_opinion_modifier, create_wargoal, set_autonomy]
relatedIds: [focus-wargoal-annex, focus-create-faction, custom-autonomy-state, focus-transfer-state-core, state-file-basics, "wiki:effect:create_wargoal", "wiki:effect:add_opinion_modifier", "wiki:effect:create_faction", "wiki:effect:add_to_faction", "wiki:effect:set_autonomy", "wiki:effect:diplomatic_relation", "wiki:effect:give_guarantee", "wiki:trigger:has_wargoal_against", "wiki:trigger:is_in_faction_with", "wiki:trigger:is_subject_of", "wiki:trigger:has_opinion"]
updatedAt: "2026-09-15"
verification: static-checked
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：外交"
    url: "https://hoi4.paradoxwikis.com/Diplomacy"
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki：触发条件"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki：国家创建"
    url: "https://hoi4.paradoxwikis.com/Country_creation"
---

# 外交：开局关系、运行时效果与定义文件各写在哪

## 这个系统负责什么

玩家在外交界面看到的一切：谁和谁有互不侵犯条约、谁保障了谁、谁在哪个阵营、谁是谁的附庸、两国之间的好感数字、能不能宣战。这些状态有的在开局就存在，有的在游戏进行中被国策、事件、决议改掉。

作为制作者你真正要决定的只有三件事：

1. **开局时两国是什么关系**（写在历史文件里，游戏开始那一刻生效一次）
2. **游戏中什么时候改变关系**（写在国策奖励、事件选项、决议效果里）
3. **改变关系时引用的「东西」是什么**（宣战理由的类型、好感修正的名字、自治档的名字，各自要先在定义文件里存在）

三件事对应三种文件位置。大多数外交脚本出错都是把一层的写法放到了另一层。

## 核心概念：状态、效果、定义

外交脚本里的名词分三类，写法完全不同：

| 类别 | 例子 | 写在哪 | 生效时机 |
| --- | --- | --- | --- |
| **开局状态** | 保障比利时、和土耳其有互不侵犯条约、在某个阵营里、是某国的附庸 | `history/countries/` 该国的历史文件 | 开局那一刻，只执行一次 |
| **运行时效果** | 国策完成后获得对邻国的宣战理由、事件里把某国拉进阵营、决议提升好感 | 国策的 `completion_reward`、事件的 `option`、决议的效果块 | 那个效果执行的时候 |
| **定义** | 宣战理由类型 `annex_everything`、好感修正 `holds_our_cores`、自治档 `autonomy_puppet` | `common/wargoals/`、`common/opinion_modifiers/`、`common/autonomous_states/` | 被效果引用时才起作用；自身不做任何事 |

开局状态和运行时效果用的往往是**同一批效果**。历史文件本质上就是一段开局执行的效果脚本，所以 `give_guarantee`、`add_to_faction`、`set_autonomy`、`diplomatic_relation` 既能写在历史文件里，也能写在国策里。区别只在于执行时机。

定义则不是效果。`common/opinion_modifiers/` 里写一个 `my_friendship = { value = 30 }` 什么都不会发生；只有某处执行了 `add_opinion_modifier = { target = GER modifier = my_friendship }`，德国对你的好感才会变。

## 文件结构与执行方式

### 开局关系：历史文件

原版每个国家的历史文件里，外交语句直接写在顶层（1936 那一段之前）：

```
add_to_faction = PRC

give_guarantee = AFA

set_autonomy = {
    target = COG
    autonomous_state = autonomy_colony
    freedom_level = 0.2
}

diplomatic_relation = {
    country = TUR
    relation = non_aggression_pact
    active = yes
}

add_opinion_modifier = {
    target = GER
    modifier = my_opinion_modifier
}
```

原版历史文件里 `diplomatic_relation` 的 `relation` 只出现过六种值：`non_aggression_pact`（139 处）、`guarantee`（110 处）、`military_access`（35 处）、`docking_rights`（14 处）、`puppet` 和 `air_base_access`（各 2 处）。这是全量计数，不是列举。

### 运行时效果：国策、事件、决议

同样这批效果搬进 `completion_reward` 或 `option` 就是运行时版本。原版国策里最常见的外交效果是 `create_wargoal`（1072 处），其次是 `add_opinion_modifier`（913 处）、`add_state_claim`（507 处）、`set_autonomy`（149 处）；事件里 `add_to_faction` 出现 494 处。

```
focus = {
    id = my_focus_claim_neighbour
    icon = GFX_goal_generic_territory_or_war
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        add_state_claim = 6
        create_wargoal = {
            type = annex_everything
            target = BEL
        }
    }
}
```

### 定义：三个目录

| 目录 | 外层块 | 一条定义长什么样 |
| --- | --- | --- |
| `common/wargoals/` | `wargoal_types = { ... }` | `annex_everything = { allowed = { always = no } threat = 0.5 }` |
| `common/opinion_modifiers/` | `opinion_modifiers = { ... }` | `my_friendship = { value = 30 months = 12 decay = 1 }` |
| `common/autonomous_states/` | 每个文件一个 `autonomy_state = { id = ... }` | 见给附庸做一个自定义自治等级 (`custom-autonomy-state`) |

好感修正的字段就是原版文件头部注释列的那几个：`value`（好感值）、`min_trust` / `max_trust`、`decay`（每日衰减）、`days` / `months` / `years`（计时器）、`trade`。原版里带计时器的写法如 `months = 12` 加 `decay = 1`，表示一年内每天衰减一点直到消失。

### 判断条件

外交状态也有一整套判断条件，写在国策的 `available`、事件的 `trigger`、决议的可见条件里。这几个最常用，名字都是原版文档里的写法：

| 想判断 | 写法 | 说明 |
| --- | --- | --- |
| 对某国有宣战理由 | `has_wargoal_against = BEL` | 也可以 `has_wargoal_against = { target = BEL type = annex_everything }` 限定类型 |
| 和某国同阵营 | `is_in_faction_with = BEL` | 在任何阵营里用 `is_in_faction = yes` |
| 是某国的附庸 | `is_subject_of = GER` | 傀儡一档专用 `is_puppet_of` |
| 处于某个自治档 | `has_autonomy_state = autonomy_puppet` | 值是自治档的 `id` |
| 保障了某国 | `has_guaranteed = BEL` | |
| 对某国好感高于某值 | `has_opinion = { target = GER value > 50 }` | 原版写法是 `value >`，不是 `=` |
| 有某条规则 | `has_rule = can_create_factions` | 规则由 `set_rule` 设置 |

## 常见制作目标与入口

| 你想在游戏里看到的 | 从这里开始 | 关键区分 |
| --- | --- | --- |
| 开局就和邻国有互不侵犯条约 | 历史文件里 `diplomatic_relation` | 不要写进国策；那样要等国策完成才有 |
| 国策完成后可以对邻国宣战 | 国策完成后获得对邻国的宣战理由 (`focus-wargoal-annex`) | `type` 必须是 `common/wargoals/` 里存在的类型 |
| 国策建立自己的阵营并拉邻国进来 | 国策建立阵营并把邻国拉进来 (`focus-create-faction`) | 阵营名是本地化键；规则管玩家手动建，不管效果 |
| 把战败国设为特殊的附庸等级 | 给附庸做一个自定义自治等级 (`custom-autonomy-state`) | 先在 `common/autonomous_states/` 定义档，再用 `set_autonomy` |
| 事件里让两国好感上升 | `add_opinion_modifier`，修正先在 `common/opinion_modifiers/` 定义 | 修正定义里可以带计时和衰减 |
| 国策完成后对某州加宣称 | `add_state_claim = 6`，国家层 | 州层的 `add_claim_by` 见转交州 (`focus-transfer-state-core`) |

## 容易混淆的事

### 建阵营的那行规则管的是玩家，不是效果

原版 61 处 `create_faction` 里有 31 处前面紧跟着 `set_rule = { can_create_factions = yes }`。实测不写它效果照样建出阵营；这条规则管的是外交界面上玩家手动建阵营的按钮。原版先给规则，是为了国策之后玩家还能自己操作。见国策建立阵营 (`focus-create-faction`)。

### `create_faction` 与 `create_faction_from_template`

游戏自带文档把 `create_faction` 标为「已弃用，建议用 `create_faction_from_template`」。但模板系统需要「No Compromise, No Surrender」DLC，原版里 171 处 `create_faction_from_template` 全部包在 `has_dlc` 条件里。不要求 DLC 的 Mod 仍然用 `create_faction = 本地化键`，1.19.2 原版自己也还有 27 处国策这么写。

### `set_autonomy` 有两种拼写

原版历史文件里写 `autonomous_state =`（23 处），国策和事件里几乎都写 `autonomy_state =`（540 处），游戏自带文档用的是后者。两种拼写都被引擎接受。钢之工坊的历史文件导出器用前者，与原版历史文件一致；你在国策里照文档写后者即可。

### 好感效果有两个，方向相反

`add_opinion_modifier = { target = B }` 写在 A 的作用域里，改的是 **A 对 B** 的评价；`reverse_add_opinion_modifier` 改的是 **B 对 A** 的评价。这是游戏提示框文案写明的：前者显示「获得 X（对 B 的评价 +N）」，后者显示「B 获得 X（对他们的评价改变 +N）」。想让别国喜欢你，用 `reverse_` 那个；写反了不报错。案例国策建立阵营 (`focus-create-faction`)里有对照表。

### 好感修正必须先定义

`add_opinion_modifier` 引用的名字必须在 `common/opinion_modifiers/` 里存在。引用一个不存在的修正不会让游戏崩溃，只是什么都不发生。原版所有国家通用的修正都在 `00_opinion_modifiers.txt` 里，自己的写在新文件里，不要改原版文件。

### 「宣战理由」不等于「宣战」

`create_wargoal` 只是给了宣战的权利，仍然要玩家（或 AI）自己去点宣战。要国策完成时直接开战，用 `declare_war_on = { target = BEL type = annex_everything }`，原版国策里 64 处这么写。

## 在钢之工坊中

- 开局关系：历史档工作台。导出的历史文件会写出 `give_guarantee`、`add_to_faction`、`set_autonomy`、`diplomatic_relation` 四种语句，形状与上面原版一致。
- 宣战理由类型：战争目标工作台，导出到 `common/wargoals/`，外层 `wargoal_types` 由导出器补齐。
- 自治档：自治状态工作台，导出到 `common/autonomous_states/`。
- 好感修正：在角色工作台的自定义特质面板里有一个好感修正子面板，导出到 `common/opinion_modifiers/`。入口位置不直观，但它在。
- 运行时效果：国策工作台的完成效果、事件工作台的选项效果，原样填写即可。

## 接下来看什么

- 国策完成后获得对邻国的宣战理由 (`focus-wargoal-annex`)
- 国策建立阵营并把邻国拉进来 (`focus-create-faction`)
- 给附庸做一个自定义自治等级 (`custom-autonomy-state`)
- 加宣称与转交州：国策完成后转交一个州并加核心 (`focus-transfer-state-core`)
