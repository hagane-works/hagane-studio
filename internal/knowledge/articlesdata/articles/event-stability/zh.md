---
contentId: event-stability
kind: case
slug: event-stability
locale: zh
status: published
title: "做一个增加稳定度的事件选项"
summary: "从一次性决议打开事件，玩家选择后增加 5 个百分点基础稳定度。包含事件、调用入口、中文本地化与待执行的检查步骤。"
primaryTopic: politics
topics: [politics]
scenarios: [event, decision]
gameTerms: [稳定度, 基础稳定度, 事件, 决议]
aliases: [事件增加稳定度, 稳定度奖励, 选择后稳定度加5%, 决议触发事件]
relatedIds: [stability, "wiki:effect:add_stability", "wiki:trigger:has_stability", "wiki:modifier:stability_weekly"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "事件选项增加稳定度 一次增加5个百分点 基础稳定度加5% 正向奖励 完整事件 决议触发 country_event add_stability 中文本地化 不是每周增长"
sources:
  - label: "HOI4 Wiki：事件制作"
    url: "https://hoi4.paradoxwikis.com/Event_modding"
  - label: "HOI4 Wiki：决议制作"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
  - label: "HOI4 Wiki：稳定度效果"
    url: "https://hoi4.paradoxwikis.com/Effects#add_stability"
  - label: "HOI4 Wiki：本地化"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

这个例子完成一条明确的流程：玩家点击免费的一次性决议 → 看到事件 → 选择“按计划推进” → 基础稳定度增加 5 个百分点。

本例已经核对资料，但尚未在游戏中执行。下面的游戏检查是交给制作者的验收步骤，不是已通过记录。

## 开始前

- 已有一个可以正常启用并进入游戏的 Mod。本例只补充内容，不新建国家、地图或国策树。
- 在 Mod 自己的目录里新增下列文件，不修改游戏安装目录。文件路径均相对于 Mod 根目录。
- 使用简体中文游戏界面。`hag_knowledge_stability` 前缀在你的 Mod 及其依赖中不能与其他内容重复。
- 使用仍保留原版基础资源的环境。事件引用已存在的 `GFX_report_event_generic_parliament`，没有附带新的图片；大型整合 Mod 若替换它，需要换成自己已经注册的事件图片。

调用入口没有国家标签限制，但关闭了 AI 主动选择；每个国家只能使用这个普通决议一次。正式剧情若只适用于一个国家，应另外加上经过核对的国家条件。

## 1. 定义事件

新建 `events/hag_knowledge_stability.txt`，以下是完整文件：

```pdx
add_namespace = hag_knowledge_stability

country_event = {
    id = hag_knowledge_stability.1
    title = hag_knowledge_stability.1.t
    desc = hag_knowledge_stability.1.d
    picture = GFX_report_event_generic_parliament

    is_triggered_only = yes

    option = {
        name = hag_knowledge_stability.1.a
        add_stability = 0.05
    }
}
```

`add_namespace` 放在事件外。`is_triggered_only` 表示这个事件依靠调用，不自行随机触发；这里不添加 `mean_time_to_happen`。

奖励直接放在 `option` 内，不放在 `immediate`，也不再套 `effect`。所以这个设计的奖励点是选项执行时，而不是弹窗刚出现时。普通事件若超时自动选择选项，也会执行相应选项效果；它不是必须点击才可能结算的永久等待窗口。

## 2. 提供一个可以点击的入口

新建 `common/decisions/categories/hag_knowledge_stability.txt`：

```pdx
hag_knowledge_stability_category = {
    allowed = {
        always = yes
    }
}
```

再新建 `common/decisions/hag_knowledge_stability.txt`：

```pdx
hag_knowledge_stability_category = {
    hag_knowledge_open_stability_event = {
        cost = 0
        fire_only_once = yes

        available = {
            has_stability < 0.95
        }

        complete_effect = {
            country_event = {
                id = hag_knowledge_stability.1
            }
        }

        ai_will_do = {
            factor = 0
        }
    }
}
```

类别键必须与上一文件相同。决议的 `complete_effect` 在选择时调用事件，事件接收国就是使用决议的国家；整个流程不切换到其他国家或地区。

`has_stability < 0.95` 是严格低于 95% 的演示门槛，便于避开接近上限的情况，不代表所有稳定度奖励都必须使用这个条件。它不扣稳定度。一次限制放在决议上，并不阻止其他脚本再次调用同一事件；不要额外重复接入多个调用入口。

## 3. 补上中文名称

新建 `localisation/simp_chinese/hag_knowledge_stability_l_simp_chinese.yml`，以 UTF-8 BOM 编码保存。下列代码块给出完整文本，复制之后仍需确认编辑器的保存编码：

```yaml
l_simp_chinese:
 hag_knowledge_stability_category:0 "地方事务"
 hag_knowledge_stability_category_desc:0 "协调地方事务，恢复民众对日常行政的信心。"
 hag_knowledge_open_stability_event:0 "召开地方事务会议"
 hag_knowledge_open_stability_event_desc:0 "听取地方代表的意见，确定接下来的工作安排。"
 hag_knowledge_stability.1.t:0 "会议达成共识"
 hag_knowledge_stability.1.d:0 "各地代表已就近期的行政安排达成共识。相关部门将按会议决定推进工作。"
 hag_knowledge_stability.1.a:0 "按计划推进。"
```

文件名后缀、第一行语言名与实际游戏语言要对应。事件标题、描述和选项键各不相同，不用同一个 `.t` 键同时充当按钮文字。

## 4. 在游戏中检查什么

这些步骤尚未执行。请使用可丢弃的新测试存档，不直接修改重要进度。

1. 启用 Mod，进入一个稳定度低于 95% 的国家。确认决议名称可读；若稳定度达到 95%，应检查条件不满足的提示，而不是将其误判为加载失败。
2. 暂停并记录稳定度提示中的基础值和其他来源。选择决议后，确认弹出的标题、正文与选项均已本地化。
3. 在事件选项尚未执行时检查，不应因为这个事件本身已增加稳定度。悬停选项，查看游戏生成的稳定度效果提示。
4. 选择“按计划推进”，对照基础稳定度变化。在没有其他变化或限制的简单测试条件下，50% 增加 5 个百分点应为 55%，不是 52.5%。
5. 确认同一国家不能再次使用该决议，其他国家没有因这次选择自动领取奖励。查看 `error.log` 是否出现与本例 ID、命名空间或本地化有关的新错误。

如果只看显示总数没有变化，先核对基础值、其他稳定度来源与限制；如果事件完全没有出现，核对调用 ID 与命名空间；如果出现原始键名，核对本地化路径、语言头和 BOM 编码。这三类问题应分别排查。

## 想改成其他效果？

- 一次性惩罚可参照 add_stability 的负数用法 (`wiki:effect:add_stability`)，同时修改剧情文案。本例本身是正向奖励，不是减益案例。
- 想让稳定度每周增加，不要反复调用本事件；看每周稳定度 (`wiki:modifier:stability_weekly`)。
- 只需要限制是否可用，看稳定度门槛 (`wiki:trigger:has_stability`)。
