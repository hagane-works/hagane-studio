---
contentId: custom-character-traits
kind: case
slug: custom-character-traits
locale: zh
status: published
title: "给人物做一个自定义特质"
summary: "领袖特质和将领特质是两种文件形状：领袖特质直接列修正，将领特质把修正包在块里并声明适用兵种。没起名字的特质会把键名原样显示在人物卡上。"
primaryTopic: people
topics: [people]
scenarios: [traits]
gameTerms: [特质, 领袖特质, 将领特质, 人物卡, 修正, 科学家特质]
aliases: [自定义特质, 加一个特质, 特质显示键名, 特质没有名字, 领袖特质怎么写, 将领特质怎么写, custom trait, leader trait, unit leader trait]
relatedIds: [character-system, leader-succession, political-power, "wiki:modifier:political_power_factor", "wiki:modifier:max_dig_in"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：特质"
    url: "https://hoi4.paradoxwikis.com/Traits"
  - label: "HOI4 Wiki：修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
  - label: "HOI4 Wiki：本地化"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

# 给人物做一个自定义特质

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到两件事：

1. 领袖卡上出现一个新特质，名字是中文，悬停能看到它给的修正。
2. 一名将军的特质栏里出现一个新特质，悬停能看到它给的修正。

如果特质出现了但名字是一串小写字母加下划线，那是本地化没接上；本文第 4 节专门讲这个。

## 开始前需要什么

- 一个已有领袖人物和一个已有将军人物，把特质挂上去用。
- 不需要任何 DLC。科学家特质另有限制，见常见错误。
- 建议先读做一个人物：领袖、将领、顾问各管什么 (`character-system`)。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 领袖特质定义 | `common/country_leader/` | 领袖和顾问共用这一类 |
| 将领特质定义 | `common/unit_leader/` | 将军、元帅、海军将领用这一类 |
| 人物 | `common/characters/` | 在身份块的 `traits` 里引用特质 |
| 名称与说明 | `localisation/` | 决定人物卡上显示中文还是键名 |

工作台里这两类都在「管理自定义特质」面板里建，分别是「自定义 country_leader 特质」和「自定义 unit_leader 特质」；人物编辑页会按角色类型只列出兼容的那一类。

## 1. 领袖特质：修正直接列在特质下面

领袖特质文件的形状很扁：一个 `leader_traits = { ... }` 包着所有特质，每个特质下面直接列修正键：

```
leader_traits = {
    my_leader_trait = {
        random = no
        political_power_factor = 0.5
        stability_factor = 0.1
    }
}
```

`random = no` 是原版给具名人物特质的惯例写法，表示不把它分给随机生成的领袖。修正键就是国家修正，`political_power_factor = 0.5` 是每日政治点 +50%，`stability_factor = 0.1` 是稳定度 +10%。这里**没有** `modifier = { ... }` 这一层，和下一节的将领特质正好相反。

工作台里：「管理自定义特质」→「新建 Trait」→ 选 country_leader 类型 → 填「显示名称」→ 在修正里填键和数值。导出器写出的就是上面这个形状。

## 2. 将领特质：修正包在块里，还要声明适用兵种

将领特质多两样东西：`type` 说它适用于哪类将领，修正要包进 `modifier` 块：

```
leader_traits = {
    my_commander_trait = {
        type = land
        trait_type = personality_trait
        modifier = {
            max_dig_in = 2
        }
    }
}
```

`type = land` 表示陆军将领；海军用 `navy`，两者都适用写 `type = { land navy }`。`trait_type = personality_trait` 是原版给"人物自带、不靠经验学"的特质的分类。`max_dig_in = 2` 是掘壕上限 +2，悬停将军的特质时能看到。

工作台里：「新建 Trait」→ 选 unit_leader 类型 → 填「显示名称」→ 修正。「高级字段」里的树内位置（`gui_row`、`gui_column`）、前置和 `cost` 只在你想让玩家用经验去学这个特质时才需要；直接挂在人物身上的特质用不到。

## 3. 把特质挂到人物上

在人物的身份块里引用特质的 ID：

```
country_leader = {
    ideology = conservatism
    traits = { my_leader_trait }
    expire = "1965.1.1.1"
    id = -1
}
```

```
corps_commander = {
    traits = { my_commander_trait }
    skill = 3
    attack_skill = 3
    defense_skill = 2
    planning_skill = 2
    logistics_skill = 2
    legacy_id = -1
}
```

工作台里打开人物编辑页，「特质」区会列出你建的自定义特质，勾上即可。领袖特质只出现在领袖和顾问的列表里，将领特质只出现在将军、元帅、海军将领的列表里，不需要自己分。

## 4. 起名字：没起名会把键名直接显示出来

特质的本地化键就是特质 ID，说明再加 `_desc` 后缀：

```
l_simp_chinese:
 my_leader_trait: "改革派"
 my_leader_trait_desc: "推动改革的领袖。"
 my_commander_trait: "工事专家"
 my_commander_trait_desc: "擅长构筑阵地。"
```

🔴 这一步漏掉的后果已经实测：特质照样生效，修正照样算，但领袖卡上显示的是 `my_leader_trait` 这串键名。没有任何报错。工作台的「显示名称」至少要填一种语言，两种都留空时导出的就是这个结果。

## 5. 逐项核对

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 领袖卡上的特质 | 显示中文名 | 本地化文件的语言目录与键拼写 |
| 悬停领袖特质 | 看到每日政治点 +50%、稳定度 +10% | 修正键是否直接写在特质下，没有多包一层 |
| 将军特质栏 | 显示中文名 | 同上 |
| 悬停将军特质 | 掘壕上限 +2 | 修正是否包在 `modifier` 块里；`type` 是否匹配 |
| 特质完全没出现 | 应该出现 | 人物身份块的 `traits` 里 ID 是否拼对；工作台会在导出前丢弃不存在的 ID 并提示 |

## 常见错误

- 🔴 **没起名**：特质生效但显示键名。补本地化。
- **把将领特质写成领袖特质的扁平形状**：将领特质的修正必须在 `modifier` 块里。这一条来自原版文件形状，本文没有实测写反的后果。
- **给科学家特质填常规科研修正**：原版科学家特质文件首行注释明写，科学家特质只作用于特殊计划，只能用适用于特殊计划的修正。`research_speed_factor` 之类不会生效。
- **特质 ID 拼错**：工作台在导出前把既不在原版表、也不在你自定义特质里的 ID 丢掉并提示，人物会少一个特质。

## 想改成别的效果？

- 想让特质只对某个国家生效（比如"对英国好感"）：领袖特质支持 `targeted_modifier`，工作台叫「定向国家修正」。它和「外交好感修正」是两回事，后者是另一套定义。
- 想让特质每天多给政治点：看《政治点：奖励、门槛与每日获取》（文章 `political-power`）里固定项和比例项的区别。
- 想在剧情里给在任领袖临时加一个特质：用给领袖添加特质 (`wiki:effect:add_country_leader_trait`)，特质本身还是按本文定义。
