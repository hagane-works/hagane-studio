---
contentId: focus-grant-trait
kind: case
slug: focus-grant-trait
locale: zh
status: published
title: "国策或事件给领袖或将领加特质"
summary: "国策完成时给在任领袖换一个特质，事件里给某位将领加一个限时特质；讲清领袖特质国家作用域就能给、将领特质必须先进人物作用域、换特质一步到位、限时写法。原版国策里给领袖特质 164 处、换特质 78 处、给将领 60 处。"
primaryTopic: people
topics: [people, politics]
scenarios: [traits, focus, event]
gameTerms: [特质, 领袖, 将领, 国策, 事件, 完成效果, 限时特质, 换特质, 作用域]
aliases: [国策给特质, 事件加特质, 领袖换特质, 将领加特质, 限时特质, 移除特质, add_country_leader_trait, swap_country_leader_traits, add_unit_leader_trait, add_timed_unit_leader_trait, random_army_leader]
relatedIds: [trait-system, general-trait-earned-in-combat, leader-succession, custom-character-traits, "wiki:effect:add_country_leader_trait", "wiki:effect:remove_country_leader_trait", "wiki:effect:swap_country_leader_traits", "wiki:effect:add_unit_leader_trait", "wiki:effect:add_timed_unit_leader_trait", "wiki:effect:remove_unit_leader_trait", "wiki:effect:random_army_leader", "wiki:trigger:has_trait"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki：作用域"
    url: "https://hoi4.paradoxwikis.com/Scopes"
  - label: "HOI4 Wiki：特质制作"
    url: "https://hoi4.paradoxwikis.com/Trait_modding"
---

# 国策或事件给领袖或将领加特质

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到三件事：

1. 国策完成的那一刻，领袖卡上的一个特质换成了另一个，提示框写着「某某获得：新特质」。
2. 一个事件之后，某位将领多了一个特质，悬停显示「获得 X 特质，持续 90 日」。
3. 九十天后那个特质自己消失。

## 开始前需要什么

- 一个已经能正常导出的 Mod，一条国策和一个事件。本文用占位 ID `my_focus_reform`、`my_events`。
- 要加的特质已经定义好：领袖特质 `my_leader_reformer`（在 `common/country_leader/`），将领特质 `my_general_wounded`（在 `common/unit_leader/`，`trait_type = status_trait`，不用 `gui_row`）。写法见给人物做一个自定义特质 (`custom-character-traits`)。
- 不需要任何 DLC。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 国策 | `common/national_focus/` | 换领袖特质 |
| 事件 | `events/` | 给将领限时特质 |
| 两条特质定义 | 两个目录 | 被引用；本文不改它们 |

## 1. 领袖：国家作用域就能给

游戏自带文档写明 `add_country_leader_trait` 支持国家和人物两种作用域：在国家作用域执行，加给**当前在任领袖**。所以国策奖励里直接写：

```
focus = {
    id = my_focus_reform
    icon = GFX_goal_generic_political_pressure
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        add_country_leader_trait = my_leader_reformer
    }
}
```

要**换**一个特质，原版国策里 78 处用的是一步到位的写法，不用先移除再添加：

```
completion_reward = {
    swap_country_leader_traits = {
        remove = my_leader_old
        add = my_leader_reformer
    }
}
```

`swap_country_leader_traits` 只有人物作用域；写在国策奖励里时，原版是先进领袖的人物作用域再换，比如比利时国策里 `BEL_leopold = { swap_country_leader_traits = { ... } }`。一个人物有多个领袖身份时，`add` / `remove` 都要加 `ideology = ...` 指明改哪个身份，文档的例子是 `HUN_miklos_horthy = { add_country_leader_trait = { ideology = oligarchism trait = anti_communist } }`。

移除用 `remove_country_leader_trait`，作用域规则和添加一样。

## 2. 将领：必须先进人物作用域

`add_unit_leader_trait` 只支持人物作用域。国策或事件本身是国家作用域，所以要先跳到某位将领身上。原版 60 处国策用法几乎都是这个形状：

```
random_army_leader = {
    limit = {
        NOT = { has_trait = my_general_wounded }
    }
    add_unit_leader_trait = my_general_wounded
}
```

`random_army_leader` 在本国将领里随机挑一个满足 `limit` 的；要指定某个人，直接用他的人物 ID 打开作用域：`MYT_my_general = { add_unit_leader_trait = ... }`。要全部将领都加，用 `every_army_leader`。

限时的写法：

```
add_namespace = my_events

country_event = {
    id = my_events.1
    title = my_events.1.t
    desc = my_events.1.d
    is_triggered_only = yes
    option = {
        name = my_events.1.a
        random_army_leader = {
            add_timed_unit_leader_trait = {
                trait = my_general_wounded
                days = 90
            }
        }
    }
}
```

原版事件里 5 处这么写，瑞典的「示威中受伤」120 天、「对政府失望」270 天。到期自动移除。手动移除用 `remove_unit_leader_trait`，同样要在人物作用域。

限时的对应物只在将领这边有：`add_timed_country_leader_trait` 原版一处都没用过，本文不推荐。

## 3. 特质本身怎么写

给将领的状态特质不用进树，所以不写 `gui_row`、`cost`、`gain_xp`；只写类型和修正：

```
leader_traits = {
    my_general_wounded = {
        type = land
        trait_type = status_trait
        non_shared_modifier = {
            experience_gain_factor = -0.25
        }
    }
}
```

领袖特质照领袖的形状，修正平铺：

```
leader_traits = {
    my_leader_reformer = {
        random = no
        stability_factor = 0.05
    }
}
```

## 4. 判断有没有

- 国家作用域判断在任领袖：`has_country_leader_with_trait = my_leader_reformer`，原版国策 45 处。
- 人物或战斗作用域判断将领：`has_trait = my_general_wounded`，通常写在 `random_army_leader` 的 `limit` 里。

## 5. 逐项核对

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 国策完成后领袖卡 | 旧特质没了，新特质在 | `swap` 里两个名字；特质是否定义在 `common/country_leader/` |
| 国策提示框 | 「某某获得：新特质」 | 特质本地化 |
| 事件后某位将领 | 多了一个特质，悬停显示「持续 90 日」 | 是否在 `random_army_leader` 里；`days` |
| 90 天后 | 特质消失 | 用的是 `add_timed_unit_leader_trait` 不是 `add_unit_leader_trait` |
| `error.log` | 没有提到你的文件 | 作用域错时这里会记 |

## 常见错误

- 🔴 **在国家作用域直接写 `add_unit_leader_trait`**：它只认人物作用域。先 `random_army_leader` 或人物 ID。
- 🔴 **换特质写成先移除再添加**：能跑，但 `swap_country_leader_traits` 一行的事，原版 78 处都这么写。
- **多身份领袖不指明 `ideology`**：文档写明有多个领袖身份时要加，否则不知道改哪个。
- **限时特质用了永久的效果**：`add_unit_leader_trait` 不会到期，要 `add_timed_unit_leader_trait`。
- **特质名拼错**：不存在的特质名，`error.log` 会记；先在定义文件里确认。

## 想改成别的效果？

- 想让全部将领都受影响：`every_army_leader = { ... }`。
- 想给海军将领：`random_navy_leader`；特质定义的 `type` 要是 `navy` 或 `all`。
- 想在换领袖时一起处理特质：见换领袖 (`leader-succession`)。
- 想让玩家自己去解锁而不是脚本给：见攒经验的特质 (`general-trait-earned-in-combat`)。
