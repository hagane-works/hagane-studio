---
contentId: leader-succession
kind: case
slug: leader-succession
locale: en
status: published
title: "Changing the leader: standby candidates, succession by event and replacing a base-game leader"
summary: "A second leader made in advance never takes over at game start; an event has to promote him or retire the incumbent, and promoting someone who is no longer there does nothing at all. Replacing a base-game leader takes a different route, and Hagane Works generates that script for you."
primaryTopic: people
topics: [people, politics]
scenarios: [event, history]
gameTerms: [country leader, succession, change leader, retire, leader death, ruling party, standby leader]
aliases: [swap the leader, leader does not take over, second leader does nothing, change leader by event, retire leader, replace vanilla leader, promote_character, retire_country_leader, create_country_leader]
relatedIds: [character-system, advisor-unlocked-by-focus, "wiki:effect:promote_character", "wiki:effect:retire_country_leader", "wiki:effect:kill_country_leader", "wiki:effect:create_country_leader", "wiki:effect:retire_character"]
updatedAt: "2026-09-10"
verification: game-tested
gameVersion: "1.19.2.0"
sources:
  - label: "HOI4 Wiki: Character modding"
    url: "https://hoi4.paradoxwikis.com/Character_modding"
  - label: "HOI4 Wiki: Effects"
    url: "https://hoi4.paradoxwikis.com/Effects"
  - label: "HOI4 Wiki: Event modding"
    url: "https://hoi4.paradoxwikis.com/Event_modding"
---

# Changing the leader: standby candidates, succession by event and replacing a base-game leader

## Goal and observable result

When this page is done you should see three things in game:

1. At game start the original leader is still in charge; your standby candidate exists but is not in power.
2. After an event fires, the standby candidate becomes leader, and the leader card shows his name and traits.
3. After a second event removes him, the leader changes to the next in line within that party.

Item 1 is engine behaviour, not a mistake. Items 2 and 3 are each one effect. The last section covers a different route: replacing a base-game leader.

## What you need before starting

- A mod that already exports correctly.
- A standby leader character: role type country leader, sub-ideology matching the ruling party. This page uses the placeholder ID `my_leader_2`.
- Enough familiarity with the event workbench to write a script-triggered event, or to put `country_event` in a focus reward.
- No DLC is required.
- Read authoring a character (`character-system`) first.

## Objects involved

| Object | Location | Role in this example |
| --- | --- | --- |
| Standby leader | `common/characters/` | One record with one `country_leader` block |
| Recruitment | the country's history file under `history/countries/` | One `recruit_character` line, added by the workbench |
| Events | `events/` | Carry the promote, retire and create effects |
| Name | `localisation/` | The name shown on the leader card |

## 1. Make the standby leader: he exists, but does not start in charge

Build him like any leader:

```
characters = {
    my_leader_2 = {
        name = my_leader_2
        portraits = {
            civilian = {
                large = GFX_portrait_my_leader_2
                small = GFX_portrait_my_leader_2_small
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

Load the game and you will find that **the original leader is still in power at start**; the person you made is just one of that party's candidates. This is the tested result, not a misconfiguration. A country may have several leader characters of the same ideology; which one holds office is decided by the history file and by events.

So "make a second leader in advance and wait for him to take over" does nothing by itself. He needs an effect that puts him there.

## 2. Promote him with an event

Promote a character (`wiki:effect:promote_character`) moves a character to the head of his party; if that is the ruling party, he becomes country leader. Written into an event option:

```
add_namespace = my_events

