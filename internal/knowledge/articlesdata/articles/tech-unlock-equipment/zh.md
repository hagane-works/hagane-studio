---
contentId: tech-unlock-equipment
kind: case
slug: tech-unlock-equipment
locale: zh
status: published
title: "做一门解锁新步兵武器的科技"
summary: "在陆军分页加一个 1936 年可研究的节点：完成后生产线里出现这件装备，步兵软攻同时提高 5%。"
primaryTopic: research
topics: [research]
scenarios: [technology]
gameTerms: [科技, 解锁装备, 研究界面, 步兵装备, 软攻]
aliases: [科技解锁装备, 加一个能解锁装备的科技, 新武器科技, 科技加软攻]
relatedIds: [technology-system, research-speed, "wiki:effect:set_technology"]
updatedAt: "2026-09-10"
verification: source-checked
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：科技"
    url: "https://hoi4.paradoxwikis.com/Technology"
  - label: "HOI4 Wiki：本地化"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

# 做一门解锁新步兵武器的科技

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到三件事：

1. 研究界面的陆军分页出现一个新节点，名称是中文，不是一串下划线。
2. 研究完成后，生产线的装备下拉里能选到这件步兵装备。
3. 研究完成后，编有步兵营的师，其软攻数值比研究前高。

如果第 1 条成立但第 2 条不成立，问题几乎总在装备 ID 上；本文最后一节列了核对顺序。

## 开始前需要什么

- 一个已经能正常导出的 Mod。
- **这件步兵装备必须已经存在**。科技只负责解锁，不负责创建装备。指向一个不存在的装备 ID 时，游戏不会报错，只是这一项静默不生效——这是本篇最常见的失败方式。
- 不需要任何 DLC。本文用到的字段都属于基础游戏。
- 建议先读做一门科技：解锁什么、什么时候能研究 (`technology-system`)，了解"解锁对象"和"持续修正"为什么是两回事。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 科技定义 | `common/technologies/` 下的文件，包在 `technologies = { ... }` 里 | 节点本体 |
| 步兵装备 | `common/units/equipment/` | 被解锁的对象，必须先存在 |
| 名称与说明 | `localisation/` 下对应语言的 YML | 决定界面上显示中文还是原始 key |
| 图标 | `interface/` 下的 `.gfx` | 缺失时节点显示为空白图标，不影响研究 |

## 1. 建一个能被看到的节点

先只写"在哪、多贵、什么时候能研究"，不写任何解锁：

```
technologies = {
    my_infantry_smg_1 = {
        research_cost = 1.5
        start_year = 1936

        folder = {
            name = infantry_folder
            position = { x = 1 y = 2 }
        }
    }
}
```

`research_cost = 1.5` 在默认速度下约合 150 天；原版步兵主线常用 1.5。`start_year = 1936` 让它在开局就能正常研究——设成 1939 的话，1936 年开局的玩家研究它要多付 3 年的超时代惩罚，也就是每提前一年 +200% 成本。

`position` 里的两个数是**相对偏移**，不是绝对坐标，而且 **`y` 是横轴、`x` 是纵轴**：`y` 增加节点往右走（年份方向，原版把 1936 年定为 `y = 2`），`x` 增加往下换一行。上面用 `x = 1`、`y = 2`，就是把它放在步兵武器主线下面一行、1936 年那一列。如果新节点和旁边的节点重叠，改 `x` 换一行；想让它排在前置科技右边，加大 `y`。方向填反不报错，节点只是落进别人的链条里，一定进游戏看一眼。

做到这里进游戏，陆军分页应该已经能看到一个可点的节点，只是名字还是 `my_infantry_smg_1`。

## 2. 接上前置

让它挂在已有的步兵武器科技后面。`path` 写在**前置科技**的定义里，指向新科技：

```
infantry_weapons = {
    path = {
        leads_to_tech = my_infantry_smg_1
        research_cost_coeff = 1
    }
}
```

