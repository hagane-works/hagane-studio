---
contentId: "wiki:trigger:has_stability"
kind: reference
slug: has_stability
locale: zh
status: published
title: "检查稳定度是否达到门槛"
summary: "用 has_stability 判断稳定度。50% 写成 0.5；至少 50% 与超过 50% 不同，条件不会消耗稳定度。"
primaryTopic: politics
topics: [politics]
scenarios: [decision, focus, event]
gameTerms: [稳定度, 决议条件, 国策条件]
aliases: [稳定度门槛, 至少50%稳定度, 稳定度不足, 稳定度低于50%]
relatedIds: [stability, "wiki:effect:add_stability", "wiki:modifier:stability_weekly", event-stability]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: trigger/has_stability
searchText: "has_stability 稳定度条件 稳定度不足 至少50% 低于50% 超过50% 不超过50% 检查稳定度 不扣除 不消耗 决议available"
sources:
  - label: "HOI4 Wiki：has_stability"
    url: "https://hoi4.paradoxwikis.com/Triggers#has_stability"
  - label: "HOI4 Wiki：决议制作"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
---

技术名：`has_stability` · 判断条件（Trigger）· 国家作用域。

## 1. 游戏里会发生什么，在哪里体现？

它只判断当前国家的稳定度是否满足比较条件，不会增加或扣除稳定度。

例如在普通决议的 `available` 中要求稳定度至少达到 50%，不足时决议仍可显示，但不能选择。若改放在 `visible`，影响的是是否出现；若放在事件选项的 `trigger`，影响的是那个选项是否出现。

## 2. 对谁生效，何时生效或失效？

判断当前国家，不是该国某个地区。外层内容每次检查这个条件时，会读取当时的稳定度；它不是在第一次成立后就永久解锁的标记。

“要求 50% 稳定度”与“花费 5 个百分点稳定度”是两件事。如果你的设计还需要代价，必须另行安排效果，不能把判断当作支付。

## 3. 数值或参数怎么填？

随附文档给出的稳定度刻度是 0 到 1：50% 写 `0.5`，不是 `50`，也不在脚本值里写 `%`。

| 目标 | 条件 | 恰好 50% 时 |
| --- | --- | --- |
| 低于 50% | `has_stability < 0.5` | 不成立 |
| 超过 50% | `has_stability > 0.5` | 不成立 |
| 至少 50% | `NOT = { has_stability < 0.5 }` | 成立 |
| 不超过 50% | `NOT = { has_stability > 0.5 }` | 成立 |

这里用单个比较条件取反表达含等号边界。不要用 `> 0.49` 冒充“至少 50%”，也不要根据界面四舍五入后的短数字判断精确边界。

## 4. 放在哪里，怎样写？

这是已有普通决议中的局部条件块：

```pdx
available = {
    NOT = {
        has_stability < 0.5
    }
}
```

若原来已经有 `available`，将内部条件合并进去，保留其他要求。不要再创建一个重复的同名块。

若设计是“低稳定度时才能采取补救措施”，改用 `has_stability < 0.5`。这仍是选择条件，不会自行给予稳定度奖励。

## 5. 接下来可能需要什么？

- 真正改变稳定度：立即增减稳定度 (`wiki:effect:add_stability`)。
- 做出完整事件选择：增加稳定度的事件案例 (`event-stability`)。案例是一次奖励，不是支付稳定度。
- 持续改善或恶化稳定度：每周固定变化 (`wiki:modifier:stability_weekly`)。
- 返回《稳定度专题》（文章 `stability`）。
