---
contentId: "wiki:modifier:political_power_factor"
kind: reference
slug: political_power_factor
locale: zh
status: published
title: "每日政治点：按比例调整"
summary: "political_power_factor 为政治点获取提供比例修正。0.15 表示 +15%，不是立即奖励，也不是每日固定 +0.15 点。"
primaryTopic: politics
topics: [politics]
scenarios: [ideas, focus, event, decision]
gameTerms: [政治点数, 政治点获取, 国家精神]
aliases: [政治点, 政治力, PP, 政治点获取百分比, 政治点获取倍率, 政治点加15%]
relatedIds: [political-power, "wiki:modifier:political_power_gain", "wiki:effect:add_political_power", "wiki:trigger:has_political_power", focus-political-power-reward]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/political_power_factor
searchText: "political_power_factor 每日政治点增加15% 政治点获取提高百分之十五 每日政治点减少15% 政治点获取降低百分之十五 比例修正 百分比 获取倍率 国家精神"
sources:
  - label: "HOI4 Wiki：political_power_factor"
    url: "https://hoi4.paradoxwikis.com/Modifiers#political_power_factor"
  - label: "HOI4 Wiki：国家精神制作"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---

技术名：`political_power_factor` · 调整数值（Modifier）· 本例来源：国家精神。

## 1. 游戏里会发生什么，在哪里体现？

它给政治点获取提供比例修正。国家精神中填 `0.15`，表示该来源贡献 +15% 的获取修正；填 `-0.15` 则贡献 -15%。

简中界面可能把这一项简写为“政治点数”，旁边用百分比显示；固定获取项则叫“每日政治点数获取”。要一起看名称和单位，不能只认“政治点”三个字。

它不把已有政治点余额乘以 1.15，也不会立即发放 15 点，更不是每日固定 +0.15 点。

## 2. 对谁生效，何时生效或失效？

在本例国家精神中，拥有国在该来源有效时得到修正。精神被移除或到期后，这项比例贡献停止；此前已经获得的政治点不会自动被收回。

时效由承载来源及其授予方式决定，不由 `political_power_factor` 这个数值决定。不要在参数后面添加 `days`，也不要因为它叫“修正”就理解成永久效果。

## 3. 数值或参数怎么填？

| 参数值 | 本来源提供的比例修正 | 常见误读 |
| --- | --- | --- |
| `0.15` | +15% | 不是 +0.15 点 |
| `-0.15` | -15% | 不是一次扣除 15 点 |
| `1` | +100% | 不是 +1% |
| `0` | 本来源没有比例贡献 | 不是每日获取归零 |

代码中不写 `%`。想表达 +15% 应写 `0.15`，而不是 `15`；后者会成为 +1500% 的修正。

同类比例修正作为加减项合并，不能把每一项都独立连乘。最终每日净增长还涉及固定获取和消耗，因此“再加一个 +15%”不保证让界面上原有的净增长数字恰好乘以 1.15。本文不凭这个参数推导所有 Mod 的完整政治点公式。

## 4. 放在哪里，怎样写？

以下是国家精神定义文件，可放到你的 Mod 的 `common/ideas/hag_pp_factor_example.txt`：

```pdx
ideas = {
    country = {
        HAG_pp_daily_factor = {
            allowed = { always = no }
            picture = generic_disjointed_gov
            removal_cost = -1
            modifier = {
                political_power_factor = 0.15
            }
        }
    }
}
```

本例借用原版已有图标。定义完成后，还需要在国家效果块中授予。以下是已有国策节点内的局部奖励块：

```pdx
completion_reward = {
    add_ideas = HAG_pp_daily_factor
}
```

这里的 `add_ideas` 不附带到期天数。若要在现有效果块中主动移除，用 `remove_ideas = HAG_pp_daily_factor`；若要授予固定天数，可按固定获取词条的限时示例 (`wiki:modifier:political_power_gain`)改用 `add_timed_idea`，并把 ID 改为本页的 `HAG_pp_daily_factor`。

中文名称和说明放到 `localisation/simp_chinese/hag_pp_factor_example_l_simp_chinese.yml`，保存为 UTF-8 with BOM：

```yaml
l_simp_chinese:
 HAG_pp_daily_factor:0 "政治协调"
 HAG_pp_daily_factor_desc:0 "协调各方行动，提高政治工作的效率。"
```

这仍以已有国策树为前提；不要把整个 `modifier` 定义当作国策奖励，也不要把 `0.15` 直接填到 `add_political_power` 里。科技、人物特质等来源的外层结构需另行核对，不能照搬国家精神外壳。

## 5. 接下来可能需要什么？

- 实际想要每天固定多 0.5 点：每日政治点：固定增减 (`wiki:modifier:political_power_gain`)。
- 实际想要完成国策就领 150 点：立即增加政治点 (`wiki:effect:add_political_power`) 与 完整国策奖励案例 (`focus-political-power-reward`)。
- 想要求当前政治点达到门槛：检查政治点 (`wiki:trigger:has_political_power`)。
- 想把四种用途放在一起比较：《政治点专题》（文章 `political-power`）。
