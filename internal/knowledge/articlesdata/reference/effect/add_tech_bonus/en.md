---
contentId: "wiki:effect:add_tech_bonus"
kind: reference
slug: add_tech_bonus
locale: en
status: published
title: "Give a limited-use research bonus"
summary: "Reward the next eligible research with a bonus, often through a national focus. This does not unlock a technology immediately."
primaryTopic: research
topics: [research]
scenarios: [focus, event, decision]
gameTerms: [Research bonus, Industry, National focus]
aliases: [one research bonus, limited-use research bonus, next research, technology research reward]
relatedIds: [research-speed, focus-research-bonus, "wiki:modifier:research_speed_factor", "wiki:effect:set_technology"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: effect/add_tech_bonus
searchText: "add_tech_bonus grants a limited-use research bonus to the current country. bonus is a percentage, uses is a count, category is a technology category ID, and technology can target a specific technology ID. The example gives one 50% industry research bonus."
sources:
  - label: "HOI4 Wiki: Add a research bonus"
    url: "https://hoi4.paradoxwikis.com/Effects#add_tech_bonus"
  - label: "HOI4 Wiki: Technology modding"
    url: "https://hoi4.paradoxwikis.com/Technology_modding"
---
## 1. What changes in the game, and where can I see it?

The country receives a research bonus with a limited number of uses, available for eligible technologies. A typical focus reward is one 50% research bonus for industry.

The reward is an available bonus, not a completed technology. Check the focus reward tooltip, then inspect the bonus for an eligible technology in the research interface. A technology remaining locked does not mean this effect failed.

## 2. Who receives it, and when does it apply or end?

`add_tech_bonus` is a country-scoped effect. In a focus completion reward, it normally applies to the country completing that focus. If an enclosing script changes scope, confirm which country is current.

The bonus is granted when the effect runs. `uses` sets the number of uses; this is not an ongoing research-speed modifier. This page does not establish exactly when a bonus is applied to or consumed by research already in progress. For a clear test, choose an eligible technology whose research has not started.

## 3. What do the numbers and parameters mean?

| Parameter | Example | Meaning |
| --- | --- | --- |
| `name` | `HW_industry_research_bonus` | Localisation key for the bonus name; provide display text |
| `bonus` | `0.50` | A 50% research bonus, not 50 fewer days |
| `uses` | `1` | One use, not one day |
| `category` | `industry` | A vanilla technology category ID, not its displayed name |

For a specific technology, `technology` can take a technology ID. That is not an equipment ID. The copyable example below uses only the verified `industry` category rather than combining both ways of selecting technologies.

A 50% research bonus does not guarantee that the completion time is halved. Ahead-of-time penalty reduction has separate parameters; do not treat a percentage bonus and a number of years of penalty reduction as the same value.

## 4. Where does it go, and how do I write it?

This is a **complete effect snippet for an existing effect block**, not a standalone file. Insert it inside a focus's existing `completion_reward`. Do not add another `completion_reward` wrapper around it there.

```hoi4
add_tech_bonus = {
    name = HW_industry_research_bonus
    bonus = 0.50
    uses = 1
    category = industry
}
```

The bonus also needs a display name. The following is the **complete minimal localisation file** for this example. Save it as `localisation/english/hw_research_l_english.yml` using UTF-8 with BOM.

```yaml
l_english:
 HW_industry_research_bonus: "Industrial Research Exchange"
```

The industry research focus example (`focus-research-bonus`) includes a complete focus node, icon, names and a checklist.

## 5. What might I need next?

- An ongoing bonus to research speed: Research Speed modifier (`wiki:modifier:research_speed_factor`).
- A technology granted as soon as a focus finishes: Unlock a specific technology (`wiki:effect:set_technology`).
- Help choosing between the three results: Research guide (article `research-speed`).

Checked against the bundled documentation, vanilla focuses, technology categories and reward text for 1.19.2.0. This example has not been tested in the game.
