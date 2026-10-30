---
contentId: custom-tech-folder
kind: guide
slug: custom-tech-folder
locale: zh
status: published
title: "自建科技页签：为什么科技存在却画不出来"
summary: "自定义科技不显示，不是因为做不到，而是因为界面那一层要你自己补。五处改动缺一不可，其中三种失败模式日志里一个字都没有。"
primaryTopic: research
topics: [research]
scenarios: [technology]
gameTerms: [科技树, 研究界面, 科技页签, folder, 节点模板, 本地化]
aliases: [自建科技树, 自定义科技不显示, 科技看不见, 新增科技页签, 科技树加一页, 科技画不出来]
relatedIds: [technology-system, tech-unlock-equipment, research-speed]
updatedAt: "2026-09-09"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：科技"
    url: "https://hoi4.paradoxwikis.com/Technology"
  - label: "HOI4 Wiki：界面"
    url: "https://hoi4.paradoxwikis.com/Interface"
  - label: "HOI4 Wiki：本地化"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

# 自建科技页签：为什么科技存在却画不出来

## 这一篇解决什么

网上流传一句话：

> 自定义科技不会出现在科技树里。研究界面只渲染落在 `interface/countrytechtreeview.gui` 声明的那组
> 固定格子里的科技，所以自建科技无处可画。

**这句话只对了一半。**准确的说法是：**自建科技需要你自己在那个文件里补声明，补齐了就能画。**

本文讲补什么、补在哪、漏了会怎样。如果你只是想在**原版已有的**页签下加一门科技，
不需要读这篇——看做一门科技 (`technology-system`)就够了。

## 一条科技要显示，需要三层同时存在

```
① 科技定义    common/technologies/*.txt          → 它属于哪个 folder、画在哪一格
② folder 声明 common/technology_tags/*.txt       → 这个 folder 存在、归哪本统计账
③ 界面三件套  interface/countrytechtreeview.gui  → 容器 + 格子(gridbox) + 节点模板
```

再加上页签按钮和本地化，否则玩家点不进去、或者看到的是原始 key。

**绝大多数教程只讲 ①**，于是 ② ③ 缺失，科技就"存在但不可见"。

### 一条贯穿全文的规矩：这一层必须肉眼验收

这一层的失败**有相当一部分不进 `error.log`**，还有一部分只在你真正点开那个页签之后才打印。
"日志干净"不等于"画出来了"，更不等于"画在对的位置"。

## 核心规则一：格子的命名

`gridboxtype` 是界面里的一个格子容器，写在**对应 folder 的容器内部**：

```
gridboxtype = {
    name = "support_weapons_tree"
    position = { x = 140 y = 210 }
    slotsize = { width = 70 height = 70 }
    format = "LEFT"
}
```

> **格子名 = 根科技 id + `_tree`**

同一条链上、由 `path = { leads_to_tech = … }` 串下来的科技**共用这一个格子**。

**根科技的精确定义**：在这个 folder 内，没有被同 folder 内任何一条 `path` 指向的科技。三个细节：

1. **"根"是相对 folder 而言的。** 一条科技可以写多个 `folder = { }` 同时出现在多页；它在 A 页是下游、
   在 B 页可能就是根，那么 B 页要有属于它的格子。
2. **只有 `path` 建立"同格子"关系。** 科技没有 `prerequisites` 这个键（写了会报
   `Unknown modifier: prerequisites`，原版 `common/technologies/` 里零出现）。
   两条科技之间没有 `path`，它们就是两个根、要两个格子。
3. **注释掉的 `folder` 块不算。** 原版有若干科技的 `folder` 被 `#` 注释掉了，它们不在那一页里。

原版对 12 个 folder 逐个比对过："有根科技却没格子"的情况是 **0**；多余的格子无害，不确定就留着。

## 核心规则二：坐标是相对偏移

原版把年份写成常量再用在 `y` 上：

```
@1918 = 0
@1936 = 2
@1938 = 4

folder = {
    name = infantry_folder
    position = { x = 0 y = @1936 }
}
```

🔴 **`y` 是横轴，`x` 是纵轴——和直觉相反。** `y` 增加往右移，`x` 增加往下换一行。

