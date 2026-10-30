---
contentId: general-trait-tree
kind: case
slug: general-trait-tree
locale: en
status: published
title: "Link commander traits into an unlockable tree"
summary: "Three commander traits on two rows: the first is the root, the other two require it and exclude each other, and the player unlocks them in order on the general's panel; the three ways to write a prerequisite, why exclusion is written on both sides, how rows and columns are laid out, and how the tree maps onto the editor canvas."
primaryTopic: people
topics: [people, military]
scenarios: [traits]
gameTerms: [general, trait tree, prerequisite, mutually exclusive, unlock, experience, assign]
aliases: [how to make a trait tree, trait prerequisites, mutually exclusive traits, general trait tree, gui_row, any_parent, parent, mutually_exclusive, num_parents_needed]
relatedIds: [trait-system, general-trait-earned-in-combat, custom-character-traits, "wiki:trigger:has_trait", "wiki:modifier:planning_speed", "wiki:modifier:army_defence_factor"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Trait modding"
    url: "https://hoi4.paradoxwikis.com/Trait_modding"
  - label: "HOI4 Wiki: Modifiers"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
---

# Link commander traits into an unlockable tree

## Goal and observable result

When this page is done you should see three things in game:

1. In a general's unlockable tree, the root trait sits in the left column and the two branches on the next row each take a column, both linked down from the root.
2. An exclusion marker between the two on the second row; once one is assigned, the other cannot be.
3. Until the root is assigned, hovering either branch shows "You need to unlock at least 1 parent trait before you can assign this trait".

## What you need before starting

- A mod that already exports correctly. This page uses the placeholder IDs `my_tree_root`, `my_tree_left` and `my_tree_right`.
- Ideally, a commander trait earned in combat (`general-trait-earned-in-combat`) done first; the root here is that kind of trait.
- No DLC is required.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| Three commander trait definitions | `common/unit_leader/` | one root, two branches |
| Names | `localisation/` | the key is the trait name |
| Icons | `interface/*.gfx` | `GFX_trait_` plus the trait name |

## 1. The three definitions

```
leader_traits = {
    my_tree_root = {
        type = corps_commander
        gain_xp = {
            is_fighting_in_terrain = plains
        }
        cost = 100
        modifier = {
            army_infantry_attack_factor = 0.05
        }
        ai_will_do = {
            factor = 1
        }
        gui_row = 16
    }

    my_tree_left = {
        type = corps_commander
        trait_type = assignable_trait
        gain_xp = {
            always = no
        }
        cost = 100
        modifier = {
            planning_speed = 0.1
        }
        ai_will_do = {
            factor = 1
        }
        any_parent = { my_tree_root }
        mutually_exclusive = my_tree_right
        gui_row = 17
        gui_column = 2
    }

    my_tree_right = {
        type = corps_commander
        trait_type = assignable_trait
        gain_xp = {
            always = no
        }
        cost = 100
        modifier = {
            army_defence_factor = 0.05
        }
        ai_will_do = {
            factor = 1
        }
        any_parent = { my_tree_root }
        mutually_exclusive = my_tree_left
        gui_row = 17
        gui_column = 3
    }
}
```

The root has **no `trait_type`**. Every root that serves as a prerequisite in the base game's army tree (Organizer, Infantry Leader, Trickster and the rest, 16 in all) is written this way; the engine places them in the leftmost column with assignable traits hanging below. Tested: writing the root as `trait_type = assignable_trait` left two lines in `error.log`, `my_tree_left has a wrong kind of parent (my_tree_root)`; reverting to the base-game form drew the tree correctly.

The two branches write `gain_xp = { always = no }`: not earned in combat, purchasable as soon as the prerequisite is lit and the experience covers `cost`. The base game's "Infantry Expert" and "Offensive Doctrine" are written this way, with the comment `# not gainable`.

## 2. Three ways to write a prerequisite

The header comment of the base-game file gives three, all written inside the trait definition:

| Form | Meaning | Base-game uses |
| --- | --- | --- |
| `any_parent = { a b c }` | any one lit is enough | 41, the usual one |
| `all_parents = { a b c }` | all must be lit | 0 |
| `parent = { traits = { a b c } num_parents_needed = 2 }` | N of them lit | 1: the terrain master needs two of six terrain traits |

While the prerequisite is unmet, hovering a branch shows "You need to unlock at least N parent traits before you can assign this trait", exactly the line seen in the test; `all_parents` shows "You need to unlock all parent traits before you can assign this trait". `any_parent` is the shorthand for N equal to 1.

## 3. Exclusion is written on both sides

`mutually_exclusive = my_tree_right` goes on the left branch and `mutually_exclusive = my_tree_left` on the right. Written on one side only, the other side's tooltip and link are incomplete; the base game's "Offensive Doctrine" and "Defensive Doctrine" are written as a pair.

## 4. Rows and columns

`gui_row` counts from 0 and sets the row; when `gui_column` is omitted the engine picks a column from `trait_type`: experience-earned traits with no `trait_type` in the leftmost column, assignable traits in the next, terrain traits in another. 76 base-game traits set `gui_row`, only 29 set `gui_column`, all of them in the navy tree.

🔴 **Two traits on the same row with the same `trait_type` and no column land in the same cell and hide each other**; in the test only the later-loaded one was visible. Give each branch its own `gui_column`, or put them on different rows. Column numbers map onto fixed slots of the panel: in the test, `gui_column = 3` sat at the far right, half hidden by the scrollbar; if 2 and 3 both show fully for you, keep them, otherwise lower the right one or move it down a row.

The base game's army tree uses rows up to 15, so a tree of your own starting at row 16 never shares a cell with it. Only traits with `gui_row` at or above 0 enter the tree; root and both branches need it.

## 5. Conditions beyond the prerequisite

`prerequisites = { ... }` is checked in unit-leader scope; the base game uses it 3 times, each `has_trait = some personality trait`, requiring a personality before a trait can be learned. Skill levels go the same way, `defense_skill_level > 3`. The difference from `any_parent`: the prerequisite is a line in the tree, `prerequisites` is a gate outside it, and the tooltip shows them separately.

## 6. The editor canvas

The custom traits panel in the characters workbench has a trait tree canvas: it shows only traits with `gui_row` at or above 0; dragging a node changes `gui_row` and `gui_column`; drawing from one node's lower handle to another's upper handle adds an `any_parent` on the latter. `parent` blocks and `all_parents` appear as dashed read-only lines; edit those in the raw file. Exclusion is filled in under the form's advanced fields.

## 7. Check each item

| Check | Expected | If not, look at |
| --- | --- | --- |
| Tree shape | root in the left column on row 16, the two branches on row 17 in their own columns | the three `gui_row` values, the branches' `gui_column` |
| Both branches visible | yes | whether they share a cell; whether the right column is cut off |
| Links | each branch linked to the root | the names in `any_parent`; `error.log` for `wrong kind of parent` |
| Hover a branch before the root is lit | "You need to unlock at least 1 parent trait…" | `any_parent` |
| Assign the left, look at the right | greyed, exclusion in the tooltip | whether `mutually_exclusive` is on both sides |
| Icons | all three are yours | three `GFX_trait_` sprites; a question-mark medal when none is registered |
| Bottom of the tooltip | your description | the `_desc` localisation; without it the key `name_desc` shows |

## Common mistakes

- 🔴 **A root written as `trait_type = assignable_trait`**: `error.log` reports `wrong kind of parent`. A root that serves as a prerequisite has no `trait_type`, as in the base game.
- 🔴 **Two branches on one row, same kind, no column**: one cell, one visible.
- 🔴 **Exclusion on one side only**: the other side does not know it is excluded.
- 🔴 **A branch without `gui_row`**: not in the tree, so the link has no endpoint.
- **A prerequisite naming a trait that does not exist**: no line is drawn; check the name.
- **Using `prerequisites` as a prerequisite line**: it draws nothing, it only blocks assignment.
- **Forgetting `gain_xp = { always = no }` on a branch**: what a trait without `gain_xp` does was not tested for this page; every base-game purchasable trait has one.

## Want a different effect?

- "Unlock after any two terrain traits": `parent = { traits = { a b c } num_parents_needed = 2 }`.
- A third tier: the third trait with `any_parent = { my_tree_left my_tree_right }` and `gui_row = 18`.
- Branches earned in combat too: replace `gain_xp` with a real condition, see a trait earned in combat (`general-trait-earned-in-combat`).
