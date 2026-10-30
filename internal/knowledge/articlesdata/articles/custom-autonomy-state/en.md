---
contentId: custom-autonomy-state
kind: case
slug: custom-autonomy-state
locale: en
status: published
title: "A custom autonomy level for a subject"
summary: "Add a subject level of your own beside the base game's puppet and dominion, then set a country to it from a focus or event; the definition file's shape, how to choose the freedom floor, and the two spellings of set_autonomy. Copying the skeleton of the base game's puppet.txt and changing the numbers is the safe route."
primaryTopic: diplomacy
topics: [diplomacy, politics]
scenarios: [focus, event, history]
gameTerms: [subject, puppet, autonomy, autonomy level, freedom, overlord, focus, completion reward]
aliases: [custom subject level, custom puppet, new autonomy level, subject freedom, make a puppet, set_autonomy, autonomy_state, autonomous_state, has_autonomy_state]
relatedIds: [diplomacy-system, focus-create-faction, focus-wargoal-annex, "wiki:effect:set_autonomy", "wiki:effect:end_puppet", "wiki:trigger:is_subject_of", "wiki:trigger:is_puppet_of", "wiki:trigger:has_autonomy_state", "wiki:modifier:autonomy_manpower_share"]
updatedAt: "2026-09-15"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki: Triggers"
    url: "https://hoi4.paradoxwikis.com/Triggers"
  - label: "HOI4 Wiki: Modifiers"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
  - label: "HOI4 Wiki: Autonomy state modding"
    url: "https://hoi4.paradoxwikis.com/Autonomy_state_modding"
---

# A custom autonomy level for a subject

## Goal and observable result

When this page is done you should see three things in game:

1. The moment the focus completes, the Netherlands becomes your subject, and the level name on the diplomacy panel is the name you chose, not "Puppet".
2. The autonomy bar on the subject screen sits where your chosen freedom puts it, and the level's icon is the one you defined, not a question mark.
3. A follow-up focus can only be started while "the Netherlands is at this level".

## What you need before starting

- A mod that already exports correctly, and an existing focus. This page uses the placeholder ID `my_focus_vassalise_holland`.
- An ID and a name for the level; this page uses `my_autonomy_protectorate`.
- The target's tag; this page uses the Netherlands, `HOL`, played as Luxembourg.
- No DLC is required. Half of the base game's 21 levels belong to DLCs, but a custom level does not.
- Read diplomacy: where starting relations, runtime effects and definitions each live (`diplomacy-system`) first, so that the level is understood as a definition and `set_autonomy` as the effect.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| Level definition | `common/autonomous_states/`, one file per level | the level itself: floor, manpower, rules, modifiers |
| Focus | `common/national_focus/` | `set_autonomy` in its `completion_reward` |
| Name | `localisation/` | the key is the level's `id` |
| Icon | `interface/*.gfx` | the sprite name is fixed as `GFX_<id>_icon`; without it a question mark shows |

## 1. Define the level

Create `common/autonomous_states/my_protectorate.txt`. The skeleton follows the base game's `puppet.txt`; only the numbers change:

```
autonomy_state = {
    id = my_autonomy_protectorate

    is_puppet = yes

    min_freedom_level = 0.55

    manpower_influence = 0.5

    rule = {
        desc = "RULE_DESC_IS_A_SUBJECT"
        can_not_declare_war = yes
        can_decline_call_to_war = no
    }

    modifier = {
        autonomy_manpower_share = 0.5
        extra_trade_to_overlord_factor = 0.5
    }

    ai_subject_wants_higher = {
        factor = 1.0
    }

    ai_overlord_wants_lower = {
        factor = 1.0
    }

    allowed = {
        always = yes
    }

    can_take_level = {
    }

    can_lose_level = {
    }
}
```

Field by field:

| Field | Meaning | Base-game puppet value |
| --- | --- | --- |
| `id` | the level's name, used by `set_autonomy` and `has_autonomy_state` | `autonomy_puppet` |
| `is_puppet` | whether it counts as a puppet (`is_puppet_of` is true) | `yes` |
| `min_freedom_level` | freedom floor, 0 to 1; a subject whose freedom climbs past the next level's floor moves up | `0.4` |
| `manpower_influence` | share of manpower the overlord may draw | `0.9` |
| `rule` | game rules imposed on the subject, one per line | cannot declare war, cannot decline calls to war |
| `modifier` | lasting modifiers on the subject | manpower share 0.9 and others |
| `ai_subject_wants_higher` / `ai_overlord_wants_lower` | AI weights for pushing up / holding down | 1.0 each |
| `allowed` | when this level is available at all | base game gates by DLC and ideology; a custom level writes `always = yes` |
| `can_take_level` / `can_lose_level` | extra conditions to rise to / fall from this level | empty |

`default = yes` appears only on the base-game puppet level: the "puppet" option in peace conferences and the `puppet` effect prefer a level carrying it. Do not write it on a custom level, or it competes with the base game's default.

Give different levels different `min_freedom_level` values: the engine orders levels by floor to decide promotions and demotions, and the autonomy points needed to move one level are the difference between neighbouring floors times 5000, so two equal floors can neither be ordered nor measured. The base game's 21 levels all differ.

One optional field is `peace_conference_initial_freedom`: the starting freedom when a peace conference puts a country at this level; 0.5 if omitted.

