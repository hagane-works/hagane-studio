---
contentId: "wiki:effect:add_stability"
kind: reference
slug: add_stability
locale: en
status: published
title: "Add or remove stability immediately"
summary: "Use add_stability to change base stability once. A value of 0.05 adds 5 percentage points; -0.05 removes 5. This is not weekly growth."
primaryTopic: politics
topics: [politics]
scenarios: [event, focus, decision]
gameTerms: ["Stability","Base Stability"]
aliases: ["stability +5%","stability -5%","increase stability","reduce stability","event stability reward"]
relatedIds: [stability, event-stability, "wiki:trigger:has_stability", "wiki:modifier:stability_weekly", "wiki:modifier:stability_factor"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: effect/add_stability
searchText: "add_stability increase stability immediately decrease stability once +5% -5% add 5 percentage points remove 5 percentage points event option focus reward base stability not weekly"
sources:
  - label: "HOI4 Wiki: add_stability"
    url: "https://hoi4.paradoxwikis.com/Effects#add_stability"
  - label: "HOI4 Wiki: Event modding"
    url: "https://hoi4.paradoxwikis.com/Event_modding"
---

Technical name: `add_stability` · Effect · Country scope.

## 1. What happens in the game, and where can I see it?

It adds or removes base stability from the current country once. Check the event option's effect tooltip and the country's stability tooltip.

This is not an ongoing bonus or an automatic weekly reward. Other modifiers and limits affect the final displayed stability, so do not promise an identical visible jump in every situation.

## 2. Who is affected, and when does it start or stop?

It affects the country in scope when the line executes. In an ordinary country event's `option`, with no scope change, that is the event recipient. In a focus's `completion_reward`, it is the country completing the focus.

Each execution makes one change. Closing the event does not undo it, though later effects can still change stability. A repeatable caller can grant the reward again; decide at the caller whether it should be limited.

## 3. Which values or parameters do I use?

| Intended change | Script |
| --- | --- |
| Add 5 percentage points of base stability | `add_stability = 0.05` |
| Remove 5 percentage points of base stability | `add_stability = -0.05` |

Use the fractional scale without a percent sign. `0.05` is neither 0.05 percentage points nor 5% of current stability. In a simple case without other changes or limits, 50% becomes 55%.

Do not write `5` to mean “+5%”. Some documentation dumps and older examples use integers that obscure the unit; this page follows the game display format and vanilla fractional usage.

## 4. Where does it go, and what does the script look like?

This is a single-effect snippet for an existing country event's `option = { ... }`, at the same level as its `name`:

```pdx
add_stability = 0.05
```

Do not add another `effect = { ... }` inside the option or place the reward in its `trigger`. Putting it in `immediate` runs it when the event fires, before an option is chosen.

For a penalty, replace that line with:

```pdx
add_stability = -0.05
```

These are alternatives, not two lines to add to the same option. See the event case (`event-stability`) for the complete event, caller and localisation.

## 5. What might I need next?

- Check a requirement first: stability thresholds (`wiki:trigger:has_stability`). Checking does not spend stability.
- Change stability each week: a fixed weekly change (`wiki:modifier:stability_weekly`).
- Make a bonus disappear with a national spirit: the base stability_factor reference (`wiki:modifier:stability_factor`), rather than repeatedly running this effect.
- Compare the choices: stability (article `stability`).
