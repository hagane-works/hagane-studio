---
contentId: "wiki:modifier:supply_consumption_factor"
kind: reference
slug: supply_consumption_factor
locale: zh
status: published
title: "提高或降低部队的补给消耗"
summary: "改变地面部队所需补给量：负数让部队更省补给，正数让部队更耗补给。"
primaryTopic: military
topics: [military]
scenarios: [ideas, focus, event, decision]
gameTerms: [补给消耗, 地面部队, 国家精神]
aliases: [部队补给需求, 部队少吃补给, 部队多吃补给, Supply Consumption]
relatedIds: [supply-consumption, idea-supply-consumption]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/supply_consumption_factor
searchText: "补给消耗 Supply Consumption supply_consumption_factor 改变地面部队所需补给。-0.10 是补给消耗 -10%，0.10 是补给消耗 +10%。本文使用国家精神内部的 modifier 承载。不是增加补给供给、补给枢纽能力或铁路运力。"
sources:
  - label: "HOI4 Wiki：补给消耗修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers#supply_consumption_factor"
  - label: "HOI4 Wiki：国家精神"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---
## 1. 游戏里会发生什么，在哪里体现？

游戏中的“补给消耗”指地面部队有效运作所需的补给量。降低它，是让部队少需要一些补给；提高它，则是让部队更耗补给。

在国家精神的修正说明和地面部队相关补给信息里核对方向。不要只看某一刻地图上的补给颜色：部署数量、铁路、补给枢纽等变化也会影响补给状况。

## 2. 对谁生效，何时生效或失效？

本文使用国家精神承载，作用于拥有该精神的国家的地面部队。国家精神生效期间持续提供修正，移除后不再提供。

原版已经在国家精神的 `modifier` 块中使用 `supply_consumption_factor`。文档中的军队修正分类不是“只能写在单位定义中”的意思；也不能因此把修正裸写进国策奖励。

## 3. 数值或参数怎么填？

| 写法 | 方向 |
| --- | --- |
| `supply_consumption_factor = -0.10` | 补给消耗 -10%，更省补给 |
| `supply_consumption_factor = 0.10` | 补给消耗 +10%，更耗补给 |
| `supply_consumption_factor = 0` | 本项不改变补给消耗 |

这里的 `-0.10` 不是“补给减 0.1 点”。为了让部队更省补给，应降低消耗，而不是因为“要加强部队”就填写正数。

这项修正不凭空建立铁路、补给枢纽或运输能力，也不改变燃油消耗。即使消耗下降，补给系统仍可能不足。

## 4. 放在哪里，怎样写？

下面是**国家精神定义内部的局部片段**，不是完整文件。片段已经包含 `modifier` 外层；若精神已有这个块，只把内部的修正行加入其中，不要再嵌套一层 `modifier`：

```hoi4
modifier = {
    supply_consumption_factor = -0.10
}
```

需要同时有国家精神定义和赋予它的效果。放在国策奖励中的是 `add_ideas`，而不是这条 modifier。

国家精神让部队少消耗 10% 补给 (`idea-supply-consumption`) 给出完整定义、显示文本和国策接入位置。

## 5. 接下来可能需要什么？

- 想先分清“少消耗”和“多供应”：《补给消耗专题》（文章 `supply-consumption`）。
- 已经写了修正，但国家没有收到精神：完整案例 (`idea-supply-consumption`)。

本页已对照 1.19.2.0 的修正文档、原版国家精神与中英文“补给消耗”说明；未执行游戏实测。
