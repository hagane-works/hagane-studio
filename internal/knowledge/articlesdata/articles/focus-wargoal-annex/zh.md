---
contentId: focus-wargoal-annex
kind: case
slug: focus-wargoal-annex
locale: zh
status: published
title: "国策完成后获得对邻国的宣战理由"
summary: "国策完成时给本国一个对邻国的吞并宣战理由，玩家不用花政治点正当化就能宣战；顺带讲清类型从哪来、过期怎么写、后续国策怎么用「已有宣战理由」当条件。这是原版国策里最常见的外交效果，一千多处。"
primaryTopic: diplomacy
topics: [diplomacy]
scenarios: [focus, event]
gameTerms: [宣战理由, 正当化, 宣战, 吞并, 国策, 完成效果, 世界紧张度]
aliases: [国策给宣战理由, 国策后可以宣战, 不用正当化直接宣战, 宣战理由过期, create_wargoal, has_wargoal_against, annex_everything]
relatedIds: [diplomacy-system, focus-create-faction, focus-transfer-state-core, "wiki:effect:create_wargoal", "wiki:effect:declare_war_on", "wiki:effect:add_state_claim", "wiki:trigger:has_wargoal_against"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki：触发条件"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki：国策制作"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
---

# 国策完成后获得对邻国的宣战理由

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到三件事：

1. 国策完成的那一刻，目标国家的外交面板上「宣战」变成可用，宣战理由列表里出现「吞并」。
2. 玩家没有花政治点做正当化，世界紧张度也没有因为正当化过程上升。
3. 后面一条国策只有在「已有对该国的宣战理由」时才能点。

## 开始前需要什么

- 一个已经能正常导出的 Mod，以及一条已存在的国策。本文用占位 ID `my_focus_claim_belgium`。
- 目标国家的三字母标签。本文用比利时 `BEL`，做国是卢森堡。
- 不需要任何 DLC。
- 建议先读外交：开局关系、运行时效果与定义文件各写在哪 (`diplomacy-system`)，知道「宣战理由类型」是一种定义，效果只是引用它。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 国策 | `common/national_focus/` | 效果写在它的 `completion_reward` 里 |
| 宣战理由类型 | `common/wargoals/`，原版 `00_invasion.txt` | `annex_everything` 已经存在，本文不新建类型 |
| 后续国策 | 同一棵树 | `available` 里用 `has_wargoal_against` |

## 1. 写在国策的完成效果里

```
focus = {
    id = my_focus_claim_belgium
    icon = GFX_goal_generic_territory_or_war
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        create_wargoal = {
            type = annex_everything
            target = BEL
        }
    }
}
```

国策的提示框会显示「获得针对比利时的吞并战争目标」，这是游戏自己的提示文案。`type` 是 `common/wargoals/` 里某个类型的名字。原版国策里用得最多的两个是 `annex_everything`（533 处，吞并全部）和 `puppet_wargoal_focus`（226 处，傀儡化）；`take_state_focus` 配合 `generator = { 州编号 ... }` 指定要拿的州。

`annex_everything` 的定义里写着 `allowed = { always = no }`。那个 `allowed` 管的是玩家能不能在外交界面手动正当化这种理由；国策用 `create_wargoal` 直接发放，不经过手动正当化，所以它照样能给。

## 2. 过期时间

宣战理由会过期。类型定义里有的写了 `expire`（`take_state` 是 60，`puppet_wargoal_focus` 是 730），`annex_everything` 没写。国策里可以在效果上直接覆盖：

```
create_wargoal = {
    type = annex_everything
    target = BEL
    expire = 0
}
```

原版 1072 处 `create_wargoal` 里有 448 处带 `expire`，其中 435 处写的是 `expire = 0`。实测：`annex_everything` 不写 `expire` 和写 `expire = 0`，两条理由在外交面板上都不显示逾期日期，表现相同。要一条会过期的理由，写正数天数，原版有 10 处 `expire = 730`。

## 3. 后续国策以「已有宣战理由」为条件

```
focus = {
    id = my_focus_war_belgium
    prerequisite = { focus = my_focus_claim_belgium }
    x = 0
    y = 1
    cost = 5
    available = {
        has_wargoal_against = BEL
    }
    completion_reward = {
        declare_war_on = {
            target = BEL
            type = annex_everything
        }
    }
}
```

`has_wargoal_against = BEL` 判断对比利时有任意类型的宣战理由；要限定类型写块形式 `has_wargoal_against = { target = BEL type = annex_everything }`。原版里这个条件 127 处用 `PREV`、58 处用 `ROOT` 作目标，直接写标签的情况在国策树里更常见。

`declare_war_on` 是「直接宣战」，和「给理由」不是一回事：前者国策完成时战争就开始了，后者只是让玩家可以宣战。原版国策里 `declare_war_on` 只有 64 处，`create_wargoal` 有 1072 处，绝大多数剧情都把开战的决定留给玩家。

## 4. 用事件代替国策

同样的效果原样搬进事件选项即可，测试时用控制台 `event my_events.1` 直接触发：

```
add_namespace = my_events

country_event = {
    id = my_events.1
    title = my_events.1.t
    desc = my_events.1.d
    is_triggered_only = yes
    option = {
        name = my_events.1.a
        create_wargoal = {
            type = annex_everything
            target = BEL
        }
    }
}
```

## 5. 逐项核对

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 国策完成后比利时的外交面板 | 「宣战」可用，理由列表里有「吞并」 | `type` 拼写；`target` 是不是三字母标签 |
| 世界紧张度 | 没有因正当化上升 | 如果上升了，说明玩家走的是手动正当化，不是国策给的 |
| 理由的逾期日期 | 写正数天数才显示；不写或写 0 都不显示 | 类型定义里的 `expire` |
| 后续国策 | 有理由时可点，没理由时灰 | `has_wargoal_against` 是否写在 `available` 里 |
| `error.log` | 没有提到你的文件 | 类型不存在时游戏加载阶段就在这里记一行 |

## 常见错误

- 🔴 **`type` 写了一个 `common/wargoals/` 里没有的名字**：不会有理由出现。实测游戏加载时就在 `error.log` 记一行 `invalid war goal: 你写的名字: type`，带文件名和行号，进游戏前看一眼日志就能抓到。
- 🔴 **`target` 写了不存在的国家标签**：实测什么都不发生，`error.log` 也没有任何一行。荷兰是 `HOL` 不是 `NED`，标签以原版 `common/country_tags/` 为准。
- **以为 `create_wargoal` 会开战**：它只给权利。要直接开战用 `declare_war_on`。
- **把 `has_wargoal_against` 写成 `has_war_goal_against`**：原版文档里没有后者。
- **在历史文件里写 `create_wargoal`**：原版历史文件里只有 1 处这么做。开局就要能宣战的剧情，通常是在国策树第一条或开局事件里给。

## 想改成别的效果？

- 想拿到「傀儡化」而不是「吞并」的理由：`type = puppet_wargoal_focus`。
- 想只拿某几个州：`type = take_state_focus` 加 `generator = { 州编号 ... }`。
- 想国策完成就开战：`declare_war_on`，见第 3 节。
- 想在给理由的同时加宣称：`add_state_claim = 州编号`，写在同一个 `completion_reward` 里。
