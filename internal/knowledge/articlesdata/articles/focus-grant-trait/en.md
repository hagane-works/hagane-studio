---
contentId: focus-grant-trait
kind: case
slug: focus-grant-trait
locale: en
status: published
title: "Grant a trait from a focus or event"
summary: "A focus swaps one of the sitting leader's traits for another; an event gives a general a timed trait. A leader trait can be granted from country scope, a commander trait only from inside the character's scope, swapping is one step, and there is a timed form. Base-game focuses grant leader traits 164 times, swap 78, and grant commander traits 60."
primaryTopic: people
topics: [people, politics]
scenarios: [traits, focus, event]
gameTerms: [trait, leader, general, focus, event, completion reward, timed trait, swap trait, scope]
aliases: [focus grants trait, event adds trait, leader swaps trait, add trait to general, timed trait, remove trait, add_country_leader_trait, swap_country_leader_traits, add_unit_leader_trait, add_timed_unit_leader_trait, random_army_leader]
relatedIds: [trait-system, general-trait-earned-in-combat, leader-succession, custom-character-traits, "wiki:effect:add_country_leader_trait", "wiki:effect:remove_country_leader_trait", "wiki:effect:swap_country_leader_traits", "wiki:effect:add_unit_leader_trait", "wiki:effect:add_timed_unit_leader_trait", "wiki:effect:remove_unit_leader_trait", "wiki:effect:random_army_leader", "wiki:trigger:has_trait"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki: Scopes"
    url: "https://hoi4.paradoxwikis.com/Scopes"
  - label: "HOI4 Wiki: Trait modding"
    url: "https://hoi4.paradoxwikis.com/Trait_modding"
---

# Grant a trait from a focus or event

## Goal and observable result

When this page is done you should see three things in game:

1. The moment the focus completes, one trait on the leader card is replaced by another, and the tooltip reads "So-and-so gains: new trait".
2. After an event, one general carries an extra trait, and hovering it reads "Gains X trait for 90 days".
3. Ninety days later that trait disappears on its own.

## What you need before starting

- A mod that already exports correctly, one focus and one event. This page uses the placeholder IDs `my_focus_reform` and `my_events`.
- The traits to grant already defined: a leader trait `my_leader_reformer` under `common/country_leader/`, and a commander trait `my_general_wounded` under `common/unit_leader/` with `trait_type = status_trait` and no `gui_row`. Shapes are on a custom trait for a character (`custom-character-traits`).
- No DLC is required.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| Focus | `common/national_focus/` | swaps the leader trait |
| Event | `events/` | gives a general a timed trait |
| Two trait definitions | two folders | referenced; this page does not edit them |

## 1. Leaders: country scope is enough

The game's documentation says `add_country_leader_trait` supports both country and character scope: run in country scope, it goes to the **sitting leader**. So in a focus reward:

```
focus = {
    id = my_focus_reform
    icon = GFX_goal_generic_political_pressure
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        add_country_leader_trait = my_leader_reformer
    }
}
```

To **swap** one trait for another, base-game focuses use the one-step form 78 times rather than remove-then-add:

```
completion_reward = {
    swap_country_leader_traits = {
        remove = my_leader_old
        add = my_leader_reformer
    }
}
```

`swap_country_leader_traits` is character scope only; in a focus reward the base game first enters the leader's character scope, as in the Belgian focus `BEL_leopold = { swap_country_leader_traits = { ... } }`. When a character holds several leader roles, `add` / `remove` take an `ideology = ...` to say which role; the documentation's example is `HUN_miklos_horthy = { add_country_leader_trait = { ideology = oligarchism trait = anti_communist } }`.

Removal is `remove_country_leader_trait`, with the same scope rules as adding.

## 2. Generals: enter the character's scope first

`add_unit_leader_trait` supports character scope only. A focus or event runs in country scope, so jump to a general first. Nearly all 60 base-game focus uses have this shape:

```
random_army_leader = {
    limit = {
        NOT = { has_trait = my_general_wounded }
    }
    add_unit_leader_trait = my_general_wounded
}
```

`random_army_leader` picks one of the country's generals that satisfies `limit`; for a specific person, open scope with the character ID: `MYT_my_general = { add_unit_leader_trait = ... }`. For all generals, `every_army_leader`.

The timed form:

```
add_namespace = my_events

country_event = {
    id = my_events.1
    title = my_events.1.t
    desc = my_events.1.d
    is_triggered_only = yes
    option = {
        name = my_events.1.a
        random_army_leader = {
            add_timed_unit_leader_trait = {
                trait = my_general_wounded
                days = 90
            }
        }
    }
}
```

Base-game events use it 5 times; Sweden's "wounded in the protests" runs 120 days, "disillusioned with the government" 270. It is removed automatically when time is up. Manual removal is `remove_unit_leader_trait`, again in character scope.

The timed form exists only on the commander side: `add_timed_country_leader_trait` is never used in the base game and this page does not recommend it.

## 3. The traits themselves

A status trait for a general does not enter the tree, so no `gui_row`, `cost` or `gain_xp`; only the type and modifiers:

```
leader_traits = {
    my_general_wounded = {
        type = land
        trait_type = status_trait
        non_shared_modifier = {
            experience_gain_factor = -0.25
        }
    }
}
```

A leader trait follows the leader shape, modifiers flat:

```
leader_traits = {
    my_leader_reformer = {
        random = no
        stability_factor = 0.05
    }
}
```

## 4. Checking for it

- The sitting leader, from country scope: `has_country_leader_with_trait = my_leader_reformer`, 45 times in base-game focuses.
- A general, in character or combatant scope: `has_trait = my_general_wounded`, usually inside the `limit` of `random_army_leader`.

## 5. Check each item

| Check | Expected | If not, look at |
| --- | --- | --- |
| Leader card after the focus | old trait gone, new one present | the two names in `swap`; whether the trait is defined under `common/country_leader/` |
| Focus tooltip | "So-and-so gains: new trait" | trait localisation |
| A general after the event | one more trait, hover reads "for 90 days" | whether it sits inside `random_army_leader`; `days` |
| After 90 days | trait gone | `add_timed_unit_leader_trait` rather than `add_unit_leader_trait` |
| `error.log` | no mention of your file | a scope mistake is logged here |

## Common mistakes

- 🔴 **`add_unit_leader_trait` written directly in country scope**: it only accepts character scope. `random_army_leader` or a character ID first.
- 🔴 **Swapping written as remove then add**: it runs, but `swap_country_leader_traits` does it in one line and all 78 base-game uses do.
- **No `ideology` for a leader with several roles**: the documentation says it is needed then, or the engine cannot tell which role to change.
- **The permanent effect where a timed one was meant**: `add_unit_leader_trait` never expires; use `add_timed_unit_leader_trait`.
- **A misspelled trait name**: a nonexistent trait is logged in `error.log`; confirm it in the definition file first.

## Want a different effect?

- Affect every general: `every_army_leader = { ... }`.
- An admiral instead: `random_navy_leader`; the trait's `type` must be `navy` or `all`.
- Handle traits while changing the leader: see changing the leader (`leader-succession`).
- Let the player unlock it rather than granting by script: see a trait earned in combat (`general-trait-earned-in-combat`).
