---
contentId: advisor-unlocked-by-focus
kind: case
slug: advisor-unlocked-by-focus
locale: zh
status: published
title: "做一个完成国策后才出现的政治顾问"
summary: "顾问在国策完成前不在候选里，完成后出现并可雇用；上任瞬间加一次稳定度，任期内每天多给政治点。四段脚本各管一件事，其中一种写法引擎会当场拒绝。"
primaryTopic: people
topics: [people, politics]
scenarios: [focus]
gameTerms: [顾问, 政治顾问, 内阁, 国策, 稳定度, 政治点, 雇用, 解雇]
aliases: [国策解锁顾问, 顾问不显示, 顾问灰掉不能雇, 顾问上任效果, 顾问持续修正, 隐藏顾问, advisor visible, advisor available, on_add]
relatedIds: [character-system, leader-succession, political-power, "wiki:trigger:has_completed_focus", "wiki:effect:add_stability", "wiki:modifier:political_power_gain"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：人物制作"
    url: "https://hoi4.paradoxwikis.com/Character_modding"
  - label: "HOI4 Wiki：触发条件"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki：作用域"
    url: "https://hoi4.paradoxwikis.com/Scopes"
---

# 做一个完成国策后才出现的政治顾问

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到四件事：

1. 国策没完成时，政治顾问候选里**没有**这个人。
2. 国策完成后，他出现在候选里，而且可以雇用。
3. 雇用的那一刻，稳定度上升 5 个百分点；解雇时回落。
4. 他在任期间，政治点每日获取的来源列表里多出一项，来自这个顾问。

第 1、2 条靠两个条件块，第 3 条靠上任效果，第 4 条靠持续修正。它们是四段互不相干的脚本，本文按顺序各做一段。

## 开始前需要什么

- 一个已经能正常导出的 Mod，以及一个已存在的国策，本文用占位 ID `my_focus_unlock_advisor`。
- 不需要任何 DLC。
- 建议先读做一个人物：领袖、将领、顾问各管什么 (`character-system`)，知道一个人物由档案加身份块组成。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 人物定义 | `common/characters/` 下的文件，包在 `characters = { ... }` 里 | 顾问本体，四段脚本都写在它的 `advisor` 块内 |
| 国策 | `common/national_focus/` | 被条件块引用；本文不改它 |
| 招募 | `history/countries/` 下该国的历史文件 | 一行 `recruit_character`，工作台自动补 |
| 名称 | `localisation/` 下对应语言的 YML | 决定候选列表里显示中文还是原始键 |
| 头像 | `interface/` 下的 `.gfx` 与图片 | 缺失时静默显示通用脸 |

## 1. 建一个始终可雇的顾问

先不加任何条件，让他直接出现在候选里。工作台里「角色类型」选顾问，「顾问槽位」选政治顾问，「顾问特质」选一个原版特质，本文用 `silent_workhorse`（每日政治点 +15%）。导出后的形状是：

```
characters = {
    my_advisor = {
        name = my_advisor
        portraits = {
            civilian = {
                large = GFX_portrait_my_advisor
                small = GFX_portrait_my_advisor_small
            }
        }
        advisor = {
            slot = political_advisor
            idea_token = my_advisor
            allowed = { original_tag = MYT }
            traits = { silent_workhorse }
            cost = 150
        }
    }
}
```

`allowed`、`idea_token` 和 `cost` 是工作台替你补的：`allowed` 限定只有你的国家能雇他，`idea_token` 是引擎内部用的唯一名，`cost` 默认 150 政治点。历史文件里的 `recruit_character = my_advisor` 也是自动补的。

做到这里进游戏，政治顾问候选里应该已经有他，随时可雇。

## 2. 两道门：看不看得见，能不能雇

`advisor` 块里有两个条件块，名字相近，管的事不同：

| 块 | 不满足时 | 适合做什么 |
| --- | --- | --- |
| `visible = { ... }` | 顾问**完全不在候选里**，玩家不知道有这个人 | 剧情解锁的隐藏顾问 |
| `available = { ... }` | 顾问**在候选里但灰掉**，点不了 | 玩家看得见、需要先达成条件 |

"完成国策后才出现"要的是 `visible`。在工作台的「visible (顾问显示条件)」里填一行条件，导出器会把它包成块：

```
visible = {
    has_completed_focus = my_focus_unlock_advisor
}
```

两道门可以同时用。比如国策完成后出现，但稳定度不够时灰掉：

```
visible = {
    has_completed_focus = my_focus_unlock_advisor
}
available = {
    has_stability > 0.5
}
```

这两个块在游戏里的表现已经实测：不满足 `visible` 时列表里没有他；满足 `visible` 但不满足 `available` 时他出现在列表里、呈灰色、不能雇；两个都满足时可雇。

## 3. 上任效果与任期修正

这一节是本篇最容易做错的地方。顾问块里有三个和"效果"有关的东西，行为完全不同：

| 块 | 什么时候发生 | 在谁身上执行 |
| --- | --- | --- |
| `on_add = { ... }` | 雇用的那一刻，执行一次 | **人物**作用域 |
| `on_remove = { ... }` | 解雇的那一刻，执行一次 | **人物**作用域 |
| `modifier = { ... }` | 雇用期间持续生效，解雇即消失 | 国家 |

🔴 **上任效果不在国家作用域里跑。** 直接写 `add_stability = 0.05` 这类国家效果，引擎在加载时就拒绝，`error.log` 里是这样一行：

```
Invalid scope type for effect add_stability in common/characters/my_characters.txt line : 32
```

然后雇用时什么都不发生。原版一百八十八处上任效果里**零处**这样裸写，一百二十三处用 `owner = { ... }` 先跳回国家。正确写法：

```
on_add = {
    owner = { add_stability = 0.05 }
}
on_remove = {
    owner = { add_stability = -0.05 }
}
```

⚠️ 工作台「on_add (雇用时效果)」的提示文案里给的例子是裸写的 `add_stability`，**照着填会失效**。在那个输入框里写上面这种带 `owner` 的形式即可。

持续修正直接写在国家上，不需要跳作用域：

```
modifier = {
    political_power_gain = 0.25
}
```

`political_power_gain = 0.25` 是每日固定项 +0.25 点，不是 +25%。雇用后悬停政治点，来源列表里会多出这个顾问一项；解雇后消失。这个写法原版极少用（四千多个顾问里只有一个），原版习惯把持续加成放在特质里，因为特质可以复用到多个人身上。两种都生效，选一种就行。

## 4. 补上名称

人物的本地化键就是 `name` 后面那个键：

```
l_simp_chinese:
 my_advisor: "改革派财政专家"
```

覆盖已有键时文件要放进 `localisation/<语言>/replace/`，否则会被原版或前置 Mod 的同名键压掉，而且完全没有提示。本例是新增键，不涉及覆盖。

头像找不到时游戏**不报错**，直接显示通用脸；进游戏后肉眼看一眼候选列表里的小图。

## 5. 逐项核对

开一局独立测试存档，按顺序确认，别跳步。国策可以用控制台完成，也可以临时把条件改成一个国家标记，用一个事件设置它。

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 国策未完成时打开政治顾问候选 | 没有这个人 | `visible` 块是否真的写进了 `advisor` 块内 |
| 国策完成后再看 | 出现，可雇 | 条件里的国策 ID 拼写 |
| 雇用瞬间悬停稳定度 | +5 个百分点 | `on_add` 里有没有 `owner = { ... }`；看 `error.log` 有没有作用域错误 |
| 雇用后悬停政治点每日获取 | 来源列表多一项，+0.25 | `modifier` 块的键名 |
| 解雇 | 稳定度回落，政治点来源消失 | `on_remove` 是否对称 |
| 候选列表里的小图 | 是你的图，不是通用脸 | 图片路径与 sprite 名；这一项日志不会提示 |

## 常见错误

- **上任效果裸写国家效果**：引擎拒绝并在 `error.log` 报作用域错误，上任时无事发生。包一层 `owner = { ... }`。
- **想要"看得见但灰掉"却写了 `visible`**：玩家根本找不到这个顾问。要灰掉用 `available`。
- **把持续加成写进 `on_add`**：那是一次性执行，不会持续。持续的放 `modifier` 或特质。
- **特质 ID 拼错**：工作台在导出前会把既不在原版表、也不在你自定义特质里的 ID 丢掉并给出提示，所以这个错不会进游戏，但顾问会少一个效果。
- **想做不可解雇的顾问填 `removal_cost = -1`**：工作台把 -1 当作未设置、不输出这个键；资料里也记载这个写法在 1.12.8 之后失效。原版用 `can_be_fired = no`。这一条本文没有实测。

## 想改成别的效果？

- 想让他是陆军总长、理论家或军事高层：改「顾问槽位」，这几类槽位原版都带一行 `ledger = army`（或 `navy`、`air`），工作台会一起输出。
- 想让一个将军兼任这个顾问：不要建第二个人，在将军上勾「兼任顾问身份」，见做一个人物 (`character-system`)。
- 想让上任时弹一个事件：在 `on_add` 里写 `owner = { country_event = my_events.1 }`，事件本身另写。
- 想让解锁条件是一个事件设的标记而不是国策：把条件换成 `has_country_flag = my_flag`，在事件选项里 `set_country_flag = my_flag`。
