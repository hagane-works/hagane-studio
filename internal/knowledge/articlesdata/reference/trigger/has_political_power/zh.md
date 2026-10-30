---
contentId: "wiki:trigger:has_political_power"
kind: reference
slug: has_political_power
locale: zh
status: published
title: "检查政治点是否达到门槛"
summary: "用 has_political_power 判断当前余额。至少 150、低于 150 与超过 150 是不同条件；判断不会扣费。"
primaryTopic: politics
topics: [politics]
scenarios: [focus, event, decision]
gameTerms: [政治点数, 国策条件, 决议条件]
aliases: [政治点, 政治力, PP, 政治点不足, 政治点是否足够, 政治点门槛, 至少150政治点]
relatedIds: [political-power, "wiki:effect:add_political_power", "wiki:modifier:political_power_gain", "wiki:modifier:political_power_factor", focus-political-power-reward]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: trigger/has_political_power
searchText: "has_political_power 检查政治点是否足够 至少150政治点 低于150政治点 超过150政治点 不超过150政治点 政治点不足 不扣费 决议能否点击 国策available"
sources:
  - label: "HOI4 Wiki：has_political_power"
    url: "https://hoi4.paradoxwikis.com/Triggers#has_political_power"
  - label: "HOI4 Wiki：决议制作"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
  - label: "HOI4 Wiki：国策制作"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
---

技术名：`has_political_power` · 判断条件（Trigger）· 国家作用域。

## 1. 游戏里会发生什么，在哪里体现？

它检查当前国家的政治点余额是否满足比较条件，不会增加或扣除政治点。

放在普通国策或决议的 `available` 中时，条件不满足会让内容不可选择；放在决议 `visible` 中则影响是否显示。只有结合放置位置，才能判断玩家具体会看到什么。不要为了展示“政治点不足”，误把整个决议藏起来。

## 2. 对谁生效，何时生效或失效？

判断对象是当前国家。在没有切换作用域的普通国策或决议条件里，就是使用它的国家，不是事件的另一方或某个州。

当承载这个条件的位置进行检查时，读取当时的余额。它不会锁定政治点，也不会预先扣款。某次检查成立不意味着以后始终成立；余额变化后，后续检查可能得出不同结果。

## 3. 数值或参数怎么填？

下面用 150 点演示边界，实际数值由你的设计决定：

| 目标 | 条件写法 | 恰好 150 点时 |
| --- | --- | --- |
| 低于 150 点 | `has_political_power < 150` | 不成立 |
| 超过 150 点 | `has_political_power > 150` | 不成立 |
| 至少 150 点 | `NOT = { has_political_power < 150 }` | 成立 |
| 不超过 150 点 | `NOT = { has_political_power > 150 }` | 成立 |

“至少 150”示例使用单个条件取反，避免把“超过”误写成“达到”。不要用 `has_political_power > 149` 代替：政治点可以有小数，149.5 并不够 150。

也不要照着一句“需要政治点”就复制数值。本文没有把 `=` 当作各种触发器通用的“至少”写法；不同参数的比较规则必须按对应定义核对。

## 4. 放在哪里，怎样写？

下面是已有普通国策节点内的局部条件块：

```pdx
available = {
    NOT = {
        has_political_power < 150
    }
}
```

它允许余额至少为 150 的国家满足这一项条件；不会消费这 150 点。若该国策还有其他条件，需同时满足它们。已有 `available` 时，把条件合并到原块，不要再添加一个同名块。

普通决议如果只是“支付 150 政治点才能使用”，通常应在决议本体设置政治点费用：

```pdx
cost = 150
```

`cost` 不是这条 Trigger 的参数。它属于决议本体，不能放入 `available`，也不能原样套到国策上当作政治点费用。已经使用决议 `cost` 时，不要又在 `complete_effect` 扣除同额政治点。

自定义费用或事件分支则需要分别安排条件与支付操作；仅有条件不会完成支付。

## 5. 接下来可能需要什么？

- 条件满足后需要真正奖励或扣除：立即增加或减少政治点 (`wiki:effect:add_political_power`)。
- 想做一个不收费的完成奖励：完成国策后获得 150 政治点 (`focus-political-power-reward`)。它与本页的“必须先拥有 150”不是同一设计。
- 想帮助玩家以后更快积累政治点：每日固定获取 (`wiki:modifier:political_power_gain`) 或 比例修正 (`wiki:modifier:political_power_factor`)。
- 想重新选择游戏目标：《政治点专题》（文章 `political-power`）。
