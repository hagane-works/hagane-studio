---
contentId: general-trait-earned-in-combat
kind: case
slug: general-trait-earned-in-combat
locale: zh
status: published
title: "做一个将领靠打仗攒出来的特质"
summary: "定义一个将领特质，在指定条件下打仗时自动攒经验，攒够后玩家花经验点上；讲清 gain_xp 的判断作用域、cost 是门槛不是价格标签、提示框那两句「经验值」「获取经验值当」各来自哪个字段。原版一半以上的将领特质都是这么攒的。"
primaryTopic: people
topics: [people, military]
scenarios: [traits]
gameTerms: [将领, 特质, 经验, 攒经验, 分配特质, 特质树, 地形, 平原]
aliases: [将领打仗获得特质, 自动攒经验的特质, 将领特质经验, 特质怎么解锁, 地形特质, gain_xp, cost, custom_gain_xp_trigger_tooltip, assignable_trait]
relatedIds: [trait-system, general-trait-tree, custom-character-traits, "wiki:trigger:is_fighting_in_terrain", "wiki:trigger:has_trait", "wiki:modifier:army_infantry_attack_factor", "wiki:effect:add_unit_leader_trait"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：特质制作"
    url: "https://hoi4.paradoxwikis.com/Trait_modding"
  - label: "HOI4 Wiki：触发条件"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki：修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
---

# 做一个将领靠打仗攒出来的特质

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到三件事：

1. 打开任何一位将领的面板，可解锁特质树里有你的特质，图标、名字、说明都对。
2. 鼠标悬停时提示框显示修正、「将会花费 15.00 陆军经验来分配这个特质」和你的说明；这位将领在平原打过仗、攒到一点经验之后，再多出「经验值：X/100」和「获取经验值当：在平原地形作战时」两行。
3. 这位将领在平原打过仗之后经验涨到 100，玩家点一下就能把特质分配给他，之后他带的步兵攻击提高。

## 开始前需要什么

- 一个已经能正常导出的 Mod。本文用占位 ID `my_plains_fighter`。
- 一个有将领的国家用来看效果，本文用卢森堡，开局有四位将领。
- 不需要任何 DLC。
- 建议先读特质：四类各管什么，怎么到人身上 (`trait-system`)，知道将领特质的修正是包在块里的。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 将领特质定义 | `common/unit_leader/` | 本文的主角 |
| 名字与说明 | `localisation/` | 键是特质名，说明加 `_desc` |
| 图标 | `interface/*.gfx` | 精灵名 `GFX_trait_` 加特质名 |
| 将领 | 任意 | 不用改人物档案；特质是攒出来的 |

## 1. 定义特质

新建 `common/unit_leader/my_traits.txt`：

```
leader_traits = {
    my_plains_fighter = {
        type = corps_commander
        gain_xp = {
            is_fighting_in_terrain = plains
        }
        custom_gain_xp_trigger_tooltip = my_plains_fighter_when
        cost = 100
        modifier = {
            army_infantry_attack_factor = 0.05
        }
        ai_will_do = {
            factor = 1
        }
        gui_row = 0
    }
}
```

逐字段：

| 字段 | 作用 | 说明 |
| --- | --- | --- |
| `type = corps_commander` | 只有将领能有 | 元帅写 `field_marshal`，两者都行写 `land` |
| `gain_xp = { ... }` | 攒经验的条件 | **在战斗作用域判断**：这位将领的部队正在打的每一场仗都会检查一遍 |
| `custom_gain_xp_trigger_tooltip` | 提示框里「获取经验值当：」后面那句话的本地化键 | 不写时引擎自己描述条件，复杂条件会很难读 |
| `cost = 100` | 这个特质自己的经验攒到多少才能分配 | 原版常见 500 到 1500；和分配时扣的那 15 点陆军经验是两回事 |
| `modifier` | 分配后给他带的部队的修正 | 只给部队；给将领本人的写 `non_shared_modifier` |
| `ai_will_do` | AI 愿不愿意分配 | 权重 0 则 AI 永远不点 |
| `gui_row = 0` | 树里第几行 | **不写就不进树**，玩家看不见 |

这里故意**不写 `trait_type`**。原版靠打仗攒的特质（步兵指挥官、组织者、诡计大师……16 条）都不写，引擎把它们放在树最左边那一列；写了 `assignable_trait` 它会跑到可分配那一列，本身也能用，但不能再当别的特质的前置（见特质树 (`general-trait-tree`)）。

`gain_xp` 的条件在**战斗作用域**判断，这是原版文件头注释写明的（`gain_xp = trigger # scope is a combatant`）。所以 `is_fighting_in_terrain = plains` 这类战斗条件能用，而「部队数量超过十个」这类将领条件要写进 `gain_xp_leader`，那个在将领作用域判断。不打仗就不攒，和平期这条永远是 0。

提示框里有两笔「经验」，别混：一笔是这个特质自己的进度，`cost = 100` 是它的门槛，攒到 5% 以上提示框才开始显示「经验值：X/100」（游戏定义 `LEADER_TRAITS_XP_SHOW = 0.05`）；另一笔是点分配时扣的陆军经验，全局固定 15 点（`UNIT_LEADER_ASSIGN_TRAIT_COST = 15`），提示框写「将会花费 15.00 来分配这个特质」。

## 2. 起名字和说明

`localisation/simp_chinese/` 下三行，键分别是特质名、特质名加 `_desc`、提示文案键：

```
l_simp_chinese:
 my_plains_fighter:0 "平原战专家"
 my_plains_fighter_desc:0 "在平原作战时表现突出。"
 my_plains_fighter_when:0 "在平原地形作战时"
```

游戏提示框里的那句「获取经验值当：」是引擎的固定文案，后面接你的 `my_plains_fighter_when`。

## 3. 登记图标

`interface/` 下新建一个 `.gfx`：

```
spriteTypes = {
    spriteType = {
        name = "GFX_trait_my_plains_fighter"
        textureFile = "gfx/interface/traits/my_plains_fighter.dds"
    }
}
```

精灵名是固定规则：`GFX_trait_` 加特质名。原版 181 个将领特质图标都这么登记。不登记时界面显示什么，见文末实测。

## 4. 让它一开始就在某位将领身上

攒经验是给玩家的路。剧情上要某位历史将领一开始就有，直接写进人物档案：`corps_commander = { traits = { my_plains_fighter } }`；或者事件里 `random_army_leader = { add_unit_leader_trait = my_plains_fighter }`。两种写法都不看 `cost`，也不看 `gain_xp`。

## 5. 逐项核对

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 将领面板的可解锁树 | 有你的特质，在最左列 | `gui_row` 写了没有；`type` 对不对 |
| 开局悬停提示 | 修正、「将会花费 15.00 来分配这个特质」、你的说明 | `_desc` 本地化；没写时显示 `名字_desc` |
| 攒到 5% 以上再悬停 | 多出「经验值：X/100」「获取经验值当：在平原地形作战时」 | `cost`；`custom_gain_xp_trigger_tooltip` 的本地化键 |
| 图标 | 你的图 | 精灵名是不是 `GFX_trait_` 加特质名；没登记时是问号奖章 |
| 在平原打一仗后 | 经验上涨 | `gain_xp` 里的条件是不是战斗作用域能判断的 |
| 攒够后点分配 | 特质点亮，步兵攻击 +5% | `modifier` 是否包在块里 |
| `error.log` | 没有提到你的文件 | 修正名拼错时这里会有一行 |

## 常见错误

- 🔴 **没写 `gui_row`**：特质定义没问题，但树里看不见，玩家永远点不到。
- 🔴 **把将领条件写进 `gain_xp`**：`gain_xp` 在战斗作用域判断，`num_units > 10` 这类要写进 `gain_xp_leader`。
- **把 `cost` 当成分配价格**：它是攒够的门槛；分配时扣的是全局固定的 15 点陆军经验。
- **`cost` 忘了写**：没有门槛的特质表现如何，本文没有实测；原版每一条可分配特质都写了。
- **修正平铺在特质下面**：那是领袖特质的写法，将领特质必须包进 `modifier = { }`。
- **`ai_will_do` 权重写 0**：玩家能点，AI 永远不点。

## 想改成别的效果？

- 想按兵种而不是地形攒：原版「步兵指挥官」用 `gain_xp = { check_expr = { value = num_infantry greater_than = { value = num_units multiply = 0.8 } } }`，即步兵占比超过八成。
- 想给将领本人加经验获取：`non_shared_modifier = { experience_gain_factor = 0.1 }`。
- 想让它成为一棵树的起点：加 `gui_row`，别的特质用 `any_parent = { my_plains_fighter }` 指向它，见特质树 (`general-trait-tree`)。
- 想让有这个特质的将领能当高级指挥：加 `slot = high_command` 和三档顾问特质，指南里有说明。
