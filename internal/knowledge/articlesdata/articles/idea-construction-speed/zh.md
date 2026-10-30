---
contentId: idea-construction-speed
kind: case
slug: idea-construction-speed
locale: zh
status: published
title: "获得持续 180 天的建设速度国家精神"
summary: "通过一次性决议授予建设速度 +10% 的国家精神，持续 180 天。配齐精神定义、工厂数量条件、授予入口与中文本地化。"
primaryTopic: industry
topics: [industry]
scenarios: [ideas, decision]
gameTerms: [建设速度, 国家精神, 民用工厂, 决议]
aliases: [建设速度加10%, 限时建设加速, 180天国家精神, 国家精神造厂加速]
relatedIds: [construction-speed, "wiki:modifier:production_speed_buildings_factor", "wiki:trigger:num_of_civilian_factories", "wiki:effect:add_building_construction", "wiki:effect:add_timed_idea", "wiki:effect:remove_ideas"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "国家精神建设速度增加10% 持续180天 限时建设加速 正向奖励 一次性决议 add_timed_idea production_speed_buildings_factor 中文本地化 不是立即送工厂 不是建设减速案例"
sources:
  - label: "HOI4 Wiki：国家精神制作"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
  - label: "HOI4 Wiki：决议制作"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
  - label: "HOI4 Wiki：限时国家精神"
    url: "https://hoi4.paradoxwikis.com/Effects#add_timed_idea"
  - label: "HOI4 Wiki：建设速度修正"
    url: "https://hoi4.paradoxwikis.com/Modifiers#production_speed_buildings_factor"
  - label: "HOI4 Wiki：本地化"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

目标是给玩家一段建设加速期，而不是立刻赠送工厂：民用工厂数量达到十座后，可免费使用一次决议，获得“建设协调计划”，其建设速度修正为 +10%，持续 180 天。

本例已核对资料，尚未在 HOI4 中执行。以下为完整新增内容文件与待执行的验收步骤。

## 开始前

- 已有一个能正常加载的 Mod，游戏界面为简体中文。下面的路径都相对于 Mod 根目录，不写入原版安装目录。
- `hag_knowledge_construction`、`hag_knowledge_construction_plan` 相关 ID 在你的 Mod 和依赖中不能重复。
- 保留原版 `GFX_idea_generic_production_bonus` 图片资源；若整合 Mod 已替换图标，应改用其有效的国家精神图片名。本例不附带新素材。
- 准备一个满足工厂数量条件、能正常安排建筑施工的测试国家。这里检查的是 `num_of_civilian_factories`，不是空闲民用工厂数量，也不会占用十座工厂。

决议没有国家标签限制，AI 不会主动选择，每个国家只可使用一次。它是演示入口；正式国策奖励或剧情事件应接到你自己的内容流程。

## 1. 定义国家精神

新建 `common/ideas/hag_knowledge_construction.txt`，以下是完整文件：

```pdx
ideas = {
    country = {
        hag_knowledge_construction_plan = {
            picture = generic_production_bonus
            removal_cost = -1

            modifier = {
                production_speed_buildings_factor = 0.10
            }
        }
    }
}
```

`country` 表示国家精神类别。`picture` 填的是图片名后半部分，游戏使用 `GFX_idea_` 前缀；不要在这里重复写成 `GFX_idea_generic_production_bonus`。

建设速度修正放在精神内部的 `modifier` 中。定义文件并不会自动把精神授予国家。`removal_cost = -1` 阻止手动移除，不代表限时精神不会到期。

## 2. 从决议授予，期限在这里设置

新建 `common/decisions/categories/hag_knowledge_construction.txt`：

```pdx
hag_knowledge_construction_category = {
    allowed = {
        always = yes
    }
}
```

新建 `common/decisions/hag_knowledge_construction.txt`：

```pdx
hag_knowledge_construction_category = {
    hag_knowledge_begin_construction_plan = {
        cost = 0
        fire_only_once = yes

        available = {
            NOT = {
                num_of_civilian_factories < 10
            }
        }

        complete_effect = {
            add_timed_idea = {
                idea = hag_knowledge_construction_plan
                days = 180
            }
        }

        ai_will_do = {
            factor = 0
        }
    }
}
```

`available` 表达“至少十座”，恰好十座也满足这项条件。它只检查数量，不扣除工厂。少于十座时，决议不可选择，而不是通过 `visible` 被藏起来。

`add_timed_idea` 在使用决议的国家执行。`idea` 必须精确对应第一步定义的 ID；180 天写在授予效果中，不写进 `modifier`。

这个入口只授予一次，避免演示中反复点击掩盖计时问题。本文不定义重复授予同一精神时的刷新策略，也不承诺内战等特殊状态下的继承行为。

## 3. 配齐中文名称

新建 `localisation/simp_chinese/hag_knowledge_construction_l_simp_chinese.yml`，以 UTF-8 BOM 编码保存：

```yaml
l_simp_chinese:
 hag_knowledge_construction_category:0 "建设事务"
 hag_knowledge_construction_category_desc:0 "协调建筑施工的安排。"
 hag_knowledge_begin_construction_plan:0 "启动建设协调计划"
 hag_knowledge_begin_construction_plan_desc:0 "集中协调施工安排，在一段时间内提高建设速度。"
 hag_knowledge_construction_plan:0 "建设协调计划"
 hag_knowledge_construction_plan_desc:0 "统一调配施工计划，减少工程之间的衔接延误。"
```

国家精神的说明文字不代替真实修正；数值由脚本产生的效果提示展示。复制代码块后，仍需在编辑器中确认文件编码和语言头。

## 4. 在游戏中检查什么

以下步骤尚未执行。使用可丢弃的新测试存档，并记录版本与启用的 Mod。

1. 检查少于十座、恰好十座、超过十座三种条件状态。先看条件提示，再判断脚本是否加载；涉及贸易或占领的特殊计数另行核对。
2. 使用决议前，记录国家精神列表与同一施工项目的速度提示、工厂分配情况。
3. 使用决议后，检查“建设协调计划”的名字、图标、建设速度 +10% 和剩余期限。确认并没有因此立即增加一座工厂。
4. 在其他条件尽量不变时对照施工数据。不能用“剩余工期是否正好减少 10%”作为唯一判据；速度修正与工期不是同一数值。
5. 在测试存档中推进到精神到期，确认本来源消失、不再提供建设速度修正。到期不应删除已经建成的建筑。同一国家也不应再次使用这个一次性决议。
6. 查看 `error.log` 中与本例 ID、修正名或本地化有关的新错误；不要把无关旧日志混作本例结论。

若能看到精神但没有建设加速，先检查 `modifier` 的层级、键名和数值；若只有决议没有精神，检查授予效果的 ID；若出现原始键名或空图标，再分别检查本地化和图片引用。

## 按自己的设计调整

- 改加成幅度或加入负面效果，先看建设速度正负用法 (`wiki:modifier:production_speed_buildings_factor`)，并同步修改剧情说明。本例只展示正向加速。
- 想直接赠送一座民用工厂，改看建筑效果 (`wiki:effect:add_building_construction`)，不要把修正数值填成工厂数量。
- 想取消来源，可查 remove_ideas 原始参考 (`wiki:effect:remove_ideas`)。移除精神与重新开放一次性决议是不同事情。
- 返回《建设速度专题》（文章 `construction-speed`）。
