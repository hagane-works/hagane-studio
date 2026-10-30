---
contentId: "wiki:modifier:political_power_gain"
kind: reference
slug: political_power_gain
locale: zh
status: published
title: "每日政治点：固定增减"
summary: "political_power_gain 调整每日固定获取项。0.5 表示 +0.5 点，不是 +50%；持续时间由国家精神等来源决定。"
primaryTopic: politics
topics: [politics]
scenarios: [ideas, focus, event, decision]
gameTerms: [每日政治点数获取, 政治点数, 国家精神]
aliases: [政治点, 政治力, PP, 每天加政治点, 每日固定政治点, 每日政治点不是百分比]
relatedIds: [political-power, "wiki:modifier:political_power_factor", "wiki:effect:add_political_power", "wiki:trigger:has_political_power", focus-political-power-reward]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/political_power_gain
searchText: "political_power_gain 每日政治点数获取 每天加1政治点 每日固定增加0.5政治点 每天减少0.5政治点 固定点数 非百分比 国家精神 90天"
sources:
  - label: "HOI4 Wiki：political_power_gain"
    url: "https://hoi4.paradoxwikis.com/Modifiers#political_power_gain"
  - label: "HOI4 Wiki：国家精神制作"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
  - label: "HOI4 Wiki：add_timed_idea"
    url: "https://hoi4.paradoxwikis.com/Effects#add_timed_idea"
---

技术名：`political_power_gain` · 调整数值（Modifier）· 本例来源：国家精神。

## 1. 游戏里会发生什么，在哪里体现？

国家精神生效时，它向拥有国的每日政治点获取提供固定数值贡献。`0.5` 表示每日固定获取项 +0.5 点，不是 +50%，也不是获得精神时立即领取 0.5 点。

游戏中的名称是“每日政治点数获取”。在国家精神说明和顶部政治点获取提示中核对。最终每日净增长还会受到比例修正及其他消耗影响，所以单个来源的 +0.5 不一定等于余额每天多出 0.5。

## 2. 对谁生效，何时生效或失效？

本例影响拥有该国家精神的国家。来源有效时参与每日计算，精神被移除或到期后停止提供这项贡献。

`political_power_gain` 自身不带天数。下面通过 `add_timed_idea` 授予 90 天；如果改用 `add_ideas`，则没有这次授予附带的到期天数。若另写了精神的生效条件，也要满足相应条件。

移除国家精神不会自动收回此前已经累积的政治点；若脚本另有扣款效果，那是另一项操作。

## 3. 数值或参数怎么填？

| 参数值 | 本项贡献 | 不是 |
| --- | --- | --- |
| `1` | 每日固定获取项 +1 点 | +1% |
| `0.5` | 每日固定获取项 +0.5 点 | +50% |
| `-0.5` | 每日固定获取项 -0.5 点 | 立即扣除 0.5 点 |
| `0` | 本项无贡献 | 禁止全部政治点获取 |

这是点数参数，可以使用小数。文档里的显示小数位不是计算精度保证。要做“获取增加 15%”，请看 `political_power_factor`，不要把 `0.15` 的两种单位混为一谈。

## 4. 放在哪里，怎样写？

以下定义放进你的 Mod 的 `common/ideas/hag_pp_examples.txt`。这是完整的国家精神定义文件，但单有定义还不会让国家获得它：

```pdx
ideas = {
    country = {
        HAG_pp_daily_flat = {
            allowed = { always = no }
            picture = generic_disjointed_gov
            removal_cost = -1
            modifier = {
                political_power_gain = 0.5
            }
        }
    }
}
```

本例借用原版已有图标。`allowed = { always = no }` 不妨碍脚本主动授予；`removal_cost = -1` 表示不能由玩家手动移除，不代表限时精神不会到期。

在已有国策的奖励块内授予 90 天：

```pdx
completion_reward = {
    add_timed_idea = {
        idea = HAG_pp_daily_flat
        days = 90
    }
}
```

名称和说明另写到 `localisation/simp_chinese/hag_pp_examples_l_simp_chinese.yml`，保存为 UTF-8 with BOM：

```yaml
l_simp_chinese:
 HAG_pp_daily_flat:0 "行政整顿"
 HAG_pp_daily_flat_desc:0 "改善日常行政工作，为政治事务提供稳定支持。"
```

定义中的 ID、授予的 ID 和本地化键必须一致。代码里的 `modifier` 属于国家精神定义，不能直接塞进普通国策的 `completion_reward`；国策在那里执行的是 `add_timed_idea`。

这些片段需要已有且能加载的国策树来承载授予操作，不是一份独立可启动的 Mod。已有同名文件或块时合并相应定义，不能覆盖别人的内容。

## 5. 接下来可能需要什么？

- 改成百分比加成：每日政治点：按比例调整 (`wiki:modifier:political_power_factor`)。
- 不想持续获取，只在完成时给一笔：立即增加政治点 (`wiki:effect:add_political_power`) 与 完整国策奖励案例 (`focus-political-power-reward`)。
- 想检查已经攒够多少：政治点门槛 (`wiki:trigger:has_political_power`)。
- 还在选择实现方式：《政治点专题》（文章 `political-power`）。
