---
contentId: character-system
kind: guide
slug: character-system
locale: zh
status: published
title: "做一个人物：领袖、将领、顾问各管什么"
summary: "一个人物是一份档案加一到多个身份块：领袖看子意识形态，将领看技能，顾问看槽位和条件。几种身份写在同一个人身上，不要拆成几个人。"
primaryTopic: people
topics: [people]
scenarios: [history, traits]
gameTerms: [人物, 国家领袖, 将军, 元帅, 海军将领, 顾问, 内阁, 头像, 特质, 科学家, 特工]
aliases: [加一个人物, 怎么做领袖, 加将军, 加顾问, 人物不显示, 顾问不出现, 一人两职, character, advisor]
relatedIds: [advisor-unlocked-by-focus, leader-succession, custom-character-traits, "wiki:effect:recruit_character", "wiki:effect:promote_character", "wiki:effect:retire_country_leader"]
updatedAt: "2026-09-10"
verification: static-checked
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：人物制作"
    url: "https://hoi4.paradoxwikis.com/Character_modding"
  - label: "HOI4 Wiki：特质"
    url: "https://hoi4.paradoxwikis.com/Traits"
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
---

# 做一个人物：领袖、将领、顾问各管什么

## 这个系统负责什么

国家选择屏上的那张脸、内阁里能雇的顾问、战场上能任命的将军，在脚本里都是同一种东西：**人物**。一个人物是一份档案（名字、头像）加上一到多个**身份块**。身份决定这个人出现在游戏的哪个界面、玩家能拿他做什么。

作为作者，你要决定的只有三件事：

1. **这个人是谁**：名字、头像、属于哪个国家。
2. **他有哪些身份**：领袖、将军或元帅、海军将领、顾问、科学家、特工，可以不止一个。
3. **每个身份的参数**：领袖看子意识形态和特质，将领看技能和特质，顾问看槽位、花费和出现条件。

在工作台里，这三件事分别对应「基本信息」、「角色类型」加「兼任顾问身份」、以及各身份自己的设置区。

## 核心概念：一个人可以有几个身份，但只能是一个人物

这是人物系统最容易做错、做错了又没有任何提示的地方。

原版的古德里安既是一名可招募的将军，又是可雇用的军事理论家。他在脚本里**只有一份档案**，档案下面挂着 `corps_commander` 和 `advisor` 两个块。进游戏后，将领名单里有他，理论家候选里也有他，而他是同一个人。

如果你为了"既当将军又当顾问"建了**两个人物**，只是名字相同，那么游戏里会出现**两个人**：一个只在将领名单，一个只在内阁候选，头像各自独立，互不知道对方存在。没有报错，日志里也没有任何提示。

| 游戏里的身份 | 脚本块 | 工作台里怎么选 |
| --- | --- | --- |
| 国家领袖 | `country_leader` | 「角色类型」选国家领袖，再填「子意识形态」 |
| 将军 / 元帅 | `corps_commander` / `field_marshal` | 「角色类型」选将军或元帅，填「陆军数值」 |
| 海军将领 | `navy_leader` | 「角色类型」选海军将领，填「海军数值」 |
| 顾问 | `advisor` | 「角色类型」选顾问；或在别的身份上勾「兼任顾问身份」 |
| 科学家 | `scientist` | 「角色类型」选科学家，填「专精」与等级 |
| 特工 | `operative` | 「角色类型」选特工，填国籍列表 |

工作台按你选的角色类型只列出兼容的特质，不用自己去分哪份特质属于哪种身份。

## 文件结构与运行过程

人物定义在 `common/characters/` 下的文件里，全部包在一个 `characters = { ... }` 块中，一个人一个条目：

```
characters = {
    my_leader = {
        name = my_leader
        portraits = {
            civilian = {
                large = GFX_portrait_my_leader
                small = GFX_portrait_my_leader_small
            }
        }
        country_leader = {
            ideology = conservatism
            traits = { my_leader_trait }
            expire = "1965.1.1.1"
            id = -1
        }
    }
}
```

`name` 后面跟的是本地化键，不是显示文本；显示名写在 `localisation/` 下对应语言的 YML 里。`portraits` 分 `civilian`、`army`、`navy` 三层，政治身份用 `civilian`，将领用 `army`，海军用 `navy`；`small` 是内阁和顾问列表里的小图。

一个人有几个身份就写几个块。下面是"将军兼陆军总长"的写法，两个块挂在同一份档案下：

```
my_general = {
    name = my_general
    portraits = {
        army = {
            large = GFX_portrait_my_general
            small = GFX_idea_my_general
        }
    }
    corps_commander = {
        traits = { infantry_officer }
        skill = 3
        attack_skill = 3
        defense_skill = 2
        planning_skill = 2
        logistics_skill = 2
        legacy_id = -1
    }
    advisor = {
        slot = army_chief
        idea_token = my_general_army_chief
        ledger = army
        allowed = { original_tag = MYT }
        traits = { army_chief_offensive_1 }
        cost = 100
    }
}
```

**定义了不等于在场。** 一个人物要真的属于某个国家，必须在这个国家的历史文件（`history/countries/` 下那份）里被招募：

```
recruit_character = my_leader
recruit_character = my_general
```

这一行只能写在历史文件里。写进 `common/on_actions/` 的开局钩子，引擎会拒绝，并明确告诉你它只允许出现在历史文件中。工作台里新建的人物会在导出时自动补上这一行（「新人物。启动时自动添加 recruit_character」），你不需要自己写；勾了「替换现有 vanilla 人物」的则不会补，因为原版历史文件已经招募过他。

钢之工坊导出顾问块时还会替你补三样原版必有、但表单上没有的东西：`allowed = { original_tag = 你的国家 }`、自动生成的 `idea_token`、以及默认 150 的 `cost`。

