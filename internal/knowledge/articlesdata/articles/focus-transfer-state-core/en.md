---
contentId: focus-transfer-state-core
kind: case
slug: focus-transfer-state-core
locale: en
status: published
title: "A focus that transfers a state and adds a core"
summary: "When the focus completes, a state is handed to your country with a core on it. Along the way: occupying without owning, adding a claim, adding compliance and renaming a state, and which of the six effects go at country level versus inside the state."
primaryTopic: territory
topics: [territory]
scenarios: [focus, event]
gameTerms: [state, transfer, core, claim, occupation, controller, compliance, national focus, completion reward]
aliases: [focus transfer state, focus add core, event transfer state, occupy without owning, add claim, rename state, transfer_state, add_state_core, set_state_controller]
relatedIds: [state-file-basics, "wiki:effect:transfer_state", "wiki:effect:add_state_core", "wiki:effect:set_state_controller", "wiki:effect:add_claim_by", "wiki:effect:add_compliance", "wiki:effect:set_state_name", "wiki:trigger:owns_state"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki: Scopes"
    url: "https://hoi4.paradoxwikis.com/Scopes"
  - label: "HOI4 Wiki: National focus modding"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
---

# A focus that transfers a state and adds a core

## Goal and observable result

When this page is done you should see three things in game:

1. At the moment the focus completes, the target state changes to your colour and its panel shows your country as owner.
2. Your country's core marker appears on the state; it is no longer non-core land.
3. Another country's claim marker appears on the state.

Three common variants follow: occupying without owning, adding compliance to an occupied state, and renaming a state in the story. This page treats all six effects together, because they come in two forms, and mixing the forms silently does nothing.

## What you need before starting

- A mod that already exports correctly, and an existing focus. This page uses the placeholder ID `my_focus_take_flanders`.
- The **id** of the target state. Select it in the workbench's state list; the read-only "state id" on the right is the number. This page uses the base game's Flanders, id 6, and Holland, id 7.
- No DLC is required.
- Read state files (`state-file-basics`) first, so that owner, controller and core are clearly three different fields.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| Focus | `common/national_focus/` | The effects go in its `completion_reward` |
| State | `history/states/` | The thing being changed; the file is untouched, the change happens at runtime |
| Effects | Usable in focus rewards, event options and decision effects | Six effects, two forms |

## 1. Two forms: at country level, or inside the state

Of the six effects, three are written at country level with the state id as the value; the other three open a block with the state id first and go inside it:

| What you want | Form | Applies to |
| --- | --- | --- |
| Transfer the state (owner and controller together) | `transfer_state = 6` | Country level, value is the state id |
| Add your core | `add_state_core = 6` | Country level |
| Occupy without owning | `set_state_controller = 7` | Country level |
| Add another country's claim | `6 = { add_claim_by = GER }` | **State level**, enter state 6 first |
| Add compliance | `7 = { add_compliance = 50 }` | State level |
| Rename the state | `7 = { set_state_name = "New name" }` | State level |

The bundled documentation lists a supported scope for every effect: the first three are COUNTRY, the last three are STATE. Write `add_claim_by = GER` directly in a focus reward and it has no state to act on; nothing happens.

## 2. Write it in the focus completion reward

The focus editor in the workbench has a "completion_reward" field; type the effects in as they are and the exporter wraps them in the `completion_reward` block. The exported focus node looks like this:

```
focus = {
    id = my_focus_take_flanders
    icon = GFX_goal_generic_territory_or_war
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        transfer_state = 6
        add_state_core = 6
        6 = { add_claim_by = GER }
    }
}
```

Order the three lines by the result you want: transfer first, then the core, then the other country's claim. Base-game focuses often wrap `transfer_state` in `hidden_effect` or `ROOT = { ... }`; that is for controlling tooltips or running in another scope, and this example does not need it.

## 3. Occupy without owning, then add compliance

`set_state_controller` changes only the controller; the owner stays. The state panel shows "owned by X, controlled by Y", which is what an occupied state is. Occupied states have compliance and resistance, adjusted with state-level effects:

```
completion_reward = {
    set_state_controller = 7
    7 = { add_compliance = 50 }
}
```

Tested: after this runs, state 7's controller is your country, the owner is still Holland, and compliance rises by 50.

## 4. Rename a state in the story

`set_state_name` takes display text directly, not a localisation key, and changes the running name without touching the file:

```
7 = { set_state_name = "New Occupied Zone" }
```

This is for story use. For a name that is different from day one, use the state file's localisation key, see the guide.

## 5. Use an event instead of a focus

The same six effects go straight into an event option:

```
add_namespace = my_events

country_event = {
    id = my_events.1
    title = my_events.1.t
    desc = my_events.1.d
    is_triggered_only = yes
    option = {
        name = my_events.1.a
        transfer_state = 6
        add_state_core = 6
    }
}
```

For testing, fire it from the console with `event my_events.1`, which is faster than waiting for a focus to complete.

## 6. Check each item

| Check | Expected | If not, look at |
| --- | --- | --- |
| Colour of state 6 after the focus | Your country | Whether a province id was written instead of a state id |
| State 6 panel | Owner is you, with your core | Whether `add_state_core` is there alongside `transfer_state` |
| State 6 panel | Has a German claim | Whether `add_claim_by` is inside `6 = { ... }` |
| State 7 panel | Owner Holland, controller you | `set_state_controller` was used, not `transfer_state` |
| State 7 compliance | 50 higher than before | Whether `add_compliance` is at state level |
| State 7 name | Your text | Whether `set_state_name` is at state level |

## Common mistakes

- **State-level effects written at country level**: `add_claim_by`, `add_compliance` and `set_state_name` do nothing when written directly in the reward. Open a block with the state id first.
- **A province id where a state id belongs**: the two numbering schemes are unrelated. State ids are in the workbench's "state id"; province ids are in the "provinces" list.
- **Transfer without a core**: the country gets non-core land, with manpower and factories reduced by the non-core rules. For land meant to be home territory, write both lines.
- **Wanting an occupation but writing a transfer**: a transfer changes the owner too, so the compliance and resistance mechanics of an occupation never come into play.

## Want a different effect?

- Remove another country's core: remove a core (`wiki:effect:remove_state_core`), written at country level like adding one.
- Make the focus available only while a state is owned: use check ownership of a state (`wiki:trigger:owns_state`) in `available`.
- Transfer several states at once: one line per state id; `transfer_state` does not take a list.
