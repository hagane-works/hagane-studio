---
contentId: focus-political-power-reward
kind: case
slug: focus-political-power-reward
locale: en
status: published
title: "Gain 150 political power on completing a focus"
summary: "Add a complete reward node to an existing focus tree, give it an English name, and check its one-off political power reward."
primaryTopic: politics
topics: [politics]
scenarios: [focus]
gameTerms: ["Political Power","National Focus","Focus Reward"]
aliases: ["focus political power reward","gain 150 PP after a focus","focus completion reward"]
relatedIds: [political-power, "wiki:effect:add_political_power", "wiki:trigger:has_political_power", "wiki:modifier:political_power_gain", "wiki:modifier:political_power_factor"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "focus completion gain 150 political power one-off reward completion_reward add_political_power positive reward English localisation existing focus tree"
sources:
  - label: "HOI4 Wiki: National focus modding"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
  - label: "HOI4 Wiki: add_political_power"
    url: "https://hoi4.paradoxwikis.com/Effects#add_political_power"
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

## What will the player see?

The target country's existing focus tree gains an “Administrative Reform” node. Completing it runs `add_political_power = 150` once, granting 150 political power. This case creates no national spirit and does not change daily gain.

The example is source-checked, not game-tested. The checks below are work still to perform, not a record of a successful run.

## Before you start

- Use a mod that already loads successfully and a country that actually uses the chosen focus tree in your start date.
- Locate the tree's file under `common/national_focus/` and its `focus_tree` ID. If the file contains several trees, select the correct block.
- Keep a recoverable copy of your changes. Edit your mod, not the game's installed vanilla files.
- Check that no other node uses `HAG_pp_administrative_reform` and that position `(12, 0)` is free. If you change the ID or position, update the corresponding localisation keys as needed.

This is a complete case for adding a reward node to an existing, loaded tree. It does not create a country, assign a focus tree or provide a standalone mod.

## 1. Insert a complete node

In the existing focus file, insert this inside the target `focus_tree = { ... }`, at the same level as its other `focus = { ... }` nodes:

```pdx
focus = {
    id = HAG_pp_administrative_reform
    icon = GFX_goal_generic_political_pressure
    x = 12
    y = 0
    cost = 5
    search_filters = { FOCUS_FILTER_POLITICAL }

    completion_reward = {
        add_political_power = 150
    }

    ai_will_do = {
        factor = 1
    }
}
```

This is a complete `focus` node, not a complete focus-tree file. Do not nest it inside another node or create another `focus_tree` with the same ID to append content.

The example uses a generic icon checked in the vanilla definitions, so no new image is needed. With no prerequisites, it is an independent starting node. To place it after another focus, add the appropriate prerequisite and adjust its position.

`cost = 5` is focus progress cost, not a price of 5 political power. Under the default rule checked for this article, one progress point takes 7 days, giving a base duration of 35 days. A mod can change that rule.

`ai_will_do` is an AI selection weight, not an extra AI reward. The political power effect belongs to `completion_reward`, so it runs on normal completion, not when the player starts the focus.

## 2. Add the English name and description

Create `localisation/english/hag_pp_focus_l_english.yml` in your mod and save it as UTF-8 with BOM:

```yaml
l_english:
 HAG_pp_administrative_reform:0 "Administrative Reform"
 HAG_pp_administrative_reform_desc:0 "Clarify the responsibilities of each department and prepare the administration for the reforms ahead."
```

The language header, filename suffix and directory all correspond to the English game interface. Keep a space before each key and quote each value. If you changed the focus ID, change both localisation keys too.

This file covers only English game text. Other game languages need their own files. Switching Hagane Works' interface language does not generate localisation for your mod.

## 3. Check placement and recipient

Check the new node in the target tree:

- `completion_reward` contains effects, not conditions such as `has_political_power`.
- There is no extra country scope such as `GER = { ... }`, so the reward goes to the country completing this ordinary focus.
- `add_political_power` takes a numeric value, not a `value` block or a percent sign.
- If you are editing an existing node's reward, add only `add_political_power = 150` to its existing reward block. Do not copy the whole example node or create a second reward block.

The case does not require the player to hold 150 political power first. For that separate requirement, see political power conditions (`wiki:trigger:has_political_power`).

## 4. Check it in the game

These checks have not been performed. Use a new test game with the mod enabled; a file parsing successfully is not sufficient.

| Check | Expected result | If it differs, inspect |
| --- | --- | --- |
| Open the target country's focus tree | “Administrative Reform” appears, not its raw ID | The tree actually loaded, node position, ID and localisation encoding |
| Inspect the reward | A one-off reward of 150 political power | Reward placement, positive value and accidental daily modifiers |
| Start the focus | No immediate payment of this reward | Whether it was placed in `select_effect` |
| Complete the focus normally | One reward is granted to that country | Scope, normal completion versus bypass, and simultaneous effects |
| Continue after completion | This line does not grant an extra 150 every day | Other spirits, repeated calls or daily scripts |
| Read logs from this run | No new errors tied to the new ID, effect or localisation | The added code and surrounding braces |

Leave enough room below the political power cap before testing. Record the balance around completion, including any simultaneous daily tick, spending or other rewards. Do not subtract the balance at the start of the focus from the balance 35 days later; that includes all income and costs during the focus.

## Adapt the example

- For a reward of 75, change the effect to `add_political_power = 75` and check the tooltip again.
- For a one-off cost, read adding and removing political power (`wiki:effect:add_political_power`) and plan both the condition and payment. This case only demonstrates a positive reward.
- For more daily gain after completion, grant a national spirit carrying a flat contribution (`wiki:modifier:political_power_gain`) or percentage modifier (`wiki:modifier:political_power_factor`). Do not put either modifier directly in the reward block.
- Compare the choices in the political power topic (article `political-power`).
