---
contentId: "wiki:effect:add_building_construction"
kind: reference
slug: add_building_construction
locale: zh
status: published
title: "在指定地区增加建筑"
summary: "用 add_building_construction 增加建筑等级。这里演示首都地区立即增加一座民用工厂，并检查所有权、控制权和空余槽位。"
primaryTopic: industry
topics: [industry]
scenarios: [focus, event, decision]
gameTerms: [民用工厂, 建筑, 建设]
aliases: [送一座民工, 立即获得工厂, 国策送工厂, 首都建厂, 瞬间建成建筑]
relatedIds: [construction-speed, "wiki:modifier:production_speed_buildings_factor", "wiki:trigger:num_of_civilian_factories", "wiki:trigger:free_building_slots"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: effect/add_building_construction
searchText: "add_building_construction 立即获得一座民用工厂 首都增加建筑 国策奖励工厂 直接送民工 type industrial_complex level 1 instant_build yes STATE作用域 不是建设速度 不是降低建筑等级"
sources:
  - label: "HOI4 Wiki：add_building_construction"
    url: "https://hoi4.paradoxwikis.com/Effects#add_building_construction"
  - label: "HOI4 Wiki：作用域"
    url: "https://hoi4.paradoxwikis.com/Scopes#capital_scope"
  - label: "HOI4 Wiki：建筑制作"
    url: "https://hoi4.paradoxwikis.com/Building_modding"
  - label: "HOI4 Wiki：建筑槽位条件"
    url: "https://hoi4.paradoxwikis.com/Triggers#free_building_slots"
---

技术名：`add_building_construction` · 执行效果（Effect）· 地区（State）作用域。

## 1. 游戏里会发生什么，在哪里体现？

为目标地区或省份安排指定等级的建筑建设。本文只演示民用工厂这一种地区建筑，并显式使用 `instant_build = yes`，让奖励直接建成；玩家应查看目标地区的工厂数。

这不是建设速度加成。若你要的是“未来盖厂更快”，应使用建设速度修正，而不是不断重复赠送建筑。

## 2. 对谁生效，何时生效或失效？

效果在目标地区执行，不能直接放在国家作用域中。本文从国家事件选项切到该国首都所在地区，并检查这个地区仍由事件接收国拥有和控制。

每次执行都会尝试增加指定等级。它没有自动到期撤销机制，也没有自带“只奖励一次”的限制。国策、事件或决议的调用入口要负责限次。

## 3. 数值或参数怎么填？

| 参数 | 本例 | 含义 |
| --- | --- | --- |
| `type` | `industrial_complex` | 民用工厂，不是军用工厂 |
| `level` | `1` | 增加一级，即本例的一座民用工厂；不是把现有总量设成 1 |
| `instant_build` | `yes` | 直接建成；`no` 用于启动正常建设 |

本文不省略 `instant_build`，避免读者猜测是否还要等待建设队列。`level` 是建筑等级数量，不是政治点或百分比。不要用负数从本例推导拆除建筑的办法。

还要满足目标地区的建筑槽位和上限。下面的检查只使用已解锁的空余共享槽位，不扩容，也不承诺适用于海军基地、铁路等所有其他建筑类型。

## 4. 放在哪里，怎样写？

这是已有普通国家事件 `option` 内的局部效果块。前提是该国家有有效首都；本段没有在更外层切换过国家，`ROOT` 是事件接收国：

```pdx
capital_scope = {
    if = {
        limit = {
            is_owned_by = ROOT
            is_controlled_by = ROOT
            free_building_slots = {
                building = industrial_complex
                size > 0
                include_locked = no
            }
        }
        add_building_construction = {
            type = industrial_complex
            level = 1
            instant_build = yes
        }
    }
}
```

这里的两层作用域分别是国家事件与首都地区。不能删除 `capital_scope` 后把建筑效果留在国家层，也不能把整个片段直接粘到地区历史的 `buildings` 内。

如果首都没有空位或不符合控制条件，本段不会赠送工厂，也不会自动寻找其他地区。正式内容应让入口条件或提示说明这个限制，或者另行设计并核对候选地区选择逻辑。

## 5. 接下来可能需要什么？

- 检查其他建筑的可用槽位：free_building_slots 原始参考 (`wiki:trigger:free_building_slots`)。
- 让之后的建设更快：建设速度修正 (`wiki:modifier:production_speed_buildings_factor`)。
- 将工厂数量作为门槛：num_of_civilian_factories (`wiki:trigger:num_of_civilian_factories`)，这不等于检查首都有没有槽位。
- 返回《建设速度专题》（文章 `construction-speed`）。
