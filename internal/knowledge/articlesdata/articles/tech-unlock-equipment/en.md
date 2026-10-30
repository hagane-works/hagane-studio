---
contentId: tech-unlock-equipment
kind: case
slug: tech-unlock-equipment
locale: en
status: published
title: "A technology that unlocks a new infantry weapon"
summary: "Add a 1936 node to the army folder: on completion the equipment appears on the production line and infantry soft attack rises by 5%."
primaryTopic: research
topics: [research]
scenarios: [technology]
gameTerms: [technology, unlock equipment, research screen, infantry equipment, soft attack]
aliases: [technology that unlocks equipment, add a tech that unlocks a weapon, new weapon technology, tech soft attack bonus]
relatedIds: [technology-system, research-speed, "wiki:effect:set_technology"]
updatedAt: "2026-09-10"
verification: source-checked
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Technology"
    url: "https://hoi4.paradoxwikis.com/Technology"
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

# A technology that unlocks a new infantry weapon

## The goal and what you should be able to see

When this is finished, three things should be true in game:

1. A new node appears in the army folder of the research screen, showing a readable name rather than a string of underscores.
2. Once researched, the infantry equipment can be selected in the production line's equipment dropdown.
3. Once researched, a division containing infantry battalions shows higher soft attack than before.

If the first is true but the second is not, the problem is almost always the equipment ID. The last section lists the order to check things in.

## Before you start

- A mod that already exports cleanly.
- **The infantry equipment must already exist.** A technology only unlocks; it does not create equipment. Pointing at an equipment ID that does not exist raises no error, the entry simply does nothing. This is the most common way this case fails.
- No DLC is required. Every field used here belongs to the base game.
- Reading authoring a technology: what it unlocks, and when it can be researched (`technology-system`) first is recommended, for why "unlocking an object" and "a lasting modifier" are different things.

## The objects involved

| Object | Where it lives | Its job here |
| --- | --- | --- |
| Technology definition | a file under `common/technologies/`, inside `technologies = { ... }` | the node itself |
| Infantry equipment | `common/units/equipment/` | the object being unlocked; must already exist |
| Name and description | the YML for your language under `localisation/` | whether the screen shows a name or a raw key |
| Icon | a `.gfx` file under `interface/` | when missing the node has a blank icon; research still works |

## 1. Create a node you can see

Start with only "where, how expensive, when available" and no unlocks at all:

```
technologies = {
    my_infantry_smg_1 = {
        research_cost = 1.5
        start_year = 1936

        folder = {
            name = infantry_folder
            position = { x = 1 y = 2 }
        }
    }
}
```

`research_cost = 1.5` is roughly 150 days at default speed, which is where the base game's infantry line usually sits. `start_year = 1936` lets it be researched normally from game start; setting 1939 instead would make a 1936 start pay three years of ahead-of-time penalty, at +200% cost per year early.

Both numbers in `position` are **relative offsets**, not absolute coordinates, and **`y` is the horizontal axis while `x` is the vertical one**: increasing `y` moves the node to the right (the year direction; the base game defines 1936 as `y = 2`), increasing `x` moves it down one row. `x = 1`, `y = 2` above puts it one row below the infantry weapons trunk, in the 1936 column. If the new node overlaps a neighbour, change `x` to move rows; to place it to the right of its prerequisite, raise `y`. Getting the direction wrong raises no error; the node simply lands in somebody else's chain, so check it in game.

Load the game at this point and the army folder should show a clickable node, still labelled `my_infantry_smg_1`.

## 2. Attach it to a prerequisite

Hang it off an existing infantry weapons technology. The `path` goes in the **prerequisite's** definition and points at the new technology:

```
infantry_weapons = {
    path = {
        leads_to_tech = my_infantry_smg_1
        research_cost_coeff = 1
    }
}
```

🔴 **Technologies have no `prerequisites` key.** `path` does both jobs at once: it draws the connecting
line and establishes "the parent must be researched first". Writing `path` is all that is needed; there is
nothing to declare on the new technology.

Writing `prerequisites = { ... }` inside a technology gets you `Unknown modifier: prerequisites` in
`error.log`, and the whole block is ignored. `prerequisites` belongs to national focuses and unit leader
traits, not to technologies — it appears zero times in the base game's `common/technologies/`.

## 3. Add the unlock and the modifier

Now add the real effects. These are two different kinds of thing living in the same technology block:

```
my_infantry_smg_1 = {
    enable_equipments = {
        my_smg_equipment_1
    }

    infantry = {
        soft_attack = 0.05
    }
}
```

`enable_equipments` **permanently unlocks an object**: from completion onward `my_smg_equipment_1` can be selected on the production line. The `infantry = { soft_attack = 0.05 }` below it is a **lasting bonus to a unit category**, where `0.05` means +5%.

Neither depends on the other. The unlock alone gives a new buildable item with unchanged numbers; the bonus alone improves numbers with no new equipment.

## 4. Give it a name

A technology's localisation key is its ID, with a `_desc` suffix for the description:

```
l_english:
 my_infantry_smg_1: "Submachine Guns"
 my_infantry_smg_1_desc: "Issue a high rate-of-fire close-quarters weapon to the infantry squad."
```

When you override an existing key, the file must go in `localisation/<language>/replace/`, otherwise the base game or a parent mod wins the collision, silently. This case adds a new key, so no override is involved.

The icon sprite name is fixed as `GFX_my_infantry_smg_1_medium`. Without it the technology still researches, the node icon is simply blank.

## 5. Check it item by item

Start a separate test save and confirm these in order, without skipping:

| Check | Expected | Where to look if it fails |
| --- | --- | --- |
| The node is visible in the army folder | not overlapping another node | the `folder` name and the x in `position` |
| The node shows a readable name | not `my_infantry_smg_1` | the language directory and key spelling in localisation |
| Not clickable before the prerequisite | greyed out | whether the parent's `path` points at this technology |
| Research takes about 150 days | not several years | whether `start_year` is later than the start year |
| Equipment selectable after completion | appears in the equipment dropdown | **the equipment ID spelling, and whether that equipment exists** |
| Infantry soft attack rises after completion | higher value in division details | the stat name and value in the `infantry` block |

## Common mistakes

- **A misspelled equipment ID.** This entry fails silently: the node still completes, there is simply nothing new on the production line. Compare the ID in the technology against the ID in the equipment definition side by side, rather than from memory.
- **Writing the bonus as an unlock.** `soft_attack` belongs in a unit category block and cannot go inside `enable_equipments`.
- **Giving a technology a `prerequisites` block.** There is no such key: it raises
  `Unknown modifier: prerequisites` and the block is ignored. The requirement comes from the parent's `path`.
- **`start_year` set too late.** Players read this as a broken technology when they are in fact paying the ahead-of-time penalty.
- **Using `allow_branch` to mean "not researchable yet".** That removes the whole branch from the screen and the player never finds it. For "visible but not clickable", use `allow`.

## Variations

- To unlock a unit type instead of equipment, swap `enable_equipments` for `enable_subunits`; the rest is unchanged.
- To fire an event at the moment of completion, use `on_research_complete`, paired with `on_research_complete_limit` to prevent repeat triggering.
- To stop the player researching it directly and grant it from a story beat instead, write `allow = { always = no }` and hand it out with unlock a specific technology directly (`wiki:effect:set_technology`) in the focus reward.
