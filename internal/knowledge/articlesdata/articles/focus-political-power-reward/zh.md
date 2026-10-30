---
contentId: focus-political-power-reward
kind: case
slug: focus-political-power-reward
locale: zh
status: published
title: "完成国策后获得 150 政治点"
summary: "在已有国策树中加入一个完整奖励节点，补齐中文名称，并检查完成时的一次性政治点奖励。"
primaryTopic: politics
topics: [politics]
scenarios: [focus]
gameTerms: [政治点数, 国策, 国策奖励]
aliases: [国策奖励政治点, 完成国策加政治点, 国策完成获得150政治点]
relatedIds: [political-power, "wiki:effect:add_political_power", "wiki:trigger:has_political_power", "wiki:modifier:political_power_gain", "wiki:modifier:political_power_factor"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "国策完成获得150政治点 国策一次性奖励150政治点 completion_reward add_political_power 正数奖励 中文本地化 现有国策树"
sources:
  - label: "HOI4 Wiki：国策制作"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
  - label: "HOI4 Wiki：add_political_power"
    url: "https://hoi4.paradoxwikis.com/Effects#add_political_power"
  - label: "HOI4 Wiki：本地化"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

## 做完后是什么样？

目标国家的现有国策树里多出“整顿行政”节点。玩家完成后，执行一次 `add_political_power = 150`，获得一笔 150 点政治点奖励。这个案例不创建国家精神，也不调整每日获取。

示例已对照资料，尚未在游戏中运行。下文的验收步骤是需要执行的检查，不是已经通过的记录。

## 开始前需要什么？

- 已有一个能够加载的 Mod，且目标国家在你的开局中确实使用这棵国策树。
- 知道那棵树对应的 `common/national_focus/` 文件及 `focus_tree` ID。若文件有多棵树，要先选对块。
- 给改动保留可恢复的副本。只改你的 Mod，不修改游戏安装目录里的原版文件。
- 确认 `HAG_pp_administrative_reform` 没有被其他节点使用；确认示例坐标 `(12, 0)` 没有占用。若有冲突，修改 ID 或坐标，并同步下面的本地化键。

这是一份“向已有、已加载的树添加奖励节点”的完整案例，不负责新建国家、分配国策树或生成可独立启动的 Mod。

## 第一步：加入完整节点

打开现有国策树文件，在目标 `focus_tree = { ... }` 内、与其他 `focus = { ... }` 平级的位置插入：

```pdx
focus = {
    id = HAG_pp_administrative_reform
    icon = GFX_goal_generic_political_pressure
    x = 12
    y = 0
    cost = 5
    search_filters = { FOCUS_FILTER_POLITICAL }

    completion_reward = {
        add_political_power = 150
    }

    ai_will_do = {
        factor = 1
    }
}
```

这是完整的 `focus` 节点，不是整份国策树文件。不要把它放到另一个节点里面，也不要另建同 ID 的 `focus_tree` 来“追加”内容。

本例使用已在原版定义中核对的通用图标，不需要新增图片。它没有前置条件，是一个独立的起始节点；如果你的树需要它接在别的节点后面，应另外添加正确的前置关系并调整位置。

`cost = 5` 是国策进度成本，不是花费 5 政治点。在本次核对的默认规则下，每点进度对应 7 天，因此基础时长为 35 天；Mod 改过相应规则时，以该 Mod 为准。

`ai_will_do` 是 AI 选择权重，不是给 AI 的额外奖励。政治点效果位于 `completion_reward`，所以正常完成时才执行，不是在点击开始时发放。

## 第二步：补齐中文名称和说明

在你的 Mod 中添加 `localisation/simp_chinese/hag_pp_focus_l_simp_chinese.yml`，并用编辑器保存为 UTF-8 with BOM：

```yaml
l_simp_chinese:
 HAG_pp_administrative_reform:0 "整顿行政"
 HAG_pp_administrative_reform_desc:0 "理顺各部门的职责，为接下来的改革做好准备。"
```

第一行语言标记、文件名后缀和目录都对应游戏的简体中文。本地化键前保留一个空格，文字使用引号。若你修改了节点 ID，这两个键要一起修改。

这份本地化只覆盖游戏简体中文；其他游戏语言需要各自的文件。切换 Hagane Works 的界面语言，不会自动替你的 Mod 生成游戏本地化。

## 第三步：检查位置与执行对象

检查目标树中的新节点：

- `completion_reward` 内只写执行效果，不把 `has_political_power` 这样的条件当作奖励。
- 没有新增 `GER = { ... }` 等国家作用域外壳，所以奖励给完成该普通国策的国家。
- `add_political_power` 使用数值参数，没有再套 `value` 块，也没有写 `%`。
- 若你是在现有节点上修改奖励，只把 `add_political_power = 150` 加进它原本的奖励块，不要复制整个示例节点或制造两个同名奖励块。

本例不要求玩家先拥有 150 政治点。如果你的设计还需要这个门槛，阅读政治点条件 (`wiki:trigger:has_political_power`)，不要把“奖励 150”与“至少拥有 150”混为一谈。

## 第四步：按这张表验收

这些检查尚未执行。准备好测试环境后，使用该 Mod 的新开局检查；不要只凭文件能解析就判定完成。

| 检查 | 预期 | 不符合时先看什么 |
| --- | --- | --- |
| 目标国家打开国策树 | 出现“整顿行政”，名称不是原始 ID | 国家实际加载的树、节点位置、ID、本地化编码 |
| 查看节点奖励 | 显示一次性政治点奖励 150 | 奖励块位置、数值是否为正、是否误写成每日修正 |
| 点击开始 | 不立即发放这笔奖励 | 是否误放进 `select_effect` |
| 正常完成 | 完成时向该国家发放一笔奖励 | 作用域、是否正常完成而非绕过、同期其他效果 |
| 完成之后 | 不会因这一行额外产生每日 +150 | 是否另有国家精神、重复调用或每日脚本 |
| 查看本次游戏日志 | 无这次新增 ID、效果和本地化相关错误 | 先从本次新增代码及其前后括号排查 |

数值测试前留出足够的政治点余额空间，避免奖励触及上限。记录完成前后的余额和同期日结、支出、其他奖励；不要拿国策开始前与 35 天后的余额直接相减，那会混入整个期间的获取和消耗。

## 常见改法

- 奖励改成 75 点：只改成 `add_political_power = 75`，并再次检查奖励提示。
- 要一次扣除政治点：先阅读立即增加或减少政治点 (`wiki:effect:add_political_power`)，确认条件和费用设计；本案例只演示正数奖励。
- 要完成后每天多获得政治点：改为授予承载固定获取 (`wiki:modifier:political_power_gain`)或比例修正 (`wiki:modifier:political_power_factor`)的国家精神，不是把每日修正塞进当前奖励块。
- 还不确定选哪种：回到《政治点专题》（文章 `political-power`）。
