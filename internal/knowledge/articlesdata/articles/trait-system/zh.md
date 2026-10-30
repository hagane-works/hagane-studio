---
contentId: trait-system
kind: guide
slug: trait-system
locale: zh
status: published
title: "特质：四类各管什么，怎么到人身上"
summary: "领袖、顾问、将领、科学家四类特质分住三个目录，文件形状各不相同；一个特质到人身上有五条路：定义时写上、随机生成、玩家花经验买、打仗攒出来、脚本直接给。分清这两件事，就不会把将领特质的写法套到领袖上，也不会写出一个永远没人能拿到的特质。"
primaryTopic: people
topics: [people]
scenarios: [traits, focus, event]
gameTerms: [特质, 领袖特质, 顾问特质, 将领特质, 科学家特质, 特质树, 经验, 高级指挥, 人格]
aliases: [特质怎么做, 特质分几类, 特质怎么加到人身上, 特质树, 将领经验, 随机人格, trait, leader_traits, trait_type, gain_xp]
relatedIds: [character-system, custom-character-traits, general-trait-earned-in-combat, general-trait-tree, focus-grant-trait, "wiki:effect:add_country_leader_trait", "wiki:effect:add_unit_leader_trait", "wiki:effect:swap_country_leader_traits", "wiki:trigger:has_trait", "wiki:modifier:experience_gain_factor"]
updatedAt: "2026-09-15"
verification: static-checked
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：特质制作"
    url: "https://hoi4.paradoxwikis.com/Trait_modding"
  - label: "HOI4 Wiki：人物制作"
    url: "https://hoi4.paradoxwikis.com/Character_modding"
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
---

# 特质：四类各管什么，怎么到人身上

## 这个系统负责什么

人物卡上那些带图标的小方块。领袖有「冷酷政客」，顾问有「沉默的工作狂」，将领有「步兵指挥官」，科学家有自己的专长。每一个都是一条特质定义，挂在一个人身上，给这个人或他带的部队一组修正。

作为制作者你要决定两件事：

1. **这个特质是哪一类、写在哪个文件里、修正长什么样**
2. **它怎么到人身上**：一直在、随机出现、玩家花经验买、打仗攒出来、还是国策事件给

第一件事决定文件形状，第二件事决定你要不要写 `gain_xp`、`cost`、`gui_row` 这些字段。大多数「特质做了没反应」都是第二件事没想清楚：定义了一个没人能拿到的特质。

## 核心概念：四类特质，三个目录，三种形状

| 类别 | 目录 | 外层块 | 修正怎么写 | 原版数量 |
| --- | --- | --- | --- | --- |
| 领袖特质 | `common/country_leader/` | `leader_traits = { }` | **平铺**在特质下面：`stability_factor = 0.05` | 与顾问共 1782 条 |
| 顾问特质 | 同上，同一批文件 | 同上 | 同上；高级指挥三档另带 `command_cap_increase`、`command_power` | 同上 |
| 将领特质 | `common/unit_leader/` | `leader_traits = { }` | **包在块里**：`modifier = { }`，另有 `non_shared_modifier`、`field_marshal_modifier`、`corps_commander_modifier`、`sub_unit_modifiers` | 178 条 |
| 科学家特质 | `common/scientist_traits/` | 直接顶层 | `modifier = { }`，只对特殊计划生效 | 少量，随附 `_documentation.md` |

领袖和顾问特质长得一样、住同一个目录，区别只在于谁引用它：`country_leader` 块引用的是领袖特质，`advisor` 块引用的是顾问特质。批次二的给人物做一个自定义特质 (`custom-character-traits`)讲过领袖与将领两种形状的最小写法，本文不重复。

将领特质多出来的几个修正块各有去处：`modifier` 给他带的部队，`non_shared_modifier` 给将领本人（比如经验获取），`field_marshal_modifier` 只在当元帅时生效，`sub_unit_modifiers` 按兵种细分。这些区分原版文件头部的注释写得很全，`common/unit_leader/00_traits.txt` 前七十行就是一份字段说明。

