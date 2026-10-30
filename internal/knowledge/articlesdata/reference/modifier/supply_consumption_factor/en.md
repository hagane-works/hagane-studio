---
contentId: "wiki:modifier:supply_consumption_factor"
kind: reference
slug: supply_consumption_factor
locale: en
status: published
title: "Increase or reduce your troops' supply consumption"
summary: "Change the supply needed by land troops. Negative values reduce consumption; positive values increase it."
primaryTopic: military
topics: [military]
scenarios: [ideas, focus, event, decision]
gameTerms: [Supply Consumption, Land troops, National spirit]
aliases: [troop supply demand, use less supply, use more supply, lower supply consumption, higher supply consumption]
relatedIds: [supply-consumption, idea-supply-consumption]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/supply_consumption_factor
searchText: "Supply Consumption supply_consumption_factor changes the supply needed by land troops. -0.10 gives Supply Consumption -10%; 0.10 gives +10%. This example uses a national spirit's modifier block. It does not increase supply availability, hub capacity or railway throughput."
sources:
  - label: "HOI4 Wiki: Supply Consumption modifier"
    url: "https://hoi4.paradoxwikis.com/Modifiers#supply_consumption_factor"
  - label: "HOI4 Wiki: National spirits"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---
## 1. What changes in the game, and where can I see it?

Supply Consumption is the supply land troops need to operate effectively. Reducing it lowers their demand; increasing it makes them consume more.

Check the national spirit's modifier tooltip and the land troops' supply information. Do not judge the result solely by the supply map's colours: deployments, railways and supply hubs also affect the supply situation.

## 2. Who receives it, and when does it apply or end?

This example uses a national spirit. It affects the land troops of the country that has the spirit, for as long as the spirit remains active. Removing the spirit stops it from providing this modifier.

Vanilla already uses `supply_consumption_factor` in national spirits' `modifier` blocks. Its army category in the documentation does not mean it can only be written in a unit definition. Nor does it make the modifier a valid effect to write directly in a focus reward.

## 3. What do the numbers and parameters mean?

| Setting | Result |
| --- | --- |
| `supply_consumption_factor = -0.10` | Supply Consumption -10%; troops need less supply |
| `supply_consumption_factor = 0.10` | Supply Consumption +10%; troops need more supply |
| `supply_consumption_factor = 0` | No change from this modifier |

`-0.10` does not mean a flat reduction of 0.1 supply. If your intention is to help troops by reducing their demand, use a negative value rather than assuming that a positive number is beneficial.

This modifier does not build railways, supply hubs or transport capacity, and it does not change fuel consumption. Supply can still be insufficient even after demand falls.

## 4. Where does it go, and how do I write it?

This is a **partial snippet inside a national spirit definition**, not a complete file. It already includes the `modifier` wrapper. If that block exists, add only the inner line rather than nesting another `modifier` inside it.

```hoi4
modifier = {
    supply_consumption_factor = -0.10
}
```

You need both the spirit definition and an effect that grants it. A focus reward uses `add_ideas`, not this modifier block directly.

A national spirit that reduces supply consumption by 10% (`idea-supply-consumption`) includes the complete definition, display text and the insertion point in an existing focus.

## 5. What might I need next?

- Distinguish using less supply from delivering more: Supply consumption guide (article `supply-consumption`).
- A defined modifier whose spirit has not reached the country: Complete spirit example (`idea-supply-consumption`).

Checked against the bundled modifier documentation, vanilla national spirits and game display text for 1.19.2.0. This example has not been tested in the game.