原版 `infantry_folder` 的证据：`infantry_weapons1`(x=0 y=@1936) → `infantry_weapons2`(x=0 y=@1938)
→ `improved_infantry_weapons`(x=0 y=@1939)，同一条链 `x` 恒为 0、只有 `y` 在涨，而它在游戏里是
一条**水平**的链。装甲车线的 `armored_car1` 是 `x=4`，画在下面第 4 行。

⚠️ **`start_year` 与 `position` 无关**：前者只管超时代惩罚的基准年，后者只管画在哪一格。
把年份常量写进 `y` 是原版的**惯例**，不是引擎强制——所以架空或古代背景的 Mod 可以自己定年份档位。

方向填错不会有任何报错，节点照样画出来，只是落进了别人的链条里。

## 五处改动

以新建一个叫 `mymod_infantry_folder` 的科技页为例。**五处缺一不可。**

### ① `common/technology_tags/00_technology.txt` —— 声明 folder

同名遮蔽原版文件，在 `technology_folders = { }` 内加：

```
mymod_infantry_folder = {
    ledger = army
}
```

`ledger` 归哪本统计账：`army` / `navy` / `air` / `civilian` / `hidden`。
可选的 `available = { }` 是触发器，不满足则**整个页签不绘制**。

⚠️ 这个文件里还有一个很大的 `technology_categories = { }` 块。科技 `categories = { }` 里出现的
每一个名字**都必须先在这里登记**（原版登记 134 个、实际用到 105 个、未登记 0 个——原版自己严格遵守）。

### ② 界面 —— folder 容器

在 `interface/countrytechtreeview.gui` 里，名为 `countrytechtreeview` 的那个容器**内部**加一个子容器。
**名字必须与 folder id 完全一致**：

```
containerWindowType = {
    name = "mymod_infantry_folder"
    position = { x=0 y=47 }
    size = { width = 100%% height = 100%% }
    margin = { top = 13 left = 13 bottom = 24 right = 25 }
    drag_scroll = { left middle }
    verticalScrollbar = "right_vertical_slider"
    horizontalScrollbar = "bottom_horizontal_slider"
    scroll_wheel_factor = 40

    background = {
        name = "Background"
        quadTextureSprite = "GFX_tiled_window_2b_border"
    }

    containerWindowType = {
        name = "techtree_stripes"
        position = { x = 0 y = 0 }
        size = {
            width = 1400 height = 1275
            min = { width = 100%% height = 100%% }
        }
        clipping = no
    }

    gridboxtype = {
        name = "MYTECH_root_tree"
        position = { x = 140 y = 210 }
        slotsize = { width = 70 height = 70 }
        format = "LEFT"
    }
}
```

🔴 **`techtree_stripes` 这个名字是固定的，不要改，也不要因为"它看起来只是背景"就省掉。**
节点是画在这块画布上的；缺了它，页签和节点会一并不显示，而且**日志一个字都不会说**。
`size` 就是画布大小，树画不下就往大调。

`%%` 是 gui 文件里的转义写法，**照抄，不要改成单个 `%`**。

名字与 folder id 对不上的症状是：页签在，点了空白。

> ⭐ **年份刻度和行标题不是引擎生成的**，是在 `techtree_stripes` 里用 `instantTextBoxType` 硬写的
> 字面量（横向 `x = 130 + 140 × n`）。所以做架空或古代背景的 Mod **可以直接写 1200、1250**，
> 完全不受 1936–1948 的限制。原版看着像"引擎画的年份轴"，其实是一行行写死的文本框。

### ③ 界面 —— 页签按钮

在同一文件名为 `folder_tabs` 的容器内：

```
buttonType = {
    name = "mymod_infantry_folder_tab"
    position = { x = 200 y = 0 }
    quadTextureSprite = "GFX_infantry_folder_tab"
    frame = 1
}
```

名字必须是 `<folder_id>_tab`。横向位置是等距的：**`x = 22 + 89 × n`**，
也就是 22 / 111 / 200 / 289 / 378 / 467 / 556 / 645 / 734 / 823 / 912。
贴图可以直接复用原版的，省去画图。

### ④ 界面 —— 节点模板 🔴 最容易漏的一处

