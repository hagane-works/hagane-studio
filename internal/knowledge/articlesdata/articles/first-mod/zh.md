---
contentId: first-mod
kind: tutorial
slug: first-mod
locale: zh
status: published
title: "第 1 步：认识 Mod 的文件结构"
summary: "先建一个独立的小 Mod，分清启动器描述文件、游戏脚本与本地化文件，再开始添加内容。"
primaryTopic: research
topics: [research]
scenarios: [ideas]
gameTerms: [Mod, 国家精神, 科研速度]
aliases: [新手入门, 从零做Mod, Mod目录, descriptor.mod, Mod structure]
relatedIds: [first-content, localisation, check-and-troubleshoot]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "HOI4 Mod 入门第一步：本地 Mod 文件夹、外部 .mod 与内部 descriptor.mod 的区别。用 hw_learning 目录存放一个科研国家精神，脚本放 common/ideas，中文显示文本放 localisation/simp_chinese。"
sources:
  - label: "HOI4 Wiki：Mod 文件结构"
    url: "https://hoi4.paradoxwikis.com/Mod_structure"
  - label: "HOI4 Wiki：国家精神"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---
这四步会做出一个小练习：定义“科研训练”国家精神，为拥有它的国家提供科研速度 +5%，再补上中文名称并检查结果。

这里讲游戏 Mod 文件本身，不要求使用某个编辑器。你也不需要先画图、写事件或创建一棵国策树。

## 先建一个单独的本地 Mod

使用游戏启动器的本地 Mod 创建功能，名称可以填“钢之工坊入门练习”，内容文件夹使用 `hw_learning`。具体按钮位置会随启动器版本变化；重点是确认生成的文件，而不是照着旧截图寻找按钮。

文件放在游戏用户数据目录下的 `mod` 文件夹，不是直接改游戏安装目录。若文档目录被系统重定向，以实际游戏用户数据目录为准。

下图是**完成前三步后应有的结构**，不是脚本：

```text
mod/
├── hw_learning.mod
└── hw_learning/
    ├── descriptor.mod
    ├── common/
    │   └── ideas/
    │       └── hw_learning_ideas.txt
    └── localisation/
        └── simp_chinese/
            └── hw_learning_l_simp_chinese.yml
```

这一步先确认两份描述文件即可，脚本和本地化文件会在后两步加入。

## 两份描述文件各做什么？

外部的 `hw_learning.mod` 告诉启动器“这个 Mod 叫什么，内容在哪里”。下面是**本练习的最小本地描述文件内容**，供核对使用；启动器生成的其他有效字段应保留：

```hoi4
name="钢之工坊入门练习"
path="mod/hw_learning"
supported_version="1.19.*"
```

内容文件夹中的 `descriptor.mod` 保存 Mod 自身的描述信息。下面是**本练习的最小内部描述文件内容**：

```hoi4
name="钢之工坊入门练习"
supported_version="1.19.*"
```

两者都不是放国家精神或国策的地方。`path` 要对应你的实际内容目录；如果启动器为实际位置生成了其他有效路径，不要机械替换成这个示例。

`supported_version` 是声明兼容范围，不会升级游戏，也不证明 Mod 已在该范围的每个版本测试通过。本系列来源核对使用 1.19.2.0。

## 脚本与显示文字分开放

| 文件 | 用途 | 本系列使用的编码 |
| --- | --- | --- |
| `.mod`、`descriptor.mod` | 告诉启动器如何加载 Mod | UTF-8，无 BOM |
| `common/ideas/*.txt` | 定义国家精神与数值 | UTF-8，无 BOM |
| `localisation/simp_chinese/*_l_simp_chinese.yml` | 游戏内简体中文名称与说明 | UTF-8-BOM |

新练习使用独立文件名与 `HW_` 前缀，不复制整份原版国家精神文件。这能减少意外覆盖，但文件和 ID 仍须在自己的 Mod 中保持唯一。

不要在描述文件里加入 `replace_path` 来“强制加载”。这个小练习只新增自己的内容，不需要把原版目录整块替换。

## 这一页完成时检查什么？

- 启动器能够找到新建的本地 Mod，指向的是你的 `hw_learning` 内容目录。
- 文件真实扩展名已显示，没有 `descriptor.mod.txt` 或 `hw_learning.mod.txt`。
- 尚未改动原版文件，也没有为了练习覆盖已有 Mod。

此时游戏里还没有新的国家精神，这是正常的。下一步：写第一份游戏内容 (`first-content`)。

本页是结构与来源核对，不是一次已完成的游戏加载测试记录。