## 一个特质到人身上的五条路

| 路 | 写法 | 适用 | 原版怎么用 |
| --- | --- | --- | --- |
| **定义时写上** | 人物档案里 `traits = { my_trait }` | 四类都行 | 最常见；历史人物的固定特质 |
| **随机生成** | 领袖特质 `random = yes`；将领特质 `trait_type = personality_trait` 加 `new_commander_weight` | 领袖、将领 | 领袖特质 1782 条里只有 23 条 `random = yes`；将领人格特质 94 条 |
| **玩家花经验买** | 将领特质 `trait_type = assignable_trait` 加 `cost`，`gui_row` 让它出现在树里 | 将领 | 46 条可分配特质，`cost` 出现 229 次 |
| **打仗攒出来** | 将领特质带 `gain_xp = { 条件 }`，条件在战斗里成立时攒经验 | 将领 | 71 条带 `gain_xp` |
| **脚本给** | `add_country_leader_trait`、`add_unit_leader_trait`、`swap_country_leader_traits` 等 | 四类都行 | 国策里给领袖 164 处、换领袖特质 78 处、给将领 60 处 |

后三条各有一篇案例：做一个将领靠打仗攒出来的特质 (`general-trait-earned-in-combat`)、把几个将领特质连成一棵可解锁的树 (`general-trait-tree`)、国策或事件给领袖或将领加特质 (`focus-grant-trait`)。

「玩家花经验买」和「打仗攒出来」在原版里通常是同一个特质的两面：`gain_xp` 定义攒经验的条件，`cost` 定义这个特质自己的经验攒够多少能点。将领面板上「经验值：X/1000」和「获取经验值当：」就是这两个字段，攒到 5% 以上才显示。点分配那一下另外扣全局固定的 15 点陆军经验（`UNIT_LEADER_ASSIGN_TRAIT_COST`），提示框写「将会花费 15.00 来分配这个特质」，和 `cost` 无关。

## 文件结构与执行方式

### 将领特质的骨架

```
leader_traits = {
    my_trait = {
        type = corps_commander
        trait_type = assignable_trait
        gain_xp = {
            is_fighting_in_terrain = hills
        }
        cost = 700
        modifier = {
            hills = {
                attack = 0.1
                defence = 0.1
            }
        }
        ai_will_do = {
            factor = 1
        }
        gui_row = 2
    }
}
```

`type` 决定谁能有它：`all`、`land`、`navy`、`operative`、`corps_commander`、`field_marshal`。`trait_type` 决定它在界面上归哪一类、能不能分配：`assignable_trait`（可买）、`personality_trait`（随机人格）、`basic_terrain_trait` / `assignable_terrain_trait`（地形）、`status_trait`（伤病、被俘这类状态）、`basic_trait`（特工）、`exile`。原版 178 条里有 16 条没写 `trait_type`，都是像「步兵指挥官」这类靠 `gain_xp` 攒的。

### 判断条件

`has_trait = my_trait` 在人物或战斗作用域里判断。判断在任领袖有没有某特质，国家作用域直接写 `has_country_leader_with_trait = my_trait`，原版国策里 45 处；判断将领特质多写在 `random_army_leader = { limit = { has_trait = ... } }` 里。

## 常见制作目标与入口

