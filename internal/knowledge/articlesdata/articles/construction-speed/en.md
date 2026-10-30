---
contentId: construction-speed
kind: topic
slug: construction-speed
locale: en
status: published
title: "Construction speed: build faster or grant a factory?"
summary: "Construction speed, an immediate building reward and a civilian-factory requirement are three different uses. Choose the player's result without confusing construction with equipment output."
primaryTopic: industry
topics: [industry]
scenarios: [ideas, focus, event, decision]
gameTerms: ["Construction Speed","Civilian Factory","National Spirit"]
aliases: ["build speed","build factories faster","civs","free factory","building construction"]
relatedIds: ["wiki:modifier:production_speed_buildings_factor", "wiki:effect:add_building_construction", "wiki:trigger:num_of_civilian_factories", "wiki:trigger:free_building_slots", "wiki:modifier:industrial_capacity_factory", idea-construction-speed]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "construction speed build faster slower national spirit construction speed +10% -10% give one civilian factory immediately at least 10 civilian factories not equipment output not daily factories"
sources:
  - label: "HOI4 Wiki: Construction speed modifier"
    url: "https://hoi4.paradoxwikis.com/Modifiers#production_speed_buildings_factor"
  - label: "HOI4 Wiki: Building effects"
    url: "https://hoi4.paradoxwikis.com/Effects#add_building_construction"
  - label: "HOI4 Wiki: Factory-count conditions"
    url: "https://hoi4.paradoxwikis.com/Triggers#num_of_civilian_factories"
  - label: "HOI4 Wiki: Idea modding"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---

“More industry” might mean building factories faster, or receiving a factory immediately. Choose the result before choosing the script.

## Choose the result

| Goal | Use | What it does not do |
| --- | --- | --- |
| Add or remove 10% construction speed while a national spirit is active | Construction speed modifier (`wiki:modifier:production_speed_buildings_factor`) | Does not grant factories or increase military-factory equipment output |
| Grant a civilian factory immediately through a focus or event | Add a building (`wiki:effect:add_building_construction`) | Does not make all future construction faster |
| Require a certain number of civilian factories | Civilian-factory threshold (`wiki:trigger:num_of_civilian_factories`) | Does not grant, remove or reserve factories |

For complete files, see a construction-speed national spirit lasting 180 days (`idea-construction-speed`). It includes the spirit, its granting decision, duration and English localisation.

## Construction speed is not equipment output

Despite the word `production` in `production_speed_buildings_factor`, its English game label is “Construction Speed”. It affects building construction.

To make military factories produce more equipment, look in a different direction, such as the base Factory Output reference (`wiki:modifier:industrial_capacity_factory`). Separate “build factories faster” from “factories produce more equipment” before interpreting “factory speed”.

## Values and duration are separate

A construction-speed modifier of `0.10` means +10%; `-0.10` means -10%. It neither grants 0.1 factories per day nor directly reduces the remaining construction time by 10%. Factory allocation, other modifiers and construction conditions also affect the duration.

The national spirit lasts 180 days, not the modifier key itself. Defining a spirit's `modifier` does not automatically grant it to a country; an effect must do that.

## An immediate building reward still needs a location

“A country receives a civilian factory” requires choosing the state in which it will be built. `add_building_construction` executes in State scope. When calling it from a country event, first scope to the target state.

Check free building slots (`wiki:trigger:free_building_slots`) too. Do not mistake “nowhere to build” for “construction speed is too low”, or change global building caps casually for a single reward.
