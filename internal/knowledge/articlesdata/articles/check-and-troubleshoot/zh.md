---
contentId: check-and-troubleshoot
kind: tutorial
slug: check-and-troubleshoot
locale: zh
status: published
title: "第 4 步：检查 Mod，定位没有生效的原因"
summary: "从文件加载、精神赋予、显示文本到实际修正逐层检查，并区分静态检查与游戏验证。"
primaryTopic: research
topics: [research]
scenarios: [ideas, focus]
gameTerms: [国家精神, 科研速度, 游戏日志]
aliases: [Mod不生效, 国家精神不显示, 排错, error.log, 检查清单, troubleshoot]
relatedIds: [first-mod, first-content, localisation, "wiki:modifier:research_speed_factor", focus-research-bonus, idea-supply-consumption]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "HOI4 Mod 入门第四步：检查是否加载正确 Mod、文件目录和扩展名、括号、国家精神赋予、中文本地化、科研速度数值。定位本次用户数据目录 logs/error.log 中自己的文件与 ID；没有日志错误不等于游戏结果正确。"
sources:
  - label: "HOI4 Wiki：Mod 结构与测试"
    url: "https://hoi4.paradoxwikis.com/Mod_structure"
  - label: "HOI4 Wiki：本地化问题"
    url: "https://hoi4.paradoxwikis.com/Localisation"
  - label: "HOI4 Wiki：控制台命令"
    url: "https://hoi4.paradoxwikis.com/Console_commands"
---
现在已经有描述文件、国家精神定义和中文文本。最后一步不是“看不到红字就完成”，而是确认同一个小例子从加载到实际作用都接上了。

## 启动前先做静态检查

- 目录与 第 1 步 (`first-mod`) 一致，启动器启用的是当前练习 Mod，没有选中同名旧副本。
- `hw_learning_ideas.txt` 的括号成对，字段仍是英文，数值是 `research_speed_factor = 0.05`。
- `HW_learning_spirit` 在定义、本地化和测试命令中拼写一致，没有重复定义。
- `hw_learning_l_simp_chinese.yml` 确实为 UTF-8-BOM，语言头为 `l_simp_chinese:`，不是 `.yml.txt`。

这些检查能发现文件问题，但不能证明游戏已经加载、国家已经获得精神或数值已经生效。

## 用一个状态清楚的测试局

保留自己的正式存档，另开本地测试局，只启用当前练习需要的 Mod。使用当前已核对的游戏版本，并记下测试前有哪些国家精神和科研修正。

本系列用 第 2 步中的控制台命令 (`first-content`) 手动给予国家精神，目的是先隔离“定义是否可用”这一步。之后改由国策等正式玩法触发时，必须重新检查触发链，不能因为控制台测试可用就认定国策也接好了。

## 按看到的问题往回查

| 观察到的问题 | 先检查什么 |
| --- | --- |
| 启动器找不到 Mod | 外部 `.mod` 文件、内容目录和实际路径 |
| 测试时找不到 `HW_learning_spirit` | 是否启用当前 Mod；定义文件位置、真实扩展名、括号和唯一 ID |
| 定义存在，但国家没有精神 | 是否实际执行赋予；不要把“文件已保存”当成“国家已获得” |
| 精神名称仍是原始 ID | 简体中文文件名、语言头、编码、键名、引号及当前游戏语言 |
| 精神存在，修正却不对 | `modifier` 层级、修正名、数值与符号 |
| 研究天数没有按预想变化 | 先核对科研速度修正；再对照其他精神、科技年份与超前惩罚，不只比较天数 |

每次只改一类问题。改完文件后重新进行对应检查，并确认读取的是这次启动的结果，而非以前的日志或存档状态。

## 日志怎么读？

从实际游戏用户数据目录中的 `logs/error.log` 找本次启动的信息，优先搜索自己的文件名、`HW_learning_spirit` 或 `research_speed_factor`。日志给出行号时，也检查前面是否缺了引号或右括号；真正的起点可能早于报错行。

如果里面同时有原版或其他 Mod 的信息，先区分来源，再改自己的文件。不要为了清掉所有提示就随意删除不相关内容。

**没有相关错误，不等于结果正确。** 完整检查至少还包括：精神给了哪个国家、名称显示什么、修正方向与数值是什么，以及正式触发有没有执行。

## 记录一个可复查的结果

可以把下面四项写在自己的测试记录中：

1. 游戏版本、启用的 Mod、测试国家与测试前状态。
2. 这次改了哪些文件，使用哪个入口给予国家精神。
3. 期望结果：得到“科研训练”，科研速度提供 +5% 修正；实际看到了什么。
4. 对应的界面截图与本次相关日志片段。分享时移除个人目录、账号等不必要信息。

记录“已核对脚本与原版文档”和“已在游戏中复现结果”时请分开写。本系列目前属于前者，作者尚未完成四步练习的游戏实测。

## 接下来可以做什么？

已经理解定义与赋予的区别，可以做 补给消耗国家精神 (`idea-supply-consumption`)；已有国策树，可以试 一次工业研究加成 (`focus-research-bonus`)。

上一步：中文本地化 (`localisation`)。回到起点：Mod 文件结构 (`first-mod`)。
