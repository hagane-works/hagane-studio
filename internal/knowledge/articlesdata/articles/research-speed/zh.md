---
contentId: research-speed
kind: topic
slug: research-speed
locale: zh
status: published
title: "科研速度、研究加成与科技解锁"
summary: "先看玩家要得到什么：长期研究更快、获得几次研究加成，还是直接拥有某项科技。"
primaryTopic: research
topics: [research]
scenarios: [ideas, focus, event, decision, history]
gameTerms: [科研速度, 研究加成, 解锁科技]
aliases: [研究更快, 科技奖励, 科研加成, Research Speed, Research bonus]
relatedIds: ["wiki:modifier:research_speed_factor", "wiki:effect:add_tech_bonus", "wiki:effect:set_technology", focus-research-bonus, first-content]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "科研速度、研究加成、解锁科技是三种不同的游戏结果。持续科研修正使用 research_speed_factor；有限次数研究奖励使用 add_tech_bonus；直接获得指定科技使用 set_technology。"
sources:
  - label: "HOI4 Wiki：修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki：科技"
    url: "https://hoi4.paradoxwikis.com/Technology_modding"
---
同样是“加快科研”，可能是在说三件事。先选游戏里想得到的结果，再看对应写法。

## 你想让玩家得到什么？

| 玩家看到的结果 | 应该看哪一页 | 关键区别 |
| --- | --- | --- |
| 国家精神提供科研速度 +10%，存在时一直生效 | 持续提高或降低科研速度 (`wiki:modifier:research_speed_factor`) | 修正需要载体，不按研究次数消耗 |
| 完成国策后获得一次 50% 工业研究加成 | 给予有限次数的研究加成 (`wiki:effect:add_tech_bonus`) | 玩家仍要研究，奖励有范围和次数 |
| 完成国策后直接拥有基础机床 | 直接解锁指定科技 (`wiki:effect:set_technology`) | 设置科技状态，不是加快研究 |

“科研速度”对应游戏中的 Research Speed。“研究加成”是科技奖励中显示的 Research bonus。它们不应该因为都带有百分号就合并成同一种参数。

## 三个容易混淆的数字

- `research_speed_factor = 0.10`：新增一项 +10% 科研速度修正。精神持续多久，要看精神怎么获得、何时移除。
- `bonus = 0.50`、`uses = 1`：一次 50% 研究加成。`1` 是次数，不是一天。
- `basic_machine_tools = 1`：在 `set_technology` 中给予指定科技。`1` 不是科研速度或研究次数。

上面只是参数对照，不是可独立加载的完整脚本。50% 的研究加成也不保证剩余研究天数直接减半；不要把百分比、天数、次数互换。

## 从一个完整例子开始

已经有国策树，可以看 国策奖励一次工业研究加成 (`focus-research-bonus`)。它包含一个完整国策节点、加成显示名称和逐项检查。

第一次接触 Mod 文件，可以从 第一份游戏内容：国家精神 (`first-content`) 开始。这个练习用持续科研速度修正，不需要先搭建国策树。

## 查错时先问这几句

1. 玩家应该“拥有科技”，还是“拥有一次研究奖励”？前者和后者不能用同一个效果替代。
2. 奖励写在目标国家的效果块里了吗？国家精神只有定义、没有赋予时，修正还没交给任何国家。
3. 科技 ID 与类别 ID 有没有混用？`basic_machine_tools` 是一项科技，`industry` 是一类科技。
4. 测试前已经拥有这项科技，或者研究已经开始了吗？选择状态清楚的独立测试局，才容易判断差别。

本专题覆盖上述三种基础用途，不把科研槽、超前惩罚、军工机构研究加成或特殊项目当成同一项功能。页面已按 1.19.2.0 来源核对，尚未进行示例的游戏实测。
