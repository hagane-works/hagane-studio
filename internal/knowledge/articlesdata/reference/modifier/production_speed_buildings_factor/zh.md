---
contentId: "wiki:modifier:production_speed_buildings_factor"
kind: reference
slug: production_speed_buildings_factor
locale: zh
status: published
title: "提高或降低建设速度"
summary: "用 production_speed_buildings_factor 修改建筑建设速度。0.10 表示 +10%，不是直接送工厂，也不是提高装备产出。"
primaryTopic: industry
topics: [industry]
scenarios: [ideas, decision]
gameTerms: [建设速度, 国家精神, 民用工厂]
aliases: [建造速度, 造厂速度, 建设速度加10%, 建设速度减10%, 施工加速, 施工减速]
relatedIds: [construction-speed, idea-construction-speed, "wiki:effect:add_building_construction", "wiki:trigger:num_of_civilian_factories", "wiki:modifier:industrial_capacity_factory"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/production_speed_buildings_factor
searchText: "production_speed_buildings_factor 建设速度 建造速度 提高建设速度 建设速度增加10% 降低建设速度 建设速度减少10% 加快民用工厂施工 国家精神 持续修正 不是直接增加工厂 不是装备产出"
sources:
  - label: "HOI4 Wiki：production_speed_buildings_factor"
    url: "https://hoi4.paradoxwikis.com/Modifiers#production_speed_buildings_factor"
  - label: "HOI4 Wiki：国家精神制作"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---

技术名：`production_speed_buildings_factor` · 持续修正（Modifier）· 国家修正。

## 1. 游戏里会发生什么，在哪里体现？

游戏显示为“建设速度”，改变所有建筑的建设速度。国家精神携带它时，玩家可以在精神提示中看到这项加成或惩罚，并在建设界面观察施工。

这里的 `production` 不代表军用工厂生产装备。它也不创建建筑；如果根本没有安排施工，不能靠这个修正直接获得一座工厂。

## 2. 对谁生效，何时生效或失效？

本文采用国家精神作为来源：国家拥有且启用这个精神时获得修正，来源失效后停止提供这项速度加成。已经建成的建筑不会因此被删除。

键名不包含持续时间，也不是每天结算一份奖励。若只想持续 180 天，要在授予国家精神时安排期限。多个来源并存时，不要把“本精神 +10%”当作当前总建设速度必然恰好乘以 1.10。

## 3. 数值或参数怎么填？

| 目标 | 修正值 |
| --- | --- |
| 建设速度 +10% | `production_speed_buildings_factor = 0.10` |
| 建设速度 -10% | `production_speed_buildings_factor = -0.10` |

这是百分比修正，用小数，不在值里写 `%`。`0.10` 不是每天增加 0.1 座工厂；+10% 速度也不等于把剩余工期直接减去 10%。其他建设修正、施工条件和分配工厂数都要一起看。

若只想加快某一种建筑，需另外核对该建筑对应的专用修正，不能把本键当作“仅民用工厂”的效果。

## 4. 放在哪里，怎样写？

下面是已有国家精神中的局部块：

```pdx
modifier = {
    production_speed_buildings_factor = 0.10
}
```

放在国家精神本体内；若已有 `modifier`，合并这一行。不要直接写在事件 `option` 或普通国策 `completion_reward` 内，它不是可执行效果。

需要国策或事件授予它时，应先定义国家精神，再执行授予精神的效果。完整国家精神案例 (`idea-construction-speed`)包含定义、一次性决议入口、180 天期限与本地化。若要减速，只替换数值和相应文案，别保留正向奖励的说明。

## 5. 接下来可能需要什么？

- 想直接拿到一座工厂：增加建筑 (`wiki:effect:add_building_construction`)。
- 想限制已有多少民用工厂才能使用：数量门槛 (`wiki:trigger:num_of_civilian_factories`)。
- 想增加军用工厂的装备产出：industrial_capacity_factory 原始参考 (`wiki:modifier:industrial_capacity_factory`)。
- 返回《建设速度专题》（文章 `construction-speed`）。
