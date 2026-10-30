---
contentId: "wiki:effect:add_political_power"
kind: reference
slug: add_political_power
locale: zh
status: published
title: "立即增加或减少政治点"
summary: "用 add_political_power 改变当前政治点余额。正数奖励、负数扣除；不会改变每日获取。"
primaryTopic: politics
topics: [politics]
scenarios: [focus, event, decision]
gameTerms: [政治点数, 国策奖励]
aliases: [政治点, 政治力, PP, 立即获得政治点, 一次性政治点奖励, 扣除政治点]
relatedIds: [political-power, "wiki:trigger:has_political_power", "wiki:modifier:political_power_gain", "wiki:modifier:political_power_factor", focus-political-power-reward]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: effect/add_political_power
searchText: "add_political_power 立即增加150政治点 立即减少50政治点 一次性奖励 扣除政治点 国策完成奖励 事件选项 决议执行效果 当前余额"
sources:
  - label: "HOI4 Wiki：add_political_power"
    url: "https://hoi4.paradoxwikis.com/Effects#add_political_power"
  - label: "HOI4 Wiki：国策制作"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
---

技术名：`add_political_power` · 执行操作（Effect）· 国家作用域。

## 1. 游戏里会发生什么，在哪里体现？

执行时，目标国家当前持有的政治点数增加或减少指定数量。国策完成奖励中的 `add_political_power = 150` 是完成时发放一笔 150 点奖励，不是每天发放 150 点，也不是把余额设为 150。

在国策奖励提示和游戏顶部的政治点余额处核对。若奖励会超过当前规则允许的余额上限，就不能用“余额必须刚好多出 150”作为验收条件；测试时先留出足够空间。

## 2. 对谁生效，何时生效或失效？

它作用于执行位置的当前国家。在普通国策的 `completion_reward` 中，没有另行切换作用域时，就是完成国策的国家。

效果只在代码被执行时发生；再次执行就会再次加减。已经发放的政治点不会在国策完成后的某天自动消失，但仍会被正常消费或其他效果改变。

这一行自身不会检查“钱是否够用”。需要先判断余额的场景，应配合条件；不能把一个负数奖励当成完整的付费交互。

## 3. 数值或参数怎么填？

| 写法 | 本次操作 | 不是 |
| --- | --- | --- |
| `add_political_power = 150` | 当前余额增加 150 点 | 每日 +150 或设为 150 |
| `add_political_power = -50` | 当前余额减少 50 点 | 只检查是否拥有 50 点 |
| `add_political_power = 0` | 本次不改变余额 | 清空政治点 |

直接填写数值，不要改成 `add_political_power = { value = 150 }`。它的参数不是百分比；`0.15` 不能表示“政治点获取 +15%”。

政治点余额受游戏或 Mod 的规则限制。本文示例使用普通整数奖励，不把默认上限、下限写成所有 Mod 永久不变的保证。

## 4. 放在哪里，怎样写？

下面是已有普通国策节点内的奖励块。它应位于该节点的 `focus = { ... }` 之内：

```pdx
completion_reward = {
    add_political_power = 150
}
```

这段是局部片段，不是整份国策文件。如果节点已经有 `completion_reward`，把这一行加入现有块中，不要再复制一个奖励块。

国策的 `available` 接受条件，不能在里面写这条奖励。事件选项和决议的执行效果也能发放政治点，但各自外层结构不同；不要原样复制国策的 `completion_reward` 外壳过去。

## 5. 接下来可能需要什么？

- 要把节点和中文名称一起做完：完成国策后获得 150 政治点 (`focus-political-power-reward`)。
- 要先确认政治点是否足够：检查政治点是否达到门槛 (`wiki:trigger:has_political_power`)。它本身不扣除点数。
- 要让国家精神持续增加每日获取：固定增减 (`wiki:modifier:political_power_gain`) 或 按比例调整 (`wiki:modifier:political_power_factor`)。
- 还没确定想改变哪一种数值：回到《政治点专题》（文章 `political-power`）。
