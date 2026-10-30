---
contentId: diplomacy-system
kind: guide
slug: diplomacy-system
locale: en
status: published
title: "Diplomacy: where starting relations, runtime effects and definitions each live"
summary: "Diplomacy in script has three layers: starting relations go in the history file, changes during play go in focus or event effects, and the three kinds of definition (war goal types, opinion modifiers, autonomy levels) each have their own folder. Mixing the layers puts a day-one relation into a focus, or references an opinion modifier nobody defined."
primaryTopic: diplomacy
topics: [diplomacy, politics]
scenarios: [focus, event, history]
gameTerms: [diplomacy, faction, war goal, opinion, subject, puppet, autonomy, guarantee, non-aggression pact, claim]
aliases: [how to do diplomacy, how to create a faction, how to grant a war goal, how to add opinion, how to make a puppet, subject level, diplomatic_relation, add_opinion_modifier, create_wargoal, set_autonomy]
relatedIds: [focus-wargoal-annex, focus-create-faction, custom-autonomy-state, focus-transfer-state-core, state-file-basics, "wiki:effect:create_wargoal", "wiki:effect:add_opinion_modifier", "wiki:effect:create_faction", "wiki:effect:add_to_faction", "wiki:effect:set_autonomy", "wiki:effect:diplomatic_relation", "wiki:effect:give_guarantee", "wiki:trigger:has_wargoal_against", "wiki:trigger:is_in_faction_with", "wiki:trigger:is_subject_of", "wiki:trigger:has_opinion"]
updatedAt: "2026-09-15"
verification: static-checked
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Diplomacy"
    url: "https://hoi4.paradoxwikis.com/Diplomacy"
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki: Triggers"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki: Country creation"
    url: "https://hoi4.paradoxwikis.com/Country_creation"
---

# Diplomacy: where starting relations, runtime effects and definitions each live

## What this system covers

Everything the player sees on the diplomacy screen: who has a non-aggression pact with whom, who guarantees whom, who is in which faction, who is whose subject, the opinion number between two countries, whether war can be declared. Some of these exist at game start; others are changed during play by focuses, events and decisions.

As an author you really decide only three things:

1. **What two countries' relation is at game start** (written in the history file, applied once when the game begins)
2. **When a relation changes during play** (written in a focus reward, an event option, a decision effect)
3. **What the "thing" being referenced is** (the war goal type, the opinion modifier's name, the autonomy level's name; each must first exist in a definition file)

Three things, three file locations. Most diplomacy script mistakes come from putting one layer's writing into another.

## Core idea: state, effect, definition

The nouns in diplomacy script fall into three groups with different shapes:

| Group | Example | Where | When it applies |
| --- | --- | --- | --- |
| **Starting state** | guarantee Belgium, a non-aggression pact with Turkey, membership of a faction, being a subject | that country's file under `history/countries/` | at game start, once |
| **Runtime effect** | a war goal on a neighbour when a focus completes, pulling a country into a faction from an event, raising opinion from a decision | a focus's `completion_reward`, an event `option`, a decision's effect block | when that effect runs |
| **Definition** | war goal type `annex_everything`, opinion modifier `holds_our_cores`, autonomy level `autonomy_puppet` | `common/wargoals/`, `common/opinion_modifiers/`, `common/autonomous_states/` | only when an effect references it; it does nothing by itself |

Starting state and runtime effects mostly use **the same effects**. A history file is essentially an effect script run at game start, so `give_guarantee`, `add_to_faction`, `set_autonomy` and `diplomatic_relation` can be written in a history file or in a focus. Only the timing differs.

A definition is not an effect. Writing `my_friendship = { value = 30 }` under `common/opinion_modifiers/` changes nothing; only when something runs `add_opinion_modifier = { target = GER modifier = my_friendship }` does Germany's opinion move.

## File structure and how it runs

### Starting relations: the history file

In the base game each country's history file carries diplomacy statements at the top level (before the 1936 block):

```
add_to_faction = PRC

give_guarantee = AFA

set_autonomy = {
    target = COG
    autonomous_state = autonomy_colony
    freedom_level = 0.2
}

diplomatic_relation = {
    country = TUR
    relation = non_aggression_pact
    active = yes
}

add_opinion_modifier = {
    target = GER
    modifier = my_opinion_modifier
}
```

Across base-game history files, focuses and events, `diplomatic_relation` only ever uses six `relation` values: `non_aggression_pact` (139), `guarantee` (110), `military_access` (35), `docking_rights` (14), `puppet` and `air_base_access` (2 each). That is an exhaustive count, not a sample.

### Runtime effects: focus, event, decision

Move the same effects into `completion_reward` or an `option` and you have the runtime version. The most common diplomacy effect in base-game focuses is `create_wargoal` (1072 uses), then `add_opinion_modifier` (913), `add_state_claim` (507) and `set_autonomy` (149); events use `add_to_faction` 494 times.

```
focus = {
    id = my_focus_claim_neighbour
    icon = GFX_goal_generic_territory_or_war
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        add_state_claim = 6
        create_wargoal = {
            type = annex_everything
            target = BEL
        }
    }
}
```

### Definitions: three folders

| Folder | Outer block | One definition looks like |
| --- | --- | --- |
| `common/wargoals/` | `wargoal_types = { ... }` | `annex_everything = { allowed = { always = no } threat = 0.5 }` |
| `common/opinion_modifiers/` | `opinion_modifiers = { ... }` | `my_friendship = { value = 30 months = 12 decay = 1 }` |
| `common/autonomous_states/` | one `autonomy_state = { id = ... }` per file | see a custom autonomy level for a subject (`custom-autonomy-state`) |