country_event = {
    id = my_events.1
    title = my_events.1.t
    desc = my_events.1.d
    is_triggered_only = yes
    option = {
        name = my_events.1.a
        promote_character = my_leader_2
    }
}
```

`is_triggered_only = yes` means the event fires only from script. Put `country_event = my_events.1` in a focus's completion reward, or in a decision; when the focus completes the event pops up and choosing the option swaps the leader. For testing, fire it from the console with `event my_events.1`.

Tested: the event appears, and on confirming, the leader card immediately changes to `my_leader_2`, traits included.

## 3. Removing the incumbent: retire or kill

Both effects replace the sitting leader; the difference is whether the person is still around afterwards:

| Effect | What happens | Base-game usage |
| --- | --- | --- |
| Retire the current leader (`wiki:effect:retire_country_leader`) `retire_country_leader = yes` | Steps down, no longer heads the party; the next in line takes over | 157 places |
| Kill the current leader (`wiki:effect:kill_country_leader`) `kill_country_leader = yes` | Dies and is **removed from the country**; the next in line takes over | 48 places |

In both cases the next in line is another leader character of the same party. In the test, the standby leader was first promoted by event, then `kill_country_leader = yes` ran: the standby leader died and the leadership returned to the original leader. Firing the promotion event again afterwards produced **the pop-up as usual, but nothing happened, and nothing appeared in the log**. The person was gone; promoting someone who no longer exists is a silent no-op.

So the choice follows the story: if he may come back later, retire him; if he will never appear again, kill him. To remove a specific character who is not currently in office, use retire a character (`wiki:effect:retire_character`) `retire_character = my_leader_2`, which strips all of that person's roles at once.

## 4. Create a leader inside the event, with no character record

If a leader only appears once in one storyline, you can skip the character and create him inline:

```
option = {
    name = my_events.2.a
    create_country_leader = {
        name = "Interim Administrator"
        picture = GFX_portrait_my_leader_2
        expire = "1965.1.1"
        ideology = conservatism
    }
}
```

Tested: after the event the leader card shows this name. He has no character record, so `promote_character` and `retire_character` cannot find him by name later. The base game's 104 uses of this form are mostly "provisional government" and "transitional figure" cases that never need referencing again.

## 5. Replacing a base-game leader with generated script

Everything above adds a leader. Replacing a base-game leader is different: the base game already defines him under `common/characters/`, and defining another character with the same identifier makes the engine report a duplicate character tag.

The workbench's approach: on the character, fill in the "override vanilla character ID" field and leave "generate the trait overlay event on export" ticked. On export Hagane Works does **not** write this person into `common/characters/`; instead it generates a hidden event that runs on the first day, retires the base-game leader, then creates the new leader from what you filled in:

```
country_event = {
    id = my_overlay.1
    title = my_overlay.1.t
    desc = my_overlay.1.d
    hidden = yes
    is_triggered_only = yes
    immediate = {
        MYT = {
            retire_country_leader = yes
            create_country_leader = {
                name = my_leader_2
                picture = GFX_portrait_my_leader_2
                expire = "1965.1.1"
                ideology = conservatism
                traits = { my_leader_trait }
            }
        }
    }
}
```

The same retire-and-create pair is also written into the country's history file, so the country selection screen and the bookmark view show your leader rather than waiting for day one. The exporter generates this script; you only fill in the character in the workbench.

If you are not using Hagane Works, there are two hand-written options: the same retire-and-create pair in an event, or overriding the base game's whole `common/characters/<TAG>.txt` and deleting that person's block. Defining a second character with the same identifier reports `Multiple character have the tag`; that message was checked against the engine's own text, but triggering it was not tested for this page.

## 6. Check each item

| Check | Expected | If not, look at |
| --- | --- | --- |
| Leader card at game start | The original leader is in power | This is normal; do not assume the standby leader is broken |
| Fire the promotion event | The leader changes to the standby candidate, traits shown | The character ID in the event; whether the history file recruited him |
| Fire the retirement event | The leader changes to the party's next in line | Whether the party has another leader character |
| Fire the kill event, then promote the same person again | Pop-up appears, leader unchanged | This is normal; the person was removed |
| Fire the create event | A new leader with no character record appears | Whether `ideology` is a sub-ideology |
| After replacing a base-game leader, open country selection | Your leader is shown | Whether the history file has the retire-and-create pair |

## Common mistakes

- 🔴 **Making a second leader in advance and expecting him to take over**: he will not. A promote or retire effect is needed.
- 🔴 **Promoting a character who has died or left**: pop-up as usual, nothing happens, nothing in the log. If a story might kill someone and later promote him, retire him instead of killing him.
- **Using kill when you meant swap**: killing removes the person from the country; later events can never find him again.
- **Creating inline and then wanting to reference the leader by name**: a created leader has no character record, so `promote_character` and `retire_character` cannot find him.
- **Replacing a base-game leader and also writing a character with the same identifier**: the engine reports a duplicate tag. Use the workbench's override feature, or override the whole base-game file.

## Want a different effect?

- To change the sitting leader's ideology rather than the person: `set_country_leader_ideology`, not tested for this page.
- To add a trait to the sitting leader: add a country leader trait (`wiki:effect:add_country_leader_trait`).
- To have the trait show a proper name on the leader card instead of a raw key: see a custom trait for a character (`custom-character-traits`).
