---
contentId: "wiki:modifier:research_speed_factor"
kind: reference
slug: research_speed_factor
locale: zh
status: published
title: "持续提高或降低科研速度"
summary: "用百分比修正改变科研速度；国家精神存在时持续生效，不按研究次数消耗。"
primaryTopic: research
topics: [research]
scenarios: [ideas, focus, event, decision]
gameTerms: [科研速度, 研究速度, 国家精神]
aliases: [持续科研加速, 科研速度加成, 科研速度惩罚, 所有科技研究更快, Research Speed]
relatedIds: [research-speed, "wiki:effect:add_tech_bonus", "wiki:effect:set_technology", first-content]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/research_speed_factor
searchText: "科研速度 Research Speed research_speed_factor，持续提高或降低研究速度。0.10 是 +10% 修正，-0.10 是 -10% 修正。国家精神的 modifier 块是本文采用的承载位置；持续时间由国家精神是否存在决定。"
sources:
  - label: "HOI4 Wiki：科研速度修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers#research_speed_factor"
  - label: "HOI4 Wiki：国家精神"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---
## 1. 游戏里会发生什么，在哪里体现？

国家获得“科研速度”的百分比修正。正数使研究更快，负数使研究更慢。它影响研究速度，不是立即完成某项科技，也不是每天发放研究点数。

在国家精神的说明和科技研究相关提示中核对这项修正。`0.10` 表示新增一项 +10% 科研速度修正，不保证所有科技的剩余天数恰好减少 10%：其他修正、超前年份惩罚等也会参与计算。

## 2. 对谁生效，何时生效或失效？

本文用国家精神承载它：效果作用于拥有该国家精神的国家，国家精神生效期间持续存在，移除后不再提供这项加成。

只写好国家精神定义不会自动把它交给某国；还要通过 `add_ideas` 等效果赋予。若用 `add_timed_idea` 赋予，期限由该效果的 `days` 决定，不是由 `research_speed_factor` 决定。

## 3. 数值或参数怎么填？

| 写法 | 本项修正 |
| --- | --- |
| `research_speed_factor = 0.10` | 科研速度 +10% |
| `research_speed_factor = -0.10` | 科研速度 -10% |
| `research_speed_factor = 0` | 本项不提供加减成 |

不要把 `10` 当成 10%。这里的小数不是天数，也不是每天固定增加的点数。正负号改变方向，不能在搜索、抄写或转换格式时丢掉。

## 4. 放在哪里，怎样写？

下面是**国家精神定义内部的局部片段**，不是完整文件。片段已经包含 `modifier` 外层；若精神已有这个块，只把内部的修正行加入其中，不要再嵌套一层 `modifier`：

```hoi4
modifier = {
    research_speed_factor = 0.10
}
```

不要把这段直接放进国策 `completion_reward`。国策奖励里写“赋予国家精神”的效果，国家精神定义里写“科研速度”的修正。

第一份游戏内容：国家精神 (`first-content`) 提供完整文件、原版图标和手动检查步骤。

## 5. 接下来可能需要什么？

- 只奖励下一次或几次研究：有限次数的研究加成 (`wiki:effect:add_tech_bonus`)。
- 完全跳过研究、直接拥有科技：直接解锁指定科技 (`wiki:effect:set_technology`)。
- 还没确定是哪一种：《科研速度、研究加成与科技解锁》（文章 `research-speed`）。

本页已对照 1.19.2.0 随附文档、原版国家精神和中英文术语；未执行游戏实测。
