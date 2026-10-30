---
contentId: idea-supply-consumption
kind: case
slug: idea-supply-consumption
locale: zh
status: published
title: "国家精神让部队少消耗 10% 补给"
summary: "定义一个补给消耗 -10% 的国家精神，再由已有国策把它交给国家；负号不能省略。"
primaryTopic: military
topics: [military]
scenarios: [ideas, focus]
gameTerms: [国家精神, 补给消耗, 国策奖励]
aliases: [减少补给消耗的国家精神, 部队少吃补给, 后勤精神]
relatedIds: [supply-consumption, "wiki:modifier:supply_consumption_factor", first-content, localisation, check-and-troubleshoot]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "通过国家精神减少地面部队补给消耗。完整 ideas/country 定义中的 modifier 使用 supply_consumption_factor = -0.10，再由现有国策 completion_reward 中的 add_ideas 赋予。"
sources:
  - label: "HOI4 Wiki：国家精神"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
  - label: "HOI4 Wiki：补给消耗"
    url: "https://hoi4.paradoxwikis.com/Modifiers#supply_consumption_factor"
  - label: "HOI4 Wiki：赋予国家精神"
    url: "https://hoi4.paradoxwikis.com/Effects#add_ideas"
---
目标是让一个国家的地面部队获得“补给消耗 -10%”修正。本例用国家精神保存修正，用已有国策发放精神。

你需要一个能正常加载的 Mod，以及一个尚未完成、可以用于测试的现有国策。下文不新建国策树。

## 1. 定义国家精神

新建 `common/ideas/hw_supply_ideas.txt`，使用 UTF-8 无 BOM。下面是**完整文件内容**：

```hoi4
ideas = {
    country = {
        HW_supply_training = {
            picture = generic_army_war_college
            removal_cost = -1
            modifier = {
                supply_consumption_factor = -0.10
            }
        }
    }
}
```

`HW_supply_training` 是本例的唯一国家精神 ID。`picture` 对应原版已存在的 `GFX_idea_generic_army_war_college`；这里不写 `GFX_idea_` 前缀，也不需要添加素材。

最重要的一行是 `supply_consumption_factor = -0.10`。负数减少消耗，正数会增加消耗。该数值是新增的一项百分比修正，不意味着在任意已有修正下，当前界面的最终数值都直接乘以 0.9。

## 2. 给它一个中文名称

新建 `localisation/simp_chinese/hw_supply_l_simp_chinese.yml`，使用 UTF-8-BOM。下面是**完整本地化文件内容**：

```yaml
l_simp_chinese:
 HW_supply_training: "后勤训练"
 HW_supply_training_desc: "改进补给使用习惯，减少地面部队的补给需求。"
```

显示文字和脚本 ID 分开保存。不要把定义中的 `HW_supply_training` 改成中文，也不要把 `supply_consumption_factor` 翻译成中文。

## 3. 让现有国策赋予精神

找到你选定国策的 `completion_reward`。把下面这个**单条完整效果**加入该奖励块内，保留已有奖励，不要把它放进国家精神的 `modifier` 块：

```hoi4
add_ideas = HW_supply_training
```

只有定义文件还不够：国家需要实际获得该精神，修正才由它提供。若国策以前已经完成，保存文件不会让完成奖励自动再执行一次；应换一局状态清楚的本地测试局。

## 4. 核对结果

1. 查看国策奖励提示，确认奖励的是“后勤训练”。
2. 完成该国策，确认国家精神出现在目标国家，并显示“补给消耗 -10%”。
3. 在其他条件尽量一致时，对照地面部队的补给需求。部队编制、部署与地图运输条件同时变化，会让比较失去意义。
4. 检查本次日志里是否出现该文件名、精神 ID 或修正名相关错误。

这不会新建铁路或补给枢纽，也不保证原本缺补给的部队立即恢复充足补给。这里改变的是需求的一侧。

以上是验收清单，不是已完成的游戏测试记录。本例已按 1.19.2.0 的原版国家精神用法、修正文档和图标核对，尚未游戏实测。

## 移除或改成负面精神

要停止提供这项修正，可以在合适的国家效果块内执行下面这条**效果片段**：

```hoi4
remove_ideas = HW_supply_training
```

如果你的设计是增加后勤负担，应另看 提高或降低补给消耗 (`wiki:modifier:supply_consumption_factor`)，先确认正负方向，再同步显示文案。不要只改标题而保留本例的负数。
