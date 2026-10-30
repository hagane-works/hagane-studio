---
contentId: trait-system
kind: guide
slug: trait-system
locale: en
status: published
title: "Traits: what the four kinds do, and how one gets onto a person"
summary: "Leader, advisor, commander and scientist traits live in three folders with three different file shapes; a trait reaches a person by one of five routes: written on at definition, rolled at random, bought by the player with experience, earned in combat, or granted by script. Keep those two questions apart and you will not write a commander trait in leader shape, or a trait nobody can ever obtain."
primaryTopic: people
topics: [people]
scenarios: [traits, focus, event]
gameTerms: [trait, leader trait, advisor trait, commander trait, scientist trait, trait tree, experience, high command, personality]
aliases: [how to make a trait, kinds of traits, how to give a trait to a character, trait tree, general experience, random personality, trait, leader_traits, trait_type, gain_xp]
relatedIds: [character-system, custom-character-traits, general-trait-earned-in-combat, general-trait-tree, focus-grant-trait, "wiki:effect:add_country_leader_trait", "wiki:effect:add_unit_leader_trait", "wiki:effect:swap_country_leader_traits", "wiki:trigger:has_trait", "wiki:modifier:experience_gain_factor"]
updatedAt: "2026-09-15"
verification: static-checked
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Trait modding"
    url: "https://hoi4.paradoxwikis.com/Trait_modding"
  - label: "HOI4 Wiki: Character modding"
    url: "https://hoi4.paradoxwikis.com/Character_modding"
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
---

# Traits: what the four kinds do, and how one gets onto a person

## What this system covers

The small icons on a character card. A leader has "Ruthless Politician", an advisor "Silent Workhorse", a general "Infantry Leader", a scientist a specialisation. Each is one trait definition attached to one person, granting that person or the troops they lead a set of modifiers.

As an author you decide two things:

1. **Which kind the trait is, which file it goes in, and what its modifiers look like**
2. **How it gets onto a person**: always there, rolled at random, bought with experience, earned in combat, or granted by a focus or event

The first decides the file shape; the second decides whether you need fields such as `gain_xp`, `cost` and `gui_row`. Most "I made a trait and nothing happened" comes from not settling the second: a trait defined that nobody can obtain.

## Core idea: four kinds, three folders, three shapes

| Kind | Folder | Outer block | How modifiers are written | Base-game count |
| --- | --- | --- | --- | --- |
| Leader trait | `common/country_leader/` | `leader_traits = { }` | **flat** under the trait: `stability_factor = 0.05` | 1782 shared with advisors |
| Advisor trait | same folder, same files | same | same; the three high-command tiers also carry `command_cap_increase`, `command_power` | same |
| Commander trait | `common/unit_leader/` | `leader_traits = { }` | **wrapped**: `modifier = { }`, plus `non_shared_modifier`, `field_marshal_modifier`, `corps_commander_modifier`, `sub_unit_modifiers` | 178 |
| Scientist trait | `common/scientist_traits/` | top level | `modifier = { }`, special projects only | a handful, with a bundled `_documentation.md` |

Leader and advisor traits look the same and share a folder; what differs is who references them: a `country_leader` block references leader traits, an `advisor` block references advisor traits. The earlier case a custom trait for a character (`custom-character-traits`) covers the minimal leader and commander shapes; this page does not repeat it.

The extra modifier blocks of a commander trait each have a destination: `modifier` for the troops led, `non_shared_modifier` for the commander personally (experience gain, say), `field_marshal_modifier` only while serving as field marshal, `sub_unit_modifiers` per unit type. The header comment of the base game's `common/unit_leader/00_traits.txt`, some seventy lines, is a complete field list.

## Five routes a trait takes onto a person

| Route | How | Applies to | In the base game |
| --- | --- | --- | --- |
| **Written at definition** | `traits = { my_trait }` in the character record | all four kinds | the most common; historical figures' fixed traits |
| **Rolled at random** | leader trait `random = yes`; commander trait `trait_type = personality_trait` with `new_commander_weight` | leaders, commanders | only 23 of 1782 leader traits are `random = yes`; 94 commander personality traits |
| **Bought by the player** | commander trait `trait_type = assignable_trait` with `cost`, and `gui_row` so it appears in the tree | commanders | 46 assignable traits; `cost` appears 229 times |
| **Earned in combat** | commander trait with `gain_xp = { condition }`, accumulating while the condition holds in battle | commanders | 71 traits carry `gain_xp` |
| **Granted by script** | `add_country_leader_trait`, `add_unit_leader_trait`, `swap_country_leader_traits` and so on | all four kinds | focuses grant leader traits 164 times, swap 78, grant commander traits 60 |

The last three each have a case: a commander trait earned in combat (`general-trait-earned-in-combat`), link commander traits into an unlockable tree (`general-trait-tree`), grant a trait from a focus or event (`focus-grant-trait`).

"Bought by the player" and "earned in combat" are usually two faces of the same base-game trait: `gain_xp` defines when experience accrues, `cost` how much of the trait's own experience is needed before it can be assigned. The commander panel's "Experience: X/1000" and "Gains experience when:" are those two fields, shown only once past 5%. Assigning also deducts a fixed 15 army experience for every trait (`UNIT_LEADER_ASSIGN_TRAIT_COST`), shown as "It will cost 15.00 to assign this trait", unrelated to `cost`.

