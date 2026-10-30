---
contentId: focus-research-bonus
kind: case
slug: focus-research-bonus
locale: en
status: published
title: "Reward a focus with one 50% industry research bonus"
summary: "Add a reward node to an existing focus tree, with a use count, technology category, vanilla icon and English names."
primaryTopic: research
topics: [research]
scenarios: [focus]
gameTerms: [National focus, Industry, Research bonus]
aliases: [one industry research bonus, focus research reward, industrial technology reward]
relatedIds: [research-speed, "wiki:effect:add_tech_bonus", "wiki:modifier:research_speed_factor", "wiki:effect:set_technology", localisation, check-and-troubleshoot]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "Grant one 50% industry research bonus after completing a national focus. The complete focus node uses add_tech_bonus with bonus = 0.50, uses = 1 and category = industry, a vanilla icon and a localised bonus name."
sources:
  - label: "HOI4 Wiki: National focus modding"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
  - label: "HOI4 Wiki: Research bonus effect"
    url: "https://hoi4.paradoxwikis.com/Effects#add_tech_bonus"
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---
The goal is to grant one 50% industry research bonus when a focus finishes. The player must still select and research a technology. The reward neither unlocks it immediately nor permanently speeds up every research project.

## Before you begin

- You have a focus tree file in your Mod's `common/national_focus` directory that the target country already loads.
- The tree does not already contain the focus ID `HW_industrial_exchange`. If that ID is taken, choose a unique one and update its localisation keys too.
- You have checked for free space in the tree. `x = 20` and `y = 0` are example coordinates, not a guarantee against overlaps in your tree.

This example does not create the rules that assign a focus tree to a country. If the tree itself is not appearing, resolve that first; adding a node cannot replace loading the tree.

## 1. Add a complete focus node

This is a **complete focus node, not a complete focus tree file**. Place it inside the existing `focus_tree = { ... }`, alongside the other `focus` nodes. Keep the tree ID, country-selection rules and other nodes intact.

```hoi4
focus = {
    id = HW_industrial_exchange
    icon = GFX_goal_generic_scientific_exchange
    x = 20
    y = 0
    cost = 10

    completion_reward = {
        add_tech_bonus = {
            name = HW_industry_research_bonus
            bonus = 0.50
            uses = 1
            category = industry
        }
    }
}
```

This node has no prerequisite. If your design needs one, add it separately and check that the referenced focus exists. The icon is a verified vanilla scientific-exchange sprite, so this example needs no new GFX file.

`industry` is a vanilla technology category ID. `bonus` sets the percentage and `uses` sets the count. A one-use research bonus is not the same as a one-day national spirit.

## 2. Add the focus and bonus names

Create `localisation/english/hw_research_l_english.yml` in the Mod, using UTF-8 with BOM. This is the **complete localisation file for the example**:

```yaml
l_english:
 HW_industrial_exchange: "Industrial Research Exchange"
 HW_industrial_exchange_desc: "Exchange production methods to support our next industrial research project."
 HW_industry_research_bonus: "Industrial Research Exchange"
```

If you already created that file while following the reference page, combine the entries under one `l_english:` header. Keep one definition of the bonus-name key rather than adding it twice.

## 3. Check it in a separate test game

1. Enable the Mod and start with a country that loads this tree. Check that the new focus appears without overlaps, with the correct name and icon.
2. Inspect its reward tooltip. It should describe one 50% industry research bonus, not the immediate grant of a technology.
3. Complete the focus. Choose a technology in the `industry` category whose research has not started, then check the available bonus and its remaining uses after use. An already completed technology is not a useful test.
4. Check the logs from this game launch. A correct tooltip and no obvious log errors do not replace checking how the bonus is actually used.

These are checks for you to perform, not a completed playtest record. Research time also depends on the country's other modifiers and ahead-of-time penalties; “exactly half as many days” is not a sufficient acceptance test.

## Want a different reward?

- Faster research while a spirit remains active: Research Speed modifier (`wiki:modifier:research_speed_factor`).
- A technology granted immediately: Unlock a specific technology (`wiki:effect:set_technology`).
- A different use count or eligible range: add_tech_bonus parameters (`wiki:effect:add_tech_bonus`).

Checked against the vanilla generic focus reward, technology category list, sprite and bundled effect documentation for 1.19.2.0. This example has not been tested in the game.
