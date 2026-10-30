---
contentId: "wiki:trigger:has_stability"
kind: reference
slug: has_stability
locale: en
status: published
title: "Check a stability requirement"
summary: "Use has_stability to check a threshold. Write 50% as 0.5. At least 50% differs from more than 50%, and the condition does not spend stability."
primaryTopic: politics
topics: [politics]
scenarios: [decision, focus, event]
gameTerms: ["Stability","Decision Requirement","Focus Requirement"]
aliases: ["stability threshold","at least 50% stability","not enough stability","stability below 50%"]
relatedIds: [stability, "wiki:effect:add_stability", "wiki:modifier:stability_weekly", event-stability]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: trigger/has_stability
searchText: "has_stability stability requirement insufficient stability at least 50% below 50% more than 50% at most 50% check not spend decision available"
sources:
  - label: "HOI4 Wiki: has_stability"
    url: "https://hoi4.paradoxwikis.com/Triggers#has_stability"
  - label: "HOI4 Wiki: Decision modding"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
---

Technical name: `has_stability` · Trigger · Country scope.

## 1. What happens in the game, and where can I see it?

It checks the current country's stability against a comparison. It does not add or remove stability.

For example, requiring at least 50% in an ordinary decision's `available` leaves the decision visible but unavailable below the threshold. Using `visible` controls whether the decision appears; an event option's `trigger` controls whether that option appears.

## 2. Who is affected, and when does it start or stop?

It checks the current country, not one of its states. The surrounding content reads stability whenever it evaluates the condition. This is not a permanent unlock after the first successful check.

“Require 50% stability” and “spend 5 percentage points of stability” are different designs. If a cost is also needed, arrange a separate effect; a check does not pay it.

## 3. Which values or parameters do I use?

The bundled documentation specifies a 0–1 scale. Write 50% as `0.5`, not `50`, and omit the percent sign.

| Requirement | Condition | At exactly 50% |
| --- | --- | --- |
| Below 50% | `has_stability < 0.5` | False |
| Above 50% | `has_stability > 0.5` | False |
| At least 50% | `NOT = { has_stability < 0.5 }` | True |
| At most 50% | `NOT = { has_stability > 0.5 }` | True |

Inclusive examples negate one comparison. Do not substitute `> 0.49` for “at least 50%” or judge an exact boundary using only a rounded display value.

## 4. Where does it go, and what does the script look like?

This is a condition-block snippet in an existing ordinary decision:

```pdx
available = {
    NOT = {
        has_stability < 0.5
    }
}
```

If `available` already exists, merge the condition into it and preserve its other requirements. Do not create another block of the same name.

For a remedy that should be available only at low stability, use `has_stability < 0.5` instead. This remains a selection requirement; it does not grant a reward.

## 5. What might I need next?

- Actually change stability: an immediate stability effect (`wiki:effect:add_stability`).
- Build a complete event: the stability event case (`event-stability`), which grants a reward rather than charging stability.
- Improve or worsen stability over time: weekly stability (`wiki:modifier:stability_weekly`).
- Return to stability (article `stability`).
