---
contentId: advisor-unlocked-by-focus
kind: case
slug: advisor-unlocked-by-focus
locale: en
status: published
title: "A political advisor who only appears after a focus completes"
summary: "The advisor is absent from the candidates until a focus completes, then appears and can be hired; hiring adds stability once, and while in office he adds daily political power. Four script blocks each do one job, and one way of writing them is rejected by the engine outright."
primaryTopic: people
topics: [people, politics]
scenarios: [focus]
gameTerms: [advisor, political advisor, cabinet, national focus, stability, political power, hire, dismiss]
aliases: [advisor unlocked by focus, advisor not showing, advisor greyed out, advisor on hire effect, advisor lasting modifier, hidden advisor, advisor visible, advisor available, on_add]
relatedIds: [character-system, leader-succession, political-power, "wiki:trigger:has_completed_focus", "wiki:effect:add_stability", "wiki:modifier:political_power_gain"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Character modding"
    url: "https://hoi4.paradoxwikis.com/Character_modding"
  - label: "HOI4 Wiki: Triggers"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki: Scopes"
    url: "https://hoi4.paradoxwikis.com/Scopes"
---

# A political advisor who only appears after a focus completes

## Goal and observable result

When this page is done you should see four things in game:

1. Before the focus completes, the political advisor candidates do **not** include this person.
2. After the focus completes he appears among the candidates and can be hired.
3. At the moment of hiring, stability rises by 5 percentage points; it drops back when he is dismissed.
4. While he is in office, the source list for daily political power gain shows an extra entry from this advisor.

Items 1 and 2 come from two condition blocks, item 3 from the hire effect, item 4 from the lasting modifier. They are four unrelated pieces of script, and this page builds them in order.

## What you need before starting

- A mod that already exports correctly, and an existing focus. This page uses the placeholder ID `my_focus_unlock_advisor`.
- No DLC is required.
- Read authoring a character (`character-system`) first, so that "a record plus role blocks" is familiar.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| Character definition | a file under `common/characters/`, wrapped in `characters = { ... }` | The advisor himself; all four pieces of script sit inside his `advisor` block |
| Focus | `common/national_focus/` | Referenced by the condition; not changed here |
| Recruitment | the country's history file under `history/countries/` | One `recruit_character` line, added by the workbench |
| Name | the matching language YML under `localisation/` | Whether the candidate list shows a name or a raw key |
| Portrait | `.gfx` under `interface/` plus the image | A missing one silently shows a generic face |

## 1. Build an advisor who is always hireable

Start without any condition, so he simply appears. In the workbench set the role type to advisor, the slot to political advisor, and pick a base-game trait; this page uses `silent_workhorse` (+15% daily political power). The exported shape is:

```
characters = {
    my_advisor = {
        name = my_advisor
        portraits = {
            civilian = {
                large = GFX_portrait_my_advisor
                small = GFX_portrait_my_advisor_small
            }
        }
        advisor = {
            slot = political_advisor
            idea_token = my_advisor
            allowed = { original_tag = MYT }
            traits = { silent_workhorse }
            cost = 150
        }
    }
}
```

`allowed`, `idea_token` and `cost` are filled in by the workbench: `allowed` restricts hiring to your country, `idea_token` is the engine's internal unique name, and `cost` defaults to 150 political power. The `recruit_character = my_advisor` line in the history file is added automatically too.

Load the game at this point and he should already be in the political advisor candidates, hireable at any time.

## 2. Two gates: visible, and hireable

The `advisor` block has two condition blocks with similar names and different jobs:

| Block | When it fails | Use it for |
| --- | --- | --- |
| `visible = { ... }` | The advisor is **absent from the candidates entirely**; the player does not know he exists | Story-unlocked hidden advisors |
| `available = { ... }` | The advisor is **listed but greyed out** and cannot be clicked | The player can see him and must first meet a condition |

"Appears after a focus" wants `visible`. In the workbench's advisor visibility field, type one condition line and the exporter wraps it in a block:

```
visible = {
    has_completed_focus = my_focus_unlock_advisor
}
```

Both gates can be used together, for example: appears after the focus, but greyed out until stability is high enough:

```
visible = {
    has_completed_focus = my_focus_unlock_advisor
}
available = {
    has_stability > 0.5
}
```

Both behaviours were tested in game: with `visible` unmet he is not in the list; with `visible` met and `available` unmet he is listed, grey, and cannot be hired; with both met he can be hired.

## 3. The hire effect and the lasting modifier

This is the section most often done wrong. An advisor block has three effect-like things that behave completely differently:

| Block | When it happens | Executed on |
| --- | --- | --- |
| `on_add = { ... }` | Once, at the moment of hiring | The **character** scope |
| `on_remove = { ... }` | Once, at the moment of dismissal | The **character** scope |
| `modifier = { ... }` | Continuously while hired; gone on dismissal | The country |

🔴 **The hire effect does not run in country scope.** Write a country effect such as `add_stability = 0.05` directly and the engine rejects it at load, with a line like this in `error.log`:

```
Invalid scope type for effect add_stability in common/characters/my_characters.txt line : 32
```

Then nothing happens on hire. Of the 188 hire effects in the base game, **none** is written bare; 123 use `owner = { ... }` to step back to the country first. The correct form:

```
on_add = {
    owner = { add_stability = 0.05 }
}
on_remove = {
    owner = { add_stability = -0.05 }
}
```

⚠️ The workbench tooltip for the on-hire field shows a bare `add_stability` example, and **filling it in that way does nothing**. Type the `owner`-wrapped form above into that field.

The lasting modifier is written on the country directly; no scope change is needed:

```
modifier = {
    political_power_gain = 0.25
}
```

`political_power_gain = 0.25` is +0.25 points on the daily flat gain, not +25%. Once hired, hovering political power shows this advisor as a source; dismiss him and the entry disappears. The base game almost never writes it this way (one advisor out of more than four thousand); it prefers to carry lasting bonuses in traits, because a trait can be reused across people. Both work; pick one.

## 4. Add the name

The character's localisation key is the key after `name`:

```
l_english:
 my_advisor: "Reformist Treasury Expert"
```

When overriding an existing key the file has to go under `localisation/<language>/replace/`, or the base game's or an earlier mod's same-named key silently wins. This example adds a new key, so no override is involved.

A portrait that cannot be found raises **no error**; the game shows a generic face. Load the game and look at the small icon in the candidate list.

## 5. Check each item

Open a separate test save and confirm in order; do not skip steps. The focus can be completed with the console, or the condition can be temporarily replaced by a country flag that an event sets.

| Check | Expected | If not, look at |
| --- | --- | --- |
| Open political advisor candidates before the focus | He is not there | Whether the `visible` block really sits inside the `advisor` block |
| Look again after the focus | He appears and can be hired | The focus ID inside the condition |
| Hover stability at the moment of hiring | +5 percentage points | Whether `on_add` contains `owner = { ... }`; check `error.log` for a scope error |
| Hover daily political power after hiring | An extra source, +0.25 | The key name inside `modifier` |
| Dismiss | Stability drops back, the political power source disappears | Whether `on_remove` mirrors `on_add` |
| The small icon in the candidate list | Your image, not a generic face | Image path and sprite name; the log will not tell you |

## Common mistakes

- **Bare country effects in the hire block**: the engine rejects them and logs a scope error; nothing happens on hire. Wrap them in `owner = { ... }`.
- **Wanting "visible but greyed out" and writing `visible`**: the player cannot find the advisor at all. For greyed out, use `available`.
- **Putting a lasting bonus in `on_add`**: it runs once and does not persist. Lasting bonuses go in `modifier` or in a trait.
- **A misspelled trait ID**: the workbench drops any ID that is in neither the base-game table nor your custom traits before export, with a warning, so the mistake never reaches the game, but the advisor loses that effect.
- **Making an advisor unfireable with `removal_cost = -1`**: the workbench treats -1 as unset and does not emit the key; source material also records this form as no longer working after 1.12.8. The base game uses `can_be_fired = no`. This item was not tested in game for this page.

## Want a different effect?

- To make him chief of army, a theorist or high command: change the slot. Those slots carry a `ledger = army` line (or `navy`, `air`) in the base game, and the workbench emits it with the slot.
- To have a general hold this advisor role: do not create a second person; tick "also holds an advisor role" on the general, see authoring a character (`character-system`).
- To fire an event on hire: `owner = { country_event = my_events.1 }` inside `on_add`, with the event written separately.
- To unlock on a flag set by an event rather than a focus: use `has_country_flag = my_flag` as the condition and `set_country_flag = my_flag` in the event option.
