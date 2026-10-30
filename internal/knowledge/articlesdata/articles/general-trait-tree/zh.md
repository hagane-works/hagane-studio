---
contentId: general-trait-tree
kind: case
slug: general-trait-tree
locale: zh
status: published
title: "把几个将领特质连成一棵可解锁的树"
summary: "三个将领特质排成两行：第一个是起点，后两个以它为前置且互斥，玩家在将领面板上按顺序解锁；讲清前置的三种写法、互斥要两边都写、行列怎么排、树和编辑器画布怎么对应。"
primaryTopic: people
topics: [people, military]
scenarios: [traits]
gameTerms: [将领, 特质树, 前置, 互斥, 解锁, 经验, 分配]
aliases: [特质树怎么做, 特质前置, 特质互斥, 将领特质树, gui_row, any_parent, parent, mutually_exclusive, num_parents_needed]
relatedIds: [trait-system, general-trait-earned-in-combat, custom-character-traits, "wiki:trigger:has_trait", "wiki:modifier:planning_speed", "wiki:modifier:army_defence_factor"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：特质制作"
    url: "https://hoi4.paradoxwikis.com/Trait_modding"
  - label: "HOI4 Wiki：修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
---

# 把几个将领特质连成一棵可解锁的树

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到三件事：

1. 将领面板的可解锁树里，起点特质在左边那一列，下一行两个分支各占一列，都从起点连下来。
2. 第二行两个之间有互斥标记；点了其中一个，另一个就点不了。
3. 起点没点亮之前，两个分支悬停提示写着「你需要至少解锁 1 个前级特质才能够分配这个特质」。

## 开始前需要什么

- 一个已经能正常导出的 Mod。本文用占位 ID `my_tree_root`、`my_tree_left`、`my_tree_right`。
- 建议先做完做一个将领靠打仗攒出来的特质 (`general-trait-earned-in-combat`)，本文的起点就是那种特质。
- 不需要任何 DLC。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 三条将领特质定义 | `common/unit_leader/` | 一个起点、两个分支 |
| 名字 | `localisation/` | 键是特质名 |
| 图标 | `interface/*.gfx` | `GFX_trait_` 加特质名 |

## 1. 三条定义

```
leader_traits = {
    my_tree_root = {
        type = corps_commander
        gain_xp = {
            is_fighting_in_terrain = plains
        }
        cost = 100
        modifier = {
            army_infantry_attack_factor = 0.05
        }
        ai_will_do = {
            factor = 1
        }
        gui_row = 16
    }

    my_tree_left = {
        type = corps_commander
        trait_type = assignable_trait
        gain_xp = {
            always = no
        }
        cost = 100
        modifier = {
            planning_speed = 0.1
        }
        ai_will_do = {
            factor = 1
        }
        any_parent = { my_tree_root }
        mutually_exclusive = my_tree_right
        gui_row = 17
        gui_column = 2
    }

    my_tree_right = {
        type = corps_commander
        trait_type = assignable_trait
        gain_xp = {
            always = no
        }
        cost = 100
        modifier = {
            army_defence_factor = 0.05
        }
        ai_will_do = {
            factor = 1
        }
        any_parent = { my_tree_root }
        mutually_exclusive = my_tree_left
        gui_row = 17
        gui_column = 3
    }
}
```

起点**不写 `trait_type`**。原版陆军树里所有当前置的根特质（组织者、步兵指挥官、诡计大师……共 16 条）都这么写；引擎把它们放在树最左边那一列，可分配特质挂在它们下面。实测把起点写成 `trait_type = assignable_trait`，`error.log` 记两行 `my_tree_left has a wrong kind of parent (my_tree_root)`，改回原版写法后树正常画出。

两个分支写 `gain_xp = { always = no }`，意思是不靠打仗攒，只要前置点亮、经验够 `cost` 就能买。原版「步兵专家」「进攻学说」都是这么写的，文件里注释为 `# not gainable`。

## 2. 前置的三种写法

原版文件头注释给了三种，都写在特质定义里：

| 写法 | 含义 | 原版用量 |
| --- | --- | --- |
| `any_parent = { a b c }` | 任意一个前置点亮即可 | 41 处，最常用 |
| `all_parents = { a b c }` | 全部点亮才行 | 0 处 |
| `parent = { traits = { a b c } num_parents_needed = 2 }` | 点亮其中 N 个 | 1 处：地形大师要求六种地形特质里的两种 |

分支未满足前置时悬停显示「你需要至少解锁 N 个前级特质才能够分配这个特质」，实测就是这句；`all_parents` 对应「必须先解锁所有前级特质之后才能分配这个特质」。`any_parent` 就是 N 等于 1 的简写。

## 3. 互斥要两边都写

`mutually_exclusive = my_tree_right` 写在左边，`mutually_exclusive = my_tree_left` 写在右边。只写一边，另一边的提示和连线都不完整；原版「进攻学说」「防御学说」是成对写的。

## 4. 行和列

`gui_row` 从 0 开始数，决定第几行；`gui_column` 不写时引擎按 `trait_type` 自动挑一列：不写 `trait_type` 的攒经验特质在最左列，可分配特质在它右边那列，地形特质另有一列。原版 76 条特质写了 `gui_row`，只有 29 条写了 `gui_column`，全在海军树里。

🔴 **同一行、同一 `trait_type`、都不写列，两个特质会落在同一格互相遮挡**，实测只看得见后加载的那个。两个分支要么各写一个 `gui_column`，要么错开行。列号对应面板上固定的列位，实测 `gui_column = 3` 落在最右边、被滚动条挡住一半；两个分支写 2 和 3 时能看全的话就用，看不全就把右边那个改小或换行。

原版陆军树用到第 15 行，自己的树从第 16 行起就不会和原版同格。只有 `gui_row` 大于等于 0 的特质才进树；起点和两个分支都要写。

## 5. 除前置之外的额外条件

`prerequisites = { ... }` 在将领作用域判断，原版只有 3 处，都是 `has_trait = 某个人格特质`，用来要求某种人格才能学某个特质。要求技能等级用 `defense_skill_level > 3` 这类。它和 `any_parent` 的区别：前置是树上的连线，`prerequisites` 是树之外的门槛，提示框里分开显示。

## 6. 编辑器画布

角色工作台的自定义特质面板有一个特质树画布：只显示 `gui_row` 大于等于 0 的特质；拖节点改 `gui_row` 和 `gui_column`；从一个节点下方把手拉到另一个上方把手，就是给后者加一条 `any_parent`。`parent` 块和 `all_parents` 在画布上是虚线，只读，要改走原始文件。互斥在表单的「高级字段」里填。

## 7. 逐项核对

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 树的形状 | 起点在最左列第 16 行，两个分支在第 17 行各占一列 | 三条的 `gui_row`、分支的 `gui_column` |
| 两个分支都看得见 | 是 | 是不是同格；右边那列是不是被挡住 |
| 连线 | 两个分支各有一条连到起点 | `any_parent` 里的名字；`error.log` 有没有 `wrong kind of parent` |
| 起点未点亮时悬停分支 | 「你需要至少解锁 1 个前级特质才能够分配这个特质」 | `any_parent` |
| 点了左边后看右边 | 灰掉，提示互斥 | 两边是否都写了 `mutually_exclusive` |
| 图标 | 三个都是你的图 | 三个 `GFX_trait_` 精灵；没登记时是问号奖章 |
| 悬停底部 | 你的说明文字 | `_desc` 本地化；没写时显示 `名字_desc` 这个键 |

## 常见错误

- 🔴 **起点写了 `trait_type = assignable_trait`**：`error.log` 报 `wrong kind of parent`。当前置的根特质照原版不写 `trait_type`。
- 🔴 **两个分支同行同类不写列**：落在同一格，只看得见一个。
- 🔴 **互斥只写一边**：另一边不知道自己被互斥。
- 🔴 **分支没写 `gui_row`**：不进树，连线没有终点。
- **前置写了不存在的特质名**：连线画不出来；先确认名字。
- **把 `prerequisites` 当前置用**：它不画线，只挡分配。
- **分支忘了 `gain_xp = { always = no }`**：不写 `gain_xp` 的特质表现如何本文没有实测，原版可买特质都写了。

## 想改成别的效果？

- 想要「点亮任意两个地形特质后解锁」：`parent = { traits = { a b c } num_parents_needed = 2 }`。
- 想要三层树：第三层特质 `any_parent = { my_tree_left my_tree_right }`，`gui_row = 18`。
- 想让分支也能靠打仗攒：把 `gain_xp` 换成真实条件，见攒经验的特质 (`general-trait-earned-in-combat`)。
