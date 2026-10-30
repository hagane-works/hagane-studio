---
contentId: "wiki:trigger:has_political_power"
kind: reference
slug: has_political_power
locale: en
status: published
title: "Check a political power requirement"
summary: "Use has_political_power to check the current balance. At least 150, less than 150 and more than 150 are different conditions. Checking does not charge a cost."
primaryTopic: politics
topics: [politics]
scenarios: [focus, event, decision]
gameTerms: ["Political Power","Focus Requirement","Decision Requirement"]
aliases: ["PP","not enough political power","political power threshold","at least 150 political power","政治点够 100 才能点决议","Allow a decision with 100 political power","Разрешить решение при 100 политической власти","Entscheidung erst ab 100 politischer Macht erlauben","Permitir una decisión con 100 de poder político","Autoriser une décision à partir de 100 points de pouvoir politique","政治力が100以上ならディシジョンを実行","정치력 100 이상일 때 결정 실행"]
relatedIds: [political-power, "wiki:effect:add_political_power", "wiki:modifier:political_power_gain", "wiki:modifier:political_power_factor", focus-political-power-reward]
updatedAt: "2026-09-08"
verification: source-checked
gameVersion: "1.19.2.0"
referenceKey: trigger/has_political_power
searchText: "has_political_power check political power enough at least 150 less than 150 more than 150 at most 150 insufficient PP no payment decision available focus requirement"
sources:
  - label: "HOI4 Wiki: has_political_power"
    url: "https://hoi4.paradoxwikis.com/Triggers#has_political_power"
  - label: "HOI4 Wiki: Decision modding"
    url: "https://hoi4.paradoxwikis.com/Decision_modding"
  - label: "HOI4 Wiki: National focus modding"
    url: "https://hoi4.paradoxwikis.com/National_focus_modding"
---

Technical name: `has_political_power` · Trigger · Country scope.

## 1. What happens in the game, and where can I see it?

It checks whether the current country's political power balance satisfies a comparison. It does not add or remove points.

Inside an ordinary focus or decision's `available`, a failed condition prevents selection. Inside a decision's `visible`, it controls whether the decision appears at all. Placement determines what the player sees; do not hide a decision when you mean to show “not enough political power”.

## 2. Who is affected, and when does it start or stop?

It checks the current country. With no scope change in an ordinary focus or decision condition, that is the country using the content, not another participant or a state.

The surrounding content reads the balance whenever it checks this condition. The check neither reserves points nor charges them in advance. A condition that is true now may be false after the balance changes.

## 3. Which values or parameters do I use?

This table uses 150 points to show the boundaries. Choose the actual requirement for your design.

| Requirement | Condition | At exactly 150 |
| --- | --- | --- |
| Less than 150 | `has_political_power < 150` | False |
| More than 150 | `has_political_power > 150` | False |
| At least 150 | `NOT = { has_political_power < 150 }` | True |
| At most 150 | `NOT = { has_political_power > 150 }` | True |

The inclusive examples negate one comparison. Do not substitute `has_political_power > 149` for “at least 150”: political power can be fractional, and 149.5 is not enough.

Do not copy a number just because the design says “requires political power”. This page does not treat `=` as a universal “at least” operator for all triggers; each parameter's comparison rules need their own verification.

## 4. Where does it go, and what does the script look like?

This is a condition-block snippet inside an existing ordinary focus node:

```pdx
available = {
    NOT = {
        has_political_power < 150
    }
}
```

A country with at least 150 points satisfies this condition without spending those points. Any other conditions on the focus must also be met. Merge into an existing `available` block rather than adding a duplicate.

If an ordinary decision simply costs 150 political power, set its price on the decision itself:

```pdx
cost = 150
```

This single line is a decision property, not a parameter of this trigger. Do not put it inside `available` or use a focus's `cost` as a political power price. If a decision already uses `cost`, do not deduct the same amount again in `complete_effect`.

Custom costs and event branches need their own condition and payment effects. The condition alone does not complete a payment.

## 5. What might I need next?

- Actually grant or deduct points: add or remove political power (`wiki:effect:add_political_power`).
- Give a free completion reward: gain 150 political power after a focus (`focus-political-power-reward`). That is different from requiring 150 first.
- Help a country accumulate points: flat daily gain (`wiki:modifier:political_power_gain`) or percentage gain (`wiki:modifier:political_power_factor`).
- Choose a different goal: political power (article `political-power`).
