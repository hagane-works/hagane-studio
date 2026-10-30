---
contentId: "wiki:modifier:stability_weekly"
kind: reference
slug: stability_weekly
locale: zh
status: published
title: "每周固定增加或减少稳定度"
summary: "用 stability_weekly 调整每周稳定度变化量。0.01 是每周增加 1 个百分点，-0.01 是每周减少 1 个百分点；来源持续多久要另行安排。"
primaryTopic: politics
topics: [politics]
scenarios: [ideas, decision]
gameTerms: [每周稳定度, 稳定度, 国家精神]
aliases: [每周稳定度加1%, 每周稳定度减1%, 稳定度每周下降, 每周固定稳定度]
relatedIds: [stability, "wiki:effect:add_stability", "wiki:trigger:has_stability", "wiki:modifier:stability_weekly_factor", "wiki:effect:add_timed_idea", "wiki:effect:remove_ideas"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/stability_weekly
searchText: "stability_weekly 每周稳定度 每周增加1个百分点 每周减少1个百分点 每周稳定度加1% 每周稳定度减1% 国家精神 固定变化量 不是每日增长 不是每周比例倍率"
sources:
  - label: "HOI4 Wiki：stability_weekly"
    url: "https://hoi4.paradoxwikis.com/Modifiers#stability_weekly"
  - label: "HOI4 Wiki：国家精神制作"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
  - label: "HOI4 Wiki：决议制作"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
---

技术名：`stability_weekly` · 持续修正（Modifier）· 国家修正。

## 1. 游戏里会发生什么，在哪里体现？

改变国家的每周稳定度变化量，游戏显示名称是“每周稳定度”。例如国家精神存在期间每周增加 1 个百分点，或每周减少 1 个百分点。

它不是添加精神时就一次结算完整奖励，也不是每天变化。观察时要同时查看精神上的修正提示与稳定度提示，别只在授予后的当天比较总数。

## 2. 对谁生效，何时生效或失效？

国家精神携带它时，对拥有并启用该精神的国家生效；来源不再生效后，这项每周贡献也停止。停止后不会自动追回已经累积的稳定度变化。

这个修正键本身不设置天数。限时国家精神的期限由授予时的计时效果决定；带计时器的决议则有自己的持续时间。不要假定写了 `stability_weekly` 就自带“持续一周”。

## 3. 数值或参数怎么填？

| 目标 | 写在修正块中的值 |
| --- | --- |
| 每周固定增加 1 个百分点 | `stability_weekly = 0.01` |
| 每周固定减少 1 个百分点 | `stability_weekly = -0.01` |
| 每周固定增加 0.5 个百分点 | `stability_weekly = 0.005` |

单位是按周的固定变化量。这里的“每周 +1%”是游戏百分比刻度上的 1 个百分点，不是把当前稳定度乘以 1.01，也不是增长速度的相对倍率。

其他来源和数值限制可能影响最终观察到的结果，不要直接用“持续天数 ÷ 7 × 数值”承诺任意开局、任意结算时刻的最终稳定度。

## 4. 放在哪里，怎样写？

下面是已经定义好的国家精神内的局部块，放在该精神本体中：

```pdx
modifier = {
    stability_weekly = 0.01
}
```

若已有 `modifier`，合并其中的字段。精神必须实际授予国家才有这个来源；在事件选项里直接写 `stability_weekly = 0.01` 不是有效的授予方法。

想要负面精神，将值改为 `-0.01`。想要限时效果，通过 add_timed_idea 原始参考 (`wiki:effect:add_timed_idea`) 安排精神期限。这个片段不是完整的精神文件，名称、图标、本地化和授予入口仍需配齐。

## 5. 接下来可能需要什么？

- 只奖励或惩罚一次：add_stability (`wiki:effect:add_stability`)。
- 判断当前稳定度是否足够：has_stability (`wiki:trigger:has_stability`)。
- 想改每周增长的比例而非固定量：stability_weekly_factor 原始参考 (`wiki:modifier:stability_weekly_factor`)。
- 需要结束国家精神来源：remove_ideas 原始参考 (`wiki:effect:remove_ideas`)。
- 返回《稳定度专题》（文章 `stability`）。
