---
contentId: "wiki:modifier:political_power_factor"
kind: reference
slug: political_power_factor
locale: en
status: published
title: "Daily political power: a percentage change"
summary: "political_power_factor contributes a percentage modifier to political power gain. A value of 0.15 means +15%, not an immediate reward or a fixed +0.15 each day."
primaryTopic: politics
topics: [politics]
scenarios: [ideas, focus, event, decision]
gameTerms: ["Political Power Gain","Political Power","National Spirit"]
aliases: ["political power gain percent","PP gain multiplier","political power +15%","daily PP percentage"]
relatedIds: [political-power, "wiki:modifier:political_power_gain", "wiki:effect:add_political_power", "wiki:trigger:has_political_power", focus-political-power-reward]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/political_power_factor
searchText: "political_power_factor political power gain increase 15% decrease 15% daily PP percentage modifier national spirit not fixed daily points not immediate reward"
sources:
  - label: "HOI4 Wiki: political_power_factor"
    url: "https://hoi4.paradoxwikis.com/Modifiers#political_power_factor"
  - label: "HOI4 Wiki: Idea modding"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
---

Technical name: `political_power_factor` · Modifier · Example source: a national spirit.

## 1. What happens in the game, and where can I see it?

It contributes a percentage modifier to political power gain. A national spirit with `0.15` contributes +15%; `-0.15` contributes -15%.

The English label is “Political Power Gain”, displayed as a percentage. The flat contribution is labelled “Daily Political Power Gain”. Read both the label and the unit.

This does not multiply the existing balance by 1.15, grant 15 points immediately, or provide a fixed +0.15 points per day.

## 2. Who is affected, and when does it start or stop?

In this example, the country with the active national spirit receives the modifier. Removing or expiring the spirit stops its percentage contribution, without reclaiming political power already gained.

The source and its grant determine the duration. Do not append `days` to this value or assume that every modifier is permanent.

## 3. Which values or parameters do I use?

| Value | Percentage contribution | Common misreading |
| --- | --- | --- |
| `0.15` | +15% | Not +0.15 points |
| `-0.15` | -15% | Not an immediate deduction of 15 |
| `1` | +100% | Not +1% |
| `0` | No percentage contribution | Does not set daily gain to zero |

Do not write a percent sign in the code. Use `0.15` for +15%, not `15`, which represents +1500%.

Contributions to this modifier combine additively, rather than each applying as an independent multiplier. Flat gain and costs also affect the final net result. Adding another +15% therefore does not guarantee that the previous net daily gain is multiplied by 1.15. This page does not derive a complete political power formula for every mod.

## 4. Where does it go, and what does the script look like?

This is a complete national-spirit definition file for `common/ideas/hag_pp_factor_example.txt`:

```pdx
ideas = {
    country = {
        HAG_pp_daily_factor = {
            allowed = { always = no }
            picture = generic_disjointed_gov
            removal_cost = -1
            modifier = {
                political_power_factor = 0.15
            }
        }
    }
}
```

The example uses an existing vanilla icon. The spirit still needs to be granted in a country effect block. This is a reward-block snippet inside an existing focus:

```pdx
completion_reward = {
    add_ideas = HAG_pp_daily_factor
}
```

`add_ideas` attaches no expiry period. To remove the source from an existing effect block, use `remove_ideas = HAG_pp_daily_factor`. For a timed grant, adapt the flat-gain example (`wiki:modifier:political_power_gain`) to use `add_timed_idea` with this page's ID, `HAG_pp_daily_factor`.

Create `localisation/english/hag_pp_factor_example_l_english.yml` and save it as UTF-8 with BOM:

```yaml
l_english:
 HAG_pp_daily_factor:0 "Political Coordination"
 HAG_pp_daily_factor_desc:0 "Coordinate political efforts and improve the efficiency of government work."
```

This requires an existing focus tree; it is not a standalone mod. Do not treat the `modifier` definition as a focus reward, or put `0.15` into `add_political_power` expecting the same result. Technologies, character traits and other sources have their own outer structures; the national-spirit wrapper is not universal. The localisation file supplies English game text only.

## 5. What might I need next?

- Gain a fixed 0.5 more per day: flat daily gain (`wiki:modifier:political_power_gain`).
- Gain 150 on completing a focus: add political power (`wiki:effect:add_political_power`) and the complete case (`focus-political-power-reward`).
- Require a current balance: political power requirements (`wiki:trigger:has_political_power`).
- Compare the four choices: political power (article `political-power`).
