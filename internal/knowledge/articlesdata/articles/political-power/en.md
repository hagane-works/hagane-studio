---
contentId: political-power
kind: topic
slug: political-power
locale: en
status: published
title: "Political power: rewards, requirements and daily gain"
summary: "Give a one-off political power reward, check a requirement, or change daily gain. Start with the result you want in the game."
primaryTopic: politics
topics: [politics]
scenarios: [focus, event, decision, ideas]
gameTerms: ["Political Power","Daily Political Power Gain","Political Power Gain","National Spirit","National Focus"]
aliases: ["PP","political power reward","daily PP","political power requirement"]
relatedIds: ["wiki:effect:add_political_power", "wiki:trigger:has_political_power", "wiki:modifier:political_power_gain", "wiki:modifier:political_power_factor", focus-political-power-reward]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
searchText: "political power PP gain 150 immediately spend 50 at least 150 required daily fixed gain 1 daily political power gain increase 15% decrease national spirit focus reward"
sources:
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki: Triggers"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki: Modifiers"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
---

## What do you want to change?

The political power shown at the top of the screen is your current balance. Daily gain is one source of changes to that balance. They use different scripts.

| Goal in the game | Start here | Important distinction |
| --- | --- | --- |
| Gain 150 political power on completing a focus, or spend a one-off amount | Add or remove political power immediately (`wiki:effect:add_political_power`) · `add_political_power` | Changes the balance, not daily gain |
| Require at least 150 political power | Check a political power requirement (`wiki:trigger:has_political_power`) · `has_political_power` | Checks a condition; neither pays a cost nor grants a reward |
| Give a national spirit a fixed +0.5 daily gain contribution | Daily political power: a flat change (`wiki:modifier:political_power_gain`) · `political_power_gain` | Measured in points, not percent |
| Give a national spirit a +15% political power gain modifier | Daily political power: a percentage change (`wiki:modifier:political_power_factor`) · `political_power_factor` | `0.15` means +15%, not +0.15 points a day |

If you search for “increase political power”, first decide whether you mean “give some now” or “gain more each day”. Search keeps these uses separate rather than treating them as the same reward.

## Try a focus reward first

Gain 150 political power on completing a focus (`focus-political-power-reward`) starts with an existing focus tree and provides a complete node, its insertion point, English localisation and checks to run. It does not require a national spirit or change daily gain.

For “gain more political power every day after completing a focus”, grant a national spirit that carries the modifier. Placing `political_power_gain` directly inside `completion_reward` is not the same operation.

## Three common mix-ups

### Having enough political power is not the same as paying

A true condition only allows the surrounding logic to proceed. An ordinary decision uses `cost` for its political power price. An event option or custom cost needs a separate spending effect. Do not charge the same amount through both a decision's `cost` and an extra deduction.

### Expiring a source does not reclaim earlier gains

A daily gain modifier contributes while its national spirit is active. Removing or expiring that spirit stops its contribution; it does not automatically deduct political power already accumulated.

### A source's bonus is not the final daily net gain

Flat contributions, percentage modifiers and costs such as national focuses all affect the final result. Check both the national spirit and the top-bar political power tooltip. A single +0.5 or +15% source is not a promise that the balance will rise by that amount each day.
