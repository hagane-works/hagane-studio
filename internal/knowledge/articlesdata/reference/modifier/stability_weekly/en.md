---
contentId: "wiki:modifier:stability_weekly"
kind: reference
slug: stability_weekly
locale: en
status: published
title: "Add or remove a fixed amount of stability each week"
summary: "stability_weekly changes weekly stability. A value of 0.01 adds 1 percentage point a week; -0.01 removes 1. Set the source's duration separately."
primaryTopic: politics
topics: [politics]
scenarios: [ideas, decision]
gameTerms: ["Weekly Stability","Stability","National Spirit"]
aliases: ["weekly stability +1%","weekly stability -1%","stability falls each week","fixed weekly stability"]
relatedIds: [stability, "wiki:effect:add_stability", "wiki:trigger:has_stability", "wiki:modifier:stability_weekly_factor", "wiki:effect:add_timed_idea", "wiki:effect:remove_ideas"]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/stability_weekly
searchText: "stability_weekly Weekly Stability gain 1 percentage point each week lose 1 percentage point weekly +1% weekly -1% national spirit fixed change not daily not relative weekly multiplier"
sources:
  - label: "HOI4 Wiki: stability_weekly"
    url: "https://hoi4.paradoxwikis.com/Modifiers#stability_weekly"
  - label: "HOI4 Wiki: Idea modding"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
  - label: "HOI4 Wiki: Decision modding"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
---

Technical name: `stability_weekly` · Modifier · Country modifier.

## 1. What happens in the game, and where can I see it?

It changes the country's weekly stability contribution. The English label is “Weekly Stability”. A national spirit can, for example, contribute +1 or -1 percentage point each week while active.

This is neither an immediate full reward when the spirit is granted nor a daily change. Inspect both the spirit's modifier tooltip and stability's tooltip; comparing only the total on the day of the grant is not enough.

## 2. Who is affected, and when does it start or stop?

A national spirit carrying this modifier affects the country that has it active. When the source stops applying, its weekly contribution stops. Earlier accumulated stability changes are not automatically reclaimed.

The modifier itself has no duration. A timed national spirit gets its expiry from the granting effect; a timed decision has its own duration. Writing `stability_weekly` does not automatically mean “lasts one week”.

## 3. Which values or parameters do I use?

| Goal | Value inside the modifier block |
| --- | --- |
| Add a fixed 1 percentage point per week | `stability_weekly = 0.01` |
| Remove a fixed 1 percentage point per week | `stability_weekly = -0.01` |
| Add a fixed 0.5 percentage points per week | `stability_weekly = 0.005` |

A “weekly +1%” here means 1 percentage point on the game's stability scale. It does not multiply current stability by 1.01 or supply a relative multiplier to the growth rate.

Other sources, limits and the timing of ticks can affect the observed result. Do not promise an exact final stability using only “duration ÷ 7 × value” for every start date and observation time.

## 4. Where does it go, and what does the script look like?

This is a snippet inside an already defined national spirit:

```pdx
modifier = {
    stability_weekly = 0.01
}
```

Merge the field into an existing `modifier` block if there is one. The country must actually receive the spirit. Writing `stability_weekly = 0.01` directly in an event option does not grant it.

For a negative spirit, change the value to `-0.01`. For a timed source, consult the base add_timed_idea reference (`wiki:effect:add_timed_idea`). This is not a complete spirit file: its definition, icon, localisation and caller still need to be provided.

## 5. What might I need next?

- Reward or penalise once: add_stability (`wiki:effect:add_stability`).
- Check current stability: has_stability (`wiki:trigger:has_stability`).
- Change the percentage of weekly gain instead of its flat amount: the base stability_weekly_factor reference (`wiki:modifier:stability_weekly_factor`).
- End a national-spirit source: the base remove_ideas reference (`wiki:effect:remove_ideas`).
- Return to stability (article `stability`).
