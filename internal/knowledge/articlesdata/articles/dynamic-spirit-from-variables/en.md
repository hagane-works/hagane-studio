---
contentId: dynamic-spirit-from-variables
kind: case
slug: dynamic-spirit-from-variables
locale: en
status: published
title: "A national spirit that follows the situation: dynamic modifiers, variables and a recount hook"
summary: "A spirit whose number moves, such as +3% attack for every core state you have lost, cannot be an idea, because an idea's modifier values cannot be variables. What works is a set of three: a dynamic modifier that binds its value to a variable, a scripted effect that recounts, and an on_actions hook that decides how often. A five-file minimal version you can copy first, then how to add rules."
primaryTopic: politics
topics: [politics, military]
scenarios: [ideas, event]
gameTerms: [national spirit, dynamic modifier, variable, custom modifier tooltip, scripted effect, scripted trigger, on_actions, daily hook, military attaché, core state]
aliases: [spirit that changes in real time, spirit whose number moves, dynamic national spirit, modifier that follows the situation, dynamic_modifier, custom_modifier_tooltip, bind a modifier to a variable, plus X for every, variable in a national spirit, force_update_dynamic_modifier]
relatedIds: [political-power, stability, idea-supply-consumption, event-stability]
updatedAt: "2026-09-11"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Modifiers (Dynamic modifiers section)"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
  - label: "HOI4 Wiki: Variables"
    url: "https://hoi4.paradoxwikis.com/Variables"
  - label: "HOI4 Wiki: Localisation (variable formatting section)"
    url: "https://hoi4.paradoxwikis.com/Localisation"
  - label: "HOI4 Wiki: On actions"
    url: "https://hoi4.paradoxwikis.com/On_actions"
---

# A national spirit that follows the situation: dynamic modifiers, variables and a recount hook

## Goal and observable result

When this page is done you should see three things in game:

1. A new spirit in the national spirit bar. On hover you see the total the engine computed itself, for example "Division Attack: +12.00%". When the total is 0 the engine does not show that line.
2. Below the total, explanation lines you wrote yourself, one per rule, for example "For every core state under foreign control: Division Attack +3.00% (currently 4)". The number in brackets follows the situation.
3. After a core state changes hands, the number changes as soon as that day's midnight passes (the daily hook runs once at midnight).

The more land lost the harder the fight, the more foreign aid the more passive, the more countries of some kind the more nervous: these spirits all have the same shape. This page uses "lost land" as the main example.

## Before you start

**First, how this route relates to the workbench.** The Dynamic Modifiers page can create a modifier and fill in its values, and a non-numeric value is exported as written, so the "bind the value to a variable" step can be done there. It has no input for the custom tooltip, and the scripted effect and daily hook used for the recount have to be assembled separately in Scripted Snippets and in an event. This page describes the pure-script route, with every file written by hand; workbench users can fill in what fits and do the rest in raw files.

- You have a mod skeleton that loads. If not, read Your first mod (`first-mod`) first.
- You can start the game with `-debug` and read `error.log`. In the Steam library, right-click the game, Properties, and put `-debug` in Launch Options; the log is at `Documents\Paradox Interactive\Hearts of Iron IV\logs\error.log`. The `~` key opens the console in game.
- Script files (`.txt`) must not carry a BOM; localisation files (`.yml`) must. In VS Code, click the encoding in the bottom-right corner and save as "UTF-8 with BOM". Get it wrong and the whole file is discarded, same as in every other domain.

## Objects and relationships

| Object | File (as used here) | Role in this example |
| --- | --- | --- |
| Dynamic modifier | `common/dynamic_modifiers/my_thorn.txt` | The spirit itself; the value is a variable name the engine reads daily |
| Scripted effect | `common/scripted_effects/my_thorn_effects.txt` | Recounts and writes the result into variables |
| on_actions | `common/on_actions/my_thorn_on_actions.txt` | Decides how often the recount runs |
| Country history | `history/countries/MYT - my_thorn.txt` | Initialises the variable and attaches the modifier at start |
| Localisation | `localisation/english/my_thorn_l_english.yml` | The spirit's name and the explanation lines |
| Scripted trigger (advanced) | `common/scripted_triggers/my_thorn_triggers.txt` | Defines who counts as domestic and who counts as foreign |

File names are yours to choose; the directories are not. Every `my_` name and `MYT` is a placeholder; replace them with your own.

