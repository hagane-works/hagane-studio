---
contentId: "wiki:trigger:num_of_civilian_factories"
kind: reference
slug: num_of_civilian_factories
locale: en
status: published
title: "Check a civilian-factory requirement"
summary: "Use num_of_civilian_factories to check a country's count. At least ten differs from more than ten. Checking does not remove, reserve or grant factories."
primaryTopic: industry
topics: [industry]
scenarios: [decision, focus, event]
gameTerms: ["Civilian Factory","Decision Requirement","Focus Requirement"]
aliases: ["civilian factory count","at least 10 civs","not enough civilian factories","factory threshold"]
relatedIds: [construction-speed, idea-construction-speed, "wiki:effect:add_building_construction", "wiki:modifier:production_speed_buildings_factor", "wiki:trigger:num_of_civilian_factories_available_for_projects"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: trigger/num_of_civilian_factories
searchText: "num_of_civilian_factories check civilian factory count at least 10 fewer than 10 more than 10 at most 10 insufficient civilian factories not spend not reserve not idle factories"
sources:
  - label: "HOI4 Wiki: num_of_civilian_factories"
    url: "https://hoi4.paradoxwikis.com/Triggers#num_of_civilian_factories"
  - label: "HOI4 Wiki: Decision modding"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
---

Technical name: `num_of_civilian_factories` · Trigger · Country scope.

## 1. What happens in the game, and where can I see it?

It checks whether the current country's civilian-factory count satisfies a comparison. For example, an ordinary decision's `available` can prevent selection until enough factories are present.

It does not add, remove or occupy factories, and does not change construction speed. Civilian-factory count and military-factory equipment output are different concepts; the requirement should explain which one matters.

## 2. Who is affected, and when does it start or stop?

It checks the current country, not one state's building count or the capital's free building slots. Each evaluation reads the current result; a successful check does not reserve ten factories.

Do not interpret it as “idle civilian factories available for any new project”. The game has a separate trigger for factories available to a new project. The bundled description does not explain every trade, occupation or off-map counting rule, so verify specialised designs in the target version.

## 3. Which values or parameters do I use?

Compare counts, not percentages. Do not use `0.10` to mean ten factories.

| Requirement | Condition | At exactly ten |
| --- | --- | --- |
| Fewer than ten | `num_of_civilian_factories < 10` | False |
| More than ten | `num_of_civilian_factories > 10` | False |
| At least ten | `NOT = { num_of_civilian_factories < 10 }` | True |
| At most ten | `NOT = { num_of_civilian_factories > 10 }` | True |

Inclusive examples negate one comparison. Do not write `> 10` for “at least ten” or assume another trigger's `=` behaviour applies here.

## 4. Where does it go, and what does the script look like?

This is a condition-block snippet inside an existing ordinary decision:

```pdx
available = {
    NOT = {
        num_of_civilian_factories < 10
    }
}
```

Merge into an existing `available` block without removing its other requirements. Keeping the decision visible lets the player inspect what is missing. Putting this check in `visible` hides the decision instead.

If your design needs to pay for continuing use of factories, this check is not sufficient. Do not simulate such a cost by deleting buildings after the condition passes.

## 5. What might I need next?

- Grant a construction bonus when the requirement is met: the 180-day national-spirit case (`idea-construction-speed`).
- Grant an actual factory: add a building (`wiki:effect:add_building_construction`).
- Change construction speed: the construction-speed modifier (`wiki:modifier:production_speed_buildings_factor`).
- Design a project that uses factories: the base num_of_civilian_factories_available_for_projects reference (`wiki:trigger:num_of_civilian_factories_available_for_projects`). Check the project's reservation rules separately.
- Return to construction speed (article `construction-speed`).
