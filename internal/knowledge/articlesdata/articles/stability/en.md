---
contentId: stability
kind: topic
slug: stability
locale: en
status: published
title: "Stability: immediate changes, weekly changes and requirements"
summary: "Give stability through an event, change it each week with a national spirit, or require a threshold. Choose what should happen in the game first."
primaryTopic: politics
topics: [politics]
scenarios: [event, focus, decision, ideas]
gameTerms: ["Stability","Base Stability","Weekly Stability","National Spirit"]
aliases: ["stability +5%","not enough stability","stability falls each week","weekly stability"]
relatedIds: ["wiki:effect:add_stability", "wiki:trigger:has_stability", "wiki:modifier:stability_weekly", "wiki:modifier:stability_factor", "wiki:modifier:stability_weekly_factor", event-stability]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "stability base stability weekly stability increase decrease immediately reward 5 percentage points stability +5% at least 50% insufficient stability weekly loss not political power"
sources:
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects#add_stability"
  - label: "HOI4 Wiki: Triggers"
    url: "https://hoi4.paradoxwikis.com/Triggers#has_stability"
  - label: "HOI4 Wiki: Modifiers"
    url: "https://hoi4.paradoxwikis.com/Modifiers#stability_weekly"
---

“Stability +5%” could mean an event reward or a national spirit that stays active. First decide when the change should happen.

## What do you want to happen in the game?

| Goal | Start here | Important distinction |
| --- | --- | --- |
| Immediately add or remove stability when an event option executes | An immediate stability change (`wiki:effect:add_stability`) | Changes the base value once; does not repeat weekly |
| Require at least 50% stability | A stability requirement (`wiki:trigger:has_stability`) | Checks only; neither grants nor consumes stability |
| Add or remove stability each week while a national spirit is active | A fixed weekly change (`wiki:modifier:stability_weekly`) | Weekly, not daily; the source also needs a duration |

For complete files, see an event option that grants stability (`event-stability`). It includes a decision that opens the event and English localisation, so the event has a defined way to fire.

## How much is “5%” here?

Stability scripts normally use a 0–1 scale: `0.50` corresponds to 50%, and `0.05` to 5 percentage points.

For example, `add_stability = 0.05` adds 5 percentage points of base stability. With no other changes or limits, 50% becomes 55%, not 52.5%. A “+5%” tooltip does not always mean “multiply the current value by 1.05”.

Likewise, `stability_weekly = 0.01` adds a fixed 1-percentage-point weekly contribution. It does not multiply existing stability by 1.01 each week. Other sources can affect the displayed total, so do not use only that total to judge whether one script ran.

## Two nearby uses that are different

- For a stability bonus that disappears when a national spirit is removed, see the base reference for stability_factor (`wiki:modifier:stability_factor`). That differs from an event changing base stability once.
- To modify the percentage of existing weekly stability gain, see the base reference for stability_weekly_factor (`wiki:modifier:stability_weekly_factor`). It is not a fixed weekly amount.

These are base-reference links, not full tutorials covered by this topic. In particular, a name ending in `factor` is not enough to infer the calculation rule for every modifier.

## Placement determines what the player sees

An event's `option` contains one-off effects. Conditions belong in the relevant condition block. A national spirit's `modifier` contains ongoing modifiers. Their outer structures cannot be swapped.

When you only need to disable a button, the player should usually still be able to see the unmet requirement. An event option's `trigger` controls its visibility; an ordinary decision's `available` controls whether it can be selected. These produce different experiences.
