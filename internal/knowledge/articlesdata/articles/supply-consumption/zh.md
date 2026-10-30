---
contentId: supply-consumption
kind: topic
slug: supply-consumption
locale: zh
status: published
title: "调整部队的补给消耗"
summary: "让部队更省补给，和让地图上有更多补给，不是一回事。先确认要改的是哪一边。"
primaryTopic: military
topics: [military]
scenarios: [ideas, focus, event, decision]
gameTerms: [补给消耗, 补给需求, 地面部队]
aliases: [部队少吃补给, 部队多吃补给, 补给不够, Supply Consumption]
relatedIds: ["wiki:modifier:supply_consumption_factor", idea-supply-consumption, first-content]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "部队少吃补给对应降低补给消耗，部队多吃补给对应提高补给消耗。supply_consumption_factor 的负数减少消耗，正数增加消耗。国家精神可承载该修正；这不等于建铁路、增加补给供给或减少燃油消耗。"
sources:
  - label: "HOI4 Wiki：补给消耗修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers#supply_consumption_factor"
  - label: "HOI4 Wiki：国家精神"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---
游戏里的“补给消耗”是地面部队所需的补给量。想让部队在同样的补给条件下更容易维持，可以减少消耗；想设计后勤负担，则可以增加消耗。

## 先确认要改的是消耗还是供给

| 你的目标 | 这页是否适用？ |
| --- | --- |
| 让本国地面部队少消耗 10% 补给 | 适用，使用负数消耗修正 |
| 给国家一个让部队多消耗补给的负面精神 | 适用，使用正数消耗修正 |
| 建铁路、增加补给枢纽能力或改善输送 | 不是同一项参数，本页不提供这些写法 |
| 让坦克少消耗燃油 | 不是补给消耗修正，本页不混用燃油参数 |

“补给不足”可能有许多原因。降低消耗不代表补给一定充足，也不能替代对地图运输条件的检查。

## 数字的方向与直觉可能相反

想给部队带来好处，这里通常写负数：

- `supply_consumption_factor = -0.10`：补给消耗 -10%，部队更省补给。
- `supply_consumption_factor = 0.10`：补给消耗 +10%，部队更耗补给。

这是参数对照，不是独立文件。上述数值是一项百分比修正，不能把 `-0.10` 改写成“减少 0.1 点补给”，也不要丢掉负号。

## 选一条路继续

提高或降低部队的补给消耗 (`wiki:modifier:supply_consumption_factor`) 解释作用对象、数值和国家精神里的放置位置。

国家精神让部队少消耗 10% 补给 (`idea-supply-consumption`) 给出完整定义、本地化及接入已有国策奖励的方法。

## 为什么不直接写在国策奖励里？

`supply_consumption_factor` 是修正，不是“执行一次”的效果。本文让国家精神保存这项修正，再由国策用 `add_ideas` 把精神交给国家。

查看结果时，先核对国家确实获得精神，再核对说明中的“补给消耗 -10%”。地图颜色是否变化还受部队、运输线路等因素影响，不能单凭颜色为脚本判定成功。

本页已对照 1.19.2.0 的原版说明与国家精神用法；未执行游戏实测。
