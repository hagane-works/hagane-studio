---
contentId: research-speed
kind: topic
slug: research-speed
locale: en
status: published
title: "Research speed, research bonuses and technology unlocks"
summary: "Choose the result first: faster research over time, a bonus with limited uses, or immediate ownership of a technology."
primaryTopic: research
topics: [research]
scenarios: [ideas, focus, event, decision, history]
gameTerms: [Research Speed, Research bonus, Unlock technology]
aliases: [faster research, technology reward, research boost, grant technology]
relatedIds: ["wiki:modifier:research_speed_factor", "wiki:effect:add_tech_bonus", "wiki:effect:set_technology", focus-research-bonus, first-content]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "Research Speed, research bonuses and technology unlocks are three different results. Use research_speed_factor for an ongoing speed modifier, add_tech_bonus for a limited-use reward, and set_technology to grant a specific technology."
sources:
  - label: "HOI4 Wiki: Modifiers"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki: Technology modding"
    url: "https://hoi4.paradoxwikis.com/Technology_modding"
---
“Make research faster” can mean three different things. Choose what the player should receive before choosing the script.

## What should the player receive?

| Result in the game | Read this | Key distinction |
| --- | --- | --- |
| A national spirit gives Research Speed +10% while active | Ongoing Research Speed modifier (`wiki:modifier:research_speed_factor`) | Needs a source such as a spirit; not consumed per research |
| Completing a focus grants one 50% industry research bonus | Limited-use research bonus (`wiki:effect:add_tech_bonus`) | The player still researches; the bonus has eligibility rules and uses |
| Completing a focus immediately grants Basic Machine Tools | Unlock a specific technology (`wiki:effect:set_technology`) | Sets technology status instead of speeding up research |

Research Speed and the Research bonus shown in a technology reward are different concepts. Both can display a percentage, but their parameters are not interchangeable.

## Three easily confused numbers

- `research_speed_factor = 0.10`: adds a +10% Research Speed modifier. Its duration depends on when the spirit is granted and removed.
- `bonus = 0.50`, `uses = 1`: one 50% research bonus. `1` is a use, not a day.
- `basic_machine_tools = 1`: grants that technology inside `set_technology`. `1` is neither research speed nor a use count.

These are parameter comparisons, not standalone scripts. A 50% research bonus does not guarantee that remaining research time is halved. Keep percentages, days and use counts separate.

## Start with a complete example

If you already have a focus tree, try a focus that grants one industry research bonus (`focus-research-bonus`). It includes a complete focus node, the bonus name and a checklist.

If you are new to Mod files, start with your first national spirit (`first-content`). That exercise uses an ongoing Research Speed modifier without requiring a focus tree.

## Questions to ask when the result is wrong

1. Should the player own a technology, or own a research bonus? One effect cannot stand in for the other.
2. Is the reward in the intended country's effect block? A spirit that is defined but never granted has not given its modifier to any country.
3. Have you confused a technology ID with a category ID? `basic_machine_tools` is a technology; `industry` is a category.
4. Did the country already own the technology, or had research already started? Use a separate test game with a known starting state.

This topic covers these three basic goals, not research slots, ahead-of-time penalties, military industrial organisation bonuses or special projects. It has been checked against sources for 1.19.2.0; the examples have not been tested in the game.