## 1. Why an idea will not do

An idea's modifier block only accepts numbers. Put a variable name after `army_attack_factor` and the engine does not look the variable up; the modifier is simply invalid, and nothing is logged. The wiki's Variables page says it plainly: variables cannot be used within idea modifiers directly, dynamic modifiers can be used instead.

A dynamic modifier looks the same as an idea on screen: an icon, a slot in the spirit bar, a modifier list on hover. Two things differ. Its values may be variable names, and it is recalculated daily: every variable-bound value is reset to zero, then the modifiers are evaluated in the order they were attached.

So every "real-time" spirit is a dynamic modifier.

## 2. Minimal version: five files you can copy

This version has one rule: for every core state of yours not under your control, Division Attack +3%. Lost land is the example throughout; swapping in another rule touches two places, see section seven.

### 2.1 The modifier

`common/dynamic_modifiers/my_thorn.txt`:

```
my_thorn_in_throat = {
    icon = GFX_idea_unknown
    enable = { always = yes }
    army_attack_factor = my_thorn_attack
    custom_modifier_tooltip = my_thorn_in_throat_tt
}
```

Four lines, four jobs:

- `icon` is the picture in the spirit bar. `GFX_idea_unknown` is the base game's placeholder; get it running with that first, and register your own sprite later.
- `enable` is the condition for the values to apply. `always = yes` means they apply as long as the modifier is attached; `has_war = yes` and the like are also fine, and when the condition fails the values stop applying while the spirit stays attached.
- `army_attack_factor = my_thorn_attack` is the core: the right-hand side is not a number but the name of a country variable, read by the engine every day.
- `custom_modifier_tooltip` points at a localisation key. **The explanation lines do not grow by themselves.** The engine only renders the total line, "Division Attack: +12.00%"; every "For every ... +3.00%" line below it is text you wrote in that key.

### 2.2 The recount

`common/scripted_effects/my_thorn_effects.txt`:

```
my_thorn_recalc = {
    set_variable = { my_thorn_states = 0 }
    every_state = {
        limit = {
            is_core_of = ROOT
            NOT = { is_controlled_by = ROOT }
        }
        ROOT = { add_to_variable = { my_thorn_states = 1 } }
    }
    set_variable = { my_thorn_attack = my_thorn_states }
    multiply_variable = { my_thorn_attack = 0.03 }
    clamp_variable = { var = my_thorn_attack max = 0.5 }
}
```

Reading it: zero the counter; walk every state on the map and, for each one that is "my core and not controlled by me", add one to my counter; finally multiply the count by 0.03 for the modifier value, capped at 50%.

Two points:

- `ROOT` is the country that called the effect, that is, you. Inside `every_state` the scope has become a state, so you must switch back with `ROOT = { }` before adding, otherwise the count lands on the state and your variable stays 0.
- Fold at the end. The value is "count × 0.03". Do not add 0.03 each time you count; capping and display both get harder.

### 2.3 The hook

`common/on_actions/my_thorn_on_actions.txt`:

```
on_actions = {
    on_daily_MYT = {
        effect = {
            my_thorn_recalc = yes
        }
    }
}
```

`on_daily_<TAG>` runs once a day for the named country only, and its default scope is that country, so the effect is called directly. **Replace `MYT` with your country's TAG, including in the hook name itself.** A hook called `on_daily_MYT` when your TAG is not MYT never runs.

The global `on_daily` runs for every country, and the wiki explicitly marks it performance heavy; do not use it.

### 2.4 Initialise and attach

**Create a new file** under `history/countries/` whose first three letters are your country TAG, for example `MYT - my_thorn.txt`, containing just two lines:

```
set_variable = { my_thorn_attack = 0 }
add_dynamic_modifier = { modifier = my_thorn_in_throat }
```

Do not copy or edit the base game's history file for that country. One TAG may have several history files and the engine runs all of them in order, so your small file coexists with the base game's start setup. Without `days` it stays attached until you remove it with `remove_dynamic_modifier` or the `remove_trigger` in the modifier itself is met. The same two lines can go into a focus reward or an event option so the spirit appears as part of the story.

**Set the variable before attaching.** Otherwise the value is empty on day one, `has_variable` is false, and any check that relies on the variable existing takes the wrong branch.

### 2.5 The text

`localisation/english/my_thorn_l_english.yml`, with BOM. The file name must end in `_l_english.yml` with a lowercase L, or the engine will not treat it as an English file:

