---
contentId: character-system
kind: guide
slug: character-system
locale: en
status: published
title: "Authoring a character: what leaders, commanders and advisors each control"
summary: "A character is one record plus one or more role blocks: a leader is defined by a sub-ideology, a commander by skills, an advisor by a slot and conditions. Several roles go on the same person; do not split them into several people."
primaryTopic: people
topics: [people]
scenarios: [history, traits]
gameTerms: [character, country leader, general, field marshal, admiral, advisor, cabinet, portrait, trait, scientist, operative]
aliases: [how to add a character, how to make a leader, add a general, add an advisor, character not showing, advisor missing, one person two roles, character, advisor]
relatedIds: [advisor-unlocked-by-focus, leader-succession, custom-character-traits, "wiki:effect:recruit_character", "wiki:effect:promote_character", "wiki:effect:retire_country_leader"]
updatedAt: "2026-09-10"
verification: static-checked
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Character modding"
    url: "https://hoi4.paradoxwikis.com/Character_modding"
  - label: "HOI4 Wiki: Traits"
    url: "https://hoi4.paradoxwikis.com/Traits"
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
---

# Authoring a character: what leaders, commanders and advisors each control

## What this system covers

The face on the country selection screen, the advisors you can hire into the cabinet and the generals you can assign in the field are all the same kind of thing in script: a **character**. A character is one record (name, portraits) plus one or more **role blocks**. The roles decide which screens the person appears on and what the player can do with them.

As an author you decide three things:

1. **Who the person is**: name, portraits, which country.
2. **Which roles they hold**: country leader, general or field marshal, admiral, advisor, scientist, operative. It can be more than one.
3. **The parameters of each role**: a leader has a sub-ideology and traits, a commander has skills and traits, an advisor has a slot, a cost and appearance conditions.

In the workbench those map to the basic-information section, the role type plus the "also holds an advisor role" toggle, and each role's own settings area.

## Core idea: one person can hold several roles, but is still one character

This is the easiest thing to get wrong in the character system, and getting it wrong produces no warning at all.

The base game's Guderian is both a recruitable general and a hireable military theorist. In script he has **one record**, with a `corps_commander` block and an `advisor` block hanging off it. In game he shows up in the general list and in the theorist candidates, and he is the same person.

If you instead create **two characters** with the same name to get "general and advisor", the game shows **two people**: one only in the general list, one only in the cabinet candidates, each with its own portrait, neither aware of the other. No error, nothing in the log.

| Role in game | Script block | How to pick it in the workbench |
| --- | --- | --- |
| Country leader | `country_leader` | Role type: country leader, then fill in the sub-ideology |
| General / field marshal | `corps_commander` / `field_marshal` | Role type: general or field marshal, fill in army skills |
| Admiral | `navy_leader` | Role type: admiral, fill in navy skills |
| Advisor | `advisor` | Role type: advisor; or tick "also holds an advisor role" on another role |
| Scientist | `scientist` | Role type: scientist, fill in specialisations and levels |
| Operative | `operative` | Role type: operative, fill in the nationality list |

The workbench lists only the traits compatible with the role type you picked, so you never have to work out which trait file belongs to which role.

## File structure and how it runs

Characters live under `common/characters/`, wrapped in a single `characters = { ... }` block with one entry per person:

```
characters = {
    my_leader = {
        name = my_leader
        portraits = {
            civilian = {
                large = GFX_portrait_my_leader
                small = GFX_portrait_my_leader_small
            }
        }
        country_leader = {
            ideology = conservatism
            traits = { my_leader_trait }
            expire = "1965.1.1.1"
            id = -1
        }
    }
}
```

`name` takes a localisation key, not display text; the displayed name goes in the matching language YML under `localisation/`. `portraits` has `civilian`, `army` and `navy` layers: political roles use `civilian`, land commanders use `army`, admirals use `navy`. `small` is the icon used in the cabinet and advisor lists.

A person with several roles gets several blocks. This is "general who is also chief of army", both blocks under one record:

```
my_general = {
    name = my_general
    portraits = {
        army = {
            large = GFX_portrait_my_general
            small = GFX_idea_my_general
        }
    }
    corps_commander = {
        traits = { infantry_officer }
        skill = 3
        attack_skill = 3
        defense_skill = 2
        planning_skill = 2
        logistics_skill = 2
        legacy_id = -1
    }
    advisor = {
        slot = army_chief
        idea_token = my_general_army_chief
        ledger = army
        allowed = { original_tag = MYT }
        traits = { army_chief_offensive_1 }
        cost = 100
    }
}
```

**Defined is not the same as present.** For a character to actually belong to a country it must be recruited in that country's history file (the one under `history/countries/`):

```
recruit_character = my_leader
recruit_character = my_general
```

This line is only allowed in history files. Put it in a startup hook under `common/on_actions/` and the engine refuses it, stating outright that it may only appear in history files. Characters created in the workbench get this line added automatically on export ("new character: recruit_character is added at startup"); the ones marked "replaces an existing vanilla character" do not, because the base game's history file already recruited them.

