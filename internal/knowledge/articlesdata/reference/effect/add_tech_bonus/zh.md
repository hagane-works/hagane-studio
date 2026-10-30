---
contentId: "wiki:effect:add_tech_bonus"
kind: reference
slug: add_tech_bonus
locale: zh
status: published
title: "给予有限次数的研究加成"
summary: "给下一次或几次指定科技研究提供加成，常用于国策奖励；不会直接解锁科技。"
primaryTopic: research
topics: [research]
scenarios: [focus, event, decision]
gameTerms: [研究加成, 工业科技, 国策奖励]
aliases: [一次科研加成, 限次研究加成, 下一次研究, 科技研究奖励, Research bonus]
relatedIds: [research-speed, focus-research-bonus, "wiki:modifier:research_speed_factor", "wiki:effect:set_technology"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: effect/add_tech_bonus
searchText: "add_tech_bonus 在国家作用域立即给予有限次数研究加成。bonus 是百分比，uses 是次数，category 是科技类别 ID，technology 可指定单个科技 ID。例子给予一次 50% 工业研究加成。"
sources:
  - label: "HOI4 Wiki：给予研究加成"
    url: "https://hoi4.paradoxwikis.com/Effects#add_tech_bonus"
  - label: "HOI4 Wiki：科技"
    url: "https://hoi4.paradoxwikis.com/Technology_modding"
---
## 1. 游戏里会发生什么，在哪里体现？

国家获得若干次研究加成，供符合范围的科技研究使用。比如国策奖励里常见的“一次 50% 工业研究加成”。

它发放的是可用加成，不是当场完成科技。先看国策奖励提示，再到科技界面核对对应科技的研究加成；不要用“是否已经解锁”判断它有没有生效。

## 2. 对谁生效，何时生效或失效？

`add_tech_bonus` 是国家作用域的效果。国策完成奖励中，它通常作用于完成该国策的国家；若上层脚本切换了作用域，应先确认当前指向的是哪个国家。

效果执行时给予加成，可用次数由 `uses` 指定。它不是持续性的全局科研速度修正。此处不承诺对正在研究中的科技何时计入或消耗加成，测试时应另选尚未开始、且符合类别的科技。

## 3. 数值或参数怎么填？

| 参数 | 本例 | 含义 |
| --- | --- | --- |
| `name` | `HW_industry_research_bonus` | 这项加成的本地化键，需补显示名称 |
| `bonus` | `0.50` | 50% 研究加成，不是减 50 天 |
| `uses` | `1` | 可使用一次，不是持续一天 |
| `category` | `industry` | 原版科技类别 ID，不是界面上的中文标题 |

只针对某项科技时，可以使用 `technology` 指定科技 ID；它与装备 ID 不是一回事。本文的可复制例子只使用已核对的 `industry` 类别，不把两种范围混写。

50% 研究加成不等于保证完工时间减半。超前惩罚减免也有单独参数，不要把“研究加成”与“超前惩罚减少若干年”混成一个数字。

## 4. 放在哪里，怎样写？

下面是**效果块内的完整效果片段**，可以加入现有国策的 `completion_reward`，但不是独立文件，也不要再把 `completion_reward` 嵌套进去：

```hoi4
add_tech_bonus = {
    name = HW_industry_research_bonus
    bonus = 0.50
    uses = 1
    category = industry
}
```

本例用到的显示名称还需加入简体中文本地化文件。下面是**该本地化文件的最小完整内容**，保存为 `localisation/simp_chinese/hw_research_l_simp_chinese.yml`，编码使用 UTF-8-BOM：

```yaml
l_simp_chinese:
 HW_industry_research_bonus: "工业研究交流"
```

国策奖励一次工业研究加成 (`focus-research-bonus`) 包含国策节点、图标、名称和检查清单。

## 5. 接下来可能需要什么？

- 每一项研究都持续变快：科研速度修正 (`wiki:modifier:research_speed_factor`)。
- 国策一完成就拥有某项科技：直接解锁指定科技 (`wiki:effect:set_technology`)。
- 对比三种奖励的用途：《科研专题》（文章 `research-speed`）。

本页已对照 1.19.2.0 的国家作用域文档、原版国策、科技类别表和奖励提示文本；未执行游戏实测。
