---
contentId: custom-character-traits
kind: case
slug: custom-character-traits
locale: en
status: published
title: "A custom trait for a character"
summary: "Leader traits and commander traits are two different file shapes: a leader trait lists its modifiers directly, a commander trait wraps them in a block and declares which branch it applies to. A trait without a name shows its raw key on the character card."
primaryTopic: people
topics: [people]
scenarios: [traits]
gameTerms: [trait, leader trait, commander trait, character card, modifier, scientist trait]
aliases: [custom trait, add a trait, trait shows raw key, trait has no name, how to write a leader trait, how to write a commander trait, leader trait, unit leader trait]
relatedIds: [character-system, leader-succession, political-power, "wiki:modifier:political_power_factor", "wiki:modifier:max_dig_in"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Traits"
    url: "https://hoi4.paradoxwikis.com/Traits"
  - label: "HOI4 Wiki: Modifiers"
    url: "https://hoi4.paradoxwikis.com/Modifiers"
  - label: "HOI4 Wiki: Localisation"
    url: "https://hoi4.paradoxwikis.com/Localisation"
---

# A custom trait for a character

## Goal and observable result

When this page is done you should see two things in game:

1. A new trait on the leader card, with a proper name, whose modifiers show on hover.
2. A new trait in a general's trait row, with a proper name, whose modifiers show on hover.

If the trait is there but its name is a string of lowercase letters and underscores, the localisation is not connected; section 4 covers exactly that.

## What you need before starting

- An existing leader character and an existing general character to attach the traits to.
- No DLC is required. Scientist traits have a separate restriction, see common mistakes.
- Read authoring a character (`character-system`) first.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| Leader trait definitions | `common/country_leader/` | Shared by leaders and advisors |
| Commander trait definitions | `common/unit_leader/` | Used by generals, field marshals and admirals |
| Character | `common/characters/` | References the trait inside a role block's `traits` |
| Name and description | `localisation/` | Whether the card shows a name or a raw key |

In the workbench both kinds are created in the "manage custom traits" panel, as a custom country_leader trait or a custom unit_leader trait; the character editor then lists only the kind compatible with the role type.

## 1. Leader traits: modifiers listed directly under the trait

A leader trait file is flat: one `leader_traits = { ... }` wraps every trait, and each trait lists its modifier keys directly:

```
leader_traits = {
    my_leader_trait = {
        random = no
        political_power_factor = 0.5
        stability_factor = 0.1
    }
}
```

`random = no` is the base game's convention for traits of named characters; it keeps the trait out of the pool for randomly generated leaders. The modifier keys are ordinary country modifiers: `political_power_factor = 0.5` is +50% daily political power, `stability_factor = 0.1` is +10% stability. There is **no** `modifier = { ... }` layer here, which is exactly the opposite of the commander trait in the next section.

In the workbench: "manage custom traits" → "new trait" → pick the country_leader kind → fill in the display name → enter keys and values under modifiers. On export the exporter writes the shape above.

## 2. Commander traits: modifiers in a block, plus a branch declaration

A commander trait has two extra things: `type` says which kind of commander it applies to, and the modifiers go inside a `modifier` block:

```
leader_traits = {
    my_commander_trait = {
        type = land
        trait_type = personality_trait
        modifier = {
            max_dig_in = 2
        }
    }
}
```

`type = land` means army commanders; navy uses `navy`, and `type = { land navy }` covers both. `trait_type = personality_trait` is the base game's category for traits that a person simply has rather than learns through experience. `max_dig_in = 2` is +2 to maximum entrenchment, visible when hovering the general's trait.

In the workbench: "new trait" → pick the unit_leader kind → display name → modifiers. The advanced fields (tree position `gui_row` and `gui_column`, prerequisites, `cost`) matter only if you want players to learn the trait with experience; a trait attached directly to a character does not use them.

## 3. Attach the trait to a character

Reference the trait ID inside the role block:

```
country_leader = {
    ideology = conservatism
    traits = { my_leader_trait }
    expire = "1965.1.1.1"
    id = -1
}
```

```
corps_commander = {
    traits = { my_commander_trait }
    skill = 3
    attack_skill = 3
    defense_skill = 2
    planning_skill = 2
    logistics_skill = 2
    legacy_id = -1
}
```

In the workbench, open the character editor and the traits section lists your custom traits; tick the one you want. Leader traits only appear for leaders and advisors, commander traits only for generals, field marshals and admirals; there is nothing to sort by hand.

## 4. Name it: an unnamed trait shows its raw key

A trait's localisation key is its ID, with `_desc` appended for the description:

```
l_english:
 my_leader_trait: "Reformer"
 my_leader_trait_desc: "A leader pushing for reform."
 my_commander_trait: "Fortification Expert"
 my_commander_trait_desc: "Skilled at preparing positions."
```

🔴 Skipping this step has been tested: the trait still works and its modifiers still apply, but the leader card shows the string `my_leader_trait` as the name. There is no error. The workbench's display-name field needs at least one language; leave both empty and this is what gets exported.

## 5. Check each item

| Check | Expected | If not, look at |
| --- | --- | --- |
| The trait on the leader card | Shows a proper name | Language folder and key spelling in the localisation file |
| Hover the leader trait | +50% daily political power, +10% stability | Whether the modifiers sit directly under the trait, without an extra wrapping block |
| The general's trait row | Shows a proper name | Same as above |
| Hover the general's trait | Maximum entrenchment +2 | Whether the modifiers are inside `modifier`; whether `type` matches |
| The trait does not appear at all | It should | The ID inside the role block's `traits`; the workbench drops unknown IDs before export, with a warning |

## Common mistakes

- 🔴 **No name**: the trait works but shows its raw key. Add the localisation.
- **Writing a commander trait in the flat leader-trait shape**: commander modifiers must be inside a `modifier` block. This follows the base-game file shape; the effect of getting it backwards was not tested for this page.
- **Giving a scientist trait a regular research modifier**: the first comment line of the base game's scientist trait file states that scientist traits only apply to special projects and may only use modifiers that apply to those. `research_speed_factor` and the like do nothing there.
- **A misspelled trait ID**: the workbench drops any ID that is in neither the base-game table nor your custom traits before export, with a warning; the character simply loses that trait.

## Want a different effect?

- A trait that only applies toward one country (an "affinity with Britain" trait): leader traits support `targeted_modifier`, called a targeted country modifier in the workbench. It is a different thing from an opinion modifier, which is a separate definition.
- A trait that adds daily political power: see the distinction between the flat and the proportional item on political power: rewards, thresholds and daily gain (article `political-power`).
- Adding a trait to the sitting leader during the story: use add a country leader trait (`wiki:effect:add_country_leader_trait`); the trait itself is still defined as on this page.
