---
contentId: dynamic-spirit-from-variables
kind: case
slug: dynamic-spirit-from-variables
locale: zh
status: published
title: "让国家精神随局势实时变化：动态修正、变量和重算钩子"
summary: "「每丢一块核心州就多 3% 攻击」这种数值会动的精神，靠 idea 做不出来，因为 idea 的修正值不能是变量。走得通的是三件套：动态修正把修正值绑到变量上，一段脚本效果负责重新数一遍，一条 on_actions 钩子决定多久数一次。先给一个五个文件照抄就能跑的最小版本，再讲怎么加规则。"
primaryTopic: politics
topics: [politics, military]
scenarios: [ideas, event]
gameTerms: [国家精神, 动态修正, 变量, 自定义修正说明, 脚本效果, 脚本触发器, on_actions, 每日钩子, 军事顾问, 核心州]
aliases: [实时变化的国家精神, 数值会变的精神, 动态国家精神, 随局势变化的修正, dynamic_modifier, custom_modifier_tooltip, 修正绑变量, 每有一块就加, 每接受一个就加, 精神里放变量, force_update_dynamic_modifier]
relatedIds: [political-power, stability, idea-supply-consumption, event-stability]
updatedAt: "2026-09-11"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：修正（动态修正一节）"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
  - label: "HOI4 Wiki：变量"
    url: "https://hoi4.paradoxwikis.com/Variables"
  - label: "HOI4 Wiki：本地化（变量格式化一节）"
    url: "https://hoi4.paradoxwikis.com/Localisation"
  - label: "HOI4 Wiki：On actions"
    url: "https://hoi4.paradoxwikis.com/On_actions"
---

# 让国家精神随局势实时变化：动态修正、变量和重算钩子

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到三件事：

1. 国家精神栏里多出一个精神，鼠标悬停时能看到引擎自动算出的总数，例如「陆军师攻击：+12.00%」。总数是 0 的时候引擎不显示这一行。
2. 总数下面是你自己写的说明行，一行讲一条规则，例如「每有一块核心州被外国控制：陆军师攻击 +3.00%（当前 4 块）」，括号里的数字跟着局势走。
3. 一块核心州易手之后，当天 24:00 一过数字就变了（每日钩子在每天午夜跑一次）。

失地越多越拼命、外援越多越被动、某类国家越多越紧张，这些精神形状都一样，本文用「失地」做主例。

## 开始前需要什么

**先说这条路线和工作台的关系。** 工作台的「动态修正」页能建修正、填修正值，非数字的值会原样导出，所以「修正值绑变量」这一步可以在工作台里做。但它没有「自定义修正说明」的输入口，重算用的脚本效果和每日钩子也要分别在「脚本片段」和事件里拼。本文讲的是纯脚本路线，文件全部自己写；工作台用户可以对照着把能填的部分填进去，剩下的走原始文件。

- 有一个能进游戏的 Mod 骨架。没有的话先读第一个 Mod (`first-mod`)。
- 会用 `-debug` 参数启动游戏并读 `error.log`。Steam 库里右键游戏，属性，启动选项里填 `-debug`；日志在 `文档\Paradox Interactive\Hearts of Iron IV\logs\error.log`。游戏里按 `~` 键开控制台。
- 脚本文件（`.txt`）不能带 BOM，本地化文件（`.yml`）必须带 BOM。VS Code 里点右下角的编码，另存为「UTF-8 with BOM」就是带 BOM。写错整份文件作废，这一条和别的域一样。

## 涉及的对象与关系

| 对象 | 文件（本文示例） | 在本例中的作用 |
| --- | --- | --- |
| 动态修正 | `common/dynamic_modifiers/my_thorn.txt` | 精神本体；修正值写变量名，引擎每天代入 |
| 脚本效果 | `common/scripted_effects/my_thorn_effects.txt` | 重新数一遍，把结果写进变量 |
| on_actions | `common/on_actions/my_thorn_on_actions.txt` | 决定多久重算一次 |
| 国家历史 | `history/countries/MYT - my_thorn.txt` | 开局初始化变量并挂上修正 |
| 本地化 | `localisation/simp_chinese/my_thorn_l_simp_chinese.yml` | 精神名和那几行说明文字 |
| 脚本触发器（进阶） | `common/scripted_triggers/my_thorn_triggers.txt` | 定义「谁算本国势力、谁算外国」 |

文件名随便起，目录不能错。所有 `my_` 开头的名字和 `MYT` 都是占位，换成你自己的。

## 1. 为什么不能用 idea