```yaml
l_english:
 my_thorn_in_throat:0 "A Thorn in the Throat"
 my_thorn_in_throat_tt:0 "For every core state under foreign control: Division Attack §G+3.00%§! (currently [?my_thorn_states|Y0])"
```

`[?my_thorn_states|Y0]` is how a variable appears in text: the question mark is mandatory, and after the pipe come format characters, `Y0` meaning yellow with no decimals. Section five has the table.

That is all five files. Load the mod and the spirit and its number are there. Note that the first recount happens at the first midnight after start: hovering on day one shows 0 in the brackets and no total line; look again after 24:00.

## 3. Adding a second rule: foreign attachés accepted by other domestic factions

The minimal version counts "controlled by anyone else" as lost land, including states held by other factions split off from the same nation. If your nation has several factions and you want to tell "domestic" from "foreign", you have to draw that line yourself; the engine has no such concept.

First a scripted trigger, `common/scripted_triggers/my_thorn_triggers.txt`, listing the factions of one nation by `original_tag`:

```
my_is_domestic_faction = {
    OR = {
        original_tag = MYT
        original_tag = MYA
        original_tag = MYB
    }
}
```

Then extend the effect from section 2.2 to two rules: a core state counts as lost only when its controller is not domestic; and every foreign attaché accepted by another domestic faction counts as one item.

```
my_thorn_recalc = {
    set_variable = { my_thorn_states = 0 }
    set_variable = { my_thorn_attaches = 0 }

    # Rule one: core states of mine whose controller is not a domestic faction
    every_state = {
        limit = {
            is_core_of = ROOT
            NOT = { CONTROLLER = { my_is_domestic_faction = yes } }
        }
        ROOT = { add_to_variable = { my_thorn_states = 1 } }
    }

    # Rule two: every foreign attaché accepted by another domestic faction
    every_other_country = {
        limit = { my_is_domestic_faction = yes }
        every_other_country = {
            limit = {
                NOT = { my_is_domestic_faction = yes }
                PREV = { has_attache_from = PREV }
            }
            ROOT = { add_to_variable = { my_thorn_attaches = 1 } }
        }
    }

    set_variable = { my_thorn_attack = my_thorn_states }
    add_to_variable = { my_thorn_attack = my_thorn_attaches }
    multiply_variable = { my_thorn_attack = 0.03 }
    clamp_variable = { var = my_thorn_attack max = 0.5 }
}
```

Rule two is a nested loop; watch the scopes. The outer loop walks the other domestic factions, the inner loop walks foreign countries. Inside the inner loop `PREV` is the domestic faction from the outer loop, so `PREV = { has_attache_from = PREV }` reads "that faction has an attaché from the current foreign country". Adding to the counter still goes back to `ROOT`. What the diplomacy panel calls "Send Attaché" is `attache` in script.

Add a line to the text accordingly:

```yaml
l_english:
 my_thorn_in_throat_tt:0 "For every core state under foreign control: Division Attack §G+3.00%§! (currently [?my_thorn_states|Y0])\nFor every foreign military attaché accepted by another domestic faction: Division Attack §G+3.00%§! (currently [?my_thorn_attaches|Y0])"
```

Volunteers and lend-lease count the same way with the inner test swapped: `has_volunteers_amount_from` (with `tag` and `count`) for volunteers, `is_lend_leasing` for lend-lease. Both triggers are in the game's own trigger documentation; this page has not tested their spelling inside nested scopes.

## 4. Making the number move the same day

A daily hook means "takes effect tomorrow". To move the number the day something happens, call the recount again from the matching event hook and force a refresh. A state changing hands is `on_state_control_changed`:

```
on_actions = {
    on_state_control_changed = {
        effect = {
            if = {
                limit = { FROM.FROM = { is_core_of = MYT } }
                MYT = {
                    my_thorn_recalc = yes
                    force_update_dynamic_modifier = yes
                }
            }
        }
    }
}
```

In this hook `ROOT` is the new controller, `FROM` the old one and `FROM.FROM` the state, so first check the state is your core, then switch back to your own scope to recount. This block can go straight inside the `on_actions = { }` of the file from section 2.3.

Attachés, volunteers and lend-lease each have a hook (`on_send_attache`, `on_send_volunteers`, `on_lend_lease`); the wiki's On actions page gives their scopes. The base game's scripts do not use these three hooks and this page has not tested them; verify in game before relying on them.

