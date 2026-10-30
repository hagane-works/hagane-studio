---
contentId: technology-system
kind: guide
slug: technology-system
locale: zh
status: published
title: "做一门科技：解锁什么、什么时候能研究"
summary: "研究完成后给玩家什么，以及这门科技在什么条件下才允许研究、才看得见。两件事用的不是同一个块。"
primaryTopic: research
topics: [research]
scenarios: [technology]
gameTerms: [科技, 研究, 科技树, 研究界面, 前置科技, 超时代惩罚]
aliases: [科技怎么做, 加一门科技, 自定义科技, 科技树制作, 解锁装备的科技, technology]
relatedIds: [research-speed, tech-unlock-equipment, "wiki:effect:set_technology", "wiki:effect:add_tech_bonus", "wiki:modifier:research_speed_factor"]
updatedAt: "2026-09-09"
verification: static-checked
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：科技"
    url: "https://hoi4.paradoxwikis.com/Technology"
  - label: "HOI4 Wiki：修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
---

# 做一门科技：解锁什么、什么时候能研究

## 这个系统负责什么

研究界面里的每一个可点节点都是一门科技。玩家花研究时间点亮它，完成后得到一次性的解锁或长期的加成。

作为作者，你实际上只需要决定三件事：

1. **研究完成后给什么**（解锁一件装备？还是给国家一个持续加成？）
2. **什么条件下允许研究、什么条件下才看得见**
3. **它出现在研究界面的哪一页、哪个位置**

这三件事在脚本里是三组互不相干的字段。大部分制作问题源于把其中两件混成一件。

## 核心概念：解锁一个对象，不等于改变一个数值

科技能做的事分成三类，行为完全不同：

| 类别 | 你写的东西 | 发生的时机 |
| --- | --- | --- |
| **解锁一个对象** | `enable_equipments`、`enable_subunits`、`enable_equipment_modules`、`enable_building`、`enable_tactic` | 完成的那一刻起永久可用 |
| **持续生效的数值** | 科技块里的 `modifier = { ... }` | 完成后一直生效，和国家精神的修正叠加 |
| **只发生一次的脚本** | `on_research_complete = { ... }` | 完成的瞬间执行一次，之后不再触发 |

第一类给玩家新的**东西**（生产线上多一件装备、师模板里多一个营）；第二类改变已有的**数值**；第三类是一次性事件，用来发通知、加载编制或触发剧情。

想要"研究完这门科技后，所有步兵的软攻提高 5%"，用的是第二类；想要"研究完之后弹出一个事件"，用的是第三类。两者不能互换。

## 文件结构与运行过程

科技定义在 `common/technologies/` 下的文件里，全部包在一个 `technologies = { ... }` 块中，一门科技一个条目：

```
technologies = {
    my_technology_1 = {

    }
    my_technology_2 = {

    }
}
```

解锁类字段的写法各不相同，注意大括号的有无：

```
enable_equipments = { my_equipment_1 my_equipment_2 }
enable_subunits = { my_subunit_1 }
enable_equipment_modules = { my_module_1 }
enable_tactic = my_tactic
enable_building = {
    building = my_building
    level = 5
}
```

节点之间用 `path` 连线，位置写在 `folder` 块里：

```
path = {
    leads_to_tech = my_technology_2
    research_cost_coeff = 1
}
folder = {
    name = my_folder
    position = { x = 0 y = 1 }
}
```

科技 ID 全 Mod 唯一，只能用小写字母加下划线，不能有空格或中文。它会被 `has_tech`、`set_technology`、`add_tech_bonus` 等脚本按名字引用，改名会让所有引用同时失效。

## 常见制作目标与对应方法

| 你想在游戏里看到 | 从这里开始 | 关键区别 |
| --- | --- | --- |
| 研究完成后，生产线上多一件可造的装备 | `enable_equipments` | 装备本身要先在 `common/units/equipment/` 里存在 |
| 研究完成后，师模板里多一个可编入的营 | `enable_subunits` | 解锁的是兵种类型，不是具体某支部队 |
| 研究完成后，坦克或飞机设计器里多一个部件 | `enable_equipment_modules` | 只影响设计器可选项，不改已造出的装备 |
| 研究完成后，可以建一种新建筑 | `enable_building` | 是块写法，同时指定建筑和等级 |
| 研究完成后，国家长期获得某个加成 | 科技块内的 `modifier = { ... }` | 持续生效，和国家精神的修正是同一套 stat |
| 研究完成的瞬间发一个事件或加载编制 | `on_research_complete` | 只执行一次，需要配 `on_research_complete_limit` 防重入 |
| 这门科技不允许自己点，只能靠剧情解锁 | `allow = { always = no }`，再用国策的 直接解锁指定科技 (`wiki:effect:set_technology`) 给出去 | 节点仍然可见，只是灰掉 |
| 想让某一类研究更快，而不是解锁东西 | 持续提高或降低科研速度 (`wiki:modifier:research_speed_factor`) 或 给予有限次数的研究加成 (`wiki:effect:add_tech_bonus`) | 这两条属于科研速度，不属于科技定义 |

## 容易混淆的地方

### `allow` 管能不能点，`allow_branch` 管看不看得见

两个都是条件块，名字只差一个词，作用完全不同：

- `allow = { ... }` 持续检查。不满足时科技**仍然显示在研究界面上**，只是不能手动研究。
- `allow_branch = { ... }` 决定可见性。不满足时该科技**以及与它相连的科技一起从界面上消失**。

