---
contentId: first-content
kind: tutorial
slug: first-content
locale: en
status: published
title: "Step 2: Write a national spirit that improves research speed"
summary: "Define a spirit with Research Speed +5% in a complete ideas file. Learn the difference between defining content and granting it to a country."
primaryTopic: research
topics: [research]
scenarios: [ideas]
gameTerms: [National spirit, Research Speed]
aliases: [first national spirit, first game content, beginner script, national spirit tutorial]
relatedIds: [first-mod, localisation, check-and-troubleshoot, "wiki:modifier:research_speed_factor", idea-supply-consumption]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "HOI4 modding step two: a complete ideas/country/HW_learning_spirit definition. research_speed_factor = 0.05 inside modifier gives Research Speed +5%. Defining a spirit does not grant it; add_ideas HW_learning_spirit is a console check for a separate local test game."
sources:
  - label: "HOI4 Wiki: National spirits"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
  - label: "HOI4 Wiki: Research Speed"
    url: "https://hoi4.paradoxwikis.com/Modifiers#research_speed_factor"
  - label: "HOI4 Wiki: Console commands"
    url: "https://hoi4.paradoxwikis.com/Console_commands"
---
The previous step prepared the `hw_learning` Mod. Now add a spirit that gives Research Speed +5% to the country that has it.

This page adds a definition only. It does not alter country history or automatically give the spirit to every country.

## 1. Write the complete definition

Create `common/ideas/hw_learning_ideas.txt` inside the Mod content folder, saved as UTF-8 without BOM. This is the **complete file**:

```hoi4
ideas = {
    country = {
        HW_learning_spirit = {
            picture = generic_army_war_college
            removal_cost = -1
            modifier = {
                research_speed_factor = 0.05
            }
        }
    }
}
```

## 2. Understand the important parts

| Part | Purpose |
| --- | --- |
| `ideas` | Outer structure for ideas such as spirits and laws |
| `country` | The national-spirit category used here |
| `HW_learning_spirit` | Unique script ID, not the displayed name |
| `picture` | References an existing vanilla icon; no new artwork or registration is needed |
| `modifier` | Modifiers provided while the spirit is active |
| `research_speed_factor = 0.05` | Research Speed +5%, not 0.05 points per day |

`removal_cost = -1` prevents manual removal by paying Political Power. Scripts can still remove the spirit with `remove_ideas`.

Keep the braces, equals signs and field names as written. The next page puts the display name in a localisation file. Do not replace tokens such as `country` or `research_speed_factor` with prose.

## 3. Defining it is not the same as granting it

The file lets the game recognise `HW_learning_spirit`. A country must still receive it before it can appear there.

To check the definition first, open the in-game console manually in a separate local test game with only this exercise Mod enabled. Run the following **complete console command** for the current country. Do not use a save you want to keep or a multiplayer game for this check.

```text
add_ideas HW_learning_spirit
```

This command uses a space, not an equals sign. It is a test entry point, not syntax to paste into a `.txt` script. Normal gameplay usually grants spirits through focus, event or decision effects. The supply spirit example (`idea-supply-consumption`) shows how to connect one to an existing focus.

If you are not ready to start a test game, finish the localisation step first and check both together. There is no need to alter an existing save just to follow the tutorial.

## 4. What should I check now?

During the manual test, the country should receive a spirit with Research Speed +5%. If its name appears as `HW_learning_spirit`, the display text is still missing; do not fix that by changing the script ID.

If the command cannot find the spirit, check the actual filename, directory, braces and ID. If the spirit appears but its modifier is wrong, check that the modifier is inside `modifier` and its value is `0.05`.

The definition, icon and console command were checked against sources for 1.19.2.0. This exercise has not been tested in the game; the result above is the intended acceptance target.

Previous: Mod file structure (`first-mod`). Next: Add an English name (`localisation`).