## 5. Variable formatting in text

After the pipe in `[?variable|format]` you can stack format characters; the common ones:

| Character | Effect |
| --- | --- |
| `0` to `5` | Number of decimals |
| `%` | Multiply by 100 and append a percent sign |
| `+` | Green if positive, yellow if zero, red if negative |
| `=` | Prefix a plus sign on positive values |
| `Y`, `G`, `R` | Fixed yellow, green, red |

To show the folded percentage directly in an explanation line, write `[?my_thorn_attack|%+0]`.

This key can use variables because dynamic localisation in ideas and dynamic modifiers has been supported since 1.13. The base game uses it the same way: China's foreign-aid modifier reads "[?variable|Y0] Economic Capacity Surplus each month".

## 6. Checklist in game

| Check | Expected |
| --- | --- |
| All five files present, `.txt` without BOM, `.yml` with BOM | Pass |
| The icon appears in the spirit bar; the first hover line is the computed total | By eye |
| After the first midnight, the bracketed number in your explanation line matches the lost states | By eye |
| Make a core state change hands; the total rises once that day's midnight passes | By eye |
| Search `error.log` for the modifier name and the variable names: zero hits | Pass |
| Hover the explanation lines: no `[?` shown verbatim | By eye |

**The easiest way to make a state change hands** is a test event fired from the console. `events/my_thorn_test.txt`:

```
add_namespace = my_thorn_test

country_event = {
    id = my_thorn_test.1
    title = my_thorn_test.1.t
    desc = my_thorn_test.1.d
    is_triggered_only = yes
    immediate = {
        MYA = { set_state_controller = 123 }
    }
    option = {
        name = my_thorn_test.1.a
    }
}
```

Replace `123` with the id of one of your core states and `MYA` with any neighbour; no state of war is needed. Note that `set_state_controller` is a country-scope effect: switch to the receiving country first, then name the state; written the other way round, `123 = { set_state_controller = MYA }`, the engine logs `Invalid Scope` and does nothing. In game, type `event my_thorn_test.1` in the console, let the clock pass midnight, and hover the spirit. Delete the file when you are done.

The last row is the most common failure shape: `[?my_thorn_states|Y0]` appearing literally on screen means the text was used somewhere without dynamic localisation, or the question mark is missing.

## 7. Swapping in your own rule

This page uses lost land as its example. Changing the rule touches two places: the loop and its test inside the recount effect (what you count), and the modifier line in the modifier itself (which number it feeds; `army_attack_factor` can be any modifier). The hook, the initialisation, the text format and the cap stay as they are; the explanation lines are hand-written, so they follow the rule.

### 7.1 When the engine already has the number, no loop

The game's own `documentation/dynamic_variables_documentation.md` lists dozens of built-in variables in country scope; the useful ones:

| What you want to count | Built-in variable |
| --- | --- |
| Subjects | `num_subjects` |
| Faction members | `num_faction_members` |
| Owned, controlled, core states | `num_owned_states`, `num_of_controlled_states`, `num_core_states` |
| Civilian, military factories | `num_of_civilian_factories`, `num_of_military_factories` |
| Casualties (thousands) | `casualties_k` |
| Stability, surrender progress, strength ratio against enemies | `stability`, `surrender_progress`, `enemies_strength_ratio` |

The recount shrinks to two lines: copy the built-in variable into your own, then fold:

```
my_thorn_recalc = {
    set_variable = { my_thorn_attack = num_subjects }
    multiply_variable = { my_thorn_attack = 0.03 }
}
```

Whether a built-in variable can sit directly on the modifier line (`army_attack_factor = num_subjects`) was not tested here; the wiki's examples all bind custom variables, so this page always copies into its own variable first.

### 7.2 When you have to test it yourself, the loop has one shape

If there is no built-in number but a single trigger can decide it, write a loop. The shape is always: pick what to walk, write the test, `ROOT = { add_to_variable }`.

