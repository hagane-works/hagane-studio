---
contentId: "wiki:effect:set_technology"
kind: reference
slug: set_technology
locale: zh
status: published
title: "直接解锁指定科技"
summary: "让国家直接拥有指定科技；这是设置科技状态，不是科研速度或限次研究加成。"
primaryTopic: research
topics: [research]
scenarios: [focus, event, decision, history]
gameTerms: [解锁科技, 基础机床, 科技]
aliases: [直接获得科技, 跳过研究, 立即获得科技, Grant technology, Unlock technology]
relatedIds: [research-speed, "wiki:effect:add_tech_bonus", "wiki:modifier:research_speed_factor"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: effect/set_technology
searchText: "set_technology 设置国家的指定科技。basic_machine_tools = 1 直接给予基础机床；popup = no 关闭该效果的科技弹窗。需要真实科技 ID，不是装备 ID、类别 ID 或科研速度数值。"
sources:
  - label: "HOI4 Wiki：设置科技"
    url: "https://hoi4.paradoxwikis.com/Effects#set_technology"
  - label: "HOI4 Wiki：科技定义"
    url: "https://hoi4.paradoxwikis.com/Technology_modding"
---
## 1. 游戏里会发生什么，在哪里体现？

国家直接拥有指定科技，不必再走一次通常的研究过程。本文示例给予“基础机床”，在科技界面核对该科技的状态。

如果测试国家本来就拥有基础机床，再执行相同设置不会产生“新解锁”的对照结果。请先确认测试前的科技状态。

## 2. 对谁生效，何时生效或失效？

这是国家作用域效果。放入国策完成奖励、事件选项或其他国家效果块时，在该效果实际执行后设置当前国家的科技状态；仅写在文件里不会自行触发。

国家历史文件也可以用它设置开局科技，但更改历史文件与向正在运行的存档执行效果不是一回事。本页不讨论移除科技或撤销后续解锁。

## 3. 数值或参数怎么填？

`basic_machine_tools = 1` 中，左边是科技 ID，右边的 `1` 是该科技的设置值。它不是“科研速度 +1”，更不是让所有科技都升一级。

`popup = no` 控制是否显示科技弹窗，不改变本例要给予的科技。随附文档说明默认是 `yes`。

科技 ID 应从 `common/technologies` 中确认。不要把 `industry` 类别名、显示名称“基础机床”或某种装备 ID 填在这里。本文只说明已经核对的 `= 1` 给予用法，不把 `0` 推断成通用的安全撤销手段。

## 4. 放在哪里，怎样写？

下面是**国家作用域效果块内的完整片段**，不是独立文件。可加入现有国策的 `completion_reward`；该国策所在的树必须已经被目标国家加载：

```hoi4
set_technology = {
    basic_machine_tools = 1
    popup = no
}
```

本例直接引用原版的科技和本地化，不需要另造“基础机床”的显示名称。若换成自定义科技，必须先完成该科技本身的定义。

## 5. 接下来可能需要什么？

- 只是想让玩家研究得更快：持续科研速度修正 (`wiki:modifier:research_speed_factor`)。
- 想保留研究过程，只奖励一次加成：有限次数的研究加成 (`wiki:effect:add_tech_bonus`)。
- 不确定该选择哪种奖励：《科研专题》（文章 `research-speed`）。

本页已对照 1.19.2.0 的效果文档和原版科技 ID；未执行游戏实测。
