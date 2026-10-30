---
contentId: "wiki:modifier:research_speed_factor"
kind: reference
slug: research_speed_factor
locale: en
status: published
title: "Increase or reduce research speed over time"
summary: "Change Research Speed with a percentage modifier. A national spirit provides it while active, without consuming research uses."
primaryTopic: research
topics: [research]
scenarios: [ideas, focus, event, decision]
gameTerms: [Research Speed, National spirit]
aliases: [ongoing research bonus, research speed bonus, research speed penalty, faster research, slower research]
relatedIds: [research-speed, "wiki:effect:add_tech_bonus", "wiki:effect:set_technology", first-content]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/research_speed_factor
searchText: "Research Speed research_speed_factor increases or reduces research speed while its source remains active. 0.10 adds +10%; -0.10 adds -10%. This example uses a national spirit's modifier block. The spirit's lifetime determines the duration."
sources:
  - label: "HOI4 Wiki: Research Speed modifier"
    url: "https://hoi4.paradoxwikis.com/Modifiers#research_speed_factor"
  - label: "HOI4 Wiki: National spirits"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---
## 1. What changes in the game, and where can I see it?

The country gains a percentage modifier to Research Speed. A positive value makes research faster; a negative value makes it slower. This neither completes a technology immediately nor grants a fixed number of research points each day.

Check the national spirit's modifiers and the relevant research tooltips. `0.10` adds a +10% Research Speed modifier. It does not guarantee exactly 10% fewer remaining days for every technology: other modifiers and ahead-of-time penalties also affect the calculation.

## 2. Who receives it, and when does it apply or end?

This page uses a national spirit as the source. The country that has the spirit receives the modifier while the spirit is active. Removing that spirit stops it from providing this bonus.

Defining a spirit does not assign it to a country. Grant it with an effect such as `add_ideas`. If it is granted with `add_timed_idea`, that effect's `days` parameter controls the duration, not `research_speed_factor`.

## 3. What do the numbers and parameters mean?

| Setting | Modifier added |
| --- | --- |
| `research_speed_factor = 0.10` | Research Speed +10% |
| `research_speed_factor = -0.10` | Research Speed -10% |
| `research_speed_factor = 0` | No increase or reduction from this modifier |

Do not use `10` to mean 10%. The decimal is not a duration or a flat daily gain. Keep the sign when searching for, copying or adapting an example.

## 4. Where does it go, and how do I write it?

This is a **partial snippet inside a national spirit definition**, not a complete file. It already includes the `modifier` wrapper. If the spirit already has that block, add only the inner modifier line; do not nest a second `modifier` inside it.

```hoi4
modifier = {
    research_speed_factor = 0.10
}
```

Do not place this block directly inside a focus's `completion_reward`. The focus reward grants the spirit with an effect; the spirit definition supplies the Research Speed modifier.

Your first national spirit (`first-content`) provides a complete definition file, a vanilla icon and manual checks.

## 5. What might I need next?

- A bonus for just the next research or a few uses: Limited-use research bonus (`wiki:effect:add_tech_bonus`).
- Skip research and grant a technology directly: Unlock a specific technology (`wiki:effect:set_technology`).
- Compare the three results: Research speed, bonuses and unlocks (article `research-speed`).

Checked against the bundled documentation, vanilla national spirits and game display text for 1.19.2.0. This example has not been tested in the game.
