---
contentId: "wiki:effect:add_stability"
kind: reference
slug: add_stability
locale: zh
status: published
title: "立即增加或减少稳定度"
summary: "用 add_stability 一次改变基础稳定度。0.05 是增加 5 个百分点，-0.05 是减少 5 个百分点，不是每周增长。"
primaryTopic: politics
topics: [politics]
scenarios: [event, focus, decision]
gameTerms: [稳定度, 基础稳定度]
aliases: [稳定度加5%, 稳定度减5%, 增加稳定度, 降低稳定度, 事件稳定度奖励]
relatedIds: [stability, event-stability, "wiki:trigger:has_stability", "wiki:modifier:stability_weekly", "wiki:modifier:stability_factor"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: effect/add_stability
searchText: "add_stability 立即增加稳定度 立即减少稳定度 稳定度加5% 稳定度减5% 一次增加5个百分点 扣除5个百分点 事件选项 国策奖励 基础稳定度 不是每周"
sources:
  - label: "HOI4 Wiki：add_stability"
    url: "https://hoi4.paradoxwikis.com/Effects#add_stability"
  - label: "HOI4 Wiki：事件制作"
    url: "https://hoi4.paradoxwikis.com/Event_modding"
---

技术名：`add_stability` · 执行效果（Effect）· 国家作用域。

## 1. 游戏里会发生什么，在哪里体现？

一次增加或减少当前国家的基础稳定度。玩家可以在事件选项的效果提示和国家稳定度提示中查看变化。

这不是持续加成，也不是“以后每周都加一次”。游戏最终显示的稳定度还会受到其他修正和限制影响，所以不要承诺任何情况下总数都会恰好跳动同样幅度。

## 2. 对谁生效，何时生效或失效？

对执行这行效果的国家生效。在普通国家事件的 `option` 中、没有切换作用域时，目标就是收到事件的国家；放进国策 `completion_reward` 时，是完成国策的国家。

效果块每执行一次，就做一次数值变化。事件窗口关闭不会自动撤销；以后其他效果仍可改变稳定度。若调用入口可重复使用，奖励也可能重复执行，要在调用入口明确是否限次。

## 3. 数值或参数怎么填？

| 想要的变化 | 写法 |
| --- | --- |
| 基础稳定度增加 5 个百分点 | `add_stability = 0.05` |
| 基础稳定度减少 5 个百分点 | `add_stability = -0.05` |

数值使用比例刻度，不写百分号。`0.05` 不是“增加 0.05 个百分点”，也不是“增加当前稳定度的 5%”。在没有其他变化和限制的简单例子中，50% 增加 5 个百分点是 55%。

不要因为想写“+5%”就填 `5`。某些文档 dump 和旧教程里的整数示例会让人误解单位；本文按游戏显示格式与原版小数用法核对。

## 4. 放在哪里，怎样写？

下面是一条局部效果，放到已有国家事件的 `option = { ... }` 内，与原有 `name` 同级：

```pdx
add_stability = 0.05
```

不要在选项里面再包一层 `effect = { ... }`，也不要放进选项的 `trigger`。如果写入 `immediate`，它会在事件触发时执行，而不是等玩家选择。

减益写法只替换这一行：

```pdx
add_stability = -0.05
```

这两段是替代关系，不要为了同时展示正负示例把两行都加到同一个选项。完整事件、触发入口和本地化见事件案例 (`event-stability`)。

## 5. 接下来可能需要什么？

- 先满足某个稳定度要求：检查门槛 (`wiki:trigger:has_stability`)。判断本身不会扣除稳定度。
- 想让稳定度持续按周变化：每周固定变化 (`wiki:modifier:stability_weekly`)。
- 想让一项加成随国家精神移除而消失：stability_factor 原始参考 (`wiki:modifier:stability_factor`)，不是把本效果改成每帧重复执行。
- 不确定应该选哪种：《稳定度专题》（文章 `stability`）。
