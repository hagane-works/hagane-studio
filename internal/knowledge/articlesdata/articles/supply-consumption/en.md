---
contentId: supply-consumption
kind: topic
slug: supply-consumption
locale: en
status: published
title: "Change your troops' supply consumption"
summary: "Making troops use less supply is not the same as delivering more supply. First decide which side of the problem you want to change."
primaryTopic: military
topics: [military]
scenarios: [ideas, focus, event, decision]
gameTerms: [Supply Consumption, Supply demand, Land troops]
aliases: [troops use less supply, troops use more supply, not enough supply, lower supply demand]
relatedIds: ["wiki:modifier:supply_consumption_factor", idea-supply-consumption, first-content]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "Troops using less supply means lower supply consumption; using more means higher consumption. Negative supply_consumption_factor reduces demand, while positive values increase it. A national spirit can provide the modifier. This does not build railways, increase supply delivery or reduce fuel use."
sources:
  - label: "HOI4 Wiki: Supply Consumption modifier"
    url: "https://hoi4.paradoxwikis.com/Modifiers#supply_consumption_factor"
  - label: "HOI4 Wiki: National spirits"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---
Supply Consumption is the supply your land troops need. Reduce it to ease their demand under the same supply conditions, or increase it to create a logistical burden.

## Are you changing consumption or delivery?

| Your goal | Does this page apply? |
| --- | --- |
| Make the country's land troops consume 10% less supply | Yes: use a negative consumption modifier |
| Add a harmful spirit that makes troops need more supply | Yes: use a positive consumption modifier |
| Build railways, improve supply hubs or increase delivery | No: these are different settings, not covered here |
| Make tanks consume less fuel | No: fuel consumption is not supply consumption |

Supply shortages can have many causes. Lower demand does not guarantee adequate supply, and does not replace checking the transport network.

## A beneficial change uses a negative number here

- `supply_consumption_factor = -0.10`: Supply Consumption -10%; troops need less supply.
- `supply_consumption_factor = 0.10`: Supply Consumption +10%; troops need more supply.

These are parameter comparisons, not standalone files. They add percentage modifiers. `-0.10` is not a flat reduction of 0.1 supply; do not drop the minus sign.

## Choose a page to continue

Increase or reduce supply consumption (`wiki:modifier:supply_consumption_factor`) explains who receives the modifier, the values and where to put it in a national spirit.

A spirit that reduces supply consumption by 10% (`idea-supply-consumption`) includes a complete definition, localisation and instructions for granting it through an existing focus.

## Why not write it directly in the focus reward?

`supply_consumption_factor` is a modifier, not an effect that runs once. Here, a national spirit carries the modifier, and the focus uses `add_ideas` to give that spirit to the country.

To check the result, first confirm that the country received the spirit, then inspect its Supply Consumption -10% tooltip. Map colours are affected by troops and transport conditions too, so they cannot prove the script is correct by themselves.

Checked against the vanilla descriptions and national-spirit usage for 1.19.2.0. This example has not been tested in the game.
