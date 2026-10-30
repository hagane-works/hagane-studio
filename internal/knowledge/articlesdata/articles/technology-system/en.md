---
contentId: technology-system
kind: guide
slug: technology-system
locale: en
status: published
title: "Authoring a technology: what it unlocks, and when it can be researched"
summary: "What the player gets when research finishes, and the separate question of when the technology may be researched or even seen. These are different blocks."
primaryTopic: research
topics: [research]
scenarios: [technology]
gameTerms: [technology, research, tech tree, research screen, prerequisite, ahead of time penalty]
aliases: [how to add a technology, custom technology, tech tree modding, technology that unlocks equipment, technology]
relatedIds: [research-speed, tech-unlock-equipment, "wiki:effect:set_technology", "wiki:effect:add_tech_bonus", "wiki:modifier:research_speed_factor"]
updatedAt: "2026-09-09"
verification: static-checked
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Technology"
    url: "https://hoi4.paradoxwikis.com/Technology"
  - label: "HOI4 Wiki: Modifiers"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
---

# Authoring a technology: what it unlocks, and when it can be researched

## What this system covers

Every clickable node on the research screen is one technology. The player spends research time on it and, once it completes, receives either a permanent unlock or a lasting bonus.

As an author you really only decide three things:

1. **What the player gets when it completes** (a new piece of equipment? a country-wide bonus?)
2. **Under what conditions it may be researched, and under what conditions it is visible at all**
3. **Which page of the research screen it sits on, and where**

In script these are three unrelated groups of fields. Most authoring problems come from collapsing two of them into one.

## Core idea: unlocking an object is not the same as changing a number

What a technology can do falls into three categories that behave differently:

| Category | What you write | When it happens |
| --- | --- | --- |
| **Unlock an object** | `enable_equipments`, `enable_subunits`, `enable_equipment_modules`, `enable_building`, `enable_tactic` | Permanently available from the moment it completes |
| **A lasting numeric effect** | a `modifier = { ... }` block inside the technology | Applies from completion onward, stacking with national spirits |
| **A one-shot script** | `on_research_complete = { ... }` | Runs once at the moment of completion, never again |

The first gives the player new **things** (another item on the production line, another battalion available in the division template). The second changes **numbers** that already exist. The third is a one-time event, used to send a notification, load an order of battle, or start a story beat.

"After this technology, all infantry gain 5% soft attack" is the second category. "After this technology, fire an event" is the third. They are not interchangeable.

## File structure and how it runs

Technologies live under `common/technologies/`, wrapped in a single `technologies = { ... }` block with one entry per technology:

```
technologies = {
    my_technology_1 = {

    }
    my_technology_2 = {

    }
}
```

The unlock fields each have their own shape. Watch which ones take braces:

```
enable_equipments = { my_equipment_1 my_equipment_2 }
enable_subunits = { my_subunit_1 }
enable_equipment_modules = { my_module_1 }
enable_tactic = my_tactic
enable_building = {
    building = my_building
    level = 5
}
```

Nodes are linked with `path`, and placement goes in a `folder` block:

```
path = {
    leads_to_tech = my_technology_2
    research_cost_coeff = 1
}
folder = {
    name = my_folder
    position = { x = 0 y = 1 }
}
```

The technology ID is unique across the whole mod and may contain only lowercase letters and underscores, no spaces and no non-ASCII characters. Scripts such as `has_tech`, `set_technology` and `add_tech_bonus` refer to it by name, so renaming it breaks every reference at once.

## Common goals and where to start

| What you want to see in game | Start here | The distinction that matters |
| --- | --- | --- |
| After research, a new buildable item appears on the production line | `enable_equipments` | The equipment must already be defined under `common/units/equipment/` |
| After research, a new battalion can be added to division templates | `enable_subunits` | This unlocks a unit type, not a specific unit in the field |
| After research, a new part appears in the tank or aircraft designer | `enable_equipment_modules` | Affects designer options only; already-built equipment is untouched |
| After research, a new building can be constructed | `enable_building` | Block form: it names both the building and the level |
| After research, the country gains a lasting bonus | a `modifier = { ... }` inside the technology | Continuous, using the same stats as national spirits |
| Fire an event or load an order of battle at the moment of completion | `on_research_complete` | Runs once; pair it with `on_research_complete_limit` to prevent re-entry |
| The player may never research it directly, only receive it from a story beat | `allow = { always = no }`, then grant it with unlock a specific technology directly (`wiki:effect:set_technology`) | The node stays visible, just greyed out |
| Make a category of research faster rather than unlocking something | raise or lower research speed continuously (`wiki:modifier:research_speed_factor`) or grant a limited-use research bonus (`wiki:effect:add_tech_bonus`) | These belong to research speed, not to a technology definition |

## Easy things to get wrong

### `allow` controls clickability, `allow_branch` controls visibility

Both are condition blocks whose names differ by one word, and they do different jobs:

- `allow = { ... }` is checked continuously. When it fails the technology **still appears on the research screen**; it simply cannot be researched manually.
- `allow_branch = { ... }` governs visibility. When it fails, that technology **and the technologies connected to it disappear from the screen entirely**.

