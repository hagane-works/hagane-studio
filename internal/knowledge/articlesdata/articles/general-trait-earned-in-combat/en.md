---
contentId: general-trait-earned-in-combat
kind: case
slug: general-trait-earned-in-combat
locale: en
status: published
title: "A commander trait earned in combat"
summary: "Define a commander trait that accrues experience while fighting under a given condition, so the player can assign it once enough has built up; which scope gain_xp is checked in, why cost is a threshold rather than a price tag, and which field feeds each of the two tooltip lines \"Experience\" and \"Gains experience when\". More than half the base game's commander traits work this way."
primaryTopic: people
topics: [people, military]
scenarios: [traits]
gameTerms: [general, trait, experience, earn experience, assign trait, trait tree, terrain, plains]
aliases: [general earns trait by fighting, trait that gains experience, commander trait experience, how traits unlock, terrain trait, gain_xp, cost, custom_gain_xp_trigger_tooltip, assignable_trait]
relatedIds: [trait-system, general-trait-tree, custom-character-traits, "wiki:trigger:is_fighting_in_terrain", "wiki:trigger:has_trait", "wiki:modifier:army_infantry_attack_factor", "wiki:effect:add_unit_leader_trait"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Trait modding"
    url: "https://hoi4.paradoxwikis.com/Trait_modding"
  - label: "HOI4 Wiki: Triggers"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki: Modifiers"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
---

# A commander trait earned in combat

## Goal and observable result

When this page is done you should see three things in game:

1. Open any general's panel and your trait sits in the unlockable trait tree with the right icon, name and description.
2. Hovering it shows the modifier, "It will cost 15.00 army experience to assign this trait" and your description; once that general has fought on plains and built up some experience, two more lines appear: "Experience: X/100" and "Gains experience when: fighting on plains".
3. After that general has fought on plains, the experience reaches 100 and one click assigns the trait, after which the infantry he leads attacks harder.

## What you need before starting

- A mod that already exports correctly. This page uses the placeholder ID `my_plains_fighter`.
- A country with generals to look at; this page uses Luxembourg, which starts with four.
- No DLC is required.
- Read traits: what the four kinds do (`trait-system`) first, so that commander modifiers are understood as wrapped in a block.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| Commander trait definition | `common/unit_leader/` | the subject of this page |
| Name and description | `localisation/` | the key is the trait name, the description adds `_desc` |
| Icon | `interface/*.gfx` | sprite `GFX_trait_` plus the trait name |
| A general | any | no character record is edited; the trait is earned |

## 1. Define the trait

Create `common/unit_leader/my_traits.txt`:

```
leader_traits = {
    my_plains_fighter = {
        type = corps_commander
        gain_xp = {
            is_fighting_in_terrain = plains
        }
        custom_gain_xp_trigger_tooltip = my_plains_fighter_when
        cost = 100
        modifier = {
            army_infantry_attack_factor = 0.05
        }
        ai_will_do = {
            factor = 1
        }
        gui_row = 0
    }
}
```

Field by field:

| Field | Role | Note |
| --- | --- | --- |
| `type = corps_commander` | generals only | `field_marshal` for marshals, `land` for both |
| `gain_xp = { ... }` | the condition for accruing experience | **checked in combatant scope**: every battle this general's troops fight is tested |
| `custom_gain_xp_trigger_tooltip` | localisation key for the text after "Gains experience when:" | without it the engine describes the condition itself, which reads badly for complex ones |
| `cost = 100` | how much of this trait's own experience must accrue before it can be assigned | the base game runs 500 to 1500; separate from the 15 army experience deducted on assignment |
| `modifier` | modifiers for the troops he leads once assigned | troops only; for the general himself use `non_shared_modifier` |
| `ai_will_do` | whether the AI assigns it | weight 0 and the AI never does |
| `gui_row = 0` | row in the tree | **omit it and the trait is not in the tree**; the player never sees it |

There is deliberately **no `trait_type`**. The base game's combat-earned traits (Infantry Leader, Organizer, Trickster and the rest, 16 in all) have none, and the engine places them in the leftmost column of the tree; with `assignable_trait` it moves to the assignable column, still works on its own, but can no longer serve as another trait's prerequisite (see a trait tree (`general-trait-tree`)).

`gain_xp` is checked in **combatant scope**, as the header comment of the base-game file states (`gain_xp = trigger # scope is a combatant`). So combat conditions such as `is_fighting_in_terrain = plains` work here, while commander conditions such as "more than ten divisions" belong in `gain_xp_leader`, which is checked in unit-leader scope. No fighting, no accrual; in peacetime this stays at 0.

The tooltip carries two kinds of "experience"; keep them apart. One is this trait's own progress, with `cost = 100` as its threshold; the tooltip only starts showing "Experience: X/100" once it passes 5% (game define `LEADER_TRAITS_XP_SHOW = 0.05`). The other is the army experience deducted when you assign, a fixed 15 for every trait (`UNIT_LEADER_ASSIGN_TRAIT_COST = 15`), shown as "It will cost 15.00 to assign this trait".

## 2. Name and description

Three lines under `localisation/english/`, keyed by the trait name, the trait name plus `_desc`, and the tooltip key:

```
l_english:
 my_plains_fighter:0 "Plains Fighter"
 my_plains_fighter_desc:0 "Excels when fighting on open plains."
 my_plains_fighter_when:0 "fighting on plains"
```

The tooltip's "Gains experience when:" is fixed engine text; your `my_plains_fighter_when` follows it.

## 3. Register the icon

Create a `.gfx` file under `interface/`:

```
spriteTypes = {
    spriteType = {
        name = "GFX_trait_my_plains_fighter"
        textureFile = "gfx/interface/traits/my_plains_fighter.dds"
    }
}
```

The sprite name is a fixed rule: `GFX_trait_` plus the trait name. All 181 base-game commander trait icons are registered this way. What shows when nothing is registered is in the in-game result at the end of this page.

## 4. Have a general start with it

Earning is the player's route. For a historical general who should start with it, write it into the character record: `corps_commander = { traits = { my_plains_fighter } }`; or in an event, `random_army_leader = { add_unit_leader_trait = my_plains_fighter }`. Neither route looks at `cost` or `gain_xp`.

## 5. Check each item

| Check | Expected | If not, look at |
| --- | --- | --- |
| The general's unlockable tree | your trait is there, in the leftmost column | whether `gui_row` is written; `type` |
| Hover at game start | the modifier, "It will cost 15.00 to assign this trait", your description | the `_desc` localisation; without it the key `name_desc` shows |
| Hover once past 5% | "Experience: X/100", "Gains experience when: fighting on plains" appear | `cost`; the localisation key in `custom_gain_xp_trigger_tooltip` |
| Icon | your image | whether the sprite is named `GFX_trait_` plus the trait name; a question-mark medal when none is registered |
| After a battle on plains | experience rises | whether the `gain_xp` condition is one combatant scope can evaluate |
| Assign once enough | trait lit, infantry attack +5% | whether `modifier` is wrapped in a block |
| `error.log` | no mention of your file | a misspelled modifier leaves a line here |

## Common mistakes

- 🔴 **No `gui_row`**: the definition is fine, but the tree never shows it and the player can never assign it.
- 🔴 **A commander condition inside `gain_xp`**: `gain_xp` is combatant scope; `num_units > 10` and the like belong in `gain_xp_leader`.
- **Reading `cost` as the assignment price**: it is the accrual threshold; assignment deducts a fixed 15 army experience.
- **`cost` omitted**: what a trait without a threshold does was not tested for this page; every base-game assignable trait has one.
- **Modifiers flat under the trait**: that is the leader-trait shape; commander modifiers must be wrapped in `modifier = { }`.
- **`ai_will_do` weight 0**: the player can assign it, the AI never will.

## Want a different effect?

- Accrue by unit type rather than terrain: the base game's "Infantry Leader" uses `gain_xp = { check_expr = { value = num_infantry greater_than = { value = num_units multiply = 0.8 } } }`, i.e. more than 80% infantry.
- Faster experience for the general himself: `non_shared_modifier = { experience_gain_factor = 0.1 }`.
- Make it the root of a tree: keep `gui_row` and point other traits at it with `any_parent = { my_plains_fighter }`, see a trait tree (`general-trait-tree`).
- Let a general with this trait serve as high command: add `slot = high_command` and the three advisor traits, explained in the guide.
