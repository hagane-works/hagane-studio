---
contentId: construction-speed
kind: topic
slug: construction-speed
locale: zh
status: published
title: "建设速度：加快施工，还是直接奖励工厂？"
summary: "建设速度、立即建成建筑和民用工厂数量条件是三种用法。按你希望玩家得到的结果选择，别把施工加速写成装备产出。"
primaryTopic: industry
topics: [industry]
scenarios: [ideas, focus, event, decision]
gameTerms: [建设速度, 民用工厂, 国家精神]
aliases: [建造速度, 造厂速度, 民工, 送工厂, 建筑施工]
relatedIds: ["wiki:modifier:production_speed_buildings_factor", "wiki:effect:add_building_construction", "wiki:trigger:num_of_civilian_factories", "wiki:trigger:free_building_slots", "wiki:modifier:industrial_capacity_factory", idea-construction-speed]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "建设速度 建造速度 民用工厂 民工 建设加速 减速 国家精神 建设速度增加10% 建设速度减少10% 立即送一座工厂 民用工厂至少10座 不是装备产出 不是每日送厂"
sources:
  - label: "HOI4 Wiki：建设速度修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers#production_speed_buildings_factor"
  - label: "HOI4 Wiki：建筑效果"
    url: "https://hoi4.paradoxwikis.com/Effects#add_building_construction"
  - label: "HOI4 Wiki：工厂数量条件"
    url: "https://hoi4.paradoxwikis.com/Triggers#num_of_civilian_factories"
  - label: "HOI4 Wiki：国家精神制作"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---

玩家说“多给点工业”，可能想要工厂盖得更快，也可能想直接拿到一座工厂。先选结果，再处理脚本。

## 选择你要的结果

| 游戏目标 | 对应用法 | 不会替你做什么 |
| --- | --- | --- |
| 国家精神存在时，建设速度提高或降低 10% | 建设速度修正 (`wiki:modifier:production_speed_buildings_factor`) | 不直接送工厂，不增加军用工厂的装备产出 |
| 国策或事件立即奖励一座民用工厂 | 增加建筑 (`wiki:effect:add_building_construction`) | 不提高以后所有建筑的建设速度 |
| 民用工厂达到一定数量后才能选择内容 | 民用工厂数量门槛 (`wiki:trigger:num_of_civilian_factories`) | 不增加、不扣除，也不预留工厂 |

需要完整例子，可以看获得持续 180 天的建设速度国家精神 (`idea-construction-speed`)。例子把精神定义、授予它的决议、持续时间和中文本地化放在一起。

## 建设速度不等于生产速度

`production_speed_buildings_factor` 虽然带有 `production`，游戏内显示名称却是“建设速度”，作用是建筑施工。

如果你想让军用工厂生产更多装备，要查的是另一个方向，例如工厂产出原始参考 (`wiki:modifier:industrial_capacity_factory`)。把“造工厂更快”和“工厂造装备更多”分开，玩家输入的“工厂加速”才有明确含义。

## 数值与持续时间分开写

建设速度修正 `0.10` 表示 +10%，`-0.10` 表示 -10%。它不是每天加 0.1 座工厂，也不是把剩余建设时间直接减去 10%。最终工期还与分配的民用工厂、其他建设修正和施工条件有关。

持续 180 天的是承载修正的国家精神，不是修正键自带的计时功能。只在精神定义里写好 `modifier`，国家还不会自动获得它；需要由效果授予。

## 直接送建筑时，先确定位置

“国家得到一座民用工厂”仍然需要决定建筑落在哪个地区。`add_building_construction` 在地区（State）作用域执行；在国家事件里使用时，要先切到目标地区。

还要检查空余建筑槽位 (`wiki:trigger:free_building_slots`)。不要把“没有地方可建”误诊为“建设速度太低”，也不要为了一次奖励随意改动全局建筑上限。