The base game pairs `allow_branch` with `has_dlc` to hide a whole DLC-only branch from players who do not own it. If what you wanted was "visible but not yet clickable", writing `allow_branch` means the player never finds it at all.

### On cost stats, a minus sign is the buff

Some stats run opposite to intuition. `build_cost_ic = -0.1` means production cost **drops by 10%**, which is a good bonus; a positive value makes the unit more expensive instead. Technologies and doctrines commonly sit in the -5% to -15% range here.

Getting the sign backwards raises no error and is not caught on export. It only shows up when you compare the production queue in game. Before filling in a cost stat, confirm whether you meant cheaper or more expensive.

### A position is a relative offset, not an absolute coordinate — and you must check it in game

Both numbers in `position` are **relative offsets**, not screen coordinates. The base game declares years as constants and uses them on `y`:

```
@1936 = 2
@1938 = 4

folder = {
    name = infantry_folder
    position = { x = 0 y = @1936 }
}
```

🔴 **`y` is the horizontal axis and `x` is the vertical one — the opposite of what you would guess.**

- Increasing `y` moves the node **to the right**. Research runs left to right on screen; the year axis is horizontal.
- Increasing `x` moves the node **down one row**. That is how parallel branches are separated.

Here is what the base game does (`infantry_folder` on 1.19.2):

```
infantry_weapons1          x=0  y=2 (@1936)
infantry_weapons2          x=0  y=4 (@1938)
improved_infantry_weapons  x=0  y=6 (@1939)
```

`x` stays at 0 along the chain while `y` climbs — and in game that chain is drawn **horizontally**.
The armoured car line starts at `armored_car1` with `x=4`, four rows further down.

⚠️ **`start_year` has nothing to do with `position`.** The former only sets the baseline year for the
ahead-of-time penalty; the latter only decides which cell the node occupies. A mismatch raises no error,
though it will confuse players. Writing year constants into `y` is a base-game **convention**, not an
engine rule, so an alternate-history or ancient-era mod is free to define its own year steps.

Getting the direction wrong raises no error at all: the node is still drawn, it just **lands inside
somebody else's chain**, reading as "this tech line has a strange extra node". Two nodes on the same cell
overlap outright. `error.log` says nothing about any of this.

### 🔴 For a folder of your own, the technology definition alone draws nothing

The examples above use `infantry_folder` and similar folders that **already exist in the base game**. If you
want a research tab of your own, writing `common/technologies/` is not enough — the technologies will
exist and none of them will be drawn.

A folder needs three things present at once:

```
① technology definition   common/technologies/*.txt          → which folder, which cell
② folder declaration      common/technology_tags/*.txt       → the folder exists, and which ledger it counts under
③ the interface trio      interface/countrytechtreeview.gui  → container + grid box + node template
```

Plus a tab button and localisation, or the player cannot click in, or sees raw keys. **Most tutorials cover only ①.**

The widely repeated claim that "custom technologies never appear in the tech tree" is only half true. The
accurate statement is that **you have to add the declarations to `countrytechtreeview.gui` yourself; once
they are there it draws fine.** The full recipe is on a research tab of your own (`custom-tech-folder`).

### 🔴 Never put a new technology in a folder that a DLC replaces

Three base-game folders are swapped out wholesale by a DLC:

| Base-game folder | Replaced by |
| --- | --- |
| `armour_folder` | No Step Back → `nsb_armour_folder` |
| `air_techs_folder` | By Blood Alone → `bba_air_techs_folder` |
| `naval_folder` | Man the Guns → `mtgnavalfolder` |

All six names exist simultaneously in 1.19.2, but **only one of each pair is ever drawn**. Attach a new
technology to the replaced one and, the moment a player owns that DLC, your technology **vanishes with
no error and a completely clean log**.

`infantry_folder`, `industry_folder`, `electronics_folder`, `support_folder` and `artillery_folder` have no
replacement and are safe to use.

### Researching before the historical year is deliberately slowed

`start_year` is the technology's historical availability year. Researching it earlier triggers the ahead-of-time penalty: **each year in advance adds +200% to the cost**. This is engine behaviour, not a bug. If you want a technology to be researched comfortably from game start, set `start_year` to the start year or earlier.

Its companion is `research_cost`, a relative multiplier on research time: `research_cost = 1` is roughly 100 days at default speed. Base-game trunk technologies cluster between 1.5 and 2.5.

### Version and scope

This page was compiled against the bundled documentation and base-game scripts labelled 1.19.2.0. The designer XP acceleration fields (`xp_research_type`, `xp_boost_cost`) belong to DLC mechanics and have no effect in a save without the matching DLC.

## Where to go next

- For a research tab of your own, see why the technology exists but nothing is drawn (`custom-tech-folder`).
- Worked example: a technology that unlocks a new infantry weapon (`tech-unlock-equipment`), from an empty node through to selecting the equipment on the production line.
- To adjust research speed rather than unlock content, see research speed, research bonuses and technology unlocks (article `research-speed`).
- To grant a one-off research bonus when a focus completes, see a focus that grants one 50% industry research bonus (`focus-research-bonus`).