idea 的修正块只认数字。你在 `army_attack_factor` 后面写一个变量名，引擎不会去查变量，这条修正就是无效的，而且不报错。Wiki 的变量页写得很直接：变量不能直接用在 idea 的修正里，要用动态修正。

动态修正和 idea 在界面上长得一样：有图标、出现在国家精神栏、悬停显示修正列表。区别在两点：修正值可以是变量名；它每天重算一次，重算时先把所有绑变量的修正值清零，再按挂上的顺序逐个代入。

所以「实时变化」的精神，全部是动态修正。

## 2. 最小版本：五个文件，照抄就能跑

这一版只有一条规则：每有一块本国核心州不在自己控制下，陆军师攻击 +3%。本文全程用失地做例子；换成别的规则只改两处，见第七节。

### 2.1 修正本体

`common/dynamic_modifiers/my_thorn.txt`：

```
my_thorn_in_throat = {
    icon = GFX_idea_unknown
    enable = { always = yes }
    army_attack_factor = my_thorn_attack
    custom_modifier_tooltip = my_thorn_in_throat_tt
}
```

四行各管一件事：

- `icon` 是精神栏里的图标。`GFX_idea_unknown` 是原版自带的占位图，先用它跑通，想换自己的图再去注册 sprite。
- `enable` 是修正生效的条件。`always = yes` 表示只要挂着就生效；也可以写 `has_war = yes` 之类，条件不满足时数值不生效但精神还挂着。
- `army_attack_factor = my_thorn_attack` 是核心：等号右边不是数字，是一个国家变量的名字，引擎每天读它。
- `custom_modifier_tooltip` 指向一条本地化键。**说明行不会自己长出来**，引擎只自动渲染「陆军师攻击：+12.00%」这一行总数；下面每条「每有一块……+3.00%」都是你在这条键里手写的。

### 2.2 重算

`common/scripted_effects/my_thorn_effects.txt`：

```
my_thorn_recalc = {
    set_variable = { my_thorn_states = 0 }
    every_state = {
        limit = {
            is_core_of = ROOT
            NOT = { is_controlled_by = ROOT }
        }
        ROOT = { add_to_variable = { my_thorn_states = 1 } }
    }
    set_variable = { my_thorn_attack = my_thorn_states }
    multiply_variable = { my_thorn_attack = 0.03 }
    clamp_variable = { var = my_thorn_attack max = 0.5 }
}
```

读法：先把计数清零；遍历全图的州，凡是「是我的核心，且控制者不是我」的，给我自己的计数加一；最后把计数乘 0.03 得到修正值，封顶 50%。

两个要点：

- `ROOT` 是调用这段效果的国家，也就是你。`every_state` 里面作用域已经变成州，加计数前必须用 `ROOT = { }` 切回国家，否则计数加到州身上，你的变量永远是 0。
- 折算写在最后。修正值是「计数 × 0.03」，不要在每次加计数时直接加 0.03，那样封顶和显示都不好做。

### 2.3 钩子

`common/on_actions/my_thorn_on_actions.txt`：

```
on_actions = {
    on_daily_MYT = {
        effect = {
            my_thorn_recalc = yes
        }
    }
}
```

`on_daily_<TAG>` 只对指定国家每天跑一次，默认作用域就是这个国家，所以效果里直接调用。**`MYT` 换成你的国家 TAG，钩子名本身也要换**，写成 `on_daily_MYT` 而 TAG 不是 MYT，这条钩子永远不跑。

全局的 `on_daily` 对每个国家都跑，Wiki 明确标注它性能开销大，别用。

### 2.4 初始化并挂上

在 `history/countries/` 下**新建**一个文件，文件名前三个字母必须是你的国家 TAG，比如 `MYT - my_thorn.txt`，内容只有两行：

```
set_variable = { my_thorn_attack = 0 }
add_dynamic_modifier = { modifier = my_thorn_in_throat }
```

不要去复制或改原版那个国家的历史文件。同一个 TAG 可以有多个历史文件，引擎会按顺序全部执行，所以你的小文件和原版的开局设定并存。不写 `days` 就一直挂着，直到你用 `remove_dynamic_modifier` 摘掉，或者修正本体里的 `remove_trigger` 满足。这两行也可以放进国策奖励或事件选项里，让精神在剧情里出现。

**先 `set_variable` 再挂。** 否则第一天修正值是空的，而且 `has_variable` 为假，任何依赖这个变量存在的判断都会走错分支。

### 2.5 文案

`localisation/simp_chinese/my_thorn_l_simp_chinese.yml`，带 BOM。文件名必须以 `_l_simp_chinese.yml` 结尾，`l` 是小写字母 L，否则引擎不认这是中文文件：