## File structure and how it runs

### The skeleton of a commander trait

```
leader_traits = {
    my_trait = {
        type = corps_commander
        trait_type = assignable_trait
        gain_xp = {
            is_fighting_in_terrain = hills
        }
        cost = 700
        modifier = {
            hills = {
                attack = 0.1
                defence = 0.1
            }
        }
        ai_will_do = {
            factor = 1
        }
        gui_row = 2
    }
}
```

`type` decides who may hold it: `all`, `land`, `navy`, `operative`, `corps_commander`, `field_marshal`. `trait_type` decides where it is filed in the interface and whether it can be assigned: `assignable_trait` (purchasable), `personality_trait` (random personality), `basic_terrain_trait` / `assignable_terrain_trait` (terrain), `status_trait` (wounded, captured and the like), `basic_trait` (operatives), `exile`. Of the base game's 178, 16 have no `trait_type`; they are the "Infantry Leader" sort earned through `gain_xp`.

### Triggers

`has_trait = my_trait` checks in character or combatant scope. To check the sitting leader from country scope, write `has_country_leader_with_trait = my_trait`, used 45 times in base-game focuses; commander traits are usually checked inside `random_army_leader = { limit = { has_trait = ... } }`.

## Common goals and where to start

| What you want to see in game | Start here | The distinction that matters |
| --- | --- | --- |
| A historical general starts with a signature trait | `traits = { }` in the character record plus one commander trait definition | the definition needs `type`; no `gui_row` needed, it does not go in the tree |
| A general who has fought in the hills can unlock a hills trait | a trait earned in combat (`general-trait-earned-in-combat`) | `gain_xp` is checked in combatant scope; `cost` is the threshold |
| A trait tree with prerequisites and exclusions | a trait tree (`general-trait-tree`) | `any_parent` / `mutually_exclusive` / `gui_row`; exclusion is written on both sides |
| The leader swaps one trait for another when a focus completes | grant a trait from a focus (`focus-grant-trait`) | `swap_country_leader_traits` in one step, 78 times in focuses |
| An event wounds a general for three months | same | `add_timed_unit_leader_trait = { trait = ... days = 90 }`, inside the general's scope |
| A general with this trait can serve as high command | add `slot = high_command` and the three advisor traits to the commander trait | the three tiers are entries in the leader folder, see below |

## Easy things to get wrong

### Leader traits flat, commander traits wrapped

The same `stability_factor` is written directly inside a leader trait and must be wrapped in `modifier = { }` inside a commander trait. Reversed, the engine raises no error and the modifier simply does not apply. The earlier case has the side-by-side.

### A commander trait can define a high-command post as a side effect

33 base-game commander traits carry `slot = high_command` plus `specialist_advisor_trait` / `expert_advisor_trait` / `genius_advisor_trait`. A general with such a trait can be hired as high command, the tier following the general's skill. The three tiers reference advisor traits in the leader folder; "Infantry Leader" maps to `army_infantry_1` through `army_infantry_3`. The tooltip line "Unlocks high command role: …" is this.

### `random` is for leader traits, personality for commanders

`random = yes` on a leader trait lets it appear on randomly generated leaders; the commander counterpart is `trait_type = personality_trait` with a `new_commander_weight` block. The two mechanisms do not cross.

### A trait without `gui_row` is not in the tree

Omit `gui_row` or write `-1` and the trait never appears in the commander panel's unlockable tree, so the player can neither see nor buy it. Traits only ever granted by script (status traits, say) may omit it; anything the player should buy must have it, counted from 0. With no column written the engine assigns one from `trait_type`: combat-earned traits at the far left, assignable ones to their right; two of the same kind on one row without columns hide each other. A root that serves as a prerequisite has **no `trait_type`**, as in the base game; written as assignable it reports `wrong kind of parent`. Details on a trait tree (`general-trait-tree`).

### Icons must be registered

Commander trait icons are looked up by a fixed sprite name, `GFX_trait_` plus the trait name, defined under `interface/*.gfx`; the base game registers 181. Leader traits use `sprite = number` pointing at a cell of a strip. Tested: with nothing registered a question-mark medal shows, and with no `_desc` localisation the tooltip ends with the raw key `name_desc`.

### Scientist traits only affect special projects

The first paragraph of `common/scientist_traits/_documentation.md` says so: only special projects, so only that group of modifiers works; a plain research bonus written there does nothing. This is Götterdämmerung content.

## In Hagane Works

The custom traits panel in the characters workbench: pick the trait kind (leader / commander / scientist) and fill in modifiers; commander traits have an advanced-fields area matching the base game's `00_traits.txt` (`trait_type`, `cost`, `gui_row`, exclusion, script blocks) and a trait tree canvas where dragging nodes sets `gui_row` / `gui_column` and drawing a line adds an `any_parent` prerequisite. Export goes to each folder's own file. Effects that grant traits are written as-is in the focus workbench's completion reward and the event workbench's options.

## Where to go next

- A custom trait for a character (`custom-character-traits`): the minimal shapes
- A commander trait earned in combat (`general-trait-earned-in-combat`)
- Link commander traits into an unlockable tree (`general-trait-tree`)
- Grant a trait from a focus or event (`focus-grant-trait`)
- The characters themselves: authoring a character (`character-system`)
