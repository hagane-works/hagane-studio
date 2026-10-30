---
contentId: "wiki:modifier:political_power_gain"
kind: reference
slug: political_power_gain
locale: en
status: published
title: "Daily political power: a flat change"
summary: "political_power_gain changes a flat daily contribution. A value of 0.5 means +0.5 points, not +50%. The national spirit or other source controls its duration."
primaryTopic: politics
topics: [politics]
scenarios: [ideas, focus, event, decision]
gameTerms: ["Daily Political Power Gain","Political Power","National Spirit"]
aliases: ["daily PP","gain 1 political power each day","flat political power gain","daily political power not percent","每天多 1 点政治点","例如：每天多 1 点政治点","Gain 1 more political power per day","Political power +1/day","Получать на 1 политическую власть больше в день","Политическая власть +1/день","Jeden Tag 1 politische Macht mehr erhalten","Politische Macht +1/Tag","Ganar 1 de poder político adicional al día","Poder político +1/día","Gagner 1 point de pouvoir politique de plus par jour","Pouvoir politique +1/jour","毎日の政治力を1増やす","例：毎日の政治力を1増やす","매일 정치력 1 추가 획득","예: 매일 정치력 1 추가 획득"]
relatedIds: [political-power, "wiki:modifier:political_power_factor", "wiki:effect:add_political_power", "wiki:trigger:has_political_power", focus-political-power-reward]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: modifier/political_power_gain
searchText: "political_power_gain Daily Political Power Gain gain 1 political power each day fixed daily +0.5 daily -0.5 points not percent national spirit 90 days"
sources:
  - label: "HOI4 Wiki: political_power_gain"
    url: "https://hoi4.paradoxwikis.com/Modifiers#political_power_gain"
  - label: "HOI4 Wiki: Idea modding"
    url: "https://hoi4.paradoxwikis.com/Idea_modding"
  - label: "HOI4 Wiki: add_timed_idea"
    url: "https://hoi4.paradoxwikis.com/Effects#add_timed_idea"
---

Technical name: `political_power_gain` · Modifier · Example source: a national spirit.

## 1. What happens in the game, and where can I see it?

While the national spirit is active, it contributes a fixed amount to its country's daily political power gain. A value of `0.5` means a +0.5-point flat contribution, not +50% or an immediate 0.5-point reward.

The English game label is “Daily Political Power Gain”. Check the spirit and the top-bar political power tooltip. Percentage modifiers and other costs still affect net daily gain, so this source does not guarantee that the balance rises by exactly 0.5 a day.

## 2. Who is affected, and when does it start or stop?

The country with the active national spirit receives the contribution. Removing or expiring the spirit stops this source from contributing.

The modifier has no duration parameter. The example grants the spirit for 90 days with `add_timed_idea`. Using `add_ideas` instead does not attach that expiry period. Any additional activation conditions on the spirit must also be met.

Removing the spirit does not reclaim political power already accumulated. A separate deduction effect would be a different operation.

## 3. Which values or parameters do I use?

| Value | This source contributes | It does not mean |
| --- | --- | --- |
| `1` | +1 point to flat daily gain | +1% |
| `0.5` | +0.5 points to flat daily gain | +50% |
| `-0.5` | -0.5 points to flat daily gain | Deduct 0.5 immediately |
| `0` | No contribution from this source | Disable all political power gain |

This is a points value and may contain decimals. Documented display precision is not a guarantee about calculation precision. For a +15% modifier, use `political_power_factor` instead; the two meanings of `0.15` are not interchangeable.

## 4. Where does it go, and what does the script look like?

Create `common/ideas/hag_pp_examples.txt` in your mod. This is a complete national-spirit definition file, but defining the spirit does not grant it:

```pdx
ideas = {
    country = {
        HAG_pp_daily_flat = {
            allowed = { always = no }
            picture = generic_disjointed_gov
            removal_cost = -1
            modifier = {
                political_power_gain = 0.5
            }
        }
    }
}
```

The example uses an existing vanilla icon. `allowed = { always = no }` does not prevent a scripted grant. `removal_cost = -1` prevents manual removal; it does not prevent a timed spirit from expiring.

Grant it for 90 days in an existing focus's reward block:

```pdx
completion_reward = {
    add_timed_idea = {
        idea = HAG_pp_daily_flat
        days = 90
    }
}
```

This is a reward-block snippet, not an entire focus. Add the English name and description in `localisation/english/hag_pp_examples_l_english.yml`, saved as UTF-8 with BOM:

```yaml
l_english:
 HAG_pp_daily_flat:0 "Administrative Reorganisation"
 HAG_pp_daily_flat_desc:0 "Improve day-to-day administration and provide steady support for political work."
```

The definition ID, grant ID and localisation key must match. `modifier` belongs inside the national spirit definition, not directly inside an ordinary focus's `completion_reward`. The reward executes `add_timed_idea`.

An existing, loaded focus tree is required to carry this grant; these files do not make a standalone mod. Merge into existing files or blocks without overwriting other content. The localisation file covers English game text only.

## 5. What might I need next?

- Use a percentage bonus: daily political power: a percentage change (`wiki:modifier:political_power_factor`).
- Grant points only once: add political power (`wiki:effect:add_political_power`) and the complete focus reward case (`focus-political-power-reward`).
- Check the accumulated balance: political power requirements (`wiki:trigger:has_political_power`).
- Compare all four uses: political power (article `political-power`).
