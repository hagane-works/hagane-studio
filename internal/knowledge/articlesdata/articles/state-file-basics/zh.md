---
contentId: state-file-basics
kind: guide
slug: state-file-basics
locale: zh
status: published
title: "州文件：归属、核心、省份和建筑各写在哪"
summary: "一个州就是一份文件：谁拥有、谁有核心、由哪些省份组成、有什么建筑。改前两样在工作台里点几下就行；改省份组成会牵动地图那一层，那里有五条引擎硬规矩，违反的后果从一条警告到主菜单进不去。"
primaryTopic: territory
topics: [territory]
scenarios: [history]
gameTerms: [州, 地块, 省份, 归属, 核心, 宣称, 控制者, 人力, 胜利点, 建筑, 战略区, 州编号]
aliases: [州文件怎么写, 改州归属, 加核心, 州名不生效, 地图加载失败, 州编号, 一个省两个州, history states, state file]
relatedIds: [focus-transfer-state-core, recut-states-without-redrawing-map, reference-image-to-states, "wiki:effect:transfer_state", "wiki:effect:add_state_core", "wiki:trigger:owns_state"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki：州制作"
    url: "https://hoi4.paradoxwikis.com/State_modding"
  - label: "HOI4 Wiki：地图制作"
    url: "https://hoi4.paradoxwikis.com/Map_modding"
  - label: "HOI4 Wiki：本地化"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

# 州文件：归属、核心、省份和建筑各写在哪

## 这个系统负责什么

地图上每一块有归属的土地都是一个**州**。工作台里叫它「地块」。国家拥有哪些州、哪些州是核心、州里有几座工厂、人力多少，全写在 `history/states/` 下这个州自己的那份文件里。

作为作者，你在一个州上能决定的事分三层：

1. **它是谁的**：所有者、控制者、核心、宣称。
2. **它里面有什么**：人力、类别、资源、建筑、胜利点。
3. **它由哪些省份组成**。

前两层怎么改都只影响这一份文件，引擎不会拿地图来校验。第三层不同：省份是地图的最小单位，改一个州的省份组成，引擎会用五条硬规矩来检查，本文第五节逐条讲。这也是为什么工作台把前两层放在主区，把「省份」放在更多字段里。

## 核心概念：改属性和改省份不是同一个量级

| 你改的是 | 引擎怎么对待 | 做错了的后果 |
| --- | --- | --- |
| 所有者、控制者、核心、宣称 | 照写照读，不校验 | 最多是首都指到了不归自己的州，引擎拒绝并回落 |
| 人力、类别、资源、建筑 | 照写照读 | 建筑超过该类别的槽位上限时报一条错 |
| 名字 | 走本地化 | 键写错就显示原始键名 |
| **省份列表** | **五条硬规矩** | 从一条警告，到主菜单「地图加载失败」，到启动崩溃 |

想做"把这个州开局给另一个国家""给某国加一个核心"，只动第一层，工作台里改「所有者」「核心」就完事。想做"把两个州合并""从一个州里切一块出来"，就进了第三层，请先读完第五节。

## 文件结构与运行过程

一个州一份文件，放在 `history/states/`。原版的卢森堡长这样，只有两个省：

```
state = {
    id = 8
    name = "STATE_8"
    state_category = town
    manpower = 294748
    resources = {
        steel = 14
    }
    history = {
        owner = LUX
        add_core_of = LUX
        buildings = {
            infrastructure = 3
            arms_factory = 1
            industrial_complex = 3
        }
        victory_points = { 6583 5 }
    }
    provinces = {
        6583 13375
    }
}
```

| 键 | 含义 | 工作台里叫什么 |
| --- | --- | --- |
| `id` | 州编号，全图唯一，必须从 1 连续到最大值 | 「地块 id」，只读 |
| `name` | 州名的本地化键，不是显示文本 | 「名称」，只读；改名用「显示名」 |
| `state_category` | 类别，决定建筑槽位数 | 「地块类别」 |
| `manpower` | 人口 | 「人力」 |
| `history` 块 | 开局状态：所有者、控制者、核心、宣称、建筑、胜利点 | 主区那几项 |
| `buildings` 块 | 州级建筑直接写；省级建筑（港口、碉堡）套在省份编号下面 | 「建筑(地块级)」和「省级建筑」 |
| `victory_points` | 省份编号加分值 | 「胜利点」 |
| `provinces` | 这个州由哪些省份组成 | 「省份」 |

`history` 块里还可以放带日期的子块，比如 `1939.1.1 = { ... }`，在对应剧本开局时才执行；工作台会把它们原样保留，不在表单里展开。

### 文件名决定它是"覆盖"还是"再来一份"

同名文件替换原版；**文件名不同、里面的编号却相同**，引擎会报 `State ID conflict` 并列出两个文件，然后**后加载的那份赢**。所以覆盖原版某个州时，文件名必须和原版一模一样，包括原版的拼写习惯（卢森堡那份叫 `8-Luxemburg.txt`，不是 Luxembourg）。原版的文件名也不全是「编号-州名」，有 `560-TS 2.txt` 这种，要从原版目录里照抄，不要自己推。

工作台导出时会沿用原版或导入时的文件名，这一条你不用管。

### 州名和地图标签是两个键

州名走 `STATE_<编号>`，地图上城市旁边那行字走 `VICTORY_POINTS_<省份编号>`。只改前者，地图上顶着的还是原版地名。两个都放进本地化文件：

```
l_simp_chinese:
 STATE_8: "大公国"
 VICTORY_POINTS_6583: "卢森堡城"
```

覆盖原版的键，放在 `localisation/<语言>/replace/` 目录最稳。放在普通目录也生效，但引擎会在 `logs/text.log` 记一条"重复键"，`error.log` 里多一行"loc key collisions"。工作台的「显示名」字段写的就是这样一条覆盖，内部键不变。

## 常见制作目标与对应方法

| 你想在游戏里看到 | 从这里开始 | 关键区别 |
| --- | --- | --- |
| 开局这个州归另一个国家 | 「所有者」 | 只改所有者不加核心，那个国家拿到的是非核心领土 |
| 某国对这个州有核心或宣称 | 「核心」「宣称」 | 核心和宣称是两回事，可以同时有 |
| 开局被占领：一国拥有、另一国控制 | 「控制者」 | 留空等于和所有者相同 |
| 改州名 | 「显示名」 | 地图标签另改 `VICTORY_POINTS_<省份>` |
| 多几座工厂、更好的基建 | 「建筑(地块级)」 | 上限由「地块类别」决定 |
| 港口、碉堡 | 「省级建筑」，按省份编号放 | 港口建筑和港口位是两回事，见第五节 |
| 剧情里转交州、加核心 | 用国策或事件效果 | 见国策完成后转交一个州并加核心 (`focus-transfer-state-core`) |
| 把省份重新分组成新的州 | 「省份」框，或者直接写文件 | 第三层，见第五节和不重画地图，重切州 (`recut-states-without-redrawing-map`) |

## 容易混淆的地方

### 🔴 州编号必须从 1 连续到最大值，缺一个就进不了游戏

引擎从 1 数到最大编号，缺一个报一条 `Missing State ID`，接着一条 `Definition for state id N is either missing or invalid`，然后**主菜单直接判定地图加载失败**，单人游戏点不了。实测：原版最大编号是 1081，新建一个 1083 而跳过 1082，主菜单就进不去；改成 1082 就正常。

所以新州只能接在现有最大编号后面，一个一个排。想"按区域留空位"的直觉在这里用不了。

### 🔴 一个州的所有省份必须属于同一个战略区

战略区是 `map/strategicregions/` 里定义的硬边界。把两个战略区的省份塞进同一个州，主菜单同样地图加载失败，`error.log` 里会把两个战略区的名字都写出来：

```
MAP_ERROR: State "..."(#1082) has provinces (#13375, #3305) belonging to different strategic areas: "..." (#5), "..." (#275)
```

这条决定了重切州只能顺着战略区切，不能顺着历史区划切。

### 🔴 一个省份只能属于一个州

同一个省份写进两个州，游戏在启动阶段崩溃。这是最严重的一种。工作台的导出前地图检查会把它列为"下载前需处理"，不让你把这样的产物导出去。

### 🔴 州文件不能带 BOM

带了 BOM 的 `.txt`，引擎从第一个字符就读不懂，报 `Unexpected token: ï»¿state`，整份文件作废。那个州于是不存在，编号缺失，连锁触发上一条"地图加载失败"。本地化文件正相反，必须带 BOM。两个方向的规则相反，用编辑器另存时看一眼编码。

### 改了省份归属，`map/buildings.txt` 要跟着改

那份文件六万多行，第一列是州编号。你把一个省份从州 8 挪到州 1082，原版指向它的十几行建筑位就对不上了。引擎不崩，但把这些行全部丢弃，并且**把正确答案印在报错里**：

```
map/buildings.txt error at line 50927: map building(民用工厂) location is not within specified state. Supposed to be '8 - ...' but was '1082 - ...'for province 13375. BUILDING IGNORED!
```

"Supposed to be"后面是文件里写的，"but was"后面是引擎自己算出来的。修法就是照引擎说的改第一列，不要自己去猜省份落在哪个州。丢掉的如果是港口位，另一位作者的实测里游戏会闪退；本文这一条只实测了内陆省份，没崩。

### 首都必须指向自己拥有的州

首都指向不归自己的州，引擎报 `Attempting to set capital state #N for ..., they dont own it!`，拒绝设置并自己挑一个。不崩，但另一位作者的实测里有一次直接闪退。工作台的导出前检查会提醒这一条。

### 陆地省份没有州：不崩，进游戏弹提示

一个陆地省份不属于任何州，游戏能进，进去后弹 MAP_ERROR，`error.log` 记 `The land province N has no state`。

### 州引用了不存在的省份：那一个引用被丢，其余照常

`error.log` 记一条 `Malformed token`，州的其他省份正常。

### 版本与适用边界

本文按标记为 1.19.2.0 的原版文件整理。原版有 1081 个州、13414 个省份，其中陆地 10155、湖泊 126、海洋 3133；湖泊也属于州，切州时按"排除海洋"筛选，不要按"只要陆地"筛选，否则会漏掉湖。

## 接下来看什么

- 在国策或事件里转交州、加核心、只占领不拥有：国策完成后转交一个州并加核心 (`focus-transfer-state-core`)。
- 把原版一千多个州推倒重来而不重画地图：不重画地图，重切州 (`recut-states-without-redrawing-map`)。
- 一个纯造型的极端例子：把一张参考图切成地块 (`reference-image-to-states`)。
- 效果与条件：转交州 (`wiki:effect:transfer_state`)、添加核心 (`wiki:effect:add_state_core`)、检查是否拥有某州 (`wiki:trigger:owns_state`)。
