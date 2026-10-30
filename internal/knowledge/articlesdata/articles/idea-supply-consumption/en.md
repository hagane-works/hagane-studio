---
contentId: idea-supply-consumption
kind: case
slug: idea-supply-consumption
locale: en
status: published
title: "Use a national spirit to reduce supply consumption by 10%"
summary: "Define a spirit with Supply Consumption -10%, then grant it through an existing focus. Keep the minus sign."
primaryTopic: military
topics: [military]
scenarios: [ideas, focus]
gameTerms: [National spirit, Supply Consumption, Focus reward]
aliases: [supply-saving national spirit, troops use less supply, logistics spirit]
relatedIds: [supply-consumption, "wiki:modifier:supply_consumption_factor", first-content, localisation, check-and-troubleshoot]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "Reduce land troops' supply consumption through a national spirit. A complete ideas/country definition uses supply_consumption_factor = -0.10 inside modifier. An existing focus grants the spirit with add_ideas inside completion_reward."
sources:
  - label: "HOI4 Wiki: National spirits"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
  - label: "HOI4 Wiki: Supply Consumption"
    url: "https://hoi4.paradoxwikis.com/Modifiers#supply_consumption_factor"
  - label: "HOI4 Wiki: Grant a national spirit"
    url: "https://hoi4.paradoxwikis.com/Effects#add_ideas"
---
The goal is to give a country's land troops Supply Consumption -10%. A national spirit carries the modifier, and an existing focus grants the spirit.

You need a Mod that loads correctly and an existing, uncompleted focus you can use for testing. This example does not create a focus tree.

## 1. Define the national spirit

Create `common/ideas/hw_supply_ideas.txt` using UTF-8 without BOM. This is the **complete file**:

```hoi4
ideas = {
    country = {
        HW_supply_training = {
            picture = generic_army_war_college
            removal_cost = -1
            modifier = {
                supply_consumption_factor = -0.10
            }
        }
    }
}
```

`HW_supply_training` is this example's unique spirit ID. The `picture` value refers to the existing vanilla sprite `GFX_idea_generic_army_war_college`. Omit the `GFX_idea_` prefix in this field; no new image is needed.

The key line is `supply_consumption_factor = -0.10`. Negative values reduce consumption; positive values increase it. This adds a percentage modifier, so it does not mean that the final value shown with any other existing modifiers will always be exactly 0.9 times its previous value.

## 2. Give it an English name

Create `localisation/english/hw_supply_l_english.yml` using UTF-8 with BOM. This is the **complete localisation file**:

```yaml
l_english:
 HW_supply_training: "Logistics Training"
 HW_supply_training_desc: "Improve supply-use practices to reduce the demand of our land troops."
```

Display text is separate from script IDs. Keep `HW_supply_training` and `supply_consumption_factor` unchanged in the definition; edit the quoted localisation values to change what players read.

## 3. Grant the spirit through an existing focus

Find the chosen focus's `completion_reward`. Add this **single complete effect** inside that reward block, preserving any existing rewards. Do not put it inside the spirit's `modifier` block.

```hoi4
add_ideas = HW_supply_training
```

A definition alone is not enough: the country must receive the spirit to benefit from it. If the focus was already completed, saving the file does not make its completion reward run again. Use a separate local test game with a known starting state.

## 4. Check the result

1. Inspect the focus reward tooltip and confirm that it grants Logistics Training.
2. Complete the focus. Confirm that the target country receives the spirit and that its tooltip shows Supply Consumption -10%.
3. Compare land troop supply demand while keeping other conditions as similar as possible. Changing division composition, deployments and transport conditions at the same time makes the comparison unreliable.
4. Check this launch's logs for errors mentioning the file, spirit ID or modifier.

This does not create a railway or supply hub, and it does not guarantee that undersupplied troops immediately receive enough supply. It changes demand.

These are acceptance checks, not a completed playtest record. The example has been checked against vanilla national spirits, modifier documentation and sprites for 1.19.2.0, but has not been tested in the game.

## Remove it, or make it a harmful spirit

To stop this spirit from providing its modifier, run this **effect snippet** in an appropriate country effect block:

```hoi4
remove_ideas = HW_supply_training
```

To increase the logistical burden instead, read increase or reduce supply consumption (`wiki:modifier:supply_consumption_factor`). Confirm the sign and update the display text together; do not rename this spirit as a penalty while leaving its negative consumption modifier unchanged.