原版用 `allow_branch` 配合 `has_dlc` 把 DLC 专属分支对没有该 DLC 的玩家整条隐藏。如果你想要的是"看得到但点不了"，写成 `allow_branch` 会让玩家根本找不到它。

### 成本类修正的减号是加成

一部分 stat 的数值方向和直觉相反。`build_cost_ic = -0.1` 表示生产成本 **降低 10%**，是一个好加成；写成正数反而让单位更贵。科技和学说里这一档常见取值是 -5% 到 -15%。

符号写反不会报错，导出也不会拦，只有在游戏里对比生产队列才能看出来。填这类 stat 时先确认你要的是"更便宜"还是"更贵"。

### 位置是相对偏移，不是绝对坐标；填完一定要进游戏看

`position` 里的两个数是**相对偏移**，不是屏幕坐标。原版把年份写成常量再用在 `y` 上：

```
@1936 = 2
@1938 = 4

folder = {
    name = infantry_folder
    position = { x = 0 y = @1936 }
}
```

🔴 **`y` 是横轴，`x` 是纵轴——和直觉相反。**

- `y` 增加 = 节点**往右移**。研究界面里科技是从左往右推进的，年份轴是横着走的。
- `x` 增加 = 节点**往下换一行**。并列的几条支线靠它岔开。

原版数据是这样的（1.19.2 的 `infantry_folder`）：

```
infantry_weapons1          x=0  y=2 (@1936)
infantry_weapons2          x=0  y=4 (@1938)
improved_infantry_weapons  x=0  y=6 (@1939)
```

同一条链 `x` 始终是 0，只有 `y` 在涨——而它在游戏里是一条**水平**的链。
装甲车线的 `armored_car1` 是 `x=4`，画在下面第 4 行。

⚠️ **`start_year` 与 `position` 无关。** 前者只决定超时代惩罚的基准年，后者只决定画在哪一格。
两者不一致不会报错，但玩家会困惑——原版把年份常量写进 `y` 是**惯例**，不是引擎强制。
架空或古代背景的 Mod 完全可以自己定义年份档位。

填错方向不会有任何报错：节点照样画出来，只是**落进了别人的链条里**，看起来像"这条科技线多了个
奇怪的节点"。同一格上的两个节点会直接重叠。这一层的错误 `error.log` 一个字都不会说。

### 🔴 自建 folder 的话，光写科技是画不出来的

上面的例子用的是 `infantry_folder` 这类**原版已有的** folder。如果你想要一个属于自己的科技页签，
只写 `common/technologies/` 是不够的——科技会"存在但一个都画不出来"。

一个 folder 要能显示，需要三样东西同时到位：

```
① 科技定义    common/technologies/*.txt          → 它属于哪个 folder、画在哪一格
② folder 声明 common/technology_tags/*.txt       → 这个 folder 存在、归哪本统计账
③ 界面三件套  interface/countrytechtreeview.gui  → 容器 + 格子(gridbox) + 节点模板
```

再加上页签按钮和本地化，否则玩家点不进去、或者看到的是原始 key。**绝大多数教程只讲 ①。**

网上流传的"自定义科技不会出现在科技树里"这句话只对了一半：准确的说法是**需要你自己在
`countrytechtreeview.gui` 里补声明，补齐了就能画**。完整配方见自建科技页签 (`custom-tech-folder`)。

### 🔴 不要把新科技放进有 DLC 替换关系的 folder

原版有三个 folder 会被 DLC 整体替换：

| 基础版 folder | 被哪个 DLC 换成 |
| --- | --- |
| `armour_folder` | No Step Back → `nsb_armour_folder` |
| `air_techs_folder` | By Blood Alone → `bba_air_techs_folder` |
| `naval_folder` | Man the Guns → `mtgnavalfolder` |

六个名字在 1.19.2 里同时存在，但**同一时刻只有一个会绘制**。把新科技挂进被替换的那个，
玩家一旦开了对应 DLC，你的科技就**凭空消失，而且没有任何报错、日志完全干净**。

`infantry_folder`、`industry_folder`、`electronics_folder`、`support_folder`、`artillery_folder`
没有替换关系，可以放心用。

### 早于历史年份研究会被拖慢

`start_year` 是这门科技的历史可用年份。玩家在该年份之前研究会触发超时代惩罚：**每提前一年，研究成本增加 200%**。这不是 bug，是引擎的既定行为；如果你希望某门科技在开局就能顺畅研究，要把 `start_year` 设到开局年份或更早。

与它成对的是 `research_cost`，表示研究时间的相对倍率——`research_cost = 1` 在默认速度下约为 100 天。原版主线科技集中在 1.5 到 2.5 之间。

### 版本与适用边界

本文按标记为 1.19.2.0 的随附文档与原版脚本整理。装备设计器相关的 XP 加速字段（`xp_research_type`、`xp_boost_cost`）属于 DLC 机制，在没有对应 DLC 的存档里不会生效。

## 接下来看什么

- 想要一个属于自己的科技页签，看自建科技页签：为什么科技存在却画不出来 (`custom-tech-folder`)。
- 实战案例：做一门解锁新步兵武器的科技 (`tech-unlock-equipment`)，从空白节点一路做到能在生产线里选到这件装备。
- 想调整研究速度而不是解锁内容，看《科研速度、研究加成与科技解锁》（文章 `research-speed`）。
- 想让国策完成时送一次研究加成，看国策奖励一次 50% 工业研究加成 (`focus-research-bonus`)。