```yaml
l_simp_chinese:
 my_thorn_in_throat:0 "如鲠在喉"
 my_thorn_in_throat_tt:0 "每有一块核心州被外国控制：陆军师攻击 §G+3.00%§!（当前 [?my_thorn_states|Y0] 块）"
```

`[?my_thorn_states|Y0]` 是变量在文案里的写法：问号必须有，竖线后面是格式，`Y0` 表示黄色、不带小数。第五节有格式表。

到这里五个文件齐了，装进游戏就能看到精神和数字。注意第一次重算发生在开局后的第一个午夜：进游戏当天悬停，括号里是 0，总数行也不显示，过了 24:00 再看。

## 3. 加第二条规则：其它本国势力接受的外国军事顾问

上面的最小版本把「被别人控制」一律算作失地，包括被同一个国家分裂出来的其它势力控制的州。如果你的国家有好几个势力，想区分「本国势力」和「外国」，就要自己定义这条边界，引擎没有这个概念。

先写一个脚本触发器，`common/scripted_triggers/my_thorn_triggers.txt`，按 `original_tag` 把同一个国家分裂出来的各个势力列出来：

```
my_is_domestic_faction = {
    OR = {
        original_tag = MYT
        original_tag = MYA
        original_tag = MYB
    }
}
```

然后把第 2.2 节的效果扩成两条规则：核心州的控制者不是本国势力才算失地；其它本国势力每接受一名外国军事顾问也算一项。

```
my_thorn_recalc = {
    set_variable = { my_thorn_states = 0 }
    set_variable = { my_thorn_attaches = 0 }

    # 规则一：本国核心州里，控制者不是本国势力的
    every_state = {
        limit = {
            is_core_of = ROOT
            NOT = { CONTROLLER = { my_is_domestic_faction = yes } }
        }
        ROOT = { add_to_variable = { my_thorn_states = 1 } }
    }

    # 规则二：其它本国势力每接受一名外国军事顾问
    every_other_country = {
        limit = { my_is_domestic_faction = yes }
        every_other_country = {
            limit = {
                NOT = { my_is_domestic_faction = yes }
                PREV = { has_attache_from = PREV }
            }
            ROOT = { add_to_variable = { my_thorn_attaches = 1 } }
        }
    }

    set_variable = { my_thorn_attack = my_thorn_states }
    add_to_variable = { my_thorn_attack = my_thorn_attaches }
    multiply_variable = { my_thorn_attack = 0.03 }
    clamp_variable = { var = my_thorn_attack max = 0.5 }
}
```

规则二是两层循环，作用域要盯紧：外层遍历「其它本国势力」，内层遍历「外国」。内层里 `PREV` 是外层那个本国势力，`PREV = { has_attache_from = PREV }` 读作「外层那个势力，有来自当前这个外国的军事顾问」。加计数照样要回到 `ROOT`。游戏里外交面板上的「派遣军事顾问」，脚本里就是 attaché。

文案里对应加一行：

```yaml
l_simp_chinese:
 my_thorn_in_throat_tt:0 "每有一块核心州被外国控制：陆军师攻击 §G+3.00%§!（当前 [?my_thorn_states|Y0] 块）\n其它本国势力每接受一名外国军事顾问：陆军师攻击 §G+3.00%§!（当前 [?my_thorn_attaches|Y0] 名）"
```

志愿军和租借法案的计数同形，把内层那一条判断换掉即可：志愿军用 `has_volunteers_amount_from`（带 `tag` 和 `count`），租借用 `is_lend_leasing`。这两个触发器在游戏自带的触发器文档里有，本文没有实测它们在嵌套作用域下的写法。

## 4. 让数字当天就变

每日钩子的意思是「次日生效」。想让数字在事情发生的当天就变，在对应的事件钩子里再调一次重算，并且强制刷新修正。州易手对应 `on_state_control_changed`：

```
on_actions = {
    on_state_control_changed = {
        effect = {
            if = {
                limit = { FROM.FROM = { is_core_of = MYT } }
                MYT = {
                    my_thorn_recalc = yes
                    force_update_dynamic_modifier = yes
                }
            }
        }
    }
}
```

这条钩子里 `ROOT` 是新控制者、`FROM` 是旧控制者、`FROM.FROM` 是那块州，所以先判断州是不是你的核心，再切回你自己的作用域重算。这一段可以直接接在第 2.3 节那个文件的 `on_actions = { }` 里面。

顾问、志愿军、租借各有对应的钩子（`on_send_attache`、`on_send_volunteers`、`on_lend_lease`），Wiki 的 on_actions 页有它们的作用域说明；原版脚本没有用到这几条钩子，本文也没有实测，用之前自己进游戏验一次。

