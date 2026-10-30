---
contentId: localisation
kind: tutorial
slug: localisation
locale: zh
status: published
title: "第 3 步：给内容加上中文名称"
summary: "把脚本 ID 变成游戏里的名称和说明，检查语言文件名、语言头与 UTF-8-BOM 编码。"
primaryTopic: research
topics: [research]
scenarios: [ideas, focus]
gameTerms: [国家精神名称, 国家精神说明, 本地化]
aliases: [中文不显示, 显示原始ID, localisation, l_simp_chinese, UTF-8-BOM]
relatedIds: [first-mod, first-content, check-and-troubleshoot, focus-research-bonus]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "HOI4 Mod 入门第三步：hw_learning_l_simp_chinese.yml 使用 l_simp_chinese 语言头与 UTF-8-BOM，HW_learning_spirit 对应名称，HW_learning_spirit_desc 对应说明。显示原始 ID、文件后缀、英文引号与换行检查。"
sources:
  - label: "HOI4 Wiki：本地化"
    url: "https://hoi4.paradoxwikis.com/Localisation"
  - label: "HOI4 Wiki：国家精神显示文本"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---
上一页定义了 `HW_learning_spirit`。这个 ID 用于脚本引用，不适合直接显示给玩家。现在给它补上名称“科研训练”和一段说明。

## 1. 创建简体中文文件

在 Mod 中创建 `localisation/simp_chinese/hw_learning_l_simp_chinese.yml`。下面是**完整文件内容**：

```yaml
l_simp_chinese:
 HW_learning_spirit: "科研训练"
 HW_learning_spirit_desc: "改进研究工作的组织方式，提高科研速度。"
```

保存为 **UTF-8-BOM**。代码块只提供文字，复制代码并不会自动替你选择文件编码。

## 2. 三处语言标记保持一致

| 位置 | 本例写法 |
| --- | --- |
| 文件目录 | `localisation/simp_chinese/` |
| 文件名 | `hw_learning_l_simp_chinese.yml` |
| 第一行语言头 | `l_simp_chinese:` |

`l_` 的第一个字符是小写字母 L，不是数字 1。扩展名必须是 `.yml`，不是 `.yaml`，也不是被系统隐藏了末尾的 `.yml.txt`。

用简体中文游戏界面检查这份文件。切换 Hagane Works 的界面语言，不会自动生成其他语言的 Mod 文本，也不会改变游戏使用哪份本地化。

## 3. 键和值各自负责什么？

冒号左边是脚本要查找的键：`HW_learning_spirit` 对应名称，`HW_learning_spirit_desc` 对应说明。它们要与前一页的国家精神 ID 保持一致。

冒号右边才是给玩家看的文字。可以改“科研训练”这个名字，不必同时改脚本 ID。

注意这几项：

- 使用英文半角冒号和直双引号 `"`，不要换成中文引号。
- 中文写在值中，不要放进键名。
- 一个值保持在同一物理行内。需要游戏内换行时写 `\n`，不要直接把引号中的文字敲成两行。
- 原版中常见的 `键:0 "文字"` 里的数字是可省略的版本标记。本教程统一省略，不要把它误当成中文显示失败的原因。

下面是**替换现有说明行的局部示例，不是另加一个同名键**：

```yaml
 HW_learning_spirit_desc: "改进研究工作的组织方式。\n提高科研速度。"
```

## 4. 新增内容不需要覆盖整份原版文本

本练习使用自己的 `HW_learning_spirit` 键，普通的本地化文件即可。不要为了显示它就复制整份原版文件，也不要在多处重复定义同一个键。

如果以后要专门改原版已有名称，可以了解本地化 `replace/` 目录的优先规则。它与描述文件中的 `replace_path` 不是同一个功能，本练习不需要它们。

## 5. 回到同一个小例子检查

在测试国家已获得该精神的前提下，确认显示名称是“科研训练”，说明是你写的中文；同时保留上一页的科研速度 +5% 数值检查。名称正确不能证明修正正确，数值正确也不代表名称文件已加载。

如果仍显示原始 ID，按“真实扩展名 → 文件名中的语言 → 第一行 → 编码 → 键名与引号”的顺序检查，并查看本次日志中的相关位置。

本页已按本地指南和 1.19.2.0 原版简体中文文件核对规则，尚未执行这个练习的游戏实测。

上一步：定义国家精神 (`first-content`)。下一步：检查与排错 (`check-and-troubleshoot`)。
