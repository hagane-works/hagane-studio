---
contentId: focus-research-bonus
kind: case
slug: focus-research-bonus
locale: zh
status: published
title: "国策奖励一次 50% 工业研究加成"
summary: "在已有国策树中加入一个研究奖励节点，补齐加成次数、科技类别、图标与中文名称。"
primaryTopic: research
topics: [research]
scenarios: [focus]
gameTerms: [国策奖励, 工业科技, 研究加成]
aliases: [一次工业研究加成, 国策科研奖励, 工业科技研究奖励]
relatedIds: [research-speed, "wiki:effect:add_tech_bonus", "wiki:modifier:research_speed_factor", "wiki:effect:set_technology", localisation, check-and-troubleshoot]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "完成国策后给予一次 50% 工业研究加成。完整 focus 节点使用 add_tech_bonus，bonus = 0.50，uses = 1，category = industry；使用原版图标并补齐加成名称。"
sources:
  - label: "HOI4 Wiki：国策"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
  - label: "HOI4 Wiki：研究加成效果"
    url: "https://hoi4.paradoxwikis.com/Effects#add_tech_bonus"
  - label: "HOI4 Wiki：本地化"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---
目标是：玩家完成一个国策后，获得一次 50% 工业科技研究加成。玩家仍要选择并研究科技，不会当场解锁它，也不会永久提高所有研究的速度。

## 开始前需要什么？

- 你已经有一个被目标国家实际加载的国策树文件，位于 Mod 的 `common/national_focus` 中。
- 树中没有 `HW_industrial_exchange` 这个国策 ID。若已有同名内容，先改成自己的唯一 ID，并同步本地化键。
- 已确认国策树中的空位。下面的 `x = 20`、`y = 0` 只是本例位置，可调整，不能假设在你的树里一定不重叠。

本例不创建国策树的国家分配规则。如果你的树本来就不显示，先解决树的加载问题，不能靠追加节点代替。

## 1. 加入一个完整国策节点

下面是**完整的 focus 节点，不是完整的国策树文件**。把它放在现有 `focus_tree = { ... }` 内，与其他 `focus` 同级，保留原文件的树 ID、国家选择条件与其他节点：

```hoi4
focus = {
    id = HW_industrial_exchange
    icon = GFX_goal_generic_scientific_exchange
    x = 20
    y = 0
    cost = 10

    completion_reward = {
        add_tech_bonus = {
            name = HW_industry_research_bonus
            bonus = 0.50
            uses = 1
            category = industry
        }
    }
}
```

该节点没有前置国策；如果你的设计需要前置关系，应另行添加并确认引用的国策存在。示例使用原版已存在的科研交流图标，不需要另建 GFX 文件。

`industry` 是原版科技类别 ID。`bonus` 是加成比例，`uses` 是使用次数；不要把“只用一次”写成一天的国家精神。

## 2. 补齐国策与加成名称

在 Mod 中创建 `localisation/simp_chinese/hw_research_l_simp_chinese.yml`，使用 UTF-8-BOM。下面是**本例所需的完整本地化文件内容**：

```yaml
l_simp_chinese:
 HW_industrial_exchange: "工业研究交流"
 HW_industrial_exchange_desc: "交流生产工艺，为下一项工业研究提供支持。"
 HW_industry_research_bonus: "工业研究交流"
```

如果你已经按参考页创建了同名文件，应合并内容，保留一个 `l_simp_chinese:`，不要把同一个加成名称键重复定义两次。

## 3. 在独立测试局逐项核对

1. 启用当前 Mod，以会加载这棵树的国家开始一局测试；确认新国策出现、没有重叠，名称和图标正确。
2. 查看国策奖励提示，应该表达“一次、50%、工业科技研究加成”，而不是直接获得某项科技。
3. 完成国策后，选择一项符合 `industry` 类别、尚未开始研究的科技，核对可用研究加成与使用后的次数。不要用本来已经研究完成的科技做对照。
4. 同时检查本次启动的日志。奖励提示正确、没有明显日志错误，还不能替代对加成实际使用过程的核对。

以上是读者可执行的验收步骤，本例本身尚未执行游戏实测。具体研究天数还取决于国家其他修正与超前惩罚，不能用“是否正好减半”作为唯一标准。

## 想改成另一种奖励？

- 精神存在期间一直提高研究速度：改用 科研速度修正 (`wiki:modifier:research_speed_factor`)，由国家精神承载。
- 完成后立刻拥有某项科技：看 直接解锁指定科技 (`wiki:effect:set_technology`)。
- 想调整加成次数与适用范围：回到 add_tech_bonus 参数说明 (`wiki:effect:add_tech_bonus`)。

来源核对包含 1.19.2.0 原版通用国策中的限次工业加成、科技类别表、GFX 名称及随附效果文档。