When exporting an advisor block Hagane Works also fills in three things the base game always has but the form does not show: `allowed = { original_tag = <your country> }`, a generated `idea_token`, and a default `cost` of 150.

## Common goals and where to start

| What you want to see in game | Start here | The distinction that matters |
| --- | --- | --- |
| My leader is in charge from day one | Role type: country leader, sub-ideology matching the ruling party | If a base-game leader of the same ideology exists, the base-game one still starts in charge; see below |
| One more recruitable general | Role type: general, fill in skills | Leave the visibility condition empty and he is always recruitable |
| One more hireable advisor | Role type: advisor, pick a slot and advisor traits | Recruited in the history file like any other role; the workbench adds it |
| One person who is both a general and a chief | General plus "also holds an advisor role" | Do **not** create a second character |
| An advisor who only appears after a focus | Put `has_completed_focus` in the advisor's visibility condition | The availability condition only controls hireability, not visibility; see an advisor unlocked by a focus (`advisor-unlocked-by-focus`) |
| Swap leaders during the story | Use an event with promote a character (`wiki:effect:promote_character`) or retire the current leader (`wiki:effect:retire_country_leader`) | A pre-made second leader never takes over on his own; see changing the leader (`leader-succession`) |
| A trait the base game does not have | "Manage custom traits" | See a custom trait for a character (`custom-character-traits`) |
| An operative | Role type: operative, fill in nationalities | Needs La Résistance; the nationality list decides which countries can recruit him |
| A scientist | Role type: scientist, fill in specialisations | Needs Götterdämmerung; scientist traits only affect special projects |

## Easy things to get wrong

### 🔴 One person with two roles is one character, not two

The mechanism is above; here is how to tell. In game, look the name up once in the general list and once in the cabinet candidates. If the two portraits are **not the same picture**, and dismissing one side changes nothing on the other, you built two people. Delete one and tick "also holds an advisor role" on the survivor.

### 🔴 A leader needs a sub-ideology, or the person never appears as a leader at all

`ideology` inside `country_leader` takes a **sub-ideology** (`conservatism`, `liberalism`, `nazism`, `stalinism` and so on), not a group such as `democratic` or `fascism`. Fill in a group, or nothing, and the engine logs two lines at load: first "is not a valid ideology", then "No political party for country leader". The person's other roles are unaffected; only the leader role is void.

When the sub-ideology field is empty the workbench **does not emit the leader block at all**, so the person is exported as a record with no role. That is not an export error; you only find out when you cannot find him in game.

### 🔴 A missing portrait raises no error; you get a generic face

When `portraits` points at a sprite that does not exist, the game **writes nothing to the log** and shows the person with a generic portrait. The widely repeated claim that error.log reports "Could not find portrait" does not hold on 1.19.2. Portraits must be checked by eye: load the game and look at every character you made. A generic face means the image was not found.

When no portrait has been uploaded the exporter writes **no `portraits` block**; how the game displays that case was not tested for this page.

### 🔴 An advisor's "on hire" effect runs in character scope

The `on_add` and `on_remove` blocks of an advisor do not run in country scope. Write a country effect such as `add_stability = 0.05` directly and the engine rejects it at load ("Invalid scope type for effect add_stability"); nothing happens on hire. More than a hundred base-game hire effects almost all use `owner = { ... }` to step back to the country first. The workbench tooltip for the on-hire field shows the bare form, and **filling it in that way does nothing**; use the wrapped form from section 3 of the advisor case (`advisor-unlocked-by-focus`).

The advisor's `modifier = { ... }` block, by contrast, is a lasting modifier on the country: active while hired, gone when dismissed. It is a different thing from the hire effect.

### When a base-game leader of the same ideology exists, he still starts in charge

Give a base-game country a second leader of the same ideology and the base-game leader **is still the one in power at game start**; yours is a candidate. To put him in charge you either `promote_character` in the history file or do it through an event. The full recipe is changing the leader: standby candidates, succession by event and replacing a base-game leader (`leader-succession`).

### Scientist traits only affect special projects

The first comment line of the base game's scientist trait file says so: scientist traits apply only to special projects, so only modifiers that apply to those may be used. A regular research modifier such as `research_speed_factor` on a scientist trait does nothing. The workbench's custom trait panel carries the same note for scientist traits.

### An operative's nationality list decides who can recruit him

`nationalities` in the `operative` block lists the countries that may recruit this operative; empty means any country. This comes from source material; operatives were not tested in game for this page.

### Version and scope

This page was compiled against the base-game scripts and bundled documentation labelled 1.19.2.0. Operatives need La Résistance and scientists need Götterdämmerung; without the matching DLC those two roles do not appear.

## Where to go next

- What each of the four advisor script blocks does, and the one form the engine rejects: an advisor who only appears after a focus completes (`advisor-unlocked-by-focus`).
- Why a standby leader does not take over, how to swap leaders in an event, how to replace a base-game leader: changing the leader (`leader-succession`).
- The file shapes for leader and commander traits, and what an unnamed trait looks like: a custom trait for a character (`custom-character-traits`).
- To pay out political power when an advisor is hired, or have a leader trait add daily political power, see political power: rewards, thresholds and daily gain (article `political-power`).
