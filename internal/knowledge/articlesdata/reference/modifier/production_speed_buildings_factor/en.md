---
contentId: "wiki:modifier:production_speed_buildings_factor"
kind: reference
slug: production_speed_buildings_factor
locale: en
status: published
title: "Increase or decrease construction speed"
summary: "Use production_speed_buildings_factor to change building construction speed. A value of 0.10 means +10%; it neither grants a factory nor increases equipment output."
primaryTopic: industry
topics: [industry]
scenarios: [ideas, decision]
gameTerms: ["Construction Speed","National Spirit","Civilian Factory"]
aliases: ["build speed","factory construction speed","construction speed +10%","construction speed -10%","faster construction","slower construction"]
relatedIds: [construction-speed, idea-construction-speed, "wiki:effect:add_building_construction", "wiki:trigger:num_of_civilian_factories", "wiki:modifier:industrial_capacity_factory"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/production_speed_buildings_factor
searchText: "production_speed_buildings_factor Construction Speed increase construction speed +10% decrease construction speed -10% build civilian factories faster national spirit ongoing modifier not immediate factory not equipment output"
sources:
  - label: "HOI4 Wiki: production_speed_buildings_factor"
    url: "https://hoi4.paradoxwikis.com/Modifiers#production_speed_buildings_factor"
  - label: "HOI4 Wiki: Idea modding"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---

Technical name: `production_speed_buildings_factor` · Modifier · Country modifier.

## 1. What happens in the game, and where can I see it?

The game labels it “Construction Speed”; it modifies the construction speed of all buildings. When a national spirit carries it, inspect the spirit's bonus or penalty tooltip and the construction interface.

Here, `production` does not mean military factories producing equipment. The modifier also creates no buildings. With no construction assigned, it cannot grant a factory on its own.

## 2. Who is affected, and when does it start or stop?

This page uses a national spirit as the source. Its country receives the modifier while the spirit is present and active. When the source stops applying, its speed contribution stops; completed buildings are not removed.

The key has no duration and is not a daily reward. For a 180-day bonus, set the duration when granting the spirit. With other sources active, this spirit's +10% does not guarantee that the previous total construction speed is multiplied by exactly 1.10.

## 3. Which values or parameters do I use?

| Goal | Modifier value |
| --- | --- |
| Construction speed +10% | `production_speed_buildings_factor = 0.10` |
| Construction speed -10% | `production_speed_buildings_factor = -0.10` |

Use a fractional percentage value without a percent sign. `0.10` is not 0.1 factories per day. +10% speed is also not the same as subtracting 10% from remaining construction time. Inspect other modifiers, construction conditions and allocated factories too.

For just one building type, verify its building-specific modifier separately. This key is not limited to civilian factories.

## 4. Where does it go, and what does the script look like?

This is a snippet inside an existing national spirit:

```pdx
modifier = {
    production_speed_buildings_factor = 0.10
}
```

Place it inside the spirit definition. If `modifier` already exists, merge this line into it. Do not put it directly inside an event `option` or ordinary focus `completion_reward`; it is not an executable effect.

To grant the bonus through a focus or event, define the spirit first and then grant it with an effect. The complete national-spirit case (`idea-construction-speed`) includes the definition, one-off decision, 180-day duration and localisation. For a penalty, update both the value and the narrative instead of leaving positive-reward text.

## 5. What might I need next?

- Receive a factory immediately: add a building (`wiki:effect:add_building_construction`).
- Require a number of civilian factories: factory-count conditions (`wiki:trigger:num_of_civilian_factories`).
- Improve military-factory equipment output: the base industrial_capacity_factory reference (`wiki:modifier:industrial_capacity_factory`).
- Return to construction speed (article `construction-speed`).