Opinion modifier fields are exactly those listed in the header comment of the base-game file: `value`, `min_trust` / `max_trust`, `decay` (per day), `days` / `months` / `years` (timer), `trade`. A timed one in the base game reads `months = 12` with `decay = 1`: fading by one a day over a year until gone.

### Triggers

Diplomatic state has its own set of triggers for a focus's `available`, an event's `trigger`, a decision's visibility. These are the common ones, spelled as in the game's documentation:

| To check | Write | Note |
| --- | --- | --- |
| a war goal against a country | `has_wargoal_against = BEL` | or `has_wargoal_against = { target = BEL type = annex_everything }` to require a type |
| in the same faction as a country | `is_in_faction_with = BEL` | any faction at all: `is_in_faction = yes` |
| a subject of a country | `is_subject_of = GER` | puppet level specifically: `is_puppet_of` |
| at a given autonomy level | `has_autonomy_state = autonomy_puppet` | the value is the level's `id` |
| guaranteed a country | `has_guaranteed = BEL` | |
| opinion above a value | `has_opinion = { target = GER value > 50 }` | the base game writes `value >`, not `=` |
| has a rule | `has_rule = can_create_factions` | rules are set with `set_rule` |

## Common goals and where to start

| What you want to see in game | Start here | The distinction that matters |
| --- | --- | --- |
| A non-aggression pact with a neighbour from day one | `diplomatic_relation` in the history file | not in a focus; that would wait for the focus |
| War on a neighbour becomes possible after a focus | a war goal on a neighbour when a focus completes (`focus-wargoal-annex`) | `type` must exist under `common/wargoals/` |
| A focus founds your own faction and invites a neighbour | found a faction and invite a neighbour (`focus-create-faction`) | the faction name is a localisation key; the rule governs manual founding, not the effect |
| A defeated country at a special subject level | a custom autonomy level for a subject (`custom-autonomy-state`) | define the level under `common/autonomous_states/`, then `set_autonomy` |
| Two countries like each other more after an event | `add_opinion_modifier`, with the modifier defined under `common/opinion_modifiers/` | the definition may carry a timer and decay |
| A claim on a state after a focus | `add_state_claim = 6`, country scope | the state-scope `add_claim_by` is on transfer a state (`focus-transfer-state-core`) |

## Easy things to get wrong

### The faction rule governs the player, not the effect

Of the base game's 61 uses of `create_faction`, 31 are immediately preceded by `set_rule = { can_create_factions = yes }`. Tested: the effect founds the faction without it; the rule governs the player's manual create-faction button on the diplomacy screen. The base game grants the rule first so the player can keep managing factions afterwards. See found a faction (`focus-create-faction`).

### `create_faction` versus `create_faction_from_template`

The game's own documentation marks `create_faction` as deprecated in favour of `create_faction_from_template`. The template system belongs to the "No Compromise, No Surrender" DLC, and all 171 base-game uses of the template form sit inside a `has_dlc` condition. A mod that does not require the DLC keeps using `create_faction = localisation_key`; the 1.19.2 base game itself still does so in 27 focuses.

### `set_autonomy` has two spellings

Base-game history files write `autonomous_state =` (23 uses); focuses and events almost always write `autonomy_state =` (540), and the game's documentation uses the latter. The engine accepts both. Hagane Works' history-file exporter writes the former, matching base-game history files; in a focus, write the latter as the documentation does.

### Two opinion effects, opposite directions

`add_opinion_modifier = { target = B }` in A's scope changes **A's opinion of B**; `reverse_add_opinion_modifier` changes **B's opinion of A**. The game's tooltip text says so: the former shows "Gains X (Opinion of B +N)", the latter "B gains X (Opinion of them changed by +N)". To make another country like you, use the `reverse_` form; the wrong one raises no error. The case found a faction (`focus-create-faction`) has the side-by-side table.

### Opinion modifiers must be defined first

The name given to `add_opinion_modifier` must exist under `common/opinion_modifiers/`. An undefined name does not crash the game; nothing happens. The base game's shared modifiers live in `00_opinion_modifiers.txt`; put yours in a new file rather than editing that one.

### A war goal is not a war

`create_wargoal` only grants the right to declare war; the player (or AI) still has to declare. To start the war the moment a focus completes, use `declare_war_on = { target = BEL type = annex_everything }`, which base-game focuses do 64 times.

## In Hagane Works

- Starting relations: the history-file workbench. The exported history file carries `give_guarantee`, `add_to_faction`, `set_autonomy` and `diplomatic_relation` in the same shape as the base game above.
- War goal types: the war goals workbench, exported to `common/wargoals/` with the outer `wargoal_types` block added by the exporter.
- Autonomy levels: the autonomy states workbench, exported to `common/autonomous_states/`.
- Opinion modifiers: a sub-panel inside the custom traits panel of the characters workbench, exported to `common/opinion_modifiers/`. Not where you would look first, but it is there.
- Runtime effects: the focus workbench's completion reward and the event workbench's option effects, written as-is.

## Where to go next

- A war goal on a neighbour when a focus completes (`focus-wargoal-annex`)
- Found a faction and invite a neighbour (`focus-create-faction`)
- A custom autonomy level for a subject (`custom-autonomy-state`)
- Claims and state transfer: transfer a state and add a core when a focus completes (`focus-transfer-state-core`)