文件后半段有一批与 folder 平级的容器，名叫 `techtree_<folder_id>_item`。
**这是画科技节点本身的模板，一个 folder 一套。**

最省事的做法：整段复制原版某个同类 folder 的模板，只改名字。

```
containerWindowType = {
    name = "techtree_mymod_infantry_folder_item"
}
```

漏掉它的症状极具欺骗性：**页签出现、标题正常、年份刻度正常，只有科技一个都不画。**
看起来像"坐标又放偏了"，实际是节点渲染器根本没建起来。

### ⑤ 本地化

**folder 名的本地化 key 就是 folder id 本身**：

```
l_simp_chinese:
 mymod_infantry_folder:0 "自建步兵"
 MYTECH_root:0 "我的科技"
 MYTECH_root_desc:0 "科技说明。"
```

> 🔴 `.yml` **必须带 UTF-8 BOM**，否则整份文件静默失效。而 `.txt` 和 `.gui` **不能带 BOM**。

### ⑥ 补充：科技图标

图标精灵名的约定是 **`GFX_<tech_id>_medium`**，声明在 `interface/` 下的 `.gfx` 里。
没有对应精灵时节点显示成空框——这也是静默失败。

## 失败模式速查表

| 症状 | 原因 | 日志 |
|---|---|---|
| 科技完全不显示，其他一切正常 | 根科技没有对应格子 | ✅ `countrytechtreeview.cpp: Found no grid box for tech <id>` |
| **整页科技都不显示**，页签/标题/年份全正常 | 缺 `techtree_<folder_id>_item` | ⚠️ `gui.cpp: Undefined GUI_TYPE: …`，**只在点开该页签后才打印** |
| 页签整个不出现 | `available` 为假 / 没写 `buttonType` / 没在 `technology_folders` 声明 | ❌ 无 |
| 页签在，点了空白 | 容器名字与 folder id 不一致 | ❌ 无 |
| **页签和节点都不出现** | folder 容器里漏了 `techtree_stripes` 画布 | ❌ 完全静默 |
| 科技画出来了但在别人的链条里 | 坐标两轴搞反 | ❌ 无 |
| 科技显示成空框 | 缺 `GFX_<tech_id>_medium` | ❌ 常常没有 |
| 名称显示成 key | 本地化缺失，或 `.yml` 没有 BOM | ❌ 无 |
| **什么都不显示，日志完全干净** | 放进了**被 DLC 替换的 folder** | ❌ 完全静默 |

### 最后一条单独强调：不要用有 DLC 替换关系的 folder

| 基础版 | 被替换成 | DLC |
|---|---|---|
| `armour_folder` | `nsb_armour_folder` | No Step Back |
| `air_techs_folder` | `bba_air_techs_folder` | By Blood Alone |
| `naval_folder` | `mtgnavalfolder` | Man the Guns |

六个名字在 1.19.2 里同时存在，但**同一时刻只有一个会绘制**。玩家一旦开了对应 DLC，
挂在左列那三个上的科技就凭空消失，零报错。

`infantry_folder` / `industry_folder` / `electronics_folder` / `support_folder` / `artillery_folder`
没有替换关系，可以放心用。

## 推送前的四条自检

改 `.gui` 这种大文件，只数总数是不够的：

1. **全文件花括号平衡**——务必**先剥掉 `#` 之后的内容再数**。总数对得上也可能是两处错误互相抵消。
2. **与原版逐行 diff，删除行数必须为 0**——防止变换脚本误伤原版。
3. **每个根科技都有对应的 `<id>_tree` 格子。**
4. **每个自建 folder 都有 `techtree_<id>_item`。**
5. **每个自建 folder 容器里都有 `techtree_stripes`。** 名字固定,漏了就整页不显示且零日志。

做完这四条，**仍然要进游戏用眼睛看**：页签在不在、点进去标题和年份刻度对不对、
科技画出来没有、画在哪一格、鼠标悬停显示的是文字还是 key。

这一层里，"日志干净"和"做对了"是两件事。

## 接下来看什么

- 做一门科技：解锁什么、什么时候能研究 (`technology-system`)——用原版已有页签时只需要这一篇。
- 做一门解锁新步兵武器的科技 (`tech-unlock-equipment`)——从空白节点到生产线可选的完整案例。