| Effect you want | Loop | Test |
| --- | --- | --- |
| For every neighbour at war | `every_neighbor_country` | `has_war = yes` |
| For every fascist country | `every_other_country` | `has_government = fascism` |
| For every owned state with resistance above 20 | `every_owned_state` | `resistance > 20` |
| For every core state controlled by someone else (this page's example) | `every_state` | `is_core_of = ROOT` plus `NOT = { is_controlled_by = ROOT }` |

### 7.3 When the fold is not "plus one per item"

Steps, ratios and two-way scaling all live in the last few lines; the loop does not change.

Ratio: lost states as a share of core states, minus 5% attack per 10%:

```
set_variable = { my_thorn_attack = my_thorn_states }
divide_variable = { my_thorn_attack = num_core_states }
multiply_variable = { my_thorn_attack = -0.5 }
```

Step: only states beyond the fifth count:

```
subtract_from_variable = { my_thorn_states = 5 }
clamp_variable = { var = my_thorn_states min = 0 }
set_variable = { my_thorn_attack = my_thorn_states }
multiply_variable = { my_thorn_attack = 0.03 }
```

## 8. Showing the change in a focus or event

The spirit in section two is a "recount every day" spirit: the recount effect owns the variable, and focuses and events should not touch it, since midnight would compute it back anyway. There is a second kind of dynamic modifier, the "accumulated" one: no recount, each focus adds a little to the variable. The base game's Afghan cavalry modifier is written that way. There the player sees nothing in the focus tooltip, because changing a variable produces no tooltip of its own.

The fix is a `tooltip` argument on `add_to_variable`, pointing at a localisation key:

```
completion_reward = {
    add_to_variable = { my_cavalry_attack = 0.05 tooltip = my_cavalry_attack_tt }
}
```

```yaml
l_english:
 my_cavalry_attack_tt:0 "$MODIFIER_CAVALRY_ATTACK_FACTOR$: $RIGHT|+=0%$"
```

Two fixed tokens live in that text: `$RIGHT$` is the amount being added, `$LEFT$` the value before. The format characters after the pipe are the same as in section five: `+` green when positive and red when negative, `=` a sign, `%` times a hundred with a percent sign, `0` no decimals. `$MODIFIER_CAVALRY_ATTACK_FACTOR$` borrows the engine's own name for the modifier, so your line reads like the rest of the game. These keys follow no single rule: cavalry attack is `MODIFIER_CAVALRY_ATTACK_FACTOR`, while Division Attack is `MODIFIERS_ARMY_ATTACK_FACTOR`, with an extra S. Search the base game's `localisation/` folder for the modifier name before you write one; if there is none, write your own words.

The base game uses this argument on `add_to_variable` more than two thousand times; `set_variable` and `subtract_from_variable` take it too.

Back to the recount spirit of section two: a focus must not touch the count, but it may change "how much per state" by turning 0.03 into a variable. The last step of the recount multiplies by the variable, the focus adds to it with a tooltip, and the country history file initialises one more line:

```
multiply_variable = { my_thorn_attack = my_thorn_rate }
```

```
add_to_variable = { my_thorn_rate = 0.02 tooltip = my_thorn_rate_tt }
```

```yaml
l_english:
 my_thorn_rate_tt:0 "Division Attack per lost state: $RIGHT|+=0%$"
```

The player reads "Division Attack per lost state: +2%" on the focus, and the total in the spirit follows the next day.

## Common mistakes

- **A variable name inside an idea**: silently invalid, no error, the spirit looks attached and the value is 0.
- **TAG not replaced in the hook name**: `on_daily_MYT` runs for MYT only; if your country is not MYT the recount never runs.
- **Not initialising the variable**: the value is empty on day one; the recount creates it later, but checks that relied on `has_variable` have already gone wrong.
- **Counting into the wrong scope**: no `ROOT = { }` inside the loop, the count lands on the state or the foreign country and your variable stays 0.
- **Expecting explanation lines to be generated**: the engine renders only the total; each rule's text goes in the key `custom_modifier_tooltip` points at.
- **Recounting from the global `on_daily`**: every country runs the loop every day and the game visibly slows. Use `on_daily_<TAG>` or an event hook.
- **Illegal characters in a localisation key**: keys may only use letters, digits, underscore, dot, hyphen and apostrophe; one bad character drops every entry after it in the file.

## Want a different approach?

- **A fixed number that never moves**: use an idea; far simpler.
- **A number that follows something the engine already tracks, such as stability**: no counting needed, see section 7.1.
- **The AI should understand the modifier too**: the AI reads the values, not the explanation lines; those are for the player.
- **Per state rather than per country**: dynamic modifiers can be attached to states with the `scope` argument of `add_dynamic_modifier`; with `attacker_modifier = yes` they apply to divisions attacking that state.
