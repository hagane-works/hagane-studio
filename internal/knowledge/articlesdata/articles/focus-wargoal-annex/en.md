---
contentId: focus-wargoal-annex
kind: case
slug: focus-wargoal-annex
locale: en
status: published
title: "A war goal on a neighbour when a focus completes"
summary: "When a focus completes, the country receives an annexation war goal on a neighbour, so the player can declare war without spending political power on justification; along the way, where the type comes from, how expiry is written, and how a later focus uses \"already has a war goal\" as a condition. This is the most common diplomacy effect in base-game focuses, over a thousand uses."
primaryTopic: diplomacy
topics: [diplomacy]
scenarios: [focus, event]
gameTerms: [war goal, justification, declare war, annex, focus, completion reward, world tension]
aliases: [focus grants war goal, declare war after focus, war without justifying, war goal expiry, create_wargoal, has_wargoal_against, annex_everything]
relatedIds: [diplomacy-system, focus-create-faction, focus-transfer-state-core, "wiki:effect:create_wargoal", "wiki:effect:declare_war_on", "wiki:effect:add_state_claim", "wiki:trigger:has_wargoal_against"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki: Triggers"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki: National focus modding"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
---

# A war goal on a neighbour when a focus completes

## Goal and observable result

When this page is done you should see three things in game:

1. The moment the focus completes, "Declare war" becomes available on the target country's diplomacy panel, and the list of war goals shows "Annex".
2. The player spent no political power on justification, and world tension did not rise from a justification in progress.
3. A later focus can only be started while "a war goal against that country" exists.

## What you need before starting

- A mod that already exports correctly, and an existing focus. This page uses the placeholder ID `my_focus_claim_belgium`.
- The target country's three-letter tag. This page uses Belgium, `BEL`, played as Luxembourg.
- No DLC is required.
- Read diplomacy: where starting relations, runtime effects and definitions each live (`diplomacy-system`) first, so that a war goal type is understood as a definition the effect merely references.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| Focus | `common/national_focus/` | the effect goes in its `completion_reward` |
| War goal type | `common/wargoals/`, base-game `00_invasion.txt` | `annex_everything` already exists; this page defines no new type |
| Follow-up focus | the same tree | `has_wargoal_against` in its `available` |

## 1. Write it in the focus's completion reward

```
focus = {
    id = my_focus_claim_belgium
    icon = GFX_goal_generic_territory_or_war
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        create_wargoal = {
            type = annex_everything
            target = BEL
        }
    }
}
```

The focus tooltip reads "Gains Annex war goal against Belgium", the game's own text. `type` names a type under `common/wargoals/`. The two the base game uses most in focuses are `annex_everything` (533 uses, annex everything) and `puppet_wargoal_focus` (226, puppet the target); `take_state_focus` with `generator = { state ids ... }` names the states to take.

The definition of `annex_everything` says `allowed = { always = no }`. That `allowed` governs whether the player may justify this kind of goal by hand on the diplomacy screen; a focus grants it directly through `create_wargoal` without manual justification, so it is granted all the same.

## 2. Expiry

War goals expire. Some type definitions set `expire` (`take_state` 60, `puppet_wargoal_focus` 730); `annex_everything` sets none. A focus may override it on the effect:

```
create_wargoal = {
    type = annex_everything
    target = BEL
    expire = 0
}
```

Of the base game's 1072 `create_wargoal` uses, 448 carry `expire`, and 435 of those write `expire = 0`. Tested: `annex_everything` without `expire` and with `expire = 0` behave the same, and neither goal shows an expiry date on the diplomacy panel. For a goal that does expire, write a positive number of days; the base game has 10 focuses with `expire = 730`.

## 3. A follow-up focus conditioned on the war goal

```
focus = {
    id = my_focus_war_belgium
    prerequisite = { focus = my_focus_claim_belgium }
    x = 0
    y = 1
    cost = 5
    available = {
        has_wargoal_against = BEL
    }
    completion_reward = {
        declare_war_on = {
            target = BEL
            type = annex_everything
        }
    }
}
```

`has_wargoal_against = BEL` checks for a war goal of any type against Belgium; to require a type, use the block form `has_wargoal_against = { target = BEL type = annex_everything }`. In the base game this trigger targets `PREV` 127 times and `ROOT` 58 times; a literal tag is more usual inside a focus tree.

`declare_war_on` is "declare war now", not "grant a war goal": the former starts the war when the focus completes, the latter only lets the player declare. Base-game focuses use `declare_war_on` 64 times and `create_wargoal` 1072 times; most storylines leave the decision to go to war with the player.

## 4. Use an event instead of a focus

The same effect moves unchanged into an event option; fire it from the console with `event my_events.1` while testing:

```
add_namespace = my_events

country_event = {
    id = my_events.1
    title = my_events.1.t
    desc = my_events.1.d
    is_triggered_only = yes
    option = {
        name = my_events.1.a
        create_wargoal = {
            type = annex_everything
            target = BEL
        }
    }
}
```

## 5. Check each item

| Check | Expected | If not, look at |
| --- | --- | --- |
| Belgium's diplomacy panel after the focus | "Declare war" available; "Annex" in the war goal list | spelling of `type`; whether `target` is a three-letter tag |
| World tension | no rise from justification | if it rose, the player justified by hand rather than receiving the goal |
| The goal's expiry date | shown only for a positive `expire`; absent when omitted or 0 | `expire` in the type definition |
| Follow-up focus | startable with the goal, greyed without | whether `has_wargoal_against` sits in `available` |
| `error.log` | no mention of your file | a nonexistent type is logged here while the game loads |

## Common mistakes

- 🔴 **A `type` that does not exist under `common/wargoals/`**: no goal appears. Tested: while loading, the game writes `invalid war goal: your name: type` to `error.log` with the file name and line, so a glance at the log before playing catches it.
- 🔴 **A `target` tag that does not exist**: tested, nothing happens and nothing is logged. The Netherlands is `HOL`, not `NED`; tags come from the base game's `common/country_tags/`.
- **Expecting `create_wargoal` to start the war**: it only grants the right. Use `declare_war_on` for that.
- **Writing `has_war_goal_against`**: the game's documentation has no such trigger; it is `has_wargoal_against`.
- **`create_wargoal` in a history file**: the base game does this once. A storyline that should allow war from day one usually grants the goal in the first focus or a start-of-game event.

## Want a different effect?

- A "puppet" goal instead of "annex": `type = puppet_wargoal_focus`.
- Only certain states: `type = take_state_focus` with `generator = { state ids ... }`.
- War the moment the focus completes: `declare_war_on`, section 3.
- A claim alongside the goal: `add_state_claim = state id` in the same `completion_reward`.