## 5. 文案里的变量格式

`[?变量名|格式]` 的竖线后面可以叠多个格式字符，常用的几个：

| 字符 | 作用 |
| --- | --- |
| `0` 到 `5` | 小数位数 |
| `%` | 乘以 100 再加百分号 |
| `+` | 正数绿、零黄、负数红 |
| `=` | 正数前面加加号 |
| `Y`、`G`、`R` | 固定黄、绿、红 |

想让说明行直接显示折算后的百分比，就写 `[?my_thorn_attack|%+0]`。

这条键之所以能用变量，是因为 1.13 起 idea 和动态修正的文案支持动态本地化。原版也是这么用的：中国的美援修正在中文里就是「[?变量|Y0]经济产能每月盈余」这种形状。

## 6. 装机核对

| 检查 | 期望 |
| --- | --- |
| 五个文件都在，`.txt` 无 BOM，`.yml` 有 BOM | 通过 |
| 精神栏里出现图标，悬停第一行是自动算出的总数 | 肉眼 |
| 过了第一个午夜，说明行括号里的数字对得上失地数 | 肉眼 |
| 让一块核心州易手，当天午夜一过总数上升 | 肉眼 |
| `error.log` 里搜修正名和变量名，零命中 | 通过 |
| 悬停说明行，`[?` 没有原样显示 | 肉眼 |

**让州易手最省事的做法**是写一个测试事件，用控制台发。`events/my_thorn_test.txt`：

```
add_namespace = my_thorn_test

country_event = {
    id = my_thorn_test.1
    title = my_thorn_test.1.t
    desc = my_thorn_test.1.d
    is_triggered_only = yes
    immediate = {
        MYA = { set_state_controller = 123 }
    }
    option = {
        name = my_thorn_test.1.a
    }
}
```

`123` 换成你一块核心州的编号，`MYA` 换成任意邻国，不需要处于战争状态。注意 `set_state_controller` 是国家作用域的效果：先切到接收方国家，再写州编号；反过来写成 `123 = { set_state_controller = MYA }` 引擎会报 `Invalid Scope` 并且什么都不做。进游戏后控制台输入 `event my_thorn_test.1`，让时间走过当天午夜，悬停精神看数字。测完把这个文件删掉。

最后一行核对是最常见的失败形状：`[?my_thorn_states|Y0]` 原样出现在界面上，说明这条文案用在了不支持动态本地化的位置，或者问号漏了。

## 7. 换成你自己的规则

本文用失地做例子。换规则只改两处：重算效果里的循环和判定（数什么），修正本体里的修正项（加到哪个数值上，`army_attack_factor` 换成任何修正项都行）。钩子、初始化、文案格式、封顶一个字不用动；说明行是手写的，跟着规则改。

### 7.1 引擎已经有现成变量的，不用循环

游戏自带的 `documentation/dynamic_variables_documentation.md` 列了国家作用域下几十个内建变量，常用的：

| 想数的 | 内建变量 |
| --- | --- |
| 傀儡数 | `num_subjects` |
| 阵营成员数 | `num_faction_members` |
| 拥有的州、控制的州、核心州 | `num_owned_states`、`num_of_controlled_states`、`num_core_states` |
| 民用工厂、军用工厂 | `num_of_civilian_factories`、`num_of_military_factories` |
| 累计伤亡（千） | `casualties_k` |
| 稳定度、投降进度、敌我兵力比 | `stability`、`surrender_progress`、`enemies_strength_ratio` |

这时重算效果缩成两行，先把内建变量复制进自己的变量，再折算：

```
my_thorn_recalc = {
    set_variable = { my_thorn_attack = num_subjects }
    multiply_variable = { my_thorn_attack = 0.03 }
}
```

内建变量能不能直接写在修正本体那一行（`army_attack_factor = num_subjects`）本文没有验过，Wiki 的例子绑的都是自定义变量，所以统一走「先复制进自己的变量」这条稳的路。

### 7.2 要自己判定的，循环形状固定

引擎没有现成变量、但能用一个触发器判定的，就写循环。形状永远是：选循环对象，写判定，`ROOT = { add_to_variable }`。

| 想要的效果 | 循环 | 判定 |
| --- | --- | --- |
| 每有一个邻国在打仗 | `every_neighbor_country` | `has_war = yes` |
| 每有一个法西斯国家 | `every_other_country` | `has_government = fascism` |
| 每有一块本国州抵抗度超过 20 | `every_owned_state` | `resistance > 20` |
| 每有一块核心州被别人控制（本文主例） | `every_state` | `is_core_of = ROOT` 加 `NOT = { is_controlled_by = ROOT }` |