## 常见制作目标与对应方法

| 你想在游戏里看到 | 从这里开始 | 关键区别 |
| --- | --- | --- |
| 开局就是我做的领袖 | 「角色类型」国家领袖 + 「子意识形态」与执政党一致 | 同一意识形态已有原版领袖时，开局仍是原版那位，见下 |
| 多一个可招募的将军 | 「角色类型」将军，填技能 | 「可见条件」留空就始终可招募 |
| 多一个可雇的顾问 | 「角色类型」顾问，选「顾问槽位」和「顾问特质」 | 和别的身份一样要在历史文件里招募，工作台会自动补 |
| 一个人既是将军又是总长 | 将军 + 勾「兼任顾问身份」 | **不要**建第二个人物 |
| 完成某个国策后才出现的顾问 | 「visible (顾问显示条件)」写 `has_completed_focus` | 「available」只管能不能雇，不管看不看得见，见做一个完成国策后才出现的政治顾问 (`advisor-unlocked-by-focus`) |
| 剧情里换一个领袖 | 用事件 提拔指定人物 (`wiki:effect:promote_character`) 或 让在任领袖退休 (`wiki:effect:retire_country_leader`) | 预先建好的第二位领袖不会自己上台，见换领袖 (`leader-succession`) |
| 给人物一个原版没有的特质 | 「管理自定义特质」 | 见给人物做一个自定义特质 (`custom-character-traits`) |
| 加一名特工 | 「角色类型」特工，填国籍列表 | 需要 La Résistance；国籍列表决定哪些国家能招募他 |
| 加一名科学家 | 「角色类型」科学家，填专精 | 需要 Götterdämmerung；科学家特质只影响特殊计划 |

## 容易混淆的地方

### 🔴 一人两职是一个人物，不是两个

上一节已经说了机制，这里只说判断方法：进游戏后，在将领名单和内阁候选里各找一次这个名字。如果两边的头像**不是同一张**、解雇一边另一边毫无变化，那就是建成了两个人。修法是删掉一个，在留下的那个上勾「兼任顾问身份」。

### 🔴 领袖必须有子意识形态，否则这个人根本不会以领袖身份出现

`country_leader` 块里的 `ideology` 填的是**子意识形态**（`conservatism`、`liberalism`、`nazism`、`stalinism` 这一级），不是 `democratic`、`fascism` 这样的大类。填了大类或者不填，引擎在加载时报两条：先是"不是有效的意识形态"，接着是"该国家领袖没有对应的政党"。这个人的其他身份不受影响，只是领袖身份作废。

工作台在你没填「子意识形态」时**不会输出领袖块**，等于这个人被导出成了一个没有身份的档案。这不是导出报错，只有进游戏找不到他时才会发现。

### 🔴 头像找不到不报错，直接换成通用脸

`portraits` 里引用了一个不存在的 sprite 时，游戏**不写任何日志**，把这个人显示成一张通用头像。网上流传的"error.log 会报 Could not find portrait"在 1.19.2 上不成立。所以头像必须靠肉眼验收：进游戏后逐个看一遍你做的人物，通用脸就是没找到图。

导出器在你没上传头像时**不写 `portraits` 块**；这种情况下游戏怎么显示，本文没有实测。

### 🔴 顾问的"上任效果"在人物作用域里执行

顾问块的 `on_add` 和 `on_remove` 不是在国家作用域里跑的。直接写 `add_stability = 0.05` 这类国家效果，引擎在加载时就拒绝（"Invalid scope type for effect add_stability"），上任时什么都不发生。原版一百多处上任效果几乎全部用 `owner = { ... }` 跳回国家再写效果。工作台上「on_add (雇用时效果)」的提示文案给的是裸写的例子，**照着填会失效**，请按顾问案例 (`advisor-unlocked-by-focus`)第 3 节的写法包一层。

顾问块里的 `modifier = { ... }` 则是在国家上生效的持续修正，雇用期间一直有效，解雇即消失。它和上任效果是两回事。

### 同一意识形态已有原版领袖时，开局仍是原版那位

给一个原版国家再做一位同意识形态的领袖，开局时执政的**还是原版领袖**，你做的那位只是一个在候选里的人。要让他上台，要么在历史文件里 `promote_character`，要么走事件。完整做法见换领袖：候补人选、事件继位与替换原版领袖 (`leader-succession`)。

### 科学家特质只对特殊计划生效

原版科学家特质文件的第一行注释就写着：科学家特质只作用于特殊计划，因此只能用适用于特殊计划的修正。给科学家特质填 `research_speed_factor` 这类常规科研修正不会生效。工作台的自定义特质面板对科学家特质也有同样的提示。

### 特工的国籍列表决定谁能招募

`operative` 块的 `nationalities` 列出可以招募这名特工的国家；留空时任何国家都能招。这一条来自资料核对，本文没有在游戏里实测特工。

### 版本与适用边界

本文按标记为 1.19.2.0 的原版脚本与游戏随附文档整理。特工需要 La Résistance，科学家需要 Götterdämmerung；没有对应 DLC 时这两种身份不会出现。

## 接下来看什么

- 顾问的四段脚本各管什么，以及那个会被引擎拒绝的写法：做一个完成国策后才出现的政治顾问 (`advisor-unlocked-by-focus`)。
- 候补领袖为什么不上台、事件里怎么换人、怎么替换原版领袖：换领袖：候补人选、事件继位与替换原版领袖 (`leader-succession`)。
- 领袖特质和将领特质的文件形状、没起名会显示成什么：给人物做一个自定义特质 (`custom-character-traits`)。
- 想让顾问上任时发政治点、或让领袖特质每天多给政治点，看《政治点：奖励、门槛与每日获取》（文章 `political-power`）。
