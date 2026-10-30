---
contentId: leader-succession
kind: case
slug: leader-succession
locale: zh
status: published
title: "换领袖：候补人选、事件继位与替换原版领袖"
summary: "预先做好的第二位领袖开局不会自动上台，要靠事件提拔或让在任者退场；提拔一个已经不在的人什么都不会发生。替换原版领袖走另一条路，钢之工坊会替你生成那段脚本。"
primaryTopic: people
topics: [people, politics]
scenarios: [event, history]
gameTerms: [国家领袖, 继位, 换领袖, 退休, 领袖死亡, 执政党, 候补领袖]
aliases: [换一个领袖, 领袖不上台, 第二个领袖没用, 事件换领袖, 领袖退休, 替换原版领袖, promote_character, retire_country_leader, create_country_leader]
relatedIds: [character-system, advisor-unlocked-by-focus, "wiki:effect:promote_character", "wiki:effect:retire_country_leader", "wiki:effect:kill_country_leader", "wiki:effect:create_country_leader", "wiki:effect:retire_character"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：人物制作"
    url: "https://hoi4.paradoxwikis.com/Character_modding"
  - label: "HOI4 Wiki：效果"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki：事件制作"
    url: "https://hoi4.paradoxwikis.com/Event_modding"
---

# 换领袖：候补人选、事件继位与替换原版领袖

## 目标与最终可观察结果

做完这一篇之后，在游戏里应该能看到三件事：

1. 开局时执政的还是原来的领袖；你做的候补人选存在，但没有上台。
2. 一个事件触发后，候补人选成为领袖，领袖卡换成他的名字和特质。
3. 再一个事件让他退场，领袖换成该党的下一顺位。

第 1 条是引擎行为，不是错误；第 2、3 条各是一个效果。本文最后一节讲另一条路：替换原版领袖。

## 开始前需要什么

- 一个已经能正常导出的 Mod。
- 一个候补领袖人物：「角色类型」国家领袖，「子意识形态」与执政党一致。本文用占位 ID `my_leader_2`。
- 会用事件工作台写一个由脚本触发的事件，或者会在国策奖励里写 `country_event`。
- 不需要任何 DLC。
- 建议先读做一个人物：领袖、将领、顾问各管什么 (`character-system`)。

## 涉及的对象与关系

| 对象 | 位置 | 在本例中的作用 |
| --- | --- | --- |
| 候补领袖 | `common/characters/` | 一份档案加一个 `country_leader` 块 |
| 招募 | `history/countries/` 下该国的历史文件 | `recruit_character` 一行，工作台自动补 |
| 事件 | `events/` | 承载提拔、退休、创建三种效果 |
| 名称 | `localisation/` | 领袖卡上显示的名字 |

## 1. 做候补领袖：他存在，但开局不上台

按普通领袖做即可：

```
characters = {
    my_leader_2 = {
        name = my_leader_2
        portraits = {
            civilian = {
                large = GFX_portrait_my_leader_2
                small = GFX_portrait_my_leader_2_small
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

进游戏后你会发现：**开局执政的还是原版领袖**，你做的这位只是该党的一个候选人。这是实测结果，不是配置错了。一个国家同一意识形态可以有多位领袖人物，谁在任由历史文件和事件决定。

所以，"预先建好第二位领袖等着他上台"这件事本身不会发生任何事。他需要一个把他推上去的效果。

## 2. 用事件提拔他

提拔指定人物 (`wiki:effect:promote_character`) 把一个人物推到他所属政党的首位；如果那是执政党，他就成为国家领袖。写在事件选项里：

```
add_namespace = my_events

country_event = {
    id = my_events.1
    title = my_events.1.t
    desc = my_events.1.d
    is_triggered_only = yes
    option = {
        name = my_events.1.a
        promote_character = my_leader_2
    }
}
```

`is_triggered_only = yes` 表示这个事件只由脚本触发。在国策的完成奖励里写 `country_event = my_events.1`，或者在决议里写，国策完成时事件弹出，点选项后领袖换人。测试时用控制台 `event my_events.1` 直接触发。

实测：事件弹出、点确定，领袖卡立即换成 `my_leader_2`，他的特质一起显示。

## 3. 让在任领袖退场：退休还是死亡

两个效果都能把在任领袖换掉，区别在人还在不在：

| 效果 | 发生什么 | 原版用量 |
| --- | --- | --- |
| 让在任领袖退休 (`wiki:effect:retire_country_leader`) `retire_country_leader = yes` | 卸任，不再是该党首位；下一顺位接任 | 一百五十七处 |
| 在任领袖死亡 (`wiki:effect:kill_country_leader`) `kill_country_leader = yes` | 死亡并**从国家移除**；下一顺位接任 | 四十八处 |

两者的下一顺位都是该党另一位领袖人物。实测里，先用事件提拔了候补领袖，再执行 `kill_country_leader = yes`，候补领袖死亡，领袖回到原版那位；接着再执行一次提拔候补领袖的事件，**弹窗照常出现，但什么都没发生，日志里也没有任何一行**。人已经不在了，提拔一个不存在的人是静默无效的。

所以选哪个取决于剧情：以后还想让他回来，用退休；确定不会再出现，用死亡。想点名让某个不在任的人物退场，用 让指定人物退出 (`wiki:effect:retire_character`) `retire_character = my_leader_2`，它同时移除这个人的全部身份。

## 4. 不建人物，直接在事件里创建领袖

如果这位领袖只在一条剧情线里出现一次，可以不做人物，直接在事件里创建：

```
option = {
    name = my_events.2.a
    create_country_leader = {
        name = "临时执政"
        picture = GFX_portrait_my_leader_2
        expire = "1965.1.1"
        ideology = conservatism
    }
}
```

实测：事件后领袖卡出现这个名字。它不对应任何人物档案，所以之后不能再用 `promote_character` 或 `retire_character` 按名字找到他；原版一百零四处这种写法多用于"临时政府""过渡人物"这类不需要再引用的场合。

## 5. 替换原版领袖：钢之工坊替你生成脚本

上面几节都是"多加一位领袖"。如果你要的是"把原版的某位领袖换掉"，情况不同：原版已经在 `common/characters/` 里定义了他，你再定义一个同名人物，引擎会报人物标识重复。

工作台的做法是：在人物上填「覆盖 vanilla 人物 ID」，保留「导出时自动生成 trait_overlay 事件」。导出时钢之工坊**不会**把这个人写进 `common/characters/`，而是生成一个开局第一天执行的隐藏事件，在里面先让原版领袖退休，再按你填的内容创建新领袖：

```
country_event = {
    id = my_overlay.1
    title = my_overlay.1.t
    desc = my_overlay.1.d
    hidden = yes
    is_triggered_only = yes
    immediate = {
        MYT = {
            retire_country_leader = yes
            create_country_leader = {
                name = my_leader_2
                picture = GFX_portrait_my_leader_2
                expire = "1965.1.1"
                ideology = conservatism
                traits = { my_leader_trait }
            }
        }
    }
}
```

同一段退休加创建也会写进该国的历史文件，这样国家选择屏和书签界面看到的就是你的领袖，而不是等开局第一天才换。这段脚本由导出器生成，你只需要在工作台里填人物本身。

如果你不用钢之工坊、手写替换，两种做法：在事件里用同样的退休加创建；或者整份覆盖原版的 `common/characters/<TAG>.txt`，删掉原版那个人的块。直接再定义一个同标识的人物会报 `Multiple character have the tag`，这一条来自引擎报错原文，本文没有实测触发它。

## 6. 逐项核对

| 检查 | 期望 | 不符合时看哪里 |
| --- | --- | --- |
| 开局领袖卡 | 原版领袖在任 | 这是正常的；不要以为候补领袖坏了 |
| 触发提拔事件 | 领袖换成候补人选，特质随之显示 | 事件里的人物 ID；历史文件是否招募了他 |
| 触发退休事件 | 领袖换成该党下一顺位 | 该党是否还有其他领袖人物 |
| 触发死亡事件后再提拔同一人 | 弹窗出现但领袖不变 | 这是正常的；人已被移除 |
| 触发创建事件 | 出现一个不对应人物档案的新领袖 | `ideology` 是否是子意识形态 |
| 替换原版领袖后进国家选择屏 | 显示你的领袖 | 历史文件里有没有那段退休加创建 |

## 常见错误

- 🔴 **预先建好第二位领袖，以为他会自己上台**：不会。需要提拔效果或退休效果。
- 🔴 **提拔一个已经死亡或退出的人物**：弹窗照常、无事发生、日志零输出。剧情里如果一个人可能先死后被提拔，用退休而不是死亡。
- **想"换人"却用了死亡**：死亡把人从国家移除，以后的事件再也找不到他。
- **用创建代替建人物，之后又想按名字引用他**：创建出来的领袖没有人物档案，`promote_character` 和 `retire_character` 都找不到他。
- **替换原版领袖时又手写了一份同标识的人物**：引擎报人物标识重复。使用工作台的覆盖功能，或者整份覆盖原版文件。

## 想改成别的效果？

- 想让在任领袖换一个意识形态而不是换人：`set_country_leader_ideology`，本文没有实测。
- 想给在任领袖加一个特质：给领袖添加特质 (`wiki:effect:add_country_leader_trait`)。
- 想让特质在领袖卡上有中文名而不是一串键名：见给人物做一个自定义特质 (`custom-character-traits`)。