🔴 **科技没有 `prerequisites` 这个键。** `path` 一个键做两件事：画出连线，同时建立"必须先研究前置"
的要求。写了 `path` 就够了，不需要在新科技里再声明一次。

在科技块里写 `prerequisites = { ... }` 的后果是：`error.log` 报
`Unknown modifier: prerequisites`，而且那一整块被忽略。
`prerequisites` 属于国策和将领特质，不属于科技——原版 `common/technologies/` 里一次都没有出现过。

## 3. 填上解锁与修正

现在加真正的效果。注意这是两类不同的东西，写在同一个科技块里：

```
my_infantry_smg_1 = {
    enable_equipments = {
        my_smg_equipment_1
    }

    infantry = {
        soft_attack = 0.05
    }
}
```

`enable_equipments` 是**永久解锁一个对象**，从完成那一刻起生产线里就能选到 `my_smg_equipment_1`。下面的 `infantry = { soft_attack = 0.05 }` 是**给兵种类别的持续加成**，`0.05` 表示 +5%。

这两项互不依赖：只写解锁就是"多一件可造的装备但数值不变"，只写加成就是"数值变好但没有新装备"。

## 4. 补上名称

科技的本地化 key 就是科技 ID，说明再加 `_desc` 后缀：

```
l_simp_chinese:
 my_infantry_smg_1: "冲锋枪"
 my_infantry_smg_1_desc: "为步兵班配备高射速近战武器。"
```

覆盖已有 key 时文件要放进 `localisation/<语言>/replace/`，否则会被原版或前置 Mod 的同名 key 压掉，而且完全没有提示。本例是新增 key，不涉及覆盖。

图标的 sprite 名固定为 `GFX_my_infantry_smg_1_medium`。不定义也能研究，只是节点图标是空的。

## 5. 逐项核对

开一局独立测试存档，按顺序确认，别跳步：

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 陆军分页能看到节点 | 位置不与其他节点重叠 | `folder` 的 name 与 `position` 的 x |
| 节点显示中文名 | 不是 `my_infantry_smg_1` | 本地化文件的语言目录与 key 拼写 |
| 前置未研究时不能点 | 灰掉 | 前置科技那一侧的 `path` 是否指向了本科技 |
| 研究耗时接近 150 天 | 不是几年 | `start_year` 是否晚于开局年份 |
| 完成后生产线能选到装备 | 出现在装备下拉里 | **装备 ID 拼写，以及该装备是否真的存在** |
| 完成后步兵软攻上升 | 师详情里数值变高 | `infantry` 块的 stat 名与数值 |

## 常见错误

- **装备 ID 拼错**：这一项静默失效，节点照常研究完成，只是生产线里什么都没多。核对时把科技里的 ID 和装备定义里的 ID 并排比对，不要凭记忆。
- **把加成写成了解锁**：`soft_attack` 属于兵种类别块，不能放进 `enable_equipments`。
- **给科技写 `prerequisites`**：科技没有这个键，会报 `Unknown modifier: prerequisites` 并整块失效。
  前置只由前置科技那侧的 `path` 建立。
- **`start_year` 设得太晚**：玩家会以为科技坏了，实际是在付超时代惩罚。
- **用 `allow_branch` 想做"暂时不能研究"**：那会让整条分支从界面上消失，玩家根本找不到。要"看得见但点不了"用 `allow`。

## 想改成别的效果？

- 想让它解锁一个新兵种而不是装备，把 `enable_equipments` 换成 `enable_subunits`，其余步骤不变。
- 想让它完成瞬间弹一个事件，用 `on_research_complete`，并配 `on_research_complete_limit` 防止重复触发。
- 想让玩家不能自己研究、只能由国策授予，写 `allow = { always = no }`，再用直接解锁指定科技 (`wiki:effect:set_technology`)在国策奖励里给出去。