## 2. Name it

Under `localisation/english/`:

```
l_english:
 my_autonomy_protectorate:0 "Protectorate"
```

The key is the `id`. Base-game pairs look like `autonomy_puppet: "Puppet"`, `autonomy_dominion: "Dominion"`. Without the line, the interface shows the key itself.

## 3. Give it an icon

The icon is not part of the definition file; the engine looks up a sprite by a fixed name: `GFX_` plus the level's `id` plus `_icon`. The base-game puppet level uses `GFX_autonomy_puppet_icon`, defined in `interface/countrypoliticsview.gfx` with its texture under `gfx/interface/autonomy/`. For your own level, create a new `.gfx` file under `interface/`:

```
spriteTypes = {
    spriteType = {
        name = "GFX_my_autonomy_protectorate_icon"
        textureFile = "gfx/interface/autonomy/my_autonomy_protectorate_icon.dds"
    }
}
```

Without this sprite the level still works; the subject screen just shows a question mark beside the level name. That is exactly what the in-game test showed: the custom name rendered, the icon slot was a question mark.

## 4. Set it from a focus

```
focus = {
    id = my_focus_vassalise_holland
    icon = GFX_goal_generic_major_alliance
    x = 0
    y = 0
    cost = 10
    completion_reward = {
        set_autonomy = {
            target = HOL
            autonomy_state = my_autonomy_protectorate
            freedom_level = 0.6
        }
    }
}
```

`freedom_level` is the starting freedom; write it above the floor. The subject screen does not show this decimal; it shows the autonomy bar in points (in the test, 0.6 displayed as 150/250). What a value below the floor does was not tested for this page. The game's documentation also lists two switches, `end_wars` and `end_civil_wars`, both defaulting to `yes`: becoming a subject ends the country's wars.

### Two spellings

The form above, `autonomy_state`, follows the game's documentation and is what base-game focuses and events write 540 times. Base-game history files instead write `autonomous_state` (23 uses), and so does Hagane Works' history-file exporter. The engine accepts both; follow the convention of the kind of file you are in, and do not mix them.

## 5. A follow-up focus conditioned on the level

```
available = {
    HOL = {
        has_autonomy_state = my_autonomy_protectorate
    }
}
```

The value of `has_autonomy_state` is the level's `id`. To check only "is it my subject", use `HOL = { is_subject_of = ROOT }`; to check only "is it a puppet", use `is_puppet_of`, which looks at `is_puppet` in the level definition.

## 6. Use an event or the history file instead

Event: the same `set_autonomy` block in an option. History file: at the top level, spelled `autonomous_state`, as in the base game's Belgian history file for its colony:

```
set_autonomy = {
    target = COG
    autonomous_state = autonomy_colony
    freedom_level = 0.2
}
```

## 7. Check each item

| Check | Expected | If not, look at |
| --- | --- | --- |
| The Netherlands' diplomacy panel after the focus | your subject, shown as "Luxembourgish Netherlands", level named "Protectorate" | whether `id` and the localisation key match; `error.log` |
| The icon beside the level name | your icon | whether the sprite is named `GFX_<id>_icon` |
| The autonomy bar | positioned for 0.6, not below 0.55 | `freedom_level` against `min_freedom_level` |
| `is_puppet_of` style triggers | true | `is_puppet = yes` in the definition |
| Follow-up focus | startable while at this level | whether the `has_autonomy_state` value is the `id` |
| Promotion and demotion | level changes as freedom crosses a neighbouring floor | whether floors are all distinct |

## Common mistakes

- 🔴 **A `target` tag that does not exist**: in the test, `error.log` records `_set_autonomy_: invalid target tag ---` and the effect does nothing. The Netherlands is `HOL`, not `NED`; tags come from the base game's `common/country_tags/`.
- 🔴 **No icon**: the level works, the screen shows a question mark. Add the `GFX_<id>_icon` sprite.
- **`set_autonomy` naming an undefined level**: confirm the file is under `common/autonomous_states/` and the `id` matches; what the log says in that case was not tested.
- 🔴 **`default = yes` on a custom level**: it competes with the base-game puppet level for the default slot.
- **Two levels with the same `min_freedom_level`**: undefined promotion order.
- **A localisation key that is not the `id`**: the key must equal the `id`.
- **`autonomy_state` in a history file and `autonomous_state` in a focus**: both run, but it is the reverse of base-game habit and confuses the next reader.

## In Hagane Works

The autonomy states workbench maps directly onto `common/autonomous_states/`: ID, `is_puppet`, `default`, `min_freedom_level` (stored as an integer from 0 to 1000 and divided by 1000 on export), `manpower_influence`, the rule block, modifier fields and AI weight blocks. `set_autonomy` in a focus is written as-is in the focus workbench's completion reward; a relation that exists from day one goes through the history-file workbench.

## Want a different effect?

- Release the subject: `end_puppet = HOL`, 36 uses in base-game events.
- Use a base-game level directly: `autonomy_state = autonomy_puppet`, `autonomy_dominion`, `autonomy_colony` and so on; the 21 `id`s are in the files under `common/autonomous_states/`.
- Make the subject send troops or manpower: adjust modifiers such as `autonomy_manpower_share` in the `modifier` block.
