---
contentId: "wiki:trigger:num_of_civilian_factories"
kind: reference
slug: num_of_civilian_factories
locale: zh
status: published
title: "检查民用工厂数量是否达到门槛"
summary: "用 num_of_civilian_factories 判断国家的民用工厂数量。至少 10 座与超过 10 座不同；判断不会扣除、预留或赠送工厂。"
primaryTopic: industry
topics: [industry]
scenarios: [decision, focus, event]
gameTerms: [民用工厂, 决议条件, 国策条件]
aliases: [民工数量, 至少10座民工, 民用工厂不足, 工厂数量条件]
relatedIds: [construction-speed, idea-construction-speed, "wiki:effect:add_building_construction", "wiki:modifier:production_speed_buildings_factor", "wiki:trigger:num_of_civilian_factories_available_for_projects"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: trigger/num_of_civilian_factories
searchText: "num_of_civilian_factories 检查民用工厂数量 至少10座民用工厂 民工少于10座 超过10座 不超过10座 民用工厂不足 数量门槛 不扣除 不预留 不是空闲工厂"
sources:
  - label: "HOI4 Wiki：num_of_civilian_factories"
    url: "https://hoi4.paradoxwikis.com/Triggers#num_of_civilian_factories"
  - label: "HOI4 Wiki：决议制作"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
---

技术名：`num_of_civilian_factories` · 判断条件（Trigger）· 国家作用域。

## 1. 游戏里会发生什么，在哪里体现？

判断当前国家的民用工厂数量是否满足条件。例如在普通决议 `available` 中设置门槛，工厂不足时玩家不能选择这个决议。

它不会增加、扣除或占用工厂，也不改变建设速度。民用工厂数量和军用工厂装备产出是不同概念，不应共用同一句“工厂不够”的说明。

## 2. 对谁生效，何时生效或失效？

判断的是当前国家；不是某个地区的建筑数量，也不是首都的空余建筑槽位。每次条件检查读取当时的结果，没有“检查成立后替你保留十座工厂”的功能。

不要将它解释成“空闲的、可随时投入新项目的民用工厂”。游戏另有检查新项目可用工厂的条件。随附文档没有完整说明本键在贸易、占领、境外工厂等情况下的所有计数细节，特殊设计需要在目标版本中单独核对。

## 3. 数值或参数怎么填？

本例按工厂座数比较，不写 `0.10` 来表示十座，也不写百分号。

| 目标 | 条件 | 恰好 10 座时 |
| --- | --- | --- |
| 少于 10 座 | `num_of_civilian_factories < 10` | 不成立 |
| 超过 10 座 | `num_of_civilian_factories > 10` | 不成立 |
| 至少 10 座 | `NOT = { num_of_civilian_factories < 10 }` | 成立 |
| 不超过 10 座 | `NOT = { num_of_civilian_factories > 10 }` | 成立 |

包含边界时使用单个比较条件取反。不要把“至少十座”写成 `> 10`；也不要把其他 Trigger 的 `=` 规则直接套过来。

## 4. 放在哪里，怎样写？

这是已有普通决议的局部条件块：

```pdx
available = {
    NOT = {
        num_of_civilian_factories < 10
    }
}
```

已有 `available` 时，将内部条件合并进去，保留其他要求。条件不满足时让决议仍可见，玩家才能查到缺少什么；若把条件放进 `visible`，决议会被隐藏。

如果你想要“支付若干工厂的持续使用权”，这个判断本身不够。不要为了模拟费用，在判断通过后直接删除建筑。

## 5. 接下来可能需要什么？

- 判断通过后授予建设加速：180 天国家精神案例 (`idea-construction-speed`)。
- 真正赠送工厂：增加建筑 (`wiki:effect:add_building_construction`)。
- 真正改变施工速度：建设速度修正 (`wiki:modifier:production_speed_buildings_factor`)。
- 设计需要占用工厂的新项目：num_of_civilian_factories_available_for_projects 原始参考 (`wiki:trigger:num_of_civilian_factories_available_for_projects`)，需继续核对对应项目的占用机制。
- 返回《建设速度专题》（文章 `construction-speed`）。
