---
contentId: stability
kind: topic
slug: stability
locale: zh
status: published
title: "稳定度：立即增减、每周变化与条件门槛"
summary: "想让事件加稳定度、让国家精神每周生效，还是要求稳定度达到门槛？先选游戏里要发生的事，再看对应写法。"
primaryTopic: politics
topics: [politics]
scenarios: [event, focus, decision, ideas]
gameTerms: [稳定度, 基础稳定度, 每周稳定度, 国家精神]
aliases: [安定度, 稳定度加5%, 稳定度不足, 稳定度每周下降]
relatedIds: ["wiki:effect:add_stability", "wiki:trigger:has_stability", "wiki:modifier:stability_weekly", "wiki:modifier:stability_factor", "wiki:modifier:stability_weekly_factor", event-stability]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "稳定度 基础稳定度 每周稳定度 增加 减少 立即 奖励 5个百分点 稳定度加5% 至少50% 稳定度不足 每周下降 稳定度不是政治点"
sources:
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects#add_stability"
  - label: "HOI4 Wiki：条件"
    url: "https://hoi4.paradoxwikis.com/Triggers#has_stability"
  - label: "HOI4 Wiki：修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers#stability_weekly"
---

“稳定度 +5%”可能是一笔事件奖励，也可能是一个持续存在的国家精神。先分清什么时候变化，才能找到合适的脚本。

## 你想让游戏里发生什么？

| 游戏目标 | 从这里开始 | 需要分清的地方 |
| --- | --- | --- |
| 选择事件选项后，立刻增加或减少稳定度 | 立即增减稳定度 (`wiki:effect:add_stability`) | 一次改变基础值，不会每周自动重复 |
| 稳定度至少达到 50% 才能使用内容 | 检查稳定度门槛 (`wiki:trigger:has_stability`) | 只检查，不奖励，也不消耗稳定度 |
| 国家精神生效期间，每周增加或减少稳定度 | 每周固定变化 (`wiki:modifier:stability_weekly`) | 按周，不是按天；还要安排来源持续多久 |

想从完整文件开始，可以看做一个增加稳定度的事件选项 (`event-stability`)。案例带有决议入口、事件与中文本地化，不需要先猜事件该从哪里触发。

## “5%”在这里是多少？

稳定度脚本通常使用 0 到 1 的比例刻度：`0.50` 对应 50%，`0.05` 对应 5 个百分点。

例如 `add_stability = 0.05` 增加 5 个百分点的基础稳定度。假设没有其他变化、也没有碰到显示或数值限制，50% 加 5 个百分点是 55%，不是 52.5%。游戏提示中的“+5%”不能一律理解成“当前数值乘以 1.05”。

同样，`stability_weekly = 0.01` 表示每周固定变化量增加 1 个百分点，不是每周把现有稳定度乘以 1.01。界面上的稳定度还会受其他来源影响，别只看最终总数判断一条脚本是否执行。

## 两个容易找错的相邻用法

- “拥有这个国家精神时稳定度更高，移除后取消这项加成”：看 stability_factor 原始参考 (`wiki:modifier:stability_factor`)。这与事件的一次性基础值变化不同。
- “调整已有的每周稳定度增长比例”：看 stability_weekly_factor 原始参考 (`wiki:modifier:stability_weekly_factor`)。它与固定的每周变化量不是同一项。

这两条目前是原始参考入口，不把它们当成本专题已经展开的完整教程。尤其不要因为名字带 `factor`，就假定所有数值都按同一种倍率规则计算。

## 放置位置决定玩家看到什么

事件的 `option` 放一次性效果；条件写进对应的条件块；国家精神的 `modifier` 放持续修正。不能把这三种脚本互换外层。

如果你只想限制一个按钮能否使用，通常要让玩家仍看得到不满足的条件。把条件放进事件选项的 `trigger` 会影响选项是否出现；普通决议的 `available` 则用于能否选择。两者不是一样的体验。
