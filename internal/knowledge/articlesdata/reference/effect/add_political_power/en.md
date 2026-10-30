---
contentId: "wiki:effect:add_political_power"
kind: reference
slug: add_political_power
locale: en
status: published
title: "Add or remove political power immediately"
summary: "Use add_political_power to change the current balance. Positive values grant points; negative values deduct them. Daily gain is unchanged."
primaryTopic: politics
topics: [politics]
scenarios: [focus, event, decision]
gameTerms: ["Political Power","Focus Reward"]
aliases: ["PP","gain political power now","one-off political power reward","spend political power","完成国策后获得 50 政治点","Gain 50 political power after completing a focus","Получить 50 политической власти после завершения фокуса","Nach Abschluss eines Fokus 50 politische Macht erhalten","Obtener 50 de poder político al completar un enfoque","Gagner 50 points de pouvoir politique après une priorité nationale","国家方針の完了で政治力を50獲得","국가중점 완료 후 정치력 50 획득"]
relatedIds: [political-power, "wiki:trigger:has_political_power", "wiki:modifier:political_power_gain", "wiki:modifier:political_power_factor", focus-political-power-reward]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: effect/add_political_power
searchText: "add_political_power gain 150 political power immediately remove 50 PP one-off reward spend political power focus completion event option decision effect current balance"
sources:
  - label: "HOI4 Wiki: add_political_power"
    url: "https://hoi4.paradoxwikis.com/Effects#add_political_power"
  - label: "HOI4 Wiki: National focus modding"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
---

Technical name: `add_political_power` · Effect · Country scope.

## 1. What happens in the game, and where can I see it?

When the effect executes, the target country's current political power balance changes by the specified amount. In a focus completion reward, `add_political_power = 150` grants 150 points once. It neither grants 150 every day nor sets the balance to 150.

Check the focus reward tooltip and the political power balance at the top of the screen. Leave enough room below the applicable cap when testing; a capped balance cannot always rise by the full amount.

## 2. Who is affected, and when does it start or stop?

The effect applies to the country currently in scope. Inside an ordinary focus's `completion_reward`, without another scope change, that is the country completing the focus.

It runs only when execution reaches the line. Running it again makes another change. Granted points do not expire automatically, though normal spending and other effects can still change the balance.

This line does not check whether the country can afford a deduction. Add a condition where affordability matters; a negative reward alone is not a complete payment interaction.

## 3. Which values or parameters do I use?

| Script | This execution does | It does not mean |
| --- | --- | --- |
| `add_political_power = 150` | Add 150 points to the balance | +150 daily, or set the balance to 150 |
| `add_political_power = -50` | Remove 50 points from the balance | Only check that 50 points are available |
| `add_political_power = 0` | Leave the balance unchanged | Clear the balance |

Use a number directly, not `add_political_power = { value = 150 }`. This is not a percentage parameter: `0.15` does not mean “political power gain +15%”.

Political power limits depend on the game and mod rules. The examples use ordinary integer amounts rather than promising universal upper or lower limits.

## 4. Where does it go, and what does the script look like?

This is a reward-block snippet inside an existing ordinary `focus = { ... }` node:

```pdx
completion_reward = {
    add_political_power = 150
}
```

It is not a complete focus file. If the node already has a `completion_reward`, add the line to that block rather than creating another.

A focus's `available` block takes conditions, not rewards. Event options and decision effects can also grant political power, but use different outer structures; do not copy the focus's `completion_reward` wrapper into them.

## 5. What might I need next?

- Add the node and its English name: the complete focus reward case (`focus-political-power-reward`).
- Check affordability first: political power requirements (`wiki:trigger:has_political_power`). A check does not deduct points.
- Increase daily gain through a national spirit: flat gain (`wiki:modifier:political_power_gain`) or percentage gain (`wiki:modifier:political_power_factor`).
- Compare the different goals: political power (article `political-power`).
