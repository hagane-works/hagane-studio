---
contentId: political-power
kind: topic
slug: political-power
locale: zh
status: published
title: "政治点：奖励、门槛与每日获取"
summary: "想奖励一笔政治点、检查是否足够，还是改变每天的获取量？先选游戏里的目标，再看对应写法。"
primaryTopic: politics
topics: [politics]
scenarios: [focus, event, decision, ideas]
gameTerms: [政治点数, 每日政治点数获取, 国家精神, 国策奖励]
aliases: [政治点, 政治力, PP, political power]
relatedIds: ["wiki:effect:add_political_power", "wiki:trigger:has_political_power", "wiki:modifier:political_power_gain", "wiki:modifier:political_power_factor", focus-political-power-reward]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "政治点数 政治力 PP 立即奖励 门槛检查 每日固定获取 每日比例 政治点怎么加 国策 国家精神"
sources:
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki：条件"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki：修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
---

## 你想让政治点发生什么变化？

游戏上方显示的是当前持有的政治点数；每日获取则是余额变化的来源之一。这两件事用的不是同一条脚本。

| 游戏里的目标 | 从这里开始 | 关键区别 |
| --- | --- | --- |
| 完成国策后立即获得 150 点，或一次扣除一笔政治点 | 立即增加或减少政治点 (`wiki:effect:add_political_power`) · `add_political_power` | 改变当前余额，不修改每日获取 |
| 政治点至少达到 150 才允许使用 | 检查政治点是否达到门槛 (`wiki:trigger:has_political_power`) · `has_political_power` | 只判断条件，不扣费，也不发奖励 |
| 国家精神每天提供固定的 +0.5 点获取项 | 每日政治点：固定增减 (`wiki:modifier:political_power_gain`) · `political_power_gain` | 单位是点数，不是百分比 |
| 国家精神提供 +15% 的政治点获取修正 | 每日政治点：按比例调整 (`wiki:modifier:political_power_factor`) · `political_power_factor` | `0.15` 是 +15%，不是每日 +0.15 点 |

只搜索“增加政治点”时，先确认你说的是“现在多一笔”还是“以后每天多一些”。搜索结果会保留这些不同用途，不替你猜成同一种奖励。

## 先做一个能看懂的国策奖励

完成国策后获得 150 政治点 (`focus-political-power-reward`) 从已有国策树出发，写清节点插入位置、中文本地化、奖励数值以及验收方法。它不要求同时制作国家精神，也不会改变每日获取。

如果想做“国策完成后，每天获得更多政治点”，则需要用国策授予承载修正的国家精神。直接把 `political_power_gain` 塞进 `completion_reward` 不是同一种写法。

## 三件容易混淆的事

### 有足够政治点，不代表已经支付

条件成立只是允许后续逻辑继续。普通决议的政治点费用由 `cost` 处理；事件选项或自定义费用则要安排扣除效果。不要同时使用决议 `cost` 和同额扣除效果，造成重复收费。

### 来源到期，不会收回此前获得的政治点

每日获取修正在国家精神生效期间参与计算。国家精神到期或被移除后，这项贡献停止；它此前积累下来的政治点不会因此自动被扣回。

### 提示中的加成，不等于最后的每日净增长

固定获取项、比例修正以及国策等消耗共同影响最终结果。检查国家精神说明和顶部政治点提示，不要把单个来源的 `+0.5` 或 `+15%` 当成最终余额每天必然增加的数量。
