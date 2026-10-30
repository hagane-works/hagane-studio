---
contentId: focus-transfer-state-core
kind: case
slug: focus-transfer-state-core
locale: zh
status: published
title: "国策完成后转交一个州并加核心"
summary: "国策完成时把一个州交给本国并加上核心；顺带讲清只占领不拥有、加宣称、加顺从、改州名各用哪个效果。六个效果里有三个写在国家层、三个要先进到州里。"
primaryTopic: territory
topics: [territory]
scenarios: [focus, event]
gameTerms: [州, 转交, 核心, 宣称, 占领, 控制者, 顺从度, 国策, 完成效果]
aliases: [国策转交州, 国策加核心, 事件转交州, 占领不拥有, 加宣称, 改州名, transfer_state, add_state_core, set_state_controller]
relatedIds: [state-file-basics, "wiki:effect:transfer_state", "wiki:effect:add_state_core", "wiki:effect:set_state_controller", "wiki:effect:add_claim_by", "wiki:effect:add_compliance", "wiki:effect:set_state_name", "wiki:trigger:owns_state"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki：作用域"
    url: "https://hoi4.paradoxwikis.com/Scopes"
  - label: "HOI4 Wiki：国策制作"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
---

# 国策完成后转交一个州并加核心

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到三件事：

1. 国策完成的那一刻，目标州的颜色变成本国，州面板的所有者是本国。
2. 州面板上出现本国的核心标记，它不再是非核心领土。
3. 另一个国家对这个州出现宣称标记。

再往下还有三个常见变体：只占领不拥有、给占领区加顺从度、在剧情里给州改名。本文把六个效果放在一起讲，因为它们的写法分两种，混着写会失效。

## 开始前需要什么

- 一个已经能正常导出的 Mod，以及一个已存在的国策。本文用占位 ID `my_focus_take_flanders`。
- 目标州的**编号**。在工作台的「地块」列表里选中它，右侧只读区的「地块 id」就是；本文用原版的佛兰德斯，编号 6，荷兰是编号 7。
- 不需要任何 DLC。
- 建议先读州文件：归属、核心、省份和建筑各写在哪 (`state-file-basics`)，知道所有者、控制者、核心是三个不同的字段。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 国策 | `common/national_focus/` | 效果写在它的 `completion_reward` 里 |
| 州 | `history/states/` | 被改的对象；本文不改它的文件，运行时改 |
| 效果 | 国策奖励、事件选项、决议效果里都能写 | 六个效果，两种写法 |

## 1. 分清两种写法：写在国家层，还是先进到州里

六个效果里，三个直接写在国家层，州编号作为值；另外三个要先用州编号打开一个块，效果写在块里：

| 你想做的 | 写法 | 效果作用在谁身上 |
| --- | --- | --- |
| 转交州（所有者和控制者一起变） | `transfer_state = 6` | 国家层，值是州编号 |
| 加本国核心 | `add_state_core = 6` | 国家层 |
| 只占领不拥有 | `set_state_controller = 7` | 国家层 |
| 加某国宣称 | `6 = { add_claim_by = GER }` | **州层**，先进到州 6 里 |
| 加顺从度 | `7 = { add_compliance = 50 }` | 州层 |
| 改州名 | `7 = { set_state_name = "新名字" }` | 州层 |

游戏随附文档里每个效果都标了适用作用域：前三个是 COUNTRY，后三个是 STATE。把 `add_claim_by = GER` 直接写在国策奖励里，它拿不到州，什么都不会发生。

## 2. 在国策完成效果里写

工作台的国策编辑页有一个「completion_reward (完成效果)」输入框，把效果原样写进去，导出器会包进 `completion_reward` 块。导出后的国策节点是这样：

```
focus = {
    id = my_focus_take_flanders
    icon = GFX_goal_generic_territory_or_war
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        transfer_state = 6
        add_state_core = 6
        6 = { add_claim_by = GER }
    }
}
```

三行的顺序按你想要的结果排：先转交再加核心，最后加别国宣称。原版国策里常见把 `transfer_state` 包在 `hidden_effect` 或 `ROOT = { ... }` 里，那是为了控制提示文字或在别的作用域里执行，本例不需要。

## 3. 只占领不拥有，然后加顺从

`set_state_controller` 只改控制者，所有者不动。州面板会显示"某国拥有，某国控制"，这就是占领区的状态。占领区有顺从度和抵抗度，用州层效果调：

```
completion_reward = {
    set_state_controller = 7
    7 = { add_compliance = 50 }
}
```

实测：执行后州 7 的控制者变成本国，所有者仍是荷兰，顺从度上升 50。

## 4. 剧情里改州名

`set_state_name` 直接写显示文本，不是本地化键，改的是当前运行中的名字，不改文件：

```
7 = { set_state_name = "新占领区" }
```

这个改法适合剧情用。开局就要改名的，走州文件的本地化键，见指南。

## 5. 用事件代替国策

同样六个效果原样搬进事件选项即可：

```
add_namespace = my_events

country_event = {
    id = my_events.1
    title = my_events.1.t
    desc = my_events.1.d
    is_triggered_only = yes
    option = {
        name = my_events.1.a
        transfer_state = 6
        add_state_core = 6
    }
}
```

测试时用控制台 `event my_events.1` 直接触发，比等国策完成快。

## 6. 逐项核对

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 国策完成后州 6 的颜色 | 变成本国 | 州编号是不是写成了省份编号 |
| 州 6 面板 | 所有者本国，有本国核心 | `add_state_core` 是否和 `transfer_state` 一起写了 |
| 州 6 面板 | 有德国宣称 | `add_claim_by` 是否包在 `6 = { ... }` 里 |
| 州 7 面板 | 所有者荷兰、控制者本国 | 用的是 `set_state_controller` 不是 `transfer_state` |
| 州 7 顺从度 | 比之前高 50 | `add_compliance` 是否在州层 |
| 州 7 名字 | 变成你写的文本 | `set_state_name` 是否在州层 |

## 常见错误

- **把州层效果写在国家层**：`add_claim_by`、`add_compliance`、`set_state_name` 直接写在奖励里不起作用。先用州编号打开块。
- **把省份编号当州编号**：两套编号互不相通。州编号在工作台「地块 id」里看，省份编号在「省份」列表里看。
- **只转交不加核心**：拿到的是非核心领土，人力和工厂按非核心规则打折。想当成本土用，两行都写。
- **想要占领区却用了转交**：转交把所有者一起改了，占领区的顺从和抵抗机制就不存在了。

## 想改成别的效果？

- 想拿走别国的核心：移除核心 (`wiki:effect:remove_state_core`)，写法和加核心一样在国家层。
- 想让国策只在拥有某州时才能点：在 `available` 里用检查是否拥有某州 (`wiki:trigger:owns_state`)。
- 想一次转交一片州：一行一个州编号，`transfer_state` 不接受列表。
