---
contentId: "wiki:effect:add_building_construction"
kind: reference
slug: add_building_construction
locale: en
status: published
title: "Add a building in a chosen state"
summary: "Use add_building_construction to add building levels. This example immediately adds one civilian factory in the capital state, after checking ownership, control and free slots."
primaryTopic: industry
topics: [industry]
scenarios: [focus, event, decision]
gameTerms: ["Civilian Factory","Building","Construction"]
aliases: ["give one civilian factory","instant factory reward","focus grants a factory","factory in the capital","instant building"]
relatedIds: [construction-speed, "wiki:modifier:production_speed_buildings_factor", "wiki:trigger:num_of_civilian_factories", "wiki:trigger:free_building_slots"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: effect/add_building_construction
searchText: "add_building_construction gain one civilian factory immediately capital state building focus factory reward type industrial_complex level 1 instant_build yes state scope not construction speed not demolition"
sources:
  - label: "HOI4 Wiki: add_building_construction"
    url: "https://hoi4.paradoxwikis.com/Effects#add_building_construction"
  - label: "HOI4 Wiki: Scopes"
    url: "https://hoi4.paradoxwikis.com/Scopes#capital_scope"
  - label: "HOI4 Wiki: Building modding"
    url: "https://hoi4.paradoxwikis.com/Building_modding"
  - label: "HOI4 Wiki: Building-slot conditions"
    url: "https://hoi4.paradoxwikis.com/Triggers#free_building_slots"
---

Technical name: `add_building_construction` · Effect · State scope.

## 1. What happens in the game, and where can I see it?

It starts construction for the specified number of building levels in a state or province. This page only demonstrates a civilian factory, a state building. It explicitly sets `instant_build = yes` to complete the reward immediately; inspect the factory count in the target state.

It is not a construction-speed bonus. To make future building work faster, use a construction-speed modifier instead of repeatedly granting buildings.

## 2. Who is affected, and when does it start or stop?

The effect executes in the target state, not directly in Country scope. The example scopes from a country event option to that country's capital state and checks that the event recipient still owns and controls it.

Each execution attempts to add the specified levels. There is no automatic expiry or built-in “grant once” restriction. The focus, event or decision caller must handle repetition.

## 3. Which values or parameters do I use?

| Parameter | Example | Meaning |
| --- | --- | --- |
| `type` | `industrial_complex` | A civilian factory, not a military factory |
| `level` | `1` | Add one level—one civilian factory here; does not set the total to one |
| `instant_build` | `yes` | Complete it immediately; `no` starts normal construction |

The example does not omit `instant_build` and leave the reader to guess whether to wait for construction. `level` is a building-level count, not points or a percentage. Do not infer a demolition method by changing it to a negative number.

The target still needs suitable slots and limits. The check below uses only unlocked free shared slots. It does not expand capacity or claim to cover naval bases, railways and every other building type.

## 4. Where does it go, and what does the script look like?

This is an effect-block snippet inside an existing ordinary country event's `option`. The country must have a valid capital. No outer country-scope switch has occurred, so `ROOT` is the event recipient:

```pdx
capital_scope = {
    if = {
        limit = {
            is_owned_by = ROOT
            is_controlled_by = ROOT
            free_building_slots = {
                building = industrial_complex
                size > 0
                include_locked = no
            }
        }
        add_building_construction = {
            type = industrial_complex
            level = 1
            instant_build = yes
        }
    }
}
```

The two scopes are the country event and the capital state. Do not remove `capital_scope` and leave the building effect at country level, or paste the whole snippet into a state history file's `buildings` block.

If the capital lacks a free slot or fails the ownership/control checks, this snippet grants no factory and does not search another state. Finished content should explain this limit through its entry conditions or tooltip, or use a separately designed and verified state-selection rule.

## 5. What might I need next?

- Check slots for other buildings: the base free_building_slots reference (`wiki:trigger:free_building_slots`).
- Make later construction faster: construction speed (`wiki:modifier:production_speed_buildings_factor`).
- Require a factory count: num_of_civilian_factories (`wiki:trigger:num_of_civilian_factories`), which does not check free slots in the capital.
- Return to construction speed (article `construction-speed`).