| 你想在游戏里看到的 | 从这里开始 | 关键区分 |
| --- | --- | --- |
| 某位历史将领一开始就有一个专属特质 | 人物档案 `traits = { }` 加一条将领特质定义 | 定义要写 `type`；不写 `gui_row` 也没关系，它不需要进树 |
| 打过几场山地仗的将领能点出山地特质 | 靠打仗攒出来的特质 (`general-trait-earned-in-combat`) | `gain_xp` 在战斗作用域判断；`cost` 是攒够的门槛 |
| 一棵有前置和互斥的特质树 | 特质树 (`general-trait-tree`) | `any_parent` / `mutually_exclusive` / `gui_row`；互斥要两边都写 |
| 国策完成后领袖换一个特质 | 国策给特质 (`focus-grant-trait`) | `swap_country_leader_traits` 一步换，国策里 78 处 |
| 事件让某位将领受伤三个月 | 同上 | `add_timed_unit_leader_trait = { trait = ... days = 90 }`，要先进将领的作用域 |
| 将领靠这个特质能当高级指挥 | 将领特质里加 `slot = high_command` 和三档顾问特质 | 三档顾问特质是领袖目录里的条目，见下 |

## 容易混淆的事

### 领袖特质平铺，将领特质包块

同一个 `stability_factor` 放在领袖特质里直接写，放在将领特质里必须包进 `modifier = { }`。写反了引擎不报错，修正就是不生效。批次二那篇有对照。

### 将领特质可以顺带定义一个高级指挥职务

原版 33 条将领特质带 `slot = high_command` 加 `specialist_advisor_trait` / `expert_advisor_trait` / `genius_advisor_trait` 三行。有这三行的将领可以被雇为高级指挥，档次按将领技能。三档引用的是领袖目录里的顾问特质，比如「步兵指挥官」对应 `army_infantry_1` 到 `army_infantry_3`。游戏提示框里那句「解锁高级指挥职务：某某」就是它。

### `random` 只管领袖特质，人格只管将领

领袖特质 `random = yes` 让它可能出现在随机生成的领袖身上；将领的对应物是 `trait_type = personality_trait` 加 `new_commander_weight` 权重块。两套机制互不相通。

### 没写 `gui_row` 的特质不进树

`gui_row` 不写或写 `-1`，特质就不出现在将领面板的可解锁树里，玩家看不见也买不了。只想靠脚本给的特质（比如状态特质）可以不写；想让玩家买的必须写，从 0 开始数。列不写时引擎按 `trait_type` 自动分：攒经验的在最左，可分配的在其右；同行同类不写列会互相遮挡。当前置的根特质照原版**不写 `trait_type`**，写成可分配特质会报 `wrong kind of parent`。细节见特质树 (`general-trait-tree`)。

### 图标要自己登记

将领特质的图标按固定名字找精灵：`GFX_trait_` 加特质名，定义在 `interface/*.gfx` 里。原版登记了 181 个。领袖特质用 `sprite = 数字` 指向一张拼图的第几格。没登记时实测显示一枚问号奖章；没写 `_desc` 本地化时提示框底部直接显示 `名字_desc` 这个键。

### 科学家特质只对特殊计划生效

`common/scientist_traits/_documentation.md` 第一段就写着：只对特殊计划生效，所以只能用那一组修正；普通科研加成写进去没有作用。这是「诸神黄昏」的内容。

## 在钢之工坊中

角色工作台的自定义特质面板：选特质类型（领袖 / 将领 / 科学家），填修正；将领特质有「高级字段」区对应原版 `00_traits.txt` 的结构（`trait_type`、`cost`、`gui_row`、互斥、脚本块），还有一个特质树画布，拖节点改 `gui_row` / `gui_column`，拉线建 `any_parent` 前置。导出到三个目录各自的文件。国策事件里给特质的效果，在国策工作台完成效果和事件工作台选项里原样写。

## 接下来看什么

- 给人物做一个自定义特质 (`custom-character-traits`)：两种形状的最小写法
- 做一个将领靠打仗攒出来的特质 (`general-trait-earned-in-combat`)
- 把几个将领特质连成一棵可解锁的树 (`general-trait-tree`)
- 国策或事件给领袖或将领加特质 (`focus-grant-trait`)
- 人物本身怎么做：做一个人物：领袖、将领、顾问各管什么 (`character-system`)
