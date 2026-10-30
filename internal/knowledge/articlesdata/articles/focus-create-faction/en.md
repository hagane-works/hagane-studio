---
contentId: focus-create-faction
kind: case
slug: focus-create-faction
locale: en
status: published
title: "Found a faction and invite a neighbour"
summary: "A focus founds a faction with your country as leader, and a second focus pulls a neighbour in; along the way, the rule you must grant yourself first, the faction name being a localisation key, and which direction each of the two opinion effects changes. Half the base game's faction-founding focuses carry the same rule line in front, but the effect itself does not need it."
primaryTopic: diplomacy
topics: [diplomacy, politics]
scenarios: [focus, event]
gameTerms: [faction, found a faction, join a faction, faction leader, opinion, rule, focus, completion reward]
aliases: [focus founds faction, create my own faction, invite a country into faction, faction does not appear, faction name shows a key, add opinion, opinion direction, create_faction, add_to_faction, set_rule, add_opinion_modifier, reverse_add_opinion_modifier]
relatedIds: [diplomacy-system, focus-wargoal-annex, custom-autonomy-state, "wiki:effect:create_faction", "wiki:effect:create_faction_from_template", "wiki:effect:add_to_faction", "wiki:effect:set_rule", "wiki:effect:add_opinion_modifier", "wiki:effect:reverse_add_opinion_modifier", "wiki:trigger:is_in_faction_with", "wiki:trigger:has_rule", "wiki:trigger:has_opinion"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki: Triggers"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

# Found a faction and invite a neighbour

## Goal and observable result

When this page is done you should see three things in game:

1. The moment the first focus completes, a faction named by you appears in the top bar, with your country as its leader.
2. After the second focus completes, Belgium appears in the member list.
3. Before that, Belgium's opinion of your country has already risen through one modifier, and the diplomacy panel shows the modifier's name and value.

## What you need before starting

- A mod that already exports correctly, and two focuses in sequence. This page uses the placeholder IDs `my_focus_found_league` and `my_focus_invite_belgium`.
- A localisation entry for the faction name; this page uses the key `my_league_name`.
- A neighbour's tag; this page uses Belgium, `BEL`, played as Luxembourg.
- No DLC is required.
- Read diplomacy: where starting relations, runtime effects and definitions each live (`diplomacy-system`) first.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| Two focuses | `common/national_focus/` | the first founds, the second invites |
| Opinion modifier definition | `common/opinion_modifiers/` | make them like you before inviting |
| Faction name | `localisation/` | the value of `create_faction` is the key |
| Rule | `set_rule` on the country | the effect does not need it; the player's manual faction button does |

## 1. What that rule line governs

Of the base game's 61 uses of `create_faction`, 31 are immediately preceded by the same line:

```
set_rule = { can_create_factions = yes }
```

Tested: **without this line, `create_faction` founds the faction all the same**. In game the rule displays as "Can Create Factions"; it governs the player's own create-faction button on the diplomacy screen, not the effect. All 225 places where the base game grants it are focuses, events and decisions, none in history files, so by default no country can found a faction by hand; a focus grants the rule so the player can keep managing factions afterwards. If your focus only needs a faction to appear, you may leave it out; if the player should be able to found one manually later, write it.

## 2. Found the faction

```
focus = {
    id = my_focus_found_league
    icon = GFX_goal_generic_alliance
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        set_rule = { can_create_factions = yes }
        create_faction = my_league_name
    }
}
```

What follows `create_faction` is not display text but a localisation key. The base game writes `create_faction = AFG_united_islamic_world_faction`, matched by `AFG_united_islamic_world_faction: "Islamic League"` in the localisation files. So add a line under `localisation/english/`:

```
l_english:
 my_league_name:0 "My League"
```

Without it the faction is still founded; its name simply shows as the key.

The game's own documentation marks `create_faction` as deprecated in favour of `create_faction_from_template`. The template system belongs to the "No Compromise, No Surrender" DLC, and all 171 base-game template uses sit inside a `has_dlc` condition; a mod that does not require the DLC keeps using `create_faction`, as 27 base-game focuses still do.

## 3. Make the neighbour like you: which way the opinion effect points

`add_to_faction` ignores opinion and pulls the country straight in. Storylines usually raise opinion first, and opinion modifiers are the most common effect in this whole area (913 focus uses, 1009 event uses), so it is worth getting right here.

Direction first, because getting it backwards raises no error. The game's own tooltip text settles it:

| Effect | Tooltip shown in game | Whose opinion of whom |
| --- | --- | --- |
| `add_opinion_modifier = { target = BEL ... }` | "Gains X (Opinion of Belgium +50.)" | **yours of Belgium** |
| `reverse_add_opinion_modifier = { target = BEL ... }` | "Belgium gains X (Opinion of them changed by +50.)" | **Belgium's of you** |

To make Belgium willing to follow you, it is Belgium's opinion of you that must rise, so the effect is `reverse_add_opinion_modifier`. The base game's 87 uses of it are mostly exactly this: a treaty is signed and the other country's opinion of the signatory rises.

Define the modifier first. Create `common/opinion_modifiers/my_mod.txt`:

```
opinion_modifiers = {
    my_league_friendship = {
        value = 50
        months = 12
        decay = 1
    }
}
```

`value` is the opinion amount; `months = 12` with `decay = 1` fades it by one a day over a year until it is gone; for a permanent one write only `value`. The field names come from the header comment of the base-game file, and those are all of them.

Then reference it in the focus, with the `reverse_` form:

```
completion_reward = {
    reverse_add_opinion_modifier = {
        target = BEL
        modifier = my_league_friendship
    }
}
```

🔴 `add_opinion_modifier` would run too; it would just make you like Belgium more while Belgium's view of you stays put, and nothing in the log would say so.

## 4. Invite the neighbour

```
focus = {
    id = my_focus_invite_belgium
    prerequisite = { focus = my_focus_found_league }
    x = 0
    y = 1
    cost = 5
    available = {
        is_faction_leader = yes
    }
    completion_reward = {
        add_to_faction = BEL
    }
}
```

`add_to_faction` runs in the **faction leader's** scope, and its value is the country to bring in. The most common base-game event form is a scope jump such as `FROM = { add_to_faction = ROOT }`; a literal tag is more usual in a focus. `is_faction_leader = yes` in `available` guarantees your country already leads a faction; otherwise this focus should not be startable.

## 5. Use events instead of focuses

All three effects move unchanged into event options; fire them from the console with `event my_events.1` while testing.

## 6. Check each item

| Check | Expected | If not, look at |
| --- | --- | --- |
| Top bar after the first focus | a faction, your country as leader | whether `create_faction` is in the reward; `error.log` |
| Faction name | the localised name | whether the key matches the value of `create_faction` |
| "Belgium's opinion of you" on Belgium's diplomacy panel | the modifier shown, value matching `value` | whether the modifier is defined under `common/opinion_modifiers/`; whether the `reverse_` form was used |
| Second focus | startable once you lead a faction | `is_faction_leader` |
| After the second focus | Belgium in the member list | whether Belgium is already in another faction |

## Common mistakes

- **Assuming a faction cannot be founded without `set_rule`**: the effect ignores the rule; tested, it founds. The rule governs manual founding by the player.
- 🔴 **Display text after `create_faction`**: the value must be a localisation key; plain text is treated as a key and shows verbatim.
- 🔴 **Using `add_opinion_modifier` to make the other side like you**: that changes your opinion of them. Their opinion of you takes `reverse_add_opinion_modifier`.
- **Referencing an undefined opinion modifier**: no error, no change.
- **`add_to_faction` run by a country that is not the leader**: the target does not join. Guard with `is_faction_leader`.
- **The target is already in another faction**: check `is_in_faction = no` first, or make it leave; what `add_to_faction` does in that case was not tested for this page.

## Want a different effect?

- Join someone else's faction rather than founding one: run `add_to_faction` in the leader's scope, e.g. `GER = { add_to_faction = ROOT }`.
- Check whether two countries share a faction: `is_in_faction_with = BEL`.
- Check whether their opinion of you is high enough: `BEL = { has_opinion = { target = ROOT value > 50 } }`; the base game writes `value >`.
- A template faction for DLC owners: `create_faction_from_template = template_id` inside `has_dlc = "No Compromise, No Surrender"`.
