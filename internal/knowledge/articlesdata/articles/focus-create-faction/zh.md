---
contentId: focus-create-faction
kind: case
slug: focus-create-faction
locale: zh
status: published
title: "国策建立阵营并把邻国拉进来"
summary: "国策完成时以本国为领袖建立一个阵营，再把邻国拉进来；顺带讲清那行规则到底管什么、阵营名是本地化键、两个好感效果各改哪个方向。原版里一半的建阵营国策前面都跟着同一行规则，但效果本身不需要它。"
primaryTopic: diplomacy
topics: [diplomacy, politics]
scenarios: [focus, event]
gameTerms: [阵营, 建立阵营, 加入阵营, 阵营领袖, 好感, 规则, 国策, 完成效果]
aliases: [国策建阵营, 建立自己的阵营, 拉国家进阵营, 阵营没出现, 阵营名字不显示, 加好感, 好感方向, create_faction, add_to_faction, set_rule, add_opinion_modifier, reverse_add_opinion_modifier]
relatedIds: [diplomacy-system, focus-wargoal-annex, custom-autonomy-state, "wiki:effect:create_faction", "wiki:effect:create_faction_from_template", "wiki:effect:add_to_faction", "wiki:effect:set_rule", "wiki:effect:add_opinion_modifier", "wiki:effect:reverse_add_opinion_modifier", "wiki:trigger:is_in_faction_with", "wiki:trigger:has_rule", "wiki:trigger:has_opinion"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki：触发条件"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki：本地化"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

# 国策建立阵营并把邻国拉进来

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到三件事：

1. 国策完成的那一刻，顶栏出现一个以你的名字命名的阵营，本国是领袖。
2. 第二条国策完成后，比利时出现在阵营成员列表里。
3. 在那之前，比利时对本国的好感已经因为一条修正上升，外交面板上能看到修正的名字和数值。

## 开始前需要什么

- 一个已经能正常导出的 Mod，以及两条前后相连的国策。本文用占位 ID `my_focus_found_league` 和 `my_focus_invite_belgium`。
- 一个阵营名字的本地化条目，本文用键 `my_league_name`。
- 邻国标签，本文用比利时 `BEL`，做国是卢森堡。
- 不需要任何 DLC。
- 建议先读外交：开局关系、运行时效果与定义文件各写在哪 (`diplomacy-system`)。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 国策 ×2 | `common/national_focus/` | 第一条建阵营，第二条拉人 |
| 好感修正定义 | `common/opinion_modifiers/` | 拉人之前先让对方喜欢你 |
| 阵营名 | `localisation/` | `create_faction` 的值就是本地化键 |
| 规则 | 国家上的 `set_rule` | 效果不需要它；玩家之后手动建阵营需要 |

## 1. 那行规则管什么

原版 61 处 `create_faction` 里有 31 处前面紧跟着同一行：

```
set_rule = { can_create_factions = yes }
```

实测：**没有这一行，`create_faction` 照样建出阵营**。这条规则在游戏里显示为「可以创建阵营」，管的是外交界面上玩家自己建阵营的按钮，不拦效果。原版把它给出去的 225 处全在国策、事件、决议里，历史文件里一处都没有，所以默认没有国家能手动建阵营；国策先给规则，是为了国策之后玩家还能自己动手。你的国策如果只是要一个阵营出现，可以不写；想让玩家之后能手动操作，就写上。

## 2. 建阵营

```
focus = {
    id = my_focus_found_league
    icon = GFX_goal_generic_alliance
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        set_rule = { can_create_factions = yes }
        create_faction = my_league_name
    }
}
```

`create_faction` 后面跟的不是显示文本，是一个本地化键。原版写法如 `create_faction = AFG_united_islamic_world_faction`，对应本地化文件里 `AFG_united_islamic_world_faction: "伊斯兰联盟"`。所以要在 `localisation/simp_chinese/` 下补一行：

```
l_simp_chinese:
 my_league_name:0 "我的联盟"
```

没有这一行，阵营照样建立，只是名字显示为键名本身。

游戏自带文档把 `create_faction` 标为「已弃用」，推荐 `create_faction_from_template`。模板系统属于「No Compromise, No Surrender」DLC，原版 171 处模板写法全包在 `has_dlc` 条件里；不要求 DLC 的 Mod 继续用 `create_faction`，原版自己也还有 27 处国策这么写。

## 3. 让邻国喜欢你：好感修正的方向

`add_to_faction` 不看好感，直接拉人。但剧情上通常先给好感，而且好感修正是这一批里最常见的效果（国策 913 处、事件 1009 处），值得在这里顺手做对。

先说方向，因为写反了不报错。游戏提示框的文案把它写死了：

| 效果 | 游戏里的提示文字 | 改的是谁对谁 |
| --- | --- | --- |
| `add_opinion_modifier = { target = BEL ... }` | 「获得 X（对比利时的评价 +50。）」 | **本国对比利时** |
| `reverse_add_opinion_modifier = { target = BEL ... }` | 「比利时获得 X（对他们的评价改变 +50。）」 | **比利时对本国** |

要让比利时愿意跟你走，改的是比利时对本国的评价，用的是 `reverse_add_opinion_modifier`。原版用后者的 87 处基本都是这个场景：签了条约，让对方国家对签约方好感上升。

先定义修正。新建 `common/opinion_modifiers/my_mod.txt`：

```
opinion_modifiers = {
    my_league_friendship = {
        value = 50
        months = 12
        decay = 1
    }
}
```

`value` 是好感值，`months = 12` 加 `decay = 1` 表示一年内每天衰减一点直到消失；只要永久生效就只写 `value`。字段名取自原版文件头部的注释，就这几个。

然后在国策里引用，注意用的是 `reverse_` 那个：

```
completion_reward = {
    reverse_add_opinion_modifier = {
        target = BEL
        modifier = my_league_friendship
    }
}
```

🔴 写成 `add_opinion_modifier` 也能跑，只是变成本国更喜欢比利时，比利时对你毫无变化，而且日志里不会有任何提示。

## 4. 拉邻国进阵营

```
focus = {
    id = my_focus_invite_belgium
    prerequisite = { focus = my_focus_found_league }
    x = 0
    y = 1
    cost = 5
    available = {
        is_faction_leader = yes
    }
    completion_reward = {
        add_to_faction = BEL
    }
}
```

`add_to_faction` 由**阵营领袖**的作用域执行，值是要拉进来的国家。原版事件里最常见的写法是 `FROM = { add_to_faction = ROOT }` 这类作用域跳转，直接写标签的情况在国策里更多。`available` 里的 `is_faction_leader = yes` 保证本国已经是领袖，否则这条国策不该能点。

## 5. 用事件代替国策

三段效果原样搬进事件选项即可，测试时用控制台 `event my_events.1` 触发。

## 6. 逐项核对

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 第一条国策完成后顶栏 | 出现阵营，本国是领袖 | `create_faction` 是否在奖励里；`error.log` |
| 阵营名 | 显示中文名 | 本地化键是否与 `create_faction` 的值一致 |
| 比利时外交面板里「比利时对本国」一栏 | 好感修正出现，数值与 `value` 一致 | 修正是否定义在 `common/opinion_modifiers/`；用的是不是 `reverse_` |
| 第二条国策 | 是领袖时可点 | `is_faction_leader` |
| 第二条国策完成后 | 比利时在成员列表里 | 比利时是否已在别的阵营 |

## 常见错误

- **以为没有 `set_rule` 就建不了阵营**：效果不看规则，实测照建。规则管的是玩家手动建。
- 🔴 **`create_faction` 后面写了中文名**：值必须是本地化键；中文会被当成键名，显示成一串原文。
- 🔴 **想让对方喜欢你却用了 `add_opinion_modifier`**：那改的是你对对方的评价。对方对你的评价要用 `reverse_add_opinion_modifier`。
- **引用了没定义的好感修正**：不报错，好感不变。
- **由非领袖执行 `add_to_faction`**：对方不会进来。用 `is_faction_leader` 守住。
- **对方已在别的阵营**：先用 `is_in_faction = no` 判断，或先让它退出；这种情况下 `add_to_faction` 的表现本文没有实测。

## 想改成别的效果？

- 想让本国加入别人的阵营而不是自己建：在领袖的作用域里执行 `add_to_faction`，例如 `GER = { add_to_faction = ROOT }`。
- 想判断两国是否同阵营：`is_in_faction_with = BEL`。
- 想判断对方对你的好感够不够：`BEL = { has_opinion = { target = ROOT value > 50 } }`，原版写法是 `value >`。
- 想给 DLC 玩家用模板阵营：`create_faction_from_template = 模板id`，包在 `has_dlc = "No Compromise, No Surrender"` 里。