### 7.3 折算方式不是「每有一个就加」的

阶梯、按比例、双向，都在最后折算那几行里改，循环不动。

按比例，失地占核心州的比例，每 10% 扣 5% 攻击：

```
set_variable = { my_thorn_attack = my_thorn_states }
divide_variable = { my_thorn_attack = num_core_states }
multiply_variable = { my_thorn_attack = -0.5 }
```

阶梯，超过 5 块才开始算：

```
subtract_from_variable = { my_thorn_states = 5 }
clamp_variable = { var = my_thorn_states min = 0 }
set_variable = { my_thorn_attack = my_thorn_states }
multiply_variable = { my_thorn_attack = 0.03 }
```

## 8. 让变化显示在国策和事件里

第二节的精神是「每天重新数一遍」的，变量归重算效果管，国策和事件不要直接改它，改了当天午夜也会被算回去。但还有一类动态修正是「攒出来的」：没有重算，每完成一个国策就往变量上加一点，原版阿富汗的骑兵修正就是这种写法。这时玩家在国策提示里什么都看不到，因为改变量本身不产生任何提示。

解法是给 `add_to_variable` 带一个 `tooltip` 参数，指向一条文案：

```
completion_reward = {
    add_to_variable = { my_cavalry_attack = 0.05 tooltip = my_cavalry_attack_tt }
}
```

```yaml
l_simp_chinese:
 my_cavalry_attack_tt:0 "$MODIFIER_CAVALRY_ATTACK_FACTOR$：$RIGHT|+=0%$"
```

这条文案里有两个固定记号：`$RIGHT$` 是这次加上去的数，`$LEFT$` 是加之前的值。竖线后面的格式字符和第五节一样：`+` 正绿负红，`=` 带正负号，`%` 乘一百加百分号，`0` 不带小数。`$MODIFIER_CAVALRY_ATTACK_FACTOR$` 借的是引擎自带的修正名文案，借用它，你的提示和游戏其它地方一个口吻。这些键名没有统一规律：骑兵攻击是 `MODIFIER_CAVALRY_ATTACK_FACTOR`，陆军师攻击却是 `MODIFIERS_ARMY_ATTACK_FACTOR`，多一个 S。写之前到原版 `localisation/` 目录里搜一下修正名，找不到就自己写中文。

原版把这个参数用在 `add_to_variable` 上两千多处，`set_variable` 和 `subtract_from_variable` 也能带。

回到第二节那种重算型的精神，国策不该碰计数，但可以改「每块多少」：把 0.03 做成变量。重算里最后一步改成乘变量，国策里加变量并带提示，开局历史文件里多初始化一行：

```
multiply_variable = { my_thorn_attack = my_thorn_rate }
```

```
add_to_variable = { my_thorn_rate = 0.02 tooltip = my_thorn_rate_tt }
```

```yaml
l_simp_chinese:
 my_thorn_rate_tt:0 "每块失地的陆军师攻击加成：$RIGHT|+=0%$"
```

这样玩家在国策上看到「每块失地的陆军师攻击加成：+2%」，精神里的总数第二天跟着变。

## 常见错误

- **在 idea 里写变量名**：静默无效，游戏不报错，精神看起来挂着但数值是 0。
- **钩子名里的 TAG 没换**：`on_daily_MYT` 只对 MYT 跑，你的国家不叫 MYT 就永远不重算。
- **忘了先初始化变量**：第一天修正值为空；后面重算会把它建出来，但依赖 `has_variable` 的判断已经走错。
- **计数加到了错误的作用域**：循环里不写 `ROOT = { }`，计数落在州或外国身上，你的变量永远是 0。
- **期待说明行自动生成**：引擎只渲染总数那一行；每条规则的文字都要自己写在 `custom_modifier_tooltip` 指向的键里。
- **用全局 `on_daily` 重算**：每个国家每天都跑一遍循环，游戏会明显变卡。用 `on_daily_<TAG>`，或者事件钩子。
- **本地化键里带非法字符**：键名只能用字母、数字、下划线、点、连字符和撇号，一个非法字符会让文件后面的所有条目一起消失。

## 想改成别的做法？

- **只想要一个固定数值，不随局势变**：直接用 idea，简单得多。
- **数值随一个引擎本来就有的量变，比如稳定度**：不用自己数，见第 7.1 节。
- **想让 AI 也理解这个修正**：AI 只看修正值，不看说明行；说明行是给玩家的。
- **想按州而不是按国家生效**：动态修正也能挂在州上，`add_dynamic_modifier` 带 `scope` 参数；`attacker_modifier = yes` 时对进攻该州的师生效。
