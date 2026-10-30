---
contentId: custom-autonomy-state
kind: case
slug: custom-autonomy-state
locale: zh
status: published
title: "给附庸做一个自定义自治等级"
summary: "在原版傀儡、自治领之外再加一档自己的附庸等级，然后用国策或事件把某国设成这一档；讲清定义文件的形状、自由度下限怎么定、set_autonomy 的两种拼写。定义照原版 puppet.txt 的骨架改数值最稳。"
primaryTopic: diplomacy
topics: [diplomacy, politics]
scenarios: [focus, event, history]
gameTerms: [附庸, 傀儡, 自治, 自治等级, 自由度, 宗主国, 国策, 完成效果]
aliases: [自定义附庸等级, 自定义傀儡, 新的自治档, 附庸自由度, 设为傀儡, set_autonomy, autonomy_state, autonomous_state, has_autonomy_state]
relatedIds: [diplomacy-system, focus-create-faction, focus-wargoal-annex, "wiki:effect:set_autonomy", "wiki:effect:end_puppet", "wiki:trigger:is_subject_of", "wiki:trigger:is_puppet_of", "wiki:trigger:has_autonomy_state", "wiki:modifier:autonomy_manpower_share"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki：触发条件"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki：修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
  - label: "HOI4 Wiki：自治档制作"
    url: "https://hoi4.paradoxwikis.com/Autonomy_state_modding"
---

# 给附庸做一个自定义自治等级

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到三件事：

1. 国策完成的那一刻，荷兰成为本国的附庸，外交面板上的等级名字是你起的名字，不是「傀儡国」。
2. 附庸界面的自治进度条按点数显示，位置对应你指定的自由度；档的图标是你自己定义的，不是问号。
3. 一条后续国策只在「荷兰处于这一档」时才能点。

## 开始前需要什么

- 一个已经能正常导出的 Mod，以及一条已存在的国策。本文用占位 ID `my_focus_vassalise_holland`。
- 自治档的 ID 和名字，本文用 `my_autonomy_protectorate`。
- 目标国家标签，本文用荷兰 `HOL`，做国是卢森堡。
- 不需要任何 DLC。原版的 21 个自治档里有一半属于各 DLC，但自定义档本身不需要。
- 建议先读外交：开局关系、运行时效果与定义文件各写在哪 (`diplomacy-system`)，知道自治档是「定义」，`set_autonomy` 才是效果。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 自治档定义 | `common/autonomous_states/`，一档一个文件 | 定义等级本身：下限、人力、规则、修正 |
| 国策 | `common/national_focus/` | `set_autonomy` 写在 `completion_reward` 里 |
| 名字 | `localisation/` | 键就是自治档的 `id` |
| 图标 | `interface/*.gfx` | 精灵名固定为 `GFX_<id>_icon`；不定义就显示问号 |

## 1. 定义自治档

新建 `common/autonomous_states/my_protectorate.txt`。骨架照原版 `puppet.txt`，只改数值：

```
autonomy_state = {
    id = my_autonomy_protectorate

    is_puppet = yes

    min_freedom_level = 0.55

    manpower_influence = 0.5

    rule = {
        desc = "RULE_DESC_IS_A_SUBJECT"
        can_not_declare_war = yes
        can_decline_call_to_war = no
    }

    modifier = {
        autonomy_manpower_share = 0.5
        extra_trade_to_overlord_factor = 0.5
    }

    ai_subject_wants_higher = {
        factor = 1.0
    }

    ai_overlord_wants_lower = {
        factor = 1.0
    }

    allowed = {
        always = yes
    }

    can_take_level = {
    }

    can_lose_level = {
    }
}
```

逐字段：

| 字段 | 含义 | 原版傀儡档的值 |
| --- | --- | --- |
| `id` | 档的名字，`set_autonomy` 和 `has_autonomy_state` 都用它 | `autonomy_puppet` |
| `is_puppet` | 是否算傀儡（`is_puppet_of` 条件为真） | `yes` |
| `min_freedom_level` | 自由度下限，0 到 1；附庸的自由度爬过下一档的下限就会升级 | `0.4` |
| `manpower_influence` | 宗主能抽取的人力份额 | `0.9` |
| `rule` | 附庸受的游戏规则，每行一条 | 不能宣战、不能拒绝参战 |
| `modifier` | 附庸身上的持续修正 | 人力上交 0.9 等 |
| `ai_subject_wants_higher` / `ai_overlord_wants_lower` | AI 想升级 / 想压低的权重 | 各 1.0 |
| `allowed` | 这一档在什么条件下可用 | 原版按 DLC 和意识形态限制；自定义档写 `always = yes` |
| `can_take_level` / `can_lose_level` | 升到这档 / 从这档掉下去的额外条件 | 空 |

`default = yes` 只有原版傀儡档写了：和平会议里的「傀儡化」选项和 `puppet` 效果会优先用带这个标记的档。自定义档不要写它，否则会和原版的默认档打架。

不同档的 `min_freedom_level` 不要相同：引擎按下限排序决定升降级顺序，升降一级所需的自治点数也由相邻两档下限之差乘以 5000 算出，两档同值时既排不了序也算不出距离。原版 21 档的值各不相同。

还有一个可选字段 `peace_conference_initial_freedom`：和平会议把一国设成这一档时的初始自由度，不写按 0.5。

## 2. 起名字

`localisation/simp_chinese/` 下：

```
l_simp_chinese:
 my_autonomy_protectorate:0 "保护国"
```

键就是 `id`。原版的键值对如 `autonomy_puppet: "傀儡国"`、`autonomy_dominion: "自治领"`。没有这一行，界面上显示键名本身。

## 3. 给它一个图标

自治档的图标不在定义文件里写，引擎按固定名字去找精灵：`GFX_` 加档的 `id` 加 `_icon`。原版傀儡档对应 `GFX_autonomy_puppet_icon`，定义在 `interface/countrypoliticsview.gfx`，贴图在 `gfx/interface/autonomy/`。自己的档在 `interface/` 下新建一个 `.gfx` 文件：

```
spriteTypes = {
    spriteType = {
        name = "GFX_my_autonomy_protectorate_icon"
        textureFile = "gfx/interface/autonomy/my_autonomy_protectorate_icon.dds"
    }
}
```

不定义这个精灵，档照样生效，只是附庸界面上档名旁边是一个问号。实测就是这样：自定义档名显示正常，图标位置是问号。

## 4. 在国策里设置

```
focus = {
    id = my_focus_vassalise_holland
    icon = GFX_goal_generic_major_alliance
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        set_autonomy = {
            target = HOL
            autonomy_state = my_autonomy_protectorate
            freedom_level = 0.6
        }
    }
}
```

`freedom_level` 是初始自由度，写在下限之上；附庸界面不显示这个小数，显示的是自治进度条的点数（实测 0.6 对应 150/250）。写了低于下限的值会怎样，本文没有实测。游戏自带文档还列了 `end_wars` 和 `end_civil_wars` 两个开关，默认都是 `yes`，即设为附庸时结束它的战争。

### 两种拼写

上面按游戏自带文档写的是 `autonomy_state`，原版国策和事件里 540 处这么写。原版历史文件里写的却是 `autonomous_state`（23 处），钢之工坊的历史档工作台导出的也是这个。两种拼写引擎都接受；在哪一类文件里就照那一类的习惯写，别混。

## 5. 后续国策以「处于这一档」为条件

```
available = {
    HOL = {
        has_autonomy_state = my_autonomy_protectorate
    }
}
```

`has_autonomy_state` 的值是档的 `id`。只判断「是不是我的附庸」用 `HOL = { is_subject_of = ROOT }`；只判断「是不是傀儡」用 `is_puppet_of`，它看的是档定义里的 `is_puppet`。

## 6. 用事件或历史文件代替国策

事件：同一段 `set_autonomy` 搬进选项。历史文件：写在顶层，拼写换成 `autonomous_state`，例如原版比利时历史文件里的殖民地：

```
set_autonomy = {
    target = COG
    autonomous_state = autonomy_colony
    freedom_level = 0.2
}
```

## 7. 逐项核对

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 国策完成后荷兰的外交面板 | 是本国附庸，显示「卢森堡属荷兰」，等级名是「保护国」 | `id` 与本地化键是否一致；`error.log` |
| 档名旁的图标 | 你的图标 | 精灵名是不是 `GFX_<id>_icon` |
| 自治进度条 | 位置对应 0.6，不低于 0.55 | `freedom_level` 与 `min_freedom_level` |
| `is_puppet_of` 类条件 | 为真 | 定义里 `is_puppet = yes` |
| 后续国策 | 荷兰在这档时可点 | `has_autonomy_state` 的值是不是 `id` |
| 升降级 | 自由度爬过邻档下限时切档 | 各档 `min_freedom_level` 是否互不相同 |

## 常见错误

- 🔴 **`target` 写了不存在的国家标签**：实测 `error.log` 记一行 `_set_autonomy_: invalid target tag ---`，效果不执行。荷兰是 `HOL` 不是 `NED`，标签以原版 `common/country_tags/` 为准。
- 🔴 **没给图标**：档生效，界面上是问号。补 `GFX_<id>_icon` 精灵。
- **`set_autonomy` 引用了没定义的档**：先确认文件在 `common/autonomous_states/` 且 `id` 一致；本文没有实测这种情况的日志。
- 🔴 **自定义档写了 `default = yes`**：和原版傀儡档抢默认位。
- **两档 `min_freedom_level` 相同**：升降级顺序不确定。
- **本地化键写成别的名字**：键必须等于 `id`。
- **在历史文件里写 `autonomy_state`、在国策里写 `autonomous_state`**：都能跑，但和原版习惯相反，别人读你的文件会困惑。

## 在钢之工坊中

自治状态工作台直接对应 `common/autonomous_states/`：ID、`is_puppet`、`default`、`min_freedom_level`（工作台以 0 到 1000 的整数存，导出时除以 1000）、`manpower_influence`、规则块、修正字段、AI 权重块都能填。国策里的 `set_autonomy` 在国策工作台的完成效果里原样写；开局就是附庸的关系走历史档工作台。

## 想改成别的效果？

- 想解除附庸：`end_puppet = HOL`，原版事件里 36 处。
- 想直接用原版档：`autonomy_state = autonomy_puppet`、`autonomy_dominion`、`autonomy_colony` 等，21 个档的 `id` 见 `common/autonomous_states/` 各文件。
- 想让附庸出兵、交人力：调 `modifier` 里的 `autonomy_manpower_share` 等修正。
