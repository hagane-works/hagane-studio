---
contentId: first-content
kind: tutorial
slug: first-content
locale: zh
status: published
title: "第 2 步：写一个提高科研速度的国家精神"
summary: "用完整 ideas 文件定义科研速度 +5% 的精神，分清“定义内容”和“把内容交给国家”。"
primaryTopic: research
topics: [research]
scenarios: [ideas]
gameTerms: [国家精神, 科研速度]
aliases: [第一个国家精神, 第一份游戏内容, 新手脚本, national spirit tutorial]
relatedIds: [first-mod, localisation, check-and-troubleshoot, "wiki:modifier:research_speed_factor", idea-supply-consumption]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "HOI4 Mod 入门第二步：完整国家精神定义 ideas/country/HW_learning_spirit。modifier 中 research_speed_factor = 0.05，提高科研速度5%。定义不会自动赋予国家，独立本地测试局可用 add_ideas HW_learning_spirit 检查。"
sources:
  - label: "HOI4 Wiki：国家精神"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
  - label: "HOI4 Wiki：科研速度"
    url: "https://hoi4.paradoxwikis.com/Modifiers#research_speed_factor"
  - label: "HOI4 Wiki：控制台命令"
    url: "https://hoi4.paradoxwikis.com/Console_commands"
---
上一步已经准备好 `hw_learning` Mod。现在给它加入一个国家精神：拥有它的国家获得科研速度 +5%。

这一页只增加一个定义，不改变国家历史，也不自动给所有国家添加精神。

## 1. 写入完整定义

在 Mod 内容目录创建 `common/ideas/hw_learning_ideas.txt`，保存为 UTF-8 无 BOM。下面是**完整文件内容**：

```hoi4
ideas = {
    country = {
        HW_learning_spirit = {
            picture = generic_army_war_college
            removal_cost = -1
            modifier = {
                research_speed_factor = 0.05
            }
        }
    }
}
```

## 2. 看懂这几个位置

| 位置 | 作用 |
| --- | --- |
| `ideas` | 这个文件的国家精神、法律等内容的外层结构 |
| `country` | 本例的国家精神类别 |
| `HW_learning_spirit` | 脚本引用使用的唯一 ID，不是中文显示名称 |
| `picture` | 引用原版已存在的图标，本例无需绘制或注册图片 |
| `modifier` | 精神生效时提供的修正 |
| `research_speed_factor = 0.05` | 科研速度 +5%，不是每天获得 0.05 点 |

`removal_cost = -1` 表示不能手动花政治点移除；它不妨碍脚本用 `remove_ideas` 移除这个精神。

脚本里的括号、等号、字段名要保持原样。中文名称会写在下一页的本地化文件里，不要把 `country` 或 `research_speed_factor` 改成中文。

## 3. 定义不等于获得

这份文件只是让游戏认识 `HW_learning_spirit`。要在某个国家上看到它，还需要赋予它。

为了先检查定义，可以在只启用此练习 Mod 的独立本地测试局中，手动打开游戏控制台，对当前国家执行下面这条**完整控制台命令**。不要在要保留的正式存档或多人游戏中做这一步：

```text
add_ideas HW_learning_spirit
```

命令使用空格，没有等号。这是测试入口，不是放进 `.txt` 文件的写法。正式玩法通常由国策、事件或决议调用效果来赋予；补给国家精神案例 (`idea-supply-consumption`) 展示了接入已有国策的方法。

如果你暂时不准备启动测试局，也可以先做下一步本地化，再一起检查。不必为了完成教程就改动现有存档。

## 4. 现在应该检查什么？

手动测试时，国家应获得一个提供“科研速度 +5%”的精神。如果名称显示为 `HW_learning_spirit`，说明显示文本尚未补齐，不要先把脚本 ID 改成中文。

如果命令找不到精神，先查真实文件名、目录、括号与 ID；如果精神存在但效果不对，核对修正是否在 `modifier` 内、数值是否为 `0.05`。

本页已经按 1.19.2.0 的原版定义、图标和控制台文档核对，未由作者执行本例游戏实测。上述结果是待验证目标。

上一步：Mod 文件结构 (`first-mod`)。下一步：给内容加上中文名称 (`localisation`)。
