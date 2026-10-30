---
contentId: custom-tech-folder
kind: guide
slug: custom-tech-folder
locale: en
status: published
title: "A research tab of your own: why the technology exists but nothing is drawn"
summary: "Custom technologies not showing up is not an engine limit — the interface layer is yours to fill in. Five changes, none optional, and three of the failure modes say nothing in the log."
primaryTopic: research
topics: [research]
scenarios: [technology]
gameTerms: [tech tree, research screen, research tab, folder, node template, localisation]
aliases: [custom tech tree, custom technology not showing, technology invisible, add a research tab, tech not drawn]
relatedIds: [technology-system, tech-unlock-equipment, research-speed]
updatedAt: "2026-09-09"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Technology"
    url: "https://hoi4.paradoxwikis.com/Technology"
  - label: "HOI4 Wiki: Interface"
    url: "https://hoi4.paradoxwikis.com/Interface"
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

# A research tab of your own: why the technology exists but nothing is drawn

## What this page is for

A claim repeated in a lot of places:

> Custom technologies never appear in the tech tree. The research screen only renders technologies that
> fall inside the fixed set of grid boxes declared in `interface/countrytechtreeview.gui`, so a custom
> technology has nowhere to be drawn.

**That is only half right.** The accurate statement is: **a custom technology needs you to add the
declarations to that file yourself, and once they are there it draws fine.**

This page covers what to add, where, and what each omission looks like. If you only want to add a
technology under a folder the base game already has, you do not need this page — read
authoring a technology (`technology-system`) instead.

## A technology needs three layers present at once

```
① technology definition   common/technologies/*.txt          → which folder, which cell
② folder declaration      common/technology_tags/*.txt       → the folder exists, which ledger it counts under
③ the interface trio      interface/countrytechtreeview.gui  → container + grid box + node template
```

Plus a tab button and localisation, or the player cannot click in, or sees raw keys.

**Most tutorials cover only ①**, so ② and ③ are missing and the technology "exists but is invisible".

### One rule that runs through all of this: verify by eye

A good share of the failures in this layer **never reach `error.log`**, and some of the rest only print
once you actually open that tab. "The log is clean" does not mean "it was drawn", let alone "drawn in the
right place".

## Rule one: how a grid box is named

A `gridboxtype` is a cell container in the interface, written **inside that folder's own container**:

```
gridboxtype = {
    name = "support_weapons_tree"
    position = { x = 140 y = 210 }
    slotsize = { width = 70 height = 70 }
    format = "LEFT"
}
```

> **Grid box name = root technology id + `_tree`**

Technologies chained together by `path = { leads_to_tech = … }` **share that one box**.

**Precise definition of a root technology**: within this folder, a technology not pointed at by any `path`
from the same folder. Three details that catch people out:

1. **"Root" is relative to the folder.** A technology can declare several `folder = { }` blocks and appear
   on several pages. It may be downstream on page A and a root on page B, in which case page B needs a box of its own.
2. **Only `path` creates the shared-cell relationship.** Technologies have no `prerequisites` key —
   writing one raises `Unknown modifier: prerequisites`, and it appears zero times in the base game's
   `common/technologies/`. Two technologies with no `path` between them are two roots needing two boxes.
3. **A commented-out `folder` block does not count.** Several base-game technologies have their `folder`
   commented out with `#`; they are not on that page.

Cross-checked across the base game's 12 folders: the count of "has a root technology but no box" is **0**.
Surplus boxes are harmless — when in doubt, leave them.

## Rule two: coordinates are relative offsets

The base game declares years as constants and uses them on `y`:

```
@1918 = 0
@1936 = 2
@1938 = 4

folder = {
    name = infantry_folder
    position = { x = 0 y = @1936 }
}
```

🔴 **`y` is the horizontal axis and `x` is the vertical one — the opposite of what you would guess.**
Increasing `y` moves right; increasing `x` moves down one row.

The base game's `infantry_folder` shows it: `infantry_weapons1` (x=0 y=@1936) → `infantry_weapons2`
(x=0 y=@1938) → `improved_infantry_weapons` (x=0 y=@1939). `x` stays at 0 along the chain while `y`
climbs, and in game that chain runs **horizontally**. The armoured car line starts at `armored_car1`
with `x=4`, four rows further down.

⚠️ **`start_year` has nothing to do with `position`**: the former sets the ahead-of-time baseline year,
the latter picks the cell. Writing year constants into `y` is a base-game **convention**, not an engine
rule, so an alternate-history or ancient-era mod can define its own year steps.

Getting the direction wrong raises no error; the node is drawn, just inside somebody else's chain.

## The five changes

Taking a new page called `mymod_infantry_folder` as the example. **None of the five is optional.**

### ① `common/technology_tags/00_technology.txt` — declare the folder

Shadow the base-game file by name and add, inside `technology_folders = { }`:

```
mymod_infantry_folder = {
    ledger = army
}
```

`ledger` picks which statistics ledger it counts under: `army` / `navy` / `air` / `civilian` / `hidden`.
The optional `available = { }` is a trigger; when it fails **the whole tab is not drawn**.

⚠️ This file also holds a large `technology_categories = { }` block. Every name used in a technology's
`categories = { }` **must be registered there first** (the base game registers 134, uses 105, and leaves
0 unregistered — it follows its own rule strictly).

### ② Interface — the folder container

In `interface/countrytechtreeview.gui`, add a sub-container **inside** the container named
`countrytechtreeview`. **Its name must match the folder id exactly**:

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

🔴 **The name `techtree_stripes` is fixed. Do not rename it, and do not drop it because "it looks like
just a background".** The nodes are drawn on that canvas; without it the tab and the nodes both fail to
appear, and **the log says nothing at all**. Its `size` is the canvas size — enlarge it if the tree does
not fit.

`%%` is the escape form used in gui files. **Copy it as-is; do not reduce it to a single `%`.**

A name that does not match the folder id shows up as: the tab is there, clicking it gives a blank page.

> ⭐ **The year scale and the row titles are not generated by the engine.** They are literal
> `instantTextBoxType` entries written by hand inside `techtree_stripes` (horizontally at
> `x = 130 + 140 × n`). So an alternate-history or ancient-era mod **can simply write 1200 or 1250** and is
> not bound to 1936–1948 at all. What looks like an engine-drawn year axis is a series of hardcoded text boxes.

### ③ Interface — the tab button

Inside the container named `folder_tabs` in the same file:

```
buttonType = {
    name = "mymod_infantry_folder_tab"
    position = { x = 200 y = 0 }
    quadTextureSprite = "GFX_infantry_folder_tab"
    frame = 1
}
```

The name must be `<folder_id>_tab`. Horizontal placement is evenly spaced at **`x = 22 + 89 × n`**,
i.e. 22 / 111 / 200 / 289 / 378 / 467 / 556 / 645 / 734 / 823 / 912. You can reuse a base-game tab sprite
and skip the artwork.

### ④ Interface — the node template 🔴 the one most often missed

The later part of the file holds a set of containers at the same level as the folders, named
`techtree_<folder_id>_item`. **This is the template that draws the technology nodes themselves, one set per folder.**

The easy route: copy a similar base-game folder's template wholesale and change only the name.

```
containerWindowType = {
    name = "techtree_mymod_infantry_folder_item"
}
```

Missing it produces a deceptive symptom: **the tab appears, the title is fine, the year scale is fine, and
not a single technology is drawn.** It looks like "the coordinates are off again"; in fact the node renderer
was never built.

### ⑤ Localisation

**A folder's localisation key is the folder id itself**:

```
l_english:
 mymod_infantry_folder:0 "Foot Troops"
 MYTECH_root:0 "My Technology"
 MYTECH_root_desc:0 "What this technology does."
```

> 🔴 A `.yml` **must carry a UTF-8 BOM** or the whole file silently does nothing. A `.txt` or `.gui`
> **must not** carry one.

### ⑥ Optional: the technology icon

The sprite naming convention is **`GFX_<tech_id>_medium`**, declared in a `.gfx` file under `interface/`.
With no matching sprite the node renders as an empty frame — another silent failure.

## Failure mode reference

| Symptom | Cause | Log |
|---|---|---|
| One technology never shows, everything else fine | its root technology has no grid box | ✅ `countrytechtreeview.cpp: Found no grid box for tech <id>` |
| **No technology on the page is drawn**, tab/title/year scale fine | `techtree_<folder_id>_item` missing | ⚠️ `gui.cpp: Undefined GUI_TYPE: …`, **only printed after you open that tab** |
| Tab never appears | `available` false / no `buttonType` / not declared in `technology_folders` | ❌ none |
| Tab appears, clicking gives a blank | container name does not match the folder id | ❌ none |
| **Neither the tab nor the nodes appear** | the `techtree_stripes` canvas is missing from the folder container | ❌ fully silent |
| Technology drawn, but inside someone else's chain | the two coordinate axes were swapped | ❌ none |
| Technology renders as an empty frame | `GFX_<tech_id>_medium` missing | ❌ often none |
| Names render as keys | localisation missing, or the `.yml` has no BOM | ❌ none |
| **Nothing shows and the log is completely clean** | placed in a **DLC-replaced folder** | ❌ fully silent |

### The last row deserves its own warning

| Base game | Replaced by | DLC |
|---|---|---|
| `armour_folder` | `nsb_armour_folder` | No Step Back |
| `air_techs_folder` | `bba_air_techs_folder` | By Blood Alone |
| `naval_folder` | `mtgnavalfolder` | Man the Guns |

All six names coexist in 1.19.2, but **only one of each pair is ever drawn**. The moment a player owns the
matching DLC, technologies attached to the left column vanish, with no error at all.

`infantry_folder`, `industry_folder`, `electronics_folder`, `support_folder` and `artillery_folder` have no
replacement and are safe.

## Four checks before you ship

Editing a file the size of `.gui` needs more than counting totals:

1. **Brace balance across the file** — strip everything after `#` **before** counting. A total that adds up
   can still be two errors cancelling each other out.
2. **Line-by-line diff against the base game; deleted lines must be 0** — this catches a transform script
   damaging vanilla content.
3. **Every root technology has its `<id>_tree` box.**
4. **Every folder of your own has a `techtree_<id>_item`.**
5. **Every folder container of your own holds a `techtree_stripes`.** The name is fixed; without it the whole page fails to render, silently.

After all four, **still open the game and look**: is the tab there, do the title and year scale look right
inside it, are the technologies drawn, are they in the right cells, and does hovering show text rather than a key.

In this layer, "the log is clean" and "it is correct" are two different statements.

## Where to go next

- Authoring a technology: what it unlocks, and when it can be researched (`technology-system`) — for an existing base-game tab, that page is all you need.
- A technology that unlocks a new infantry weapon (`tech-unlock-equipment`) — a worked example from empty node to production line.
